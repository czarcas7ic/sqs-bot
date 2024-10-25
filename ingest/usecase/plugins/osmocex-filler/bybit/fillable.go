package bybit

import (
	"github.com/osmosis-labs/osmosis/osmomath"
	osmocexfillertypes "github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/types"

	"go.uber.org/zap"
)

var (
	bps = osmomath.NewBigDec(10000)
)

func (be *BybitExchange) existsArbitrageOpportunity(bybitOrders osmocexfillertypes.Orders, osmoPrice osmomath.BigDec, baseDecimals, quoteDecimals int) *ArbitrageDirection {
	switch bybitOrders.Direction() {
	case osmocexfillertypes.BID: // bid should be greater than osmo price (buy from osmo, sell on bybit)
		bybitHighestBid := bybitOrders[0]
		bybitHighestBidPrice := osmomath.MustNewBigDecFromStr(bybitHighestBid.GetPrice())

		if bybitHighestBidPrice.GT(osmoPrice) {
			// WARN:
			// - fillAmountBase is returned with base's precision, upstream must unscale it (when asks on bybit)
			fillAmountBase := computeFillableAmount(bybitOrders, osmoPrice)
			fillAmountQuote := fillAmountBase.Mul(osmoPrice)

			scaleBigDecDecimals(&fillAmountBase, baseDecimals)
			scaleBigDecDecimals(&fillAmountQuote, quoteDecimals)

			return NewArbitrageDirection(OSMO, BYBIT, fillAmountBase, fillAmountQuote)
		}

		return nil
	case osmocexfillertypes.ASK: // ask should be less than osmo price (buy from bybit, sell on osmo)
		bybitLowestAsk := bybitOrders[0]
		bybitLowestAskPrice := osmomath.MustNewBigDecFromStr(bybitLowestAsk.GetPrice())

		if bybitLowestAskPrice.LT(osmoPrice) {
			fillAmountQuote := computeFillableAmount(bybitOrders, osmoPrice)
			fillAmountBase := fillAmountQuote.Quo(osmoPrice)

			scaleBigDecDecimals(&fillAmountBase, baseDecimals)
			scaleBigDecDecimals(&fillAmountQuote, quoteDecimals)

			return NewArbitrageDirection(BYBIT, OSMO, fillAmountBase, fillAmountQuote)
		}

		return nil
	}

	be.logger.Error("invalid orders passed", zap.Any("bybitOrders", bybitOrders))
	panic("invalid orders passed")
}

func computeFillableAmount(bybitOrders osmocexfillertypes.Orders, price osmomath.BigDec) osmomath.BigDec {
	fillAmount := osmomath.ZeroBigDec()
	switch bybitOrders.Direction() {
	case osmocexfillertypes.BID: // bids should be greater than osmo price
		order := bybitOrders[0]
		for orderPrice := osmomath.MustNewBigDecFromStr(order.GetPrice()); orderPrice.GT(price); {
			fillAmount.AddMut(osmomath.MustNewBigDecFromStr(order.GetSize()))
		}
	case osmocexfillertypes.ASK: // asks should be less than osmo price
		order := bybitOrders[0]
		for orderPrice := osmomath.MustNewBigDecFromStr(order.GetPrice()); orderPrice.LT(price); {
			fillAmount.AddMut(osmomath.MustNewBigDecFromStr(order.GetSize()))
		}
	}

	return fillAmount
}
