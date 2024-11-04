package bybit

import (
	"github.com/osmosis-labs/osmosis/osmomath"
	osmocexfillertypes "github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/types"

	"go.uber.org/zap"
)

func (be *BybitExchange) existsArbitrageOpportunity(bybitOrders osmocexfillertypes.Orders, osmoPrice osmomath.BigDec, baseDecimals, quoteDecimals int) *ArbitrageDirection {
	switch bybitOrders.Direction() {
	case osmocexfillertypes.BID: // bid should be greater than osmo price (buy from osmo, sell on bybit)
		bybitHighestBid := bybitOrders[0]
		bybitHighestBidPrice := osmomath.MustNewBigDecFromStr(bybitHighestBid.GetPrice())

		if bybitHighestBidPrice.GT(osmoPrice) {
			// bids' sizes are in quote tokens
			fillAmountQuote := computeFillableAmount(bybitOrders, osmoPrice)
			fillAmountBase := fillAmountQuote.Quo(osmoPrice)

			scaleBigDecDecimals(&fillAmountBase, baseDecimals-quoteDecimals)

			return NewArbitrageDirection(OSMO, BYBIT, fillAmountBase, fillAmountQuote)
		}

		be.logger.Info("no arbitrage opportunity found", zap.Any("bybitHighestBidPrice", bybitHighestBidPrice), zap.Any("osmoPrice", osmoPrice))

		return nil
	case osmocexfillertypes.ASK: // ask should be less than osmo price (buy from bybit, sell on osmo)
		bybitLowestAsk := bybitOrders[0]
		bybitLowestAskPrice := osmomath.MustNewBigDecFromStr(bybitLowestAsk.GetPrice())

		if bybitLowestAskPrice.LT(osmoPrice) {
			fillAmountBase := computeFillableAmount(bybitOrders, osmoPrice)
			fillAmountQuote := fillAmountBase.Mul(osmoPrice)

			scaleBigDecDecimals(&fillAmountQuote, quoteDecimals-baseDecimals)

			return NewArbitrageDirection(BYBIT, OSMO, fillAmountBase, fillAmountQuote)
		}

		be.logger.Info("no arbitrage opportunity found", zap.Any("bybitLowestAskPrice", bybitLowestAskPrice), zap.Any("osmoPrice", osmoPrice))
		return nil
	}

	be.logger.Error("invalid orders passed", zap.Any("bybitOrders", bybitOrders))
	panic("invalid orders passed")
}

func computeFillableAmount(bybitOrders osmocexfillertypes.Orders, price osmomath.BigDec) osmomath.BigDec {
	var order osmocexfillertypes.OrderBasicI

	fillAmount := osmomath.ZeroBigDec()
	index := 0
	isBid := bybitOrders.Direction() == osmocexfillertypes.BID

	for index < len(bybitOrders) {
		order = bybitOrders[index]
		orderPrice := osmomath.MustNewBigDecFromStr(order.GetPrice())

		if orderPrice.GT(price) && isBid { // for bids
			fillAmount.AddMut(osmomath.MustNewBigDecFromStr(order.GetSize()))
		} else if isBid { // if bid and price is not higher
			break
		}

		if orderPrice.LT(price) && !isBid { // for asks
			fillAmount.AddMut(osmomath.MustNewBigDecFromStr(order.GetSize()))
		} else if !isBid { // if ask and price is not lower
			break
		}

		index++
	}
	return fillAmount
}
