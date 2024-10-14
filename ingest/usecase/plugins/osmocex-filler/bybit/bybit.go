package bybit

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/joho/godotenv"
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

	if os.Getenv("BYBIT_API_KEY") == "" || os.Getenv("BYBIT_SECRET_KEY") == "" {
		panic("BYBIT_API_KEY or BYBIT_SECRET_KEY not set")
	}

	wsclient := wsbybit.NewWebsocketClient().WithAuth(os.Getenv("BYBIT_API_KEY"), os.Getenv("BYBIT_SECRET_KEY"))
	svc, err := wsclient.V5().Public(wsbybit.CategoryV5Spot)
	if err != nil {
		panic(err)
	}

	httpclient := bybit.NewBybitHttpClient(os.Getenv("BYBIT_API_KEY"), os.Getenv("BYBIT_SECRET_KEY"), bybit.WithBaseURL(HTTP_URL))

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

// Signal signals the websocket callback to start matching orderbooks
// Signal is called at the beginning of each block
func (be *BybitExchange) Signal() {
	newBlockWg := sync.WaitGroup{}
	newBlockWg.Add(be.registeredPairsSize())

	for pair := range be.registeredPairs {
		go be.processPair(pair, &newBlockWg)
	}

	newBlockWg.Wait() // blocks until all orderbooks are processed for bybit block
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
		fillAmount, err := be.calculateFillAmount(pair, bybitOrderbook.BidsDescending(), osmoOrders.AsksAscending())
		if err != nil {
			be.logger.Error("failed to get fill amount and direction", zap.Error(err))
			return
		}

		// fill ask on osmo (buy on osmo)
		coinIn := sdk.NewCoin(pair.Quote, fillAmount.Dec().TruncateInt())
		be.tradeOsmosis(coinIn, pair.Base, osmoOrderbook.PoolID)

		// fill bid on bybit (sell on bybit)
		be.spot(pair, osmocexfillertypes.SELL, fillAmount.String())
	}

	// check arb from osmo to bybit exchange
	if be.existsArbFromOsmo(pair, bybitOrderbook.AsksAscending(), osmoOrders.BidsDescending()) {
		fillAmount, err := be.calculateFillAmount(pair, bybitOrderbook.AsksAscending(), osmoOrders.BidsDescending())
		if err != nil {
			be.logger.Error("failed to get fill amount and direction", zap.Error(err))
			return
		}

		// fill bid on osmo (sell on osmo)
		coinIn := sdk.NewCoin(pair.Base, fillAmount.Dec().TruncateInt())
		be.tradeOsmosis(coinIn, pair.Quote, osmoOrderbook.PoolID)

		// fill ask on bybit (buy on bybit)
		be.spot(pair, osmocexfillertypes.BUY, fillAmount.String())
	}
}

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

func (be *BybitExchange) SupportedPair(pair osmocexfillertypes.Pair) bool {
	_, ok := be.registeredPairs[pair]
	return ok
}

func (be *BybitExchange) registeredPairsSize() int { return len(be.registeredPairs) }
