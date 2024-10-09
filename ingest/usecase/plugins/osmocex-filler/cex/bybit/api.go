package bybit

import (
	wsbybit "github.com/hirokisan/bybit/v2"
	"github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/cex"
	"go.uber.org/zap"
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
			go be.callbackInternal(resp, pair)

			return nil // immediately return to not block the websocket
		},
	)

	return err
}

func (be *BybitExchange) callbackInternal(resp wsbybit.V5WebsocketPublicOrderBookResponse, pair cex.Pair) {
	if !be.newBlockSignal {
		return
	}

	defer be.newBlockWg.Done()

	// get orderbooks from CEX and DEX
	cexOrderbook := parseBybitOrderbook(resp.Data)
	osmoOrderbook, err := be.getOrderbookForPair(pair)
	if err != nil {
		be.logger.Error("failed to get orderbook for pair", zap.String("pair", pair.String()), zap.Error(err))
	}

	err = be.matchOrderbooks(cexOrderbook, osmoOrderbook)
	if err != nil {
		be.logger.Error("failed to match orderbooks", zap.String("pair", pair.String()), zap.Error(err))
	}
}
