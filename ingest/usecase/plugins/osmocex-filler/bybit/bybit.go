package bybit

import (
	"context"
	"fmt"
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
	DEFAULT_WAIT_BLOCKS_AFTER_TRADE = 5  // TODO: it takes a while for a plugin to update the orderbook state. Why?? look into it
	USDC_INTERCHAIN                 = "ibc/498A0751C798A0D9A389AA3691123DADA57DAA4FE165D5C75894505B876BA6E4"
	UOSMO                           = "uosmo"
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
	_ = currentHeight
	if be.blockUntilHeight != 0 && currentHeight < be.blockUntilHeight {
		be.logger.Info("arbitrage blocked until height", zap.Uint64("blockUntilHeight", be.blockUntilHeight))
		return
	}

	newBlockWg := sync.WaitGroup{}
	newBlockWg.Add(be.registeredPairsSize())

	var baseTokens osmocexfillertypes.Set[string]
	var quoteTokens osmocexfillertypes.Set[string]

	for pair := range be.registeredPairs {
		baseTokens.Add(pair.BaseInterchainDenom())
		quoteTokens.Add(pair.QuoteInterchainDenom())
	}

	// add osmo and usdc to the set (for gas pricing)
	baseTokens.Add(UOSMO)
	quoteTokens.Add(USDC_INTERCHAIN)

	osmoPrices, err := (*be.osmoTokensUsecase).GetPrices(be.ctx, baseTokens.Values(), quoteTokens.Values(), domain.ChainPricingSourceType)
	if err != nil {
		be.logger.Error("failed to get osmo prices", zap.Error(err))
		return
	}

	for pair := range be.registeredPairs {
		// should be ok to use osmo prices concurrently, only passed for reads
		go be.processPair(pair, osmoPrices, &newBlockWg)
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
func (be *BybitExchange) processPair(pair osmocexfillertypes.Pair, osmoPrices domain.PricesResult, wg *sync.WaitGroup) {
	defer wg.Done()

	// get orderbooks from CEX and DEX
	bybitOrderbook, err := be.getBybitOrderbookForPair(pair)
	if err != nil {
		be.logger.Error("failed to get BYBIT orderbook for pair", zap.String("pair", pair.String()), zap.Error(err))
		return
	}

	baseDecimals, quoteDecimals, err := be.getDecimalsForPair(pair)
	if err != nil {
		be.logger.Error("failed to get decimals for pair", zap.Error(err))
		return
	}

	// this is a hack to solve the issue described in QuoteBids docs
	bybitOrderbook = bybitOrderbook.QuoteBids().ScaleSize(baseDecimals, quoteDecimals)

	// get prices for tokens
	osmoPrice := osmoPrices.GetPriceForDenom(pair.BaseInterchainDenom(), pair.QuoteInterchainDenom())

	// get balances
	bybitBalances, osmoBalances, err := be.GetBotBalances()
	if err != nil {
		be.logger.Error("failed to get bot balances", zap.Error(err))
		return
	}

	buyOnOsmo := be.existsArbitrageOpportunity(bybitOrderbook.BidsDescending(), osmoPrice, baseDecimals, quoteDecimals)
	if buyOnOsmo != nil {
		// first, check that fill amount is at least the minimum amount
		minFill, ok := be.getMinFillAmount(pair.QuoteInterchainDenom())
		if !ok {
			be.logger.Error("failed to get min fill amount", zap.Error(err))
			return
		}

		if buyOnOsmo.SellAmount.LT(minFill) {
			be.logger.Info("sell amount is less than min fill amount", zap.String("pair", pair.String()), zap.String("sellAmount", buyOnOsmo.SellAmount.String()), zap.String("minFill", minFill.String()))
			return
		}

		// check that we have enough balance to perform the trade
		quoteBalanceDec := osmomath.NewDecFromInt(osmoBalances.AmountOf(pair.QuoteInterchainDenom()))
		quoteBalance := osmomath.NewBigDecFromBigInt(quoteBalanceDec.BigInt())
		if buyOnOsmo.SellAmount.GT(quoteBalance) {
			be.logger.Info("not enough balance to perform trade", zap.String("pair", pair.String()), zap.String("buyAmount", buyOnOsmo.BuyAmount.String()), zap.String("balance", osmoBalances.GetBalance(pair.BaseInterchainDenom()).String()))
			return
		}

		coinIn := sdk.NewCoin(pair.BaseInterchainDenom(), buyOnOsmo.BuyAmount.Dec().TruncateInt())
		commit, expectedQuoteOut, gasUsed, err := be.simulateOsmoTrade(coinIn, pair.QuoteInterchainDenom())
		if err != nil {
			be.logger.Error("failed to simulate osmo trade", zap.Error(err))
			return
		}

		profit := expectedQuoteOut.Sub(buyOnOsmo.SellAmount)
		if profit.IsPositive() { // profit must be positive
			be.logger.Info("arbitrage from OSMOSIS found: profitable", zap.String("pair", pair.String()), zap.String("profit", profit.String()))
			profitValue := profit.Mul(osmoPrices.GetPriceForDenom(pair.QuoteInterchainDenom(), USDC_INTERCHAIN))

			gasUsedBigDec := osmomath.NewBigDec(int64(gasUsed))
			gasValue := gasUsedBigDec.Mul(osmoPrices.GetPriceForDenom(UOSMO, USDC_INTERCHAIN))

			fmt.Println("profitValue: ", profitValue.String())
			fmt.Println("gasValue: ", gasValue.String())

			if profitValue.GTE(gasValue) { // TODO: check that decimals align for these two
				localWg := sync.WaitGroup{}
				localWg.Add(2)

				go func() {
					defer localWg.Done()
					commit()
				}()

				go func() {
					defer localWg.Done()
					be.spot(pair, SELL, buyOnOsmo.SellAmount)
				}()

				localWg.Wait()
			}
		}
	}

	buyOnBybit := be.existsArbitrageOpportunity(bybitOrderbook.AsksAscending(), osmoPrice, baseDecimals, quoteDecimals)
	if buyOnBybit != nil {
		// first, check that fill amount is at least the minimum amount
		minFill, ok := be.getMinFillAmount(pair.BaseInterchainDenom())
		if !ok {
			be.logger.Error("failed to get min fill amount", zap.Error(err))
			return
		}

		if buyOnBybit.BuyAmount.LT(minFill) {
			be.logger.Info("buy amount is less than min fill amount", zap.String("pair", pair.String()), zap.String("buyAmount", buyOnBybit.BuyAmount.String()), zap.String("minFill", minFill.String()))
			return
		}

		coinIn := sdk.NewCoin(pair.QuoteInterchainDenom(), buyOnBybit.SellAmount.Dec().TruncateInt())
		commit, expectedBaseOut, gasUsed, err := be.simulateOsmoTrade(coinIn, pair.BaseInterchainDenom())
		if err != nil {
			be.logger.Error("failed to simulate osmo trade", zap.Error(err))
			return
		}

		// expectedBaseOut is the final amount of base tokens we get when performing trade on osmo
		// buyOnBybit.BuyAmount is the amount of base tokens we would have bought on bybit
		profit := expectedBaseOut.Sub(buyOnBybit.BuyAmount)
		if profit.IsPositive() {
			be.logger.Info("arbitrage from BYBIT found: profitable", zap.String("pair", pair.String()), zap.String("profit", profit.String()))
			profitValue := profit.Mul(osmoPrices.GetPriceForDenom(pair.BaseInterchainDenom(), USDC_INTERCHAIN))

			gasUsedBigDec := osmomath.NewBigDec(int64(gasUsed))
			gasValue := gasUsedBigDec.Mul(osmoPrices.GetPriceForDenom(UOSMO, USDC_INTERCHAIN))

			fmt.Println("profitValue: ", profitValue.String())
			fmt.Println("gasValue: ", gasValue.String())

			if profitValue.GTE(gasValue) {
				localWg := sync.WaitGroup{}
				localWg.Add(2)

				go func() {
					defer localWg.Done()
					be.spot(pair, BUY, buyOnBybit.BuyAmount)
				}()

				go func() {
					defer localWg.Done()
					commit()
				}()

				localWg.Wait()
			}
		}
	}
}

// simulates trade on osmosis. Returns expectedAmountOut and gas used
// func (be *BybitExchange) simulateTrade(pair osmocexfillertypes.Pair, ad *ArbitrageDirection) (osmomath.BigDec, uint64, error) {
// 	switch ad.BuyFrom {
// 	case BYBIT: // sell on osmo
// 		sellCoin := sdk.NewCoin(pair.QuoteInterchainDenom(), ad.SellAmount.Dec().TruncateInt())
// 		adjustedGasUsed, amountOutBaseExpected, err := be.simulateOsmoTrade(sellCoin, pair.BaseInterchainDenom())
// 		if err != nil {
// 			be.logger.Error("failed to simulate osmo trade", zap.Error(err))
// 			return osmomath.NewBigDec(0), 0, err
// 		}

// 		if amountOutBaseExpected.GT(ad.BuyAmount) {
// 			return amountOutBaseExpected, adjustedGasUsed, nil
// 		}

// 		return osmomath.NewBigDec(0), 0, errors.New("expected amount out is less than buy amount")

// 	case OSMO: // buy on osmo
// 		buyCoin := sdk.NewCoin(pair.BaseInterchainDenom(), ad.BuyAmount.Dec().TruncateInt())
// 		adjustedGasUsed, amountOutQuoteExpected, err := be.simulateOsmoTrade(buyCoin, pair.QuoteInterchainDenom())
// 		if err != nil {
// 			be.logger.Error("failed to simulate osmo trade", zap.Error(err))
// 			return osmomath.NewBigDec(0), 0, err
// 		}

// 		if amountOutQuoteExpected.GT(ad.SellAmount) {
// 			return amountOutQuoteExpected, adjustedGasUsed, nil
// 		}
// 	}

// 	return osmomath.NewBigDec(0), 0, errors.New("invalid buy from")
// }

// func (be *BybitExchange)

func (be *BybitExchange) getDecimalsForPair(pair osmocexfillertypes.Pair) (int, int, error) {
	baseDecimals, err := be.getInterchainDenomDecimals(pair.BaseInterchainDenom())
	if err != nil {
		return 0, 0, err
	}

	quoteDecimals, err := be.getInterchainDenomDecimals(pair.QuoteInterchainDenom())
	if err != nil {
		return 0, 0, err
	}

	return baseDecimals, quoteDecimals, nil
}

func (be *BybitExchange) registeredPairsSize() int { return len(be.registeredPairs) }
