package bybit

import (
	"os"
	"sync"

	"github.com/osmosis-labs/sqs/domain"
	"github.com/osmosis-labs/sqs/log"

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

	osmoOrderbooks *sync.Map

	logger log.Logger
}

var _ cex.CExchangeI = (*BybitExchange)(nil)

const (
	HTTP_URL = bybit.TESTNET // dev
)

func New(logger log.Logger) *BybitExchange {
	wsclient := wsbybit.NewWebsocketClient().WithAuth(os.Getenv("BYBIT_API_KEY"), os.Getenv("BYBIT_SECRET_KEY"))
	svc, err := wsclient.V5().Public(wsbybit.CategoryV5Spot)
	if err != nil {
		panic(err)
	}

	httpclient := bybit.NewBybitHttpClient(os.Getenv("BYBIT_API_KEY"), os.Getenv("BYBIT_SECRET_KEY"), bybit.WithBaseURL(HTTP_URL))

	return &BybitExchange{
		wsclient:        svc,
		httpclient:      httpclient,
		registeredPairs: make(map[cex.Pair]struct{}),
		logger:          logger,
	}
}

// Implements cex.CExchangeI
func (be *BybitExchange) ProcessOrderbook(osmoData domain.CanonicalOrderBooksResult) error {
	return nil
}

// matchOrderbooks is a callback used by the websocket client to try and find the fillable orderbooks
func (be *BybitExchange) matchOrderbooks(thisData cex.OrderbookData, osmoData domain.CanonicalOrderBooksResult) error {
	return nil
}

// func (be *BybitExchange) ProcessOrderbook(thisData cex.OrderbookData, osmoData domain.CanonicalOrderBooksResult) error {
// 	return nil
// }

func (be *BybitExchange) RegisterPair(pair cex.Pair) error {
	be.registeredPairs[pair] = struct{}{}
	return be.subscribeOrderbook(wsbybit.SymbolV5(pair.String()), 1)
}

func (be *BybitExchange) SupportedPair(pair cex.Pair) bool {
	_, ok := be.registeredPairs[pair]
	return ok
}
