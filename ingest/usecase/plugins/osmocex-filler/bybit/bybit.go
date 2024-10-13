package bybit

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/osmosis-labs/sqs/domain/mvc"
	"github.com/osmosis-labs/sqs/log"
	"go.uber.org/zap"

	bybit "github.com/wuhewuhe/bybit.go.api"

	wsbybit "github.com/hirokisan/bybit/v2"
	orderbookplugindomain "github.com/osmosis-labs/sqs/domain/orderbook/plugin"
	osmocexfillertypes "github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/types"
)

type BybitExchange struct {
	// uses websocket for orderbook data processing for performance
	wsclient wsbybit.V5WebsocketPublicServiceI
	// http client for trade calls
	httpclient *bybit.Client

	// map of pairs that bybit exchange is configured to arb against
	registeredPairs map[osmocexfillertypes.Pair]struct{}

	// upstream pointers
	osmoPoolIdToOrders *sync.Map
	osmoPoolsUseCase   *mvc.PoolsUsecase
	osmoTokensUseCase  *mvc.TokensUsecase

	// bybit orderbooks: Symbol -> OrderbookData
	orderbooks sync.Map

	logger log.Logger
}

var _ osmocexfillertypes.ExchangeI = (*BybitExchange)(nil)

const (
	HTTP_URL = bybit.MAINNET // dev
	DDEPTH   = 50            // default depth for an orderbook
)

func New(
	logger log.Logger,
	osmoPoolIdToOrders *sync.Map,
	osmoPoolsUseCase *mvc.PoolsUsecase,
	osmoTokensUseCase *mvc.TokensUsecase,
) *BybitExchange {
	wsclient := wsbybit.NewWebsocketClient().WithAuth(os.Getenv("BYBIT_API_KEY"), os.Getenv("BYBIT_SECRET_KEY"))
	svc, err := wsclient.V5().Public(wsbybit.CategoryV5Spot)
	if err != nil {
		panic(err)
	}

	httpclient := bybit.NewBybitHttpClient(os.Getenv("BYBIT_API_KEY"), os.Getenv("BYBIT_SECRET_KEY"), bybit.WithBaseURL(HTTP_URL))

	return &BybitExchange{
		wsclient:           svc,
		httpclient:         httpclient,
		registeredPairs:    make(map[osmocexfillertypes.Pair]struct{}),
		osmoPoolIdToOrders: osmoPoolIdToOrders,
		osmoPoolsUseCase:   osmoPoolsUseCase,
		osmoTokensUseCase:  osmoTokensUseCase,
		logger:             logger,
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
	fmt.Println("OSMO ORDERS: ", osmoOrders, bybitOrderbook)

	// check arb from bybit exchange to osmo
	if be.existsArbFromBybit(pair, bybitOrderbook.BidsDescending(), osmoOrders.AsksAscending()) {

	}

	// check arb from osmo to bybit exchange
	if be.existsArbFromOsmo(pair, bybitOrderbook.AsksAscending(), osmoOrders.BidsDescending()) {

	}
}

func (be *BybitExchange) RegisterPairs(ctx context.Context) error {
	for _, pair := range ArbPairs {
		be.registeredPairs[pair] = struct{}{}
		if err := be.subscribeOrderbook(pair, DDEPTH); err != nil {
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

// func (be *BybitExchange)
