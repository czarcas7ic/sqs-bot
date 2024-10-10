package bybit

import (
	"errors"

	wsbybit "github.com/hirokisan/bybit/v2"
	"github.com/osmosis-labs/sqs/domain"
	osmocexfillertypes "github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/types"
	"go.uber.org/zap"
)

func parseBybitOrderbook(data wsbybit.V5WebsocketPublicOrderBookData) osmocexfillertypes.OrderbookData {
	bids := []osmocexfillertypes.OrderbookEntry{}
	asks := []osmocexfillertypes.OrderbookEntry{}

	for _, bid := range data.Bids {
		bids = append(bids, osmocexfillertypes.OrderbookEntry{
			Price: bid.Price,
			Size:  bid.Size,
		})
	}

	for _, ask := range data.Asks {
		asks = append(asks, osmocexfillertypes.OrderbookEntry{
			Price: ask.Price,
			Size:  ask.Size,
		})
	}

	return osmocexfillertypes.OrderbookData{
		Symbol: string(data.Symbol),
		Bids:   bids,
		Asks:   asks,
	}
}

func (be *BybitExchange) updateBybitOrderbook(data wsbybit.V5WebsocketPublicOrderBookData) {
	orderbookAny, ok := be.orderbooks.Load(data.Symbol)
	if !ok {
		be.logger.Error("orderbook not found", zap.String("symbol", string(data.Symbol)))
		return
	}

	orderbook := orderbookAny.(osmocexfillertypes.OrderbookData)

	for i, bid := range data.Bids {
		if bid.Size == "0" {
			orderbook.Bids = append(orderbook.Bids[:i], orderbook.Bids[i+1:]...)
		} else {
			orderbook.Bids[i].Size = bid.Size
		}
	}

	for i, ask := range data.Asks {
		if ask.Size == "0" {
			orderbook.Asks = append(orderbook.Asks[:i], orderbook.Asks[i+1:]...)
		} else {
			orderbook.Asks[i].Size = ask.Size
		}
	}

	be.orderbooks.Store(data.Symbol, orderbook)
}

func (be *BybitExchange) getOsmoOrderbookForPair(pair osmocexfillertypes.Pair) (domain.CanonicalOrderBooksResult, error) {
	base := SymbolToInterchain[pair.Base]
	quote := SymbolToInterchain[pair.Quote]

	osmoPoolId, contractAddress, err := (*be.osmoPoolsUseCase).GetCanonicalOrderbookPool(base, quote)
	if err != nil {
		be.logger.Error("failed to get canonical orderbook pool", zap.Error(err))
		return domain.CanonicalOrderBooksResult{}, err
	}

	return domain.CanonicalOrderBooksResult{
		Base:            pair.Base,
		Quote:           pair.Quote,
		PoolID:          osmoPoolId,
		ContractAddress: contractAddress,
	}, nil
}

func (be *BybitExchange) getBybitOrderbookForPair(pair osmocexfillertypes.Pair) (osmocexfillertypes.OrderbookData, error) {
	orderbookAny, ok := be.orderbooks.Load(pair.String())
	if !ok {
		be.logger.Error("orderbook not found", zap.String("pair", pair.String()))
		return osmocexfillertypes.OrderbookData{}, errors.New("orderbook not found")
	}

	orderbook := orderbookAny.(osmocexfillertypes.OrderbookData)

	return orderbook, nil
}
