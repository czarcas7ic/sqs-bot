package bybit

import (
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
			Price:  bid.Price,
			Amount: bid.Size,
		})
	}

	for _, ask := range data.Asks {
		asks = append(asks, osmocexfillertypes.OrderbookEntry{
			Price:  ask.Price,
			Amount: ask.Size,
		})
	}

	return osmocexfillertypes.OrderbookData{
		Symbol: string(data.Symbol),
		Bids:   bids,
		Asks:   asks,
	}
}

func (be *BybitExchange) getOrderbookForPair(pair osmocexfillertypes.Pair) (domain.CanonicalOrderBooksResult, error) {
	base := SymbolToInterchain[pair.Base]
	quote := SymbolToInterchain[pair.Quote]

	osmoPoolId, contractAddress, err := be.osmoPoolsUseCase.GetCanonicalOrderbookPool(base, quote)
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
