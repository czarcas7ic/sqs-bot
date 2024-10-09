package bybit

import (
	"errors"

	"github.com/osmosis-labs/osmosis/osmomath"
	clmath "github.com/osmosis-labs/osmosis/v25/x/concentrated-liquidity/math"
	orderbookplugindomain "github.com/osmosis-labs/sqs/domain/orderbook/plugin"
	"github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/cex"
)

func (be *BybitExchange) searchArbFromOsmo(thisAsks []cex.OrderbookEntry, osmoBids []orderbookplugindomain.Order) error {
	osmoHighestBid := osmoBids[0]
	thisLowestAsk := thisAsks[0]

	// get osmo highest bid price from tick
	osmoHighestBidPrice, err := clmath.TickToPrice(osmoHighestBid.TickId)
	if err != nil {
		return err
	}

	// get this lowest ask price converting to big dec
	thisLowestAskPrice, err := osmomath.NewBigDecFromStr(thisLowestAsk.Price)
	if err != nil {
		return err
	}

	if !osmoHighestBidPrice.GT(thisLowestAskPrice) {
		// no arb found
		return errors.New("no arb found")
	}

	return nil
}

func (be *BybitExchange) searchArbFromThis(thisBids []cex.OrderbookEntry, osmoAsks []orderbookplugindomain.Order) error {
	thisHighestBid := thisBids[0]
	osmoLowestAsk := osmoAsks[0]

	// get this highest bid price converting to big dec
	thisHighestBidPrice, err := osmomath.NewBigDecFromStr(thisHighestBid.Price)
	if err != nil {
		return err
	}

	// get osmo lowest ask price from tick
	osmoLowestAskPrice, err := clmath.TickToPrice(osmoLowestAsk.TickId)
	if err != nil {
		return err
	}

	if !thisHighestBidPrice.GT(osmoLowestAskPrice) {
		// no arb found
		return errors.New("no arb found")
	}

	return nil
}
