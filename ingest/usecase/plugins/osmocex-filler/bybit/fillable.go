package bybit

import (
	"errors"
	"fmt"

	"github.com/osmosis-labs/osmosis/osmomath"
	clmath "github.com/osmosis-labs/osmosis/v25/x/concentrated-liquidity/math"
	orderbookplugindomain "github.com/osmosis-labs/sqs/domain/orderbook/plugin"
	osmocexfillertypes "github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/types"
)

func (be *BybitExchange) checkArbFromOsmo(pair osmocexfillertypes.Pair, thisAsks []osmocexfillertypes.OrderbookEntry, osmoBids []orderbookplugindomain.Order) error {
	osmoHighestBid := osmoBids[0]
	thisLowestAsk := thisAsks[0]

	// get osmo highest bid price from tick
	osmoHighestBidPrice, _ := clmath.TickToPrice(osmoHighestBid.TickId)

	// unscale osmoHighestBidPrice
	osmoHighestBidPrice, err := be.unscalePrice(osmoHighestBidPrice, pair.Base, pair.Quote)
	if err != nil {
		panic(err)
	}

	// get this lowest ask price converting to big dec
	thisLowestAskPrice, err := osmomath.NewBigDecFromStr(thisLowestAsk.Price)
	if err != nil {
		return err
	}

	fmt.Println(osmoHighestBidPrice.String(), thisLowestAskPrice.String())
	if !osmoHighestBidPrice.GT(thisLowestAskPrice) {
		// no arb found
		return errors.New("no arb found")
	}

	return nil
}

func (be *BybitExchange) checkArbFromThis(pair osmocexfillertypes.Pair, thisBids []osmocexfillertypes.OrderbookEntry, osmoAsks []orderbookplugindomain.Order) {
	thisHighestBid := thisBids[0]
	osmoLowestAsk := osmoAsks[0]

	// get this highest bid price converting to big dec
	thisHighestBidPrice := osmomath.MustNewBigDecFromStr(thisHighestBid.Price)

	// get osmo lowest ask price from tick
	osmoLowestAskPrice, _ := clmath.TickToPrice(osmoLowestAsk.TickId)

	// unscale osmoLowestAskPrice
	osmoLowestAskPrice, err := be.unscalePrice(osmoLowestAskPrice, pair.Base, pair.Quote)
	if err != nil {
		panic(err)
	}

	fmt.Println(osmoLowestAskPrice.String(), thisHighestBidPrice.String())
	if !thisHighestBidPrice.GT(osmoLowestAskPrice) {
		// no arb found
		return
	}
}

// adjPrice = price * 10^(baseDecimals-quoteDecimals)
// this "unscales" the price that was set at the time of limit order creation due to difference in tokens' precisions
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
