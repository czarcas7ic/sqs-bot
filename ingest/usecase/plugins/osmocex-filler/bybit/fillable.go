package bybit

import (
	"errors"

	"github.com/osmosis-labs/osmosis/osmomath"
	orderbookplugindomain "github.com/osmosis-labs/sqs/domain/orderbook/plugin"
	osmocexfillertypes "github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/types"

	"go.uber.org/zap"
)

// checks if the highest bid on osmo is higher than the lowest ask on bybit
func (be *BybitExchange) existsArbFromOsmo(pair osmocexfillertypes.Pair, bybitAsks []osmocexfillertypes.OrderBasicI, osmoBids []orderbookplugindomain.Order) bool {
	osmoHighestBid := osmoBids[0]
	bybitLowestAsk := bybitAsks[0]

	// get bybit lowest ask price converting to big dec
	bybitLowestAskPrice := osmomath.MustNewBigDecFromStr(bybitLowestAsk.GetPrice())

	// get osmo highest bid price from tick
	osmoHighestBidPrice, err := be.getUnscaledPriceForOrder(pair, osmoHighestBid)
	if err != nil {
		be.logger.Error("arbitrage from OSMOSIS: failed to get highest osmo bid price", zap.Error(err))
		return false
	}

	if !osmoHighestBidPrice.GT(bybitLowestAskPrice) {
		// no arb found
		be.logger.Info("arbitrage from OSMOSIS not found. Arbitrage offset is: ", zap.String("offset", bybitLowestAskPrice.Sub(osmoHighestBidPrice).String()))
		return false
	}

	be.logger.Info("arbitrage from OSMOSIS: found")

	return true
}

// existsArbFromBybit checks if the highest bid on bybit is higher than the lowest ask on osmo
func (be *BybitExchange) existsArbFromBybit(pair osmocexfillertypes.Pair, bybitBids []osmocexfillertypes.OrderBasicI, osmoAsks []orderbookplugindomain.Order) bool {
	bybitHighestBid := bybitBids[0]
	osmoLowestAsk := osmoAsks[0]

	// get bybit highest bid price converting to big dec
	bybitHighestBidPrice := osmomath.MustNewBigDecFromStr(bybitHighestBid.GetPrice())

	// get osmo lowest ask price from tick
	osmoLowestAskPrice, err := be.getUnscaledPriceForOrder(pair, osmoLowestAsk)
	if err != nil {
		be.logger.Error("arbitrage from BYBIT: failed to get lowest osmo ask price", zap.Error(err))
		return false
	}

	if !bybitHighestBidPrice.GT(osmoLowestAskPrice) {
		// no arb found
		be.logger.Info("arbitrage from BYBIT not found. Arbitrage offset is: ", zap.String("offset", osmoLowestAskPrice.Sub(bybitHighestBidPrice).String()))
		return false
	}

	be.logger.Info("arbitrage from BYBIT: found")
	return true
}

// getFillAmountAndDirection operates on orders found profitable, calculates the amount of profitable fill and the exchange from which to buy
// fillAmount refers to the amount of tokens that should be bought on asks side
func (be *BybitExchange) calculateFillAmount(
	pair osmocexfillertypes.Pair,
	bybitOrders []osmocexfillertypes.OrderBasicI,
	osmoOrders []orderbookplugindomain.Order,
) (fillAmount osmomath.BigDec, err error) {
	if len(bybitOrders) == 0 || len(osmoOrders) == 0 {
		return osmomath.NewBigDec(0), errors.New("empty orders")
	}

	curAskIndex := 0
	curBidIndex := 0

	switch bybitOrders[0].GetDirection() {
	case "bid": // highest bid on bybit > lowest ask on osmo -> buy from osmo, sell on bybit
		curAsk := &osmoOrders[curAskIndex]
		curBid := &bybitOrders[curBidIndex]

		curAskPrice, err := be.getUnscaledPriceForOrder(pair, *curAsk)
		if err != nil {
			be.logger.Error("failed to get unscaled price for osmo order", zap.Error(err))
			return osmomath.NewBigDec(0), err
		}

		curBidPrice := osmomath.MustNewBigDecFromStr((*curBid).GetPrice())
		for curAskPrice.LT(curBidPrice) {
			// fill the max(osmoOrder level, bybitOrder level)
			askAmount := osmomath.MustNewBigDecFromStr(curAsk.Quantity)
			bidAmount := osmomath.MustNewBigDecFromStr((*curBid).GetSize())

			// if ask's size is smaller than bid's size, fill the ask and move to the next ask, reduce bid's size accordingly
			// simulates a real trade
			if askAmount.LT(bidAmount) {
				fillAmount.AddMut(askAmount)
				(*curBid).SetSize(bidAmount.Sub(askAmount).String())
				curAskIndex++
			} else {
				fillAmount.AddMut(bidAmount)
				curAsk.Quantity = askAmount.Sub(bidAmount).String()
				curBidIndex++
			}

			curAsk = &osmoOrders[curAskIndex]
			curBid = &bybitOrders[curBidIndex]
		}

	case "ask": // lowest ask on bybit < highest bid on osmo -> buy from bybit, sell on osmo
		curAsk := &bybitOrders[curAskIndex]
		curBid := &osmoOrders[curBidIndex]

		curAskPrice := osmomath.MustNewBigDecFromStr((*curAsk).GetPrice())
		curBidPrice, err := be.getUnscaledPriceForOrder(pair, *curBid)
		if err != nil {
			be.logger.Error("failed to get unscaled price for osmo order", zap.Error(err))
			return osmomath.NewBigDec(0), err
		}

		for curAskPrice.LT(curBidPrice) {
			// fill the max(osmoOrder level, bybitOrder level)
			askAmount := osmomath.MustNewBigDecFromStr((*curAsk).GetSize())
			bidAmount := osmomath.MustNewBigDecFromStr(curBid.Quantity)

			// if ask's size is smaller than bid's size, fill the ask and move to the next ask, reduce bid's size accordingly
			// simulates a real trade
			if askAmount.LT(bidAmount) {
				fillAmount.AddMut(askAmount)
				curBid.Quantity = bidAmount.Sub(askAmount).String()
				curAskIndex++
			} else {
				fillAmount.AddMut(bidAmount)
				(*curAsk).SetSize(askAmount.Sub(bidAmount).String())
				curBidIndex++
			}

			curAsk = &bybitOrders[curAskIndex]
			curBid = &osmoOrders[curBidIndex]
		}
	default:
		be.logger.Error("invalid order direction", zap.String("direction", bybitOrders[0].GetDirection()))
		return osmomath.NewBigDec(0), errors.New("invalid order direction")
	}

	if fillAmount.IsZero() {
		be.logger.Info("no amount to fill")
		return osmomath.NewBigDec(0), errors.New("no amount to fill")
	}

	return
}

// adjPrice = price * 10^(baseDecimals-quoteDecimals)
// bybit "unscales" the price that was set at the time of limit order creation due to difference in tokens' precisions
func (be *BybitExchange) unscalePrice(price osmomath.BigDec, baseDenom, quoteDenom string) (osmomath.BigDec, error) {
	baseMetadata, err := (*be.osmoTokensUsecase).GetMetadataByChainDenom(SymbolToChainDenom[baseDenom])
	if err != nil {
		return osmomath.BigDec{}, err
	}

	quoteMetadata, err := (*be.osmoTokensUsecase).GetMetadataByChainDenom(SymbolToChainDenom[quoteDenom])
	if err != nil {
		return osmomath.BigDec{}, err
	}

	baseDecimals := baseMetadata.Precision
	quoteDecimals := quoteMetadata.Precision

	power := baseDecimals - quoteDecimals
	mul := true
	if power < 0 {
		mul = false
		power = -power
	}

	if mul {
		adjPrice := price.Mul(osmomath.NewBigDec(10).PowerInteger(uint64(power)))
		return adjPrice, nil
	} else {
		adjPrice := price.Quo(osmomath.NewBigDec(10).PowerInteger(uint64(power)))
		return adjPrice, nil
	}
}
