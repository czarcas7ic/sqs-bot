package bybit

import (
	"os"

	"github.com/osmosis-labs/sqs/log"

	"github.com/hirokisan/bybit/v2"
	"github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/cex"
)

type BybitExchange struct {
	ws     bybit.V5WebsocketPublicServiceI
	logger log.Logger
}

var _ cex.CExchangeI = (*BybitExchange)(nil)

func New(logger log.Logger) *BybitExchange {
	wsClient := bybit.NewWebsocketClient().WithAuth(os.Getenv("BYBIT_API_KEY"), os.Getenv("BYBIT_SECRET_KEY"))
	svc, err := wsClient.V5().Public(bybit.CategoryV5Spot)
	if err != nil {
		panic(err)
	}

	return &BybitExchange{
		ws:     svc,
		logger: logger,
	}
}

func (be *BybitExchange) ProcessOrderbookDataAsync(resp bybit.V5WebsocketPublicOrderBookResponse) <-chan error {
	ec := make(chan error)
	defer close(ec)

	// capture resp
	go func() {
		orderbook := parseBybitOrderbook(resp.Data)
		ec <- be.ProcessOrderbook(orderbook)
	}()

	return ec
}

func (be *BybitExchange) ProcessOrderbook(cex.OrderbookData) error {
	return nil
}
