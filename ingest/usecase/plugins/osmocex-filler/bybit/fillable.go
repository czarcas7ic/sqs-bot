package bybit

import (
	"fmt"

	"github.com/osmosis-labs/osmosis/osmomath"
	clmath "github.com/osmosis-labs/osmosis/v25/x/concentrated-liquidity/math"
	orderbookplugindomain "github.com/osmosis-labs/sqs/domain/orderbook/plugin"
	osmocexfillertypes "github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/types"

	"go.uber.org/zap"
)

// checks if the highest bid on osmo is higher than the lowest ask on bybit
func (be *BybitExchange) existsArbFromOsmo(pair osmocexfillertypes.Pair, bybitAsks []osmocexfillertypes.OrderBasicI, osmoBids []orderbookplugindomain.Order) bool {
	osmoHighestBid := osmoBids[0]
	bybitLowestAsk := bybitAsks[0]

	// get osmo highest bid price from tick
	osmoHighestBidPrice, _ := clmath.TickToPrice(osmoHighestBid.TickId)

	// unscale osmoHighestBidPrice
	osmoHighestBidPrice, err := be.unscalePrice(osmoHighestBidPrice, pair.Base, pair.Quote)
	if err != nil {
		be.logger.Error("Arb from OSMOSIS: unscaling error:", zap.Error(err))
		return false
	}

	// get bybit lowest ask price converting to big dec
	bybitLowestAskPrice, err := osmomath.NewBigDecFromStr(bybitLowestAsk.GetPrice())
	if err != nil {
		be.logger.Error("Arb from OSMOSIS: parsing error:", zap.Error(err))
		return false
	}

	fmt.Println(osmoHighestBidPrice.String(), bybitLowestAskPrice.String())
	if !osmoHighestBidPrice.GT(bybitLowestAskPrice) {
		// no arb found
		be.logger.Info("Arb from OSMOSIS: not found")
		return false
	}

	be.logger.Info("Arb from OSMOSIS: found")

	return true
}

func (be *BybitExchange) existsArbFromBybit(pair osmocexfillertypes.Pair, bybitBids []osmocexfillertypes.OrderBasicI, osmoAsks []orderbookplugindomain.Order) bool {
	bybitHighestBid := bybitBids[0]
	osmoLowestAsk := osmoAsks[0]

	// get bybit highest bid price converting to big dec
	bybitHighestBidPrice := osmomath.MustNewBigDecFromStr(bybitHighestBid.GetPrice())

	// get osmo lowest ask price from tick
	osmoLowestAskPriceUnscaled, err := clmath.TickToPrice(osmoLowestAsk.TickId)
	if err != nil {
		be.logger.Error("Arb from BYBIT: tick to price error:", zap.Error(err))
		return false
	}

	// unscale osmoLowestAskPrice
	osmoLowestAskPrice, err := be.unscalePrice(osmoLowestAskPriceUnscaled, pair.Base, pair.Quote)
	if err != nil {
		be.logger.Error("Arb from BYBIT: unscaling error:", zap.Error(err))
		return false
	}

	fmt.Println(osmoLowestAskPrice.String(), bybitHighestBidPrice.String())
	if !bybitHighestBidPrice.GT(osmoLowestAskPrice) {
		// no arb found
		be.logger.Info("Arb from BYBIT: not found")
		return false
	}

	be.logger.Info("Arb from BYBIT: found")
	return true
}

// func (be *BybitExchange)

// adjPrice = price * 10^(baseDecimals-quoteDecimals)
// bybit "unscales" the price that was set at the time of limit order creation due to difference in tokens' precisions
func (be *BybitExchange) unscalePrice(price osmomath.BigDec, baseDenom, quoteDenom string) (osmomath.BigDec, error) {
	baseMetadata, err := (*be.osmoTokensUseCase).GetMetadataByChainDenom(SymbolToChainDenom[baseDenom])
	if err != nil {
		return osmomath.BigDec{}, err
	}

	quoteMetadata, err := (*be.osmoTokensUseCase).GetMetadataByChainDenom(SymbolToChainDenom[quoteDenom])
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
