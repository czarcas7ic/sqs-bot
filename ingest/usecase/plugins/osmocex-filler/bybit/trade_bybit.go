package bybit

import (
	"fmt"

	"github.com/osmosis-labs/osmosis/osmomath"
	osmocexfillertypes "github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/types"
	bybit "github.com/wuhewuhe/bybit.go.api"
	"go.uber.org/zap"
)

const (
	DEFAULT_CATEGORY   = "spot"
	DEFAULT_ORDER_TYPE = "Market"
)

// spot executes a spot trade on bybit
// quantity is in quote tokens for sells and in base tokens for buys
func (be *BybitExchange) spot(pair osmocexfillertypes.Pair, _type osmocexfillertypes.TradeType, qty osmomath.BigDec) {
	decimals, err := be.getInterchainDenomDecimals(pair.Quote)
	if err != nil {
		be.logger.Error("failed to get interchain denom decimals", zap.Error(err))
		return
	}

	side := "Buy"
	if _type == osmocexfillertypes.SELL {
		side = "Sell"
		decimals, err = be.getInterchainDenomDecimals(pair.Base)
		if err != nil {
			be.logger.Error("failed to get interchain denom decimals", zap.Error(err))
			return
		}
	}

	// unscale the quantity back
	qty = qty.Quo(osmomath.NewBigDec(10).Power(osmomath.NewBigDec(int64(decimals))))
	quantityFloat, err := qty.Float64()
	if err != nil {
		be.logger.Error("failed to convert quantity to float64", zap.Error(err))
		return
	}

	quantityString := fmt.Sprintf("%f", quantityFloat) // convert to float because of bybit's api error of too many decimals

	order := be.httpclient.NewPlaceOrderService(DEFAULT_CATEGORY, pair.String(), side, DEFAULT_ORDER_TYPE, quantityString)
	// fmt.Println("order params: ", DEFAULT_CATEGORY, pair.String(), side, DEFAULT_ORDER_TYPE, quantityString, qty.String())
	result, err := order.Do(be.ctx)
	if err != nil {
		be.logger.Error("failed to place spot order", zap.Error(err))
		return
	}

	be.logger.Info("spot order placed", zap.String("result", bybit.PrettyPrint(result)))
}
