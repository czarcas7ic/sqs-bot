package bybit

import (
	osmocexfillertypes "github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/types"
	bybit "github.com/wuhewuhe/bybit.go.api"
	"go.uber.org/zap"
)

const (
	DEFAULT_CATEGORY   = "spot"
	DEFAULT_ORDER_TYPE = "Market"
)

// spot executes a spot trade on bybit
func (be *BybitExchange) spot(pair osmocexfillertypes.Pair, _type osmocexfillertypes.TradeType, qty string) {
	side := "Buy"

	if _type == osmocexfillertypes.SELL {
		side = "Sell"
	}

	order := be.httpclient.NewPlaceOrderService(DEFAULT_CATEGORY, pair.String(), side, DEFAULT_ORDER_TYPE, qty)
	result, err := order.Do(be.ctx)
	if err != nil {
		be.logger.Error("failed to place spot order", zap.Error(err))
		return
	}

	be.logger.Info("spot order placed", zap.String("result", bybit.PrettyPrint(result)))
}
