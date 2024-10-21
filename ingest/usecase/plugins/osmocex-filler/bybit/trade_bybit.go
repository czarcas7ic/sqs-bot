package bybit

import (
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
	side := "Buy"
	denomInterchain := pair.QuoteInterchainDenom()
	denomHuman := pair.Quote
	if _type == osmocexfillertypes.SELL {
		side = "Sell"
		denomInterchain = pair.BaseInterchainDenom()
		denomHuman = pair.Base
	}

	denomDecimals, err := be.getInterchainDenomDecimals(denomInterchain)
	if err != nil {
		be.logger.Error("failed to get interchain denom decimals", zap.Error(err))
		return
	}

	// unscale the quantity back
	qty = qty.Quo(osmomath.NewBigDec(10).Power(osmomath.NewBigDec(int64(denomDecimals))))

	// bybit has a limit on the number of decimals for quantity
	// denomToDecimals is a map from a token to the number of decimals bybit's api expects (maximum)
	qtyString := keepXPrecision(&qty, be.arbitrageConfig.denomToDecimals[denomHuman])

	order := be.httpclient.NewPlaceOrderService(DEFAULT_CATEGORY, pair.String(), side, DEFAULT_ORDER_TYPE, qtyString)
	// fmt.Println("order params: ", DEFAULT_CATEGORY, pair.String(), side, DEFAULT_ORDER_TYPE, qtyString, qty.String())
	result, err := order.Do(be.ctx)
	if err != nil {
		be.logger.Error("failed to place spot order", zap.Error(err))
		return
	}

	be.logger.Info("spot order placed", zap.String("result", bybit.PrettyPrint(result)))
}

// CONTRACT: x <= osmomath.BigDecPrecision
func keepXPrecision(bd *osmomath.BigDec, x int) string {
	strbd := bd.String()
	droppedDecimals := osmomath.BigDecPrecision - x

	strbd = strbd[:len(strbd)-droppedDecimals]
	return strbd
}
