package bybit

import (
	"errors"

	wsbybit "github.com/hirokisan/bybit/v2"
	"github.com/osmosis-labs/sqs/domain"
	osmocexfillertypes "github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/types"
	"go.uber.org/zap"
)

func parseBybitOrderbook(data wsbybit.V5WebsocketPublicOrderBookData) *osmocexfillertypes.OrderbookData {
	bids := make(map[string]string)
	asks := make(map[string]string)

	for _, bid := range data.Bids {
		bids[bid.Price] = bid.Size
	}

	for _, ask := range data.Asks {
		asks[ask.Price] = ask.Size
	}

	return osmocexfillertypes.NewOrderbookData(string(data.Symbol), bids, asks)
}

func (be *BybitExchange) updateBybitOrderbook(data wsbybit.V5WebsocketPublicOrderBookData) {
	orderbookAny, ok := be.orderbooks.Load(data.Symbol)
	if !ok {
		be.logger.Error("orderbook not found", zap.String("symbol", string(data.Symbol)))
		return
	}

	orderbook := orderbookAny.(*osmocexfillertypes.OrderbookData)

	for _, bid := range data.Bids {
		if bid.Size == "0" {
			orderbook.RemoveBid(bid.Price)
		} else {
			orderbook.SetBid(bid.Price, bid.Size)
		}
	}

	for _, ask := range data.Asks {
		if ask.Size == "0" {
			orderbook.RemoveAsk(ask.Price)
		} else {
			orderbook.SetAsk(ask.Price, ask.Size)
		}
	}

	be.orderbooks.Store(string(data.Symbol), orderbook)
}

func (be *BybitExchange) getOsmoOrderbookForPair(pair osmocexfillertypes.Pair) (domain.CanonicalOrderBooksResult, error) {
	base := SymbolToChainDenom[pair.Base]
	quote := SymbolToChainDenom[pair.Quote]

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

func (be *BybitExchange) getBybitOrderbookForPair(pair osmocexfillertypes.Pair) (*osmocexfillertypes.OrderbookData, error) {
	orderbookAny, ok := be.orderbooks.Load(pair.String())
	if !ok {
		be.logger.Error("orderbook not found", zap.String("pair", pair.String()))
		return nil, errors.New("orderbook not found")
	}

	orderbook := orderbookAny.(*osmocexfillertypes.OrderbookData)

	return orderbook, nil
}
