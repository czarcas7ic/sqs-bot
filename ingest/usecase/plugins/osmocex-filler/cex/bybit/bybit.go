package bybit

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/osmosis-labs/sqs/domain"
	"github.com/osmosis-labs/sqs/domain/mvc"
	"github.com/osmosis-labs/sqs/log"
	"go.uber.org/zap"

	bybit "github.com/wuhewuhe/bybit.go.api"

	wsbybit "github.com/hirokisan/bybit/v2"
	"github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/cex"
)

type BybitExchange struct {
	// uses websocket for orderbook data processing for performance
	wsclient wsbybit.V5WebsocketPublicServiceI
	// http client for trade calls
	httpclient *bybit.Client

	// map of pairs that this exchange is configured to arb against
	registeredPairs map[cex.Pair]struct{}

	// newBlockSignal is a signal to the websocket client to continue matching orderbooks
	newBlockSignal bool
	newBlockWg     sync.WaitGroup

	// upstream pointers
	osmoPoolIdToOrders *sync.Map
	osmoPoolsUseCase   mvc.PoolsUsecase

	logger log.Logger
}

var _ cex.CExchangeI = (*BybitExchange)(nil)

const (
	HTTP_URL = bybit.TESTNET // dev
)

func New(
	logger log.Logger,
	osmoPoolIdToOrders *sync.Map,
	osmoPoolsUseCase mvc.PoolsUsecase,
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
		registeredPairs:    make(map[cex.Pair]struct{}),
		osmoPoolIdToOrders: osmoPoolIdToOrders,
		osmoPoolsUseCase:   osmoPoolsUseCase,
		logger:             logger,
	}
}

// Signal signals the websocket callback to start matching orderbooks
// Signal is called at the beginning of each block
func (be *BybitExchange) Signal() {
	be.startBlock()
	defer be.endBlock()

	be.newBlockWg.Add(be.registeredPairsSize())
	be.newBlockWg.Wait() // blocks until all orderbooks are processed for this block
}

// matchOrderbooks is a callback used by the websocket client to try and find the fillable orderbooks
func (be *BybitExchange) matchOrderbooks(thisData cex.OrderbookData, osmoData domain.CanonicalOrderBooksResult) error {
	osmoOrdersAny, ok := be.osmoPoolIdToOrders.Load(osmoData.PoolID)
	if !ok {
		be.logger.Error("failed to load osmo orders", zap.Uint64("poolID", osmoData.PoolID))
		return errors.New("failed to load osmo orders")
	}

	osmoOrders := osmoOrdersAny.(domain.CanonicalOrderBooksResult)
	fmt.Println("osmoOrders", osmoOrders)
	return nil
}

func (be *BybitExchange) RegisterPair(pair cex.Pair) error {
	be.registeredPairs[pair] = struct{}{}
	return be.subscribeOrderbook(pair, 1)
}

func (be *BybitExchange) SupportedPair(pair cex.Pair) bool {
	_, ok := be.registeredPairs[pair]
	return ok
}

func (be *BybitExchange) registeredPairsSize() int { return len(be.registeredPairs) }
func (be *BybitExchange) startBlock() { be.newBlockSignal = true }
func (be *BybitExchange) endBlock() { be.newBlockSignal = false }
