package bybit

import (
	"errors"

	wsbybit "github.com/hirokisan/bybit/v2"
	osmocexfillertypes "github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/types"
)

// subscribeOrderbook: https://bybit-exchange.github.io/docs/v5/websocket/public/orderbook
func (be *BybitExchange) subscribeOrderbook(pair osmocexfillertypes.Pair, depth int) error {
	_, err := be.wsclient.SubscribeOrderBook(
		wsbybit.V5WebsocketPublicOrderBookParamKey{
			Depth:  depth,
			Symbol: wsbybit.SymbolV5(pair.String()),
		},
		// These callbacks are ran sequentially.
		func(resp wsbybit.V5WebsocketPublicOrderBookResponse) error {
			return be.acknowledgeResponse(resp)
		},
	)

	return err
}

func (be *BybitExchange) acknowledgeResponse(resp wsbybit.V5WebsocketPublicOrderBookResponse) error {
	switch resp.Type {
	case "snapshot": // first response, construct initial orderbook
		orderbook := parseBybitOrderbook(resp.Data)
		be.orderbooks.Store(resp.Data.Symbol, orderbook)
		return nil
	case "delta": // subsequent responses, update orderbook
		be.updateBybitOrderbook(resp.Data)
		return nil
	default:
		return errors.New("unknown response type")
	}
}
