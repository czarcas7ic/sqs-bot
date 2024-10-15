package bybit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

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

	logger log.Logger
}

var _ osmocexfillertypes.ExchangeI = (*BybitExchange)(nil)

const (
	HTTP_URL      = bybit.MAINNET
	DEFAULT_DEPTH = 50 // default depth for an orderbook
)

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
	// Get the path of the current file
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

	if os.Getenv("BYBIT_API_KEY") == "" || os.Getenv("BYBIT_API_SECRET") == "" {
		panic("BYBIT_API_KEY or BYBIT_API_SECRET not set")
	}

	wsclient := wsbybit.NewWebsocketClient().WithAuth(os.Getenv("BYBIT_API_KEY"), os.Getenv("BYBIT_API_SECRET"))
	svc, err := wsclient.V5().Public(wsbybit.CategoryV5Spot)
	if err != nil {
		panic(err)
	}

	httpclient := bybit.NewBybitHttpClient(os.Getenv("BYBIT_API_KEY"), os.Getenv("BYBIT_API_SECRET"), bybit.WithBaseURL(HTTP_URL))

	return &BybitExchange{
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
		logger:                    logger,
	}
}

// RegisterPairs implements osmocexfillertypes.ExchangeI
func (be *BybitExchange) Signal() {
	newBlockWg := sync.WaitGroup{}
	newBlockWg.Add(be.registeredPairsSize())

	for pair := range be.registeredPairs {
		go be.processPair(pair, &newBlockWg)
	}

	// results, _ := (*be.osmoPoolsUsecase).GetAllCanonicalOrderbookPoolIDs()
	// for _, result := range results {
	// 	fmt.Println(result.Base, result.Quote)
	// }

	newBlockWg.Wait() // blocks until all orderbooks are processed for bybit block
}

// RegisterPairs implements osmocexfillertypes.ExchangeI
func (be *BybitExchange) RegisterPairs(ctx context.Context) error {
	for _, pair := range ArbPairs {
		be.registeredPairs[pair] = struct{}{}
		if err := be.subscribeOrderbook(pair, DEFAULT_DEPTH); err != nil {
			return err
		}
	}

	go be.wsclient.Start(ctx, nil)

	return nil
}

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
func (be *BybitExchange) processPair(pair osmocexfillertypes.Pair, wg *sync.WaitGroup) {
	defer wg.Done()

	// get orderbooks from CEX and DEX
	bybitOrderbook, err := be.getBybitOrderbookForPair(pair)
	if err != nil {
		be.logger.Error("failed to get BYBIT orderbook for pair", zap.String("pair", pair.String()), zap.Error(err))
		return
	}

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

	// check arb from bybit exchange to osmo
	// if true -> bybit.highestBid > osmo.lowestAsk
	// so, we need to buy from osmo and sell on bybit
	if be.existsArbFromBybit(pair, bybitOrderbook.BidsDescending(), osmoOrders.AsksAscending()) {
		defer time.Sleep(10 * time.Second) // wait for block inclusion with buffer to avoid sequence mismatch and same order processing

		// scale bybit orderbook
		basePrecision, err := be.getInterchainDenomDecimals(pair.Base)
		if err != nil {
			be.logger.Error("failed to get base precision", zap.Error(err))
			return
		}

		// operate on a copy of an orderbook
		bybitOrderbook := bybitOrderbook.ScaleSize(basePrecision)

		// get total available fill amount
		fillAmount, err := be.calculateFillAmount(pair, bybitOrderbook.BidsDescending(), osmoOrders.AsksAscending())
		if err != nil {
			be.logger.Error("failed to get fill amount and direction", zap.Error(err))
			return
		}

		// get balances (do not move outside of for loop )
		bybitBalances, osmoBalances, err := be.GetBotBalances()
		if err != nil {
			be.logger.Error("failed to get bot balances", zap.Error(err))
			return
		}

		// get balances in big dec
		osmoBalanceBaseBigDec := osmomath.NewBigDecFromBigInt(osmoBalances.AmountOf(SymbolToChainDenom[pair.Base]).BigInt())
		bybitBalanceBaseBigDec := bybitBalances[pair.Base].BigDecBalance(basePrecision)

		// get final fill amount and reversed fill amount
		fillAmount = be.adjustFillAmount(fillAmount, bybitBalanceBaseBigDec, osmoBalanceBaseBigDec) // TODO: also make sure there is enough quote balance
		quoteFillAmount, err := be.reverseFillAmount(fillAmount, pair, osmocexfillertypes.OSMO)
		if err != nil {
			be.logger.Error("failed to reverse fill amount", zap.Error(err))
			return
		}

		// fill ask on osmo (buy base on osmo)
		coinIn := sdk.NewCoin(SymbolToChainDenom[pair.Quote], quoteFillAmount.Dec().TruncateInt())
		go be.tradeOsmosis(coinIn, SymbolToChainDenom[pair.Base], osmoOrderbook.PoolID)

		// fill bid on bybit (sell on bybit)
		go be.spot(pair, osmocexfillertypes.SELL, fillAmount)

		be.logger.Info("arbitrage from BYBIT: executed", zap.String("pair", pair.String()))
		return
	}

	// Actually Got: 11.4008152374
	// Paid: 11.5246

	// check arb from osmo to bybit exchange
	// if true -> osmo.highestBid > bybit.lowestAsk
	// so, we need to buy from bybit and sell on osmo
	if be.existsArbFromOsmo(pair, bybitOrderbook.AsksAscending(), osmoOrders.BidsDescending()) {
		defer time.Sleep(10 * time.Second) // wait for block inclusion with buffer to avoid sequence mismatch and same order processing

		// scale bybit orderbook
		basePrecision, err := be.getInterchainDenomDecimals(pair.Base)
		if err != nil {
			be.logger.Error("failed to get base precision", zap.Error(err))
			return
		}

		bybitOrderbook.ScaleSize(basePrecision)

		// get total available fill amount
		fillAmount, err := be.calculateFillAmount(pair, bybitOrderbook.AsksAscending(), osmoOrders.BidsDescending())
		if err != nil {
			be.logger.Error("failed to get fill amount and direction", zap.Error(err))
			return
		}

		// get balances
		bybitBalances, osmoBalances, err := be.GetBotBalances()
		if err != nil {
			be.logger.Error("failed to get bot balances", zap.Error(err))
			return
		}

		// get balances in big dec
		osmoBalanceBigDec := osmomath.NewBigDecFromBigInt(osmoBalances.AmountOf(SymbolToChainDenom[pair.Base]).BigInt())
		bybitBalanceBigDec := bybitBalances[pair.Quote].BigDecBalance(basePrecision)

		// get final fill amount and reversed fill amount
		fillAmount = be.adjustFillAmount(fillAmount, osmoBalanceBigDec, bybitBalanceBigDec)
		quoteFillAmount, err := be.reverseFillAmount(fillAmount, pair, osmocexfillertypes.BYBIT)
		if err != nil {
			be.logger.Error("failed to reverse fill amount", zap.Error(err))
			return
		}

		// fill bid on osmo (sell base on osmo)
		coinIn := sdk.NewCoin(pair.Base, fillAmount.Dec().TruncateInt())
		be.tradeOsmosis(coinIn, pair.Quote, osmoOrderbook.PoolID)

		// fill ask on bybit (buy base on bybit)
		be.spot(pair, osmocexfillertypes.BUY, quoteFillAmount)

		be.logger.Info("arbitrage from OSMOSIS: executed", zap.String("pair", pair.String()))

		return
	}
}

// reverseFillAmount computes the amount of quote tokens from fillAmount in base tokens
func (be *BybitExchange) reverseFillAmount(fillAmount osmomath.BigDec, pair osmocexfillertypes.Pair, on osmocexfillertypes.ExchangeType) (osmomath.BigDec, error) {
	price, err := be.getBasePriceInQuote(pair, on)
	if err != nil {
		return osmomath.NewBigDec(0), err
	}

	// fillAmount is computed in base tokens, scale it to precision of quote tokens
	baseDecimals, err := be.getInterchainDenomDecimals(pair.Base)
	if err != nil {
		return osmomath.NewBigDec(0), err
	}

	quoteDecimals, err := be.getInterchainDenomDecimals(pair.Quote)
	if err != nil {
		return osmomath.NewBigDec(0), err
	}

	// fillAmount_scaled = fillAmount * 10^(quoteDecimals - baseDecimals)
	var fillAmountScaled osmomath.BigDec
	power := int64(quoteDecimals - baseDecimals)
	if power >= 0 {
		fillAmountScaled = fillAmount.Mul(osmomath.NewBigDec(10).Power(osmomath.NewBigDec(power)))
	} else {
		fillAmountScaled = fillAmount.Quo(osmomath.NewBigDec(10).Power(osmomath.NewBigDec(-power)))
	}

	// the formula for quote amount is: fillAmountScaled * price (scaled) and fillAmountScaled * price / 10^quoteDecimals (unscaled)
	return fillAmountScaled.Mul(price), nil
}

func (be *BybitExchange) getBasePriceInQuote(pair osmocexfillertypes.Pair, on osmocexfillertypes.ExchangeType) (osmomath.BigDec, error) {
	if on == osmocexfillertypes.OSMO {
		prices, err := (*be.osmoTokensUsecase).GetPrices(be.ctx, []string{SymbolToChainDenom[pair.Base]}, []string{SymbolToChainDenom[pair.Quote]}, domain.ChainPricingSourceType)
		if err != nil {
			return osmomath.NewBigDec(0), err
		}

		return prices.GetPriceForDenom(SymbolToChainDenom[pair.Base], SymbolToChainDenom[pair.Quote]), nil
	} else {
		// for bybit orderbooks, define a price as the price at which you can immediately sell (which is a highest bid)
		orderbookAny, ok := be.orderbooks.Load(pair.String())
		if !ok {
			return osmomath.NewBigDec(0), errors.New("orderbook not found")
		}

		orderbook := orderbookAny.(*osmocexfillertypes.OrderbookData)
		price := orderbook.BidsDescending()[0].GetPrice()

		priceBigDec := osmomath.MustNewBigDecFromStr(price)
		return priceBigDec, nil
	}
}

func (be *BybitExchange) registeredPairsSize() int { return len(be.registeredPairs) }
