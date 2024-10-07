package bybit

import (
	wsbybit "github.com/hirokisan/bybit/v2"
	"github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/cex"
)

func parseBybitOrderbook(data wsbybit.V5WebsocketPublicOrderBookData) cex.OrderbookData {
	bids := []cex.OrderbookEntry{}
	asks := []cex.OrderbookEntry{}

	for _, bid := range data.Bids {
		bids = append(bids, cex.OrderbookEntry{
			Price:  bid.Price,
			Amount: bid.Size,
		})
	}

	for _, ask := range data.Asks {
		asks = append(asks, cex.OrderbookEntry{
			Price:  ask.Price,
			Amount: ask.Size,
		})
	}

	return cex.OrderbookData{
		Symbol: string(data.Symbol),
		Bids:   bids,
		Asks:   asks,
	}
}
