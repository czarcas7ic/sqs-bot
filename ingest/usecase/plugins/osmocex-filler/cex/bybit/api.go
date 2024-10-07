package bybit

import (
	"time"

	wsbybit "github.com/hirokisan/bybit/v2"
	"github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/cex"
)

// subscribeOrderbook: https://bybit-exchange.github.io/docs/v5/websocket/public/orderbook
func (be *BybitExchange) subscribeOrderbook(pair cex.Pair, depth int) error {
	_, err := be.wsclient.SubscribeOrderBook(
		wsbybit.V5WebsocketPublicOrderBookParamKey{
			Depth:  depth,
			Symbol: wsbybit.SymbolV5(pair.String()),
		},
		// These callbacks are ran sequentially. Introduced concurrency is needed for performance.
		func(resp wsbybit.V5WebsocketPublicOrderBookResponse) error {
			if !be.newBlockSignal.Load() {
				return nil
			}
			defer be.newBlockWg.Done()

			ec := make(chan error)
			defer close(ec)

			go func() {
				// get orderbooks from CEX and DEX
				cexOrderbook := parseBybitOrderbook(resp.Data)
				osmoOrderbook, err := be.getOrderbookForPair(pair)
				if err != nil {
					ec <- err
				}

				ec <- be.matchOrderbooks(cexOrderbook, osmoOrderbook)
			}()

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
