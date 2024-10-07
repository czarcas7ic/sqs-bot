package bybit

import (
	"time"

	"github.com/hirokisan/bybit/v2"
)

func (be *BybitExchange) SubscribeOrderbook(symbol bybit.SymbolV5, depth int) error {
	_, err := be.ws.SubscribeOrderBook(
		bybit.V5WebsocketPublicOrderBookParamKey{
			Depth:  depth,
			Symbol: symbol,
		},
		func(resp bybit.V5WebsocketPublicOrderBookResponse) error {
			select {
			case err := <-be.ProcessOrderbookDataAsync(resp):
				return err
			case <-time.After(5 * time.Second):
				be.logger.Info("timeout waiting for ProcessOrderbookDataAsync")
				return nil
			}
		},
	)

	return err
}
