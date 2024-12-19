package osmocexfiller

import (
	"context"

	"github.com/osmosis-labs/sqs/domain"
	orderbookdomain "github.com/osmosis-labs/sqs/domain/orderbook"
	orderbookplugindomain "github.com/osmosis-labs/sqs/domain/orderbook/plugin"
)

func (oc *osmocexFillerIngestPlugin) fetchTicksForOrderbook(ctx context.Context, orderbook domain.CanonicalOrderBooksResult) error {
	orderBookPool, err := oc.poolsUseCase.GetPool(orderbook.PoolID)
	if err != nil {
		return err
	}

	ticks := orderBookPool.GetSQSPoolModel().CosmWasmPoolModel.Data.Orderbook.Ticks

	orderResult := orderbookplugindomain.OrdersResponse{
		AskOrders: []orderbookdomain.Order{},
		BidOrders: []orderbookdomain.Order{},
	}
	for _, tick := range ticks {
		orders, err := oc.orderbookCWAPIClient.GetOrdersByTick(ctx, orderbook.ContractAddress, tick.TickId)
		if err != nil {
			continue
		}

		for _, order := range orders {
			// Process order
			if order.OrderDirection == "ask" {
				orderResult.AskOrders = append(orderResult.AskOrders, order)
			} else {
				orderResult.BidOrders = append(orderResult.BidOrders, order)
			}
		}
	}

	// Store the orderbook orders
	oc.orderMapByPoolID.Store(orderbook.PoolID, orderResult)

	return nil
}
