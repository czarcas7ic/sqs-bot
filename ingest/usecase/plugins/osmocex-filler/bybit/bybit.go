package bybit

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/joho/godotenv"
	"github.com/osmosis-labs/osmosis/osmomath"
	"github.com/osmosis-labs/sqs/domain"
	"github.com/osmosis-labs/sqs/domain/mvc"
	"github.com/osmosis-labs/sqs/log"
	"go.uber.org/zap"

	bybit "github.com/wuhewuhe/bybit.go.api"

	wsbybit "github.com/hirokisan/bybit/v2"
	"github.com/osmosis-labs/sqs/domain/keyring"
	orderbookplugindomain "github.com/osmosis-labs/sqs/domain/orderbook/plugin"
	passthroughdomain "github.com/osmosis-labs/sqs/domain/passthrough"
	osmocexfillertypes "github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/types"
)

type BybitExchange struct {
	// arb config
	*arbitrageConfig

	ctx context.Context

	// uses websocket for orderbook data processing for performance
	wsclient wsbybit.V5WebsocketPublicServiceI
	// http client for trade calls
	httpclient *bybit.Client

	// map of pairs that bybit exchange is configured to arb against
	registeredPairs map[osmocexfillertypes.Pair]struct{}

	// upstream pointers
	osmoPoolIdToOrders        *sync.Map
	osmoPoolsUsecase          *mvc.PoolsUsecase
	osmoTokensUsecase         *mvc.TokensUsecase
	osmoRouterUsecase         *mvc.RouterUsecase
	osmoKeyring               *keyring.Keyring
	osmoPassthroughGRPCClient *passthroughdomain.PassthroughGRPCClient

	// bybit orderbooks: Symbol -> OrderbookData
	orderbooks sync.Map

	// blockUntilHeight is the height until which the exchange will not search for arbs
	// set after a trade is executed to wait until block inclusion
	blockUntilHeight uint64

	logger log.Logger
}

var _ osmocexfillertypes.ExchangeI = (*BybitExchange)(nil)

const (
	HTTP_URL                        = bybit.MAINNET
	DEFAULT_DEPTH                   = 50 // default depth for an orderbook
	DEFAULT_WAIT_BLOCKS_AFTER_TRADE = 10
	USDC_INTERCHAIN                 = "ibc/498A0751C798A0D9A389AA3691123DADA57DAA4FE165D5C75894505B876BA6E4"
)

func init() {
	// Init environmental variables from .env file
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("No caller information")
	}

	// Get the directory of the current file
	currentDir := filepath.Dir(currentFile)

	err := godotenv.Load(currentDir + "/.env")
	if err != nil {
		panic(err)
	}
}

func New(
	ctx context.Context,
	osmoPoolIdToOrders *sync.Map,
	osmoPoolsUsecase *mvc.PoolsUsecase,
	osmoTokensUsecase *mvc.TokensUsecase,
	osmoRouterUsecase *mvc.RouterUsecase,
	osmoKeyring *keyring.Keyring,
	osmoPassthroughGRPCClient *passthroughdomain.PassthroughGRPCClient,
	logger log.Logger,
) *BybitExchange {
	if os.Getenv("BYBIT_API_KEY") == "" || os.Getenv("BYBIT_API_SECRET") == "" {
		panic("BYBIT_API_KEY or BYBIT_API_SECRET not set")
	}

	wsclient := wsbybit.NewWebsocketClient().WithAuth(os.Getenv("BYBIT_API_KEY"), os.Getenv("BYBIT_API_SECRET"))
	svc, err := wsclient.V5().Public(wsbybit.CategoryV5Spot)
	if err != nil {
		panic(err)
	}

	httpclient := bybit.NewBybitHttpClient(os.Getenv("BYBIT_API_KEY"), os.Getenv("BYBIT_API_SECRET"), bybit.WithBaseURL(HTTP_URL))

	be := &BybitExchange{
		ctx:                       ctx,
		wsclient:                  svc,
		httpclient:                httpclient,
		registeredPairs:           make(map[osmocexfillertypes.Pair]struct{}),
		osmoPoolIdToOrders:        osmoPoolIdToOrders,
		osmoPoolsUsecase:          osmoPoolsUsecase,
		osmoTokensUsecase:         osmoTokensUsecase,
		osmoRouterUsecase:         osmoRouterUsecase,
		osmoKeyring:               osmoKeyring,
		osmoPassthroughGRPCClient: osmoPassthroughGRPCClient,
		blockUntilHeight:          0,
		logger:                    logger,
	}

	be.initConfig()

	return be
}

// RegisterPairs implements osmocexfillertypes.ExchangeI
func (be *BybitExchange) Signal(currentHeight uint64) {
	if be.blockUntilHeight != 0 && currentHeight < be.blockUntilHeight {
		be.logger.Info("arbitrage blocked until height", zap.Uint64("blockUntilHeight", be.blockUntilHeight))
		return
	}

	newBlockWg := sync.WaitGroup{}
	newBlockWg.Add(be.registeredPairsSize())

	for pair := range be.registeredPairs {
		go be.processPair(pair, currentHeight, &newBlockWg)
	}

	newBlockWg.Wait() // blocks until all orderbooks are processed for bybit block
}

// RegisterPairs implements osmocexfillertypes.ExchangeI
func (be *BybitExchange) RegisterPairs(ctx context.Context) error {
	for _, pair := range pairs {
		be.registeredPairs[pair] = struct{}{}
		if err := be.subscribeOrderbook(pair, DEFAULT_DEPTH); err != nil {
			return err
		}
	}

	go be.wsclient.Start(ctx, nil)

	return nil
}

/*
{
  "orders_by_tick": {
    "tick_id": 23749165
  }
}
*/

func (be *BybitExchange) GetBotBalances() (map[string]osmocexfillertypes.CoinBalanceI, sdk.Coins, error) {
	// get bybit balances
	bybitBalances, err := be.getBybitBalances()
	if err != nil {
		be.logger.Error("failed to get bybit balances", zap.Error(err))
		return nil, nil, err
	}

	// get osmo balancess
	osmoBalances, err := (*be.osmoPassthroughGRPCClient).AllBalances(be.ctx, (*be.osmoKeyring).GetAddress().String())
	if err != nil {
		be.logger.Error("failed to get osmo balances", zap.Error(err))
		return nil, nil, err
	}

	return bybitBalances, osmoBalances, nil
}

// processPair tries to find an orderbook profitable route between the two exchanges for a given pair
// returns true if trade happened
func (be *BybitExchange) processPair(pair osmocexfillertypes.Pair, currentHeight uint64, wg *sync.WaitGroup) {
	defer wg.Done()

	// get orderbooks from CEX and DEX
	bybitOrderbook, err := be.getBybitOrderbookForPair(pair)
	if err != nil {
		be.logger.Error("failed to get BYBIT orderbook for pair", zap.String("pair", pair.String()), zap.Error(err))
		return
	}

	// this is a hack to solve the issue described in QuoteBids docs
	bybitOrderbook = bybitOrderbook.QuoteBids()

	osmoOrderbook, err := be.getOsmoOrderbookForPair(pair)
	if err != nil {
		be.logger.Error("failed to get OSMOSIS orderbook for pair", zap.String("pair", pair.String()), zap.Error(err))
		return
	}

	osmoOrdersAny, ok := be.osmoPoolIdToOrders.Load(osmoOrderbook.PoolID)
	if !ok {
		be.logger.Error("failed to load osmo orders", zap.Uint64("poolID", osmoOrderbook.PoolID))
		return

	}

	osmoOrders := osmoOrdersAny.(orderbookplugindomain.OrdersResponse)

	var executedFromBybit, executedFromOsmo bool

	// check arb from bybit exchange to osmo
	// if true -> bybit.highestBid > osmo.lowestAsk
	// so, we need to buy from osmo and sell on bybit
	if be.existsArbFromBybit(pair, bybitOrderbook.BidsDescending(), osmoOrders.AsksAscending()) {
		executedFromBybit = be.processArbitrageFromBybit(pair, bybitOrderbook, osmoOrders, osmoOrderbook.PoolID)
	}

	// if true -> osmo.highestBid > bybit.lowestAsk
	// so, we need to buy from bybit and sell on osmo
	if be.existsArbFromOsmo(pair, bybitOrderbook.AsksAscending(), osmoOrders.BidsDescending()) {
		executedFromOsmo = be.processArbitrageFromOsmo(pair, bybitOrderbook, osmoOrders, osmoOrderbook.PoolID)
	}

	if executedFromBybit || executedFromOsmo {
		be.block(currentHeight)
	}
}

func (be *BybitExchange) processArbitrageFromOsmo(pair osmocexfillertypes.Pair, bybitOrderbook *osmocexfillertypes.OrderbookData, osmoOrders orderbookplugindomain.OrdersResponse, osmoOrderbookPoolId uint64) bool {
	// scale bybit orderbook
	baseDecimals, err := be.getInterchainDenomDecimals(pair.BaseInterchainDenom())
	if err != nil {
		be.logger.Error("failed to get base precision", zap.Error(err))
		return false
	}

	quoteDecimals, err := be.getInterchainDenomDecimals(pair.QuoteInterchainDenom())
	if err != nil {
		be.logger.Error("failed to get quote precision", zap.Error(err))
		return false
	}

	// operate on a copy
	bybitOrderbook = bybitOrderbook.ScaleSize(baseDecimals, quoteDecimals)

	// get total available fill amount
	fillAmountBase, fillAmountQuote, err := be.computeFillAmountsSkewed(pair, bybitOrderbook.AsksAscending(), osmoOrders.BidsDescending())
	if err != nil {
		be.logger.Error("failed to get fill amount and direction", zap.Error(err))
		return false
	}

	// see WARN in computeFillAmountsSkewed
	scaleBigDecDecimals(&fillAmountBase, baseDecimals-quoteDecimals)

	// get balances
	bybitBalances, osmoBalances, err := be.GetBotBalances()
	if err != nil {
		be.logger.Error("failed to get bot balances", zap.Error(err))
		return false
	}

	// get balances in big dec
	osmoBalanceBaseBigDec := osmomath.NewBigDecFromBigInt(osmoBalances.AmountOf(pair.BaseInterchainDenom()).BigInt())
	bybitBalanceQuoteBigDec := bybitBalances[pair.Quote].BigDecBalance(quoteDecimals)

	// get final fill amount and reversed fill amount
	fillAmountBase = be.adjustFillAmount(fillAmountBase, osmoBalanceBaseBigDec)
	fillAmountQuote = be.adjustFillAmount(fillAmountQuote, bybitBalanceQuoteBigDec)

	// check if fill amount is sufficient
	if !be.sufficientFillAmount(pair.BaseInterchainDenom(), fillAmountBase) {
		be.logger.Info("arbitrage from OSMOSIS found: insufficient fill value", zap.String("pair", pair.String()))
		return false
	}

	// fill bid on osmo (sell base on osmo)
	coinIn := sdk.NewCoin(pair.BaseInterchainDenom(), fillAmountBase.Dec().TruncateInt())

	wg := sync.WaitGroup{}
	wg.Add(2)

	// perform osmosis trade
	go func(coinIn sdk.Coin, quoteInterchainDenom string, osmoOrderbookPoolId uint64) {
		defer wg.Done()
		be.tradeOsmosis(coinIn, quoteInterchainDenom, osmoOrderbookPoolId)
	}(coinIn, pair.QuoteInterchainDenom(), osmoOrderbookPoolId)

	// fill ask on bybit (buy base on bybit)
	go func(pair osmocexfillertypes.Pair, fillAmountQuote osmomath.BigDec) {
		defer wg.Done()
		be.spot(pair, osmocexfillertypes.BUY, fillAmountQuote)
	}(pair, fillAmountQuote)

	wg.Wait()

	be.logger.Info("arbitrage from OSMOSIS: executed", zap.String("pair", pair.String()))

	return true
}

func (be *BybitExchange) processArbitrageFromBybit(pair osmocexfillertypes.Pair, bybitOrderbook *osmocexfillertypes.OrderbookData, osmoOrders orderbookplugindomain.OrdersResponse, osmoOrderbookPoolId uint64) bool {
	baseDecimals, err := be.getInterchainDenomDecimals(pair.BaseInterchainDenom())
	if err != nil {
		be.logger.Error("failed to get base precision", zap.Error(err))
		return false
	}

	quoteDecimals, err := be.getInterchainDenomDecimals(pair.QuoteInterchainDenom())
	if err != nil {
		be.logger.Error("failed to get quote precision", zap.Error(err))
		return false
	}

	// operate on a copy of an orderbook
	bybitOrderbook = bybitOrderbook.ScaleSize(baseDecimals, quoteDecimals)

	// get total available fill amount
	fillAmountBase, fillAmountQuote, err := be.computeFillAmountsSkewed(pair, bybitOrderbook.BidsDescending(), osmoOrders.AsksAscending())
	if err != nil {
		be.logger.Error("failed to get fill amount and direction", zap.Error(err))
		return false
	}

	// see WARN in computeFillAmountsSkewed
	scaleBigDecDecimals(&fillAmountQuote, quoteDecimals-baseDecimals)

	// get balances (do not move outside of for loop)
	bybitBalances, osmoBalances, err := be.GetBotBalances()
	if err != nil {
		be.logger.Error("failed to get bot balances", zap.Error(err))
		return false
	}

	// get balances in big dec
	osmoBalanceQuoteBigDec := osmomath.NewBigDecFromBigInt(osmoBalances.AmountOf(pair.QuoteInterchainDenom()).BigInt())
	bybitBalanceBaseBigDec := bybitBalances[pair.Base].BigDecBalance(baseDecimals)

	// get final fill amount and reversed fill amount
	fillAmountBase = be.adjustFillAmount(fillAmountBase, bybitBalanceBaseBigDec)
	fillAmountQuote = be.adjustFillAmount(fillAmountQuote, osmoBalanceQuoteBigDec)

	if !be.sufficientFillAmount(pair.BaseInterchainDenom(), fillAmountBase) {
		be.logger.Info("arbitrage from BYBIT found: insufficient fill value", zap.String("pair", pair.String()))
		return false
	}

	// fill ask on osmo (buy base on osmo)
	coinIn := sdk.NewCoin(pair.QuoteInterchainDenom(), fillAmountQuote.Dec().TruncateInt())

	wg := sync.WaitGroup{}
	wg.Add(2)

	// trade on osmosis
	go func(coinIn sdk.Coin, baseInterchainDenom string, osmoOrderbookPoolId uint64) {
		defer wg.Done()
		be.tradeOsmosis(coinIn, baseInterchainDenom, osmoOrderbookPoolId)
	}(coinIn, pair.BaseInterchainDenom(), osmoOrderbookPoolId)

	// fill bid on bybit (sell on bybit)
	go func(pair osmocexfillertypes.Pair, fillAmountBase osmomath.BigDec) {
		defer wg.Done()
		be.spot(pair, osmocexfillertypes.SELL, fillAmountBase)
	}(pair, fillAmountBase)

	wg.Wait()

	be.logger.Info("arbitrage from BYBIT: executed", zap.String("pair", pair.String()))

	return true
}

// sufficientFillAmount checks if the fill amount is valued at least the amount in the arbitrage config (default 10$)
// - interchainDenom must be in the interchain form
func (be *BybitExchange) sufficientFillAmount(interchainDenom string, fillAmount osmomath.BigDec) bool {
	if minFillAmount, ok := be.arbitrageConfig.getMinFillAmount(interchainDenom); ok {
		return fillAmount.GTE(minFillAmount)
	}

	price, err := (*be.osmoTokensUsecase).GetPrices(be.ctx, []string{interchainDenom}, []string{USDC_INTERCHAIN}, domain.ChainPricingSourceType)
	if err != nil {
		be.logger.Error("failed to get price", zap.Error(err))
		return false
	}

	decimals, err := be.getInterchainDenomDecimals(interchainDenom)
	if err != nil {
		be.logger.Error("failed to get interchain denom decimals", zap.Error(err))
		return false
	}

	minFillValue := osmomath.NewBigDec(MINIMUM_FILL_VALUE)
	addBigDecDecimals(&minFillValue, decimals)

	return fillAmount.Mul(price.GetPriceForDenom(interchainDenom, USDC_INTERCHAIN)).GTE(minFillValue)
}

// block prevents the exchange from searching for arbs until the blockUntilHeight is reached
func (be *BybitExchange) block(currentHeight uint64) {
	be.blockUntilHeight = currentHeight + DEFAULT_WAIT_BLOCKS_AFTER_TRADE
}

// reverseFillAmount computes the amount of quote tokens from fillAmount in base tokens
// func (be *BybitExchange) reverseFillAmount(fillAmount osmomath.BigDec, pair osmocexfillertypes.Pair, on osmocexfillertypes.ExchangeType) (osmomath.BigDec, error) {
// 	price, err := be.getBasePriceInQuote(pair, on)
// 	if err != nil {
// 		return osmomath.NewBigDec(0), err
// 	}

// 	// fillAmount is computed in base tokens, scale it to precision of quote tokens
// 	baseDecimals, err := be.getInterchainDenomDecimals(pair.BaseInterchainDenom())
// 	if err != nil {
// 		return osmomath.NewBigDec(0), err
// 	}

// 	quoteDecimals, err := be.getInterchainDenomDecimals(pair.QuoteInterchainDenom())
// 	if err != nil {
// 		return osmomath.NewBigDec(0), err
// 	}

// 	// fillAmount_scaled = fillAmount * 10^(quoteDecimals - baseDecimals)
// 	var fillAmountScaled osmomath.BigDec
// 	power := int64(quoteDecimals - baseDecimals)
// 	if power >= 0 {
// 		fillAmountScaled = fillAmount.Mul(osmomath.NewBigDec(10).Power(osmomath.NewBigDec(power)))
// 	} else {
// 		fillAmountScaled = fillAmount.Quo(osmomath.NewBigDec(10).Power(osmomath.NewBigDec(-power)))
// 	}

// 	// the formula for quote amount is: fillAmountScaled * price (scaled) and fillAmountScaled * price / 10^quoteDecimals (unscaled)
// 	return fillAmountScaled.Mul(price), nil
// }

// func (be *BybitExchange) getBasePriceInQuote(pair osmocexfillertypes.Pair, on osmocexfillertypes.ExchangeType) (osmomath.BigDec, error) {
// 	if on == osmocexfillertypes.OSMO {
// 		prices, err := (*be.osmoTokensUsecase).GetPrices(be.ctx, []string{pair.BaseInterchainDenom()}, []string{pair.QuoteInterchainDenom()}, domain.ChainPricingSourceType)
// 		if err != nil {
// 			return osmomath.NewBigDec(0), err
// 		}

// 		return prices.GetPriceForDenom(pair.BaseInterchainDenom(), pair.QuoteInterchainDenom()), nil
// 	} else {
// 		// for bybit orderbooks, define a price as the price at which you can immediately sell (which is a highest bid)
// 		orderbookAny, ok := be.orderbooks.Load(pair.String())
// 		if !ok {
// 			return osmomath.NewBigDec(0), errors.New("orderbook not found")
// 		}

// 		orderbook := orderbookAny.(*osmocexfillertypes.OrderbookData)
// 		price := orderbook.BidsDescending()[0].GetPrice()

// 		priceBigDec := osmomath.MustNewBigDecFromStr(price)
// 		return priceBigDec, nil
// 	}
// }

func (be *BybitExchange) registeredPairsSize() int { return len(be.registeredPairs) }
