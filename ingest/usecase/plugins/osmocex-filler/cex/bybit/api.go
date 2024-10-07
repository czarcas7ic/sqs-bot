package bybit

import (
	"time"

	wsbybit "github.com/hirokisan/bybit/v2"
	"github.com/osmosis-labs/sqs/domain"
)

// subscribeOrderbook: https://bybit-exchange.github.io/docs/v5/websocket/public/orderbook
func (be *BybitExchange) subscribeOrderbook(symbol wsbybit.SymbolV5, depth int) error {
	_, err := be.wsclient.SubscribeOrderBook(
		wsbybit.V5WebsocketPublicOrderBookParamKey{
			Depth:  depth,
			Symbol: symbol,
		},
		func(resp wsbybit.V5WebsocketPublicOrderBookResponse) error {
			ec := make(chan error)
			defer close(ec)

			cexOrderbook := parseBybitOrderbook(resp.Data)

			go be.matchOrderbooks(cexOrderbook, domain.CanonicalOrderBooksResult{}) // TODO: osmoData

			select {
			case err := <-ec:
				return err
			case <-time.After(5 * time.Second):
				be.logger.Info("timeout waiting for ProcessOrderbookDataAsync")
				return nil
			}
		},
	)

	return err
}
