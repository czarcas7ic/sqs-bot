package orderbookplugindomain

import "sort"

// Order represents an order in the orderbook returned by the orderbook contract.
type Order struct {
	TickId         int64  `json:"tick_id"`
	OrderId        int64  `json:"order_id"`
	OrderDirection string `json:"order_direction"`
	Owner          string `json:"owner"`
	Quantity       string `json:"quantity"`
	Etas           string `json:"etas"`
	ClaimBounty    string `json:"claim_bounty"`
	PlacedQuantity string `json:"placed_quantity"`
	PlacedAt       string `json:"placed_at"`
}

// OrdersResponse represents the response from the orderbook contract containing the orders for a given tick.
type OrdersResponse struct {
	Address   string  `json:"address"`
	BidOrders []Order `json:"bid_orders"`
	AskOrders []Order `json:"ask_orders"`
}

func (or OrdersResponse) BidsDescending() []Order {
	bids := make([]Order, len(or.BidOrders))
	copy(bids, or.BidOrders)

	sort.Slice(bids, func(i, j int) bool {
		return bids[i].TickId > bids[j].TickId
	})

	return bids
}

func (or OrdersResponse) AsksAscending() []Order {
	asks := make([]Order, len(or.AskOrders))
	copy(asks, or.AskOrders)

	sort.Slice(asks, func(i, j int) bool {
		return asks[i].TickId < asks[j].TickId
	})

	return asks
}
