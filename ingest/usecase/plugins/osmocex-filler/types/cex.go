package osmocexfillertypes

import (
	"context"
)

type ExchangeI interface {
	// RegisterPair adds a pair of tokens to the list of arb-able pairs
	RegisterPairs(ctx context.Context) error

	// Signal starts the arb process at the beginning of each block
	Signal()
}

type OrderBasicI interface {
	GetPrice() string
	GetSize() string
	GetDirection() string

	SetSize(string)
}

type ExchangeType int

const (
	OSMO ExchangeType = iota
	BYBIT
)

// type OrderbookBasicI interface {
// 	Asks() []OrderBasicI
// 	Bids() []OrderBasicI

// 	SetAsk(price, size string)
// 	SetBid(price, size string)

// 	RemoveAsk(price string)
// 	RemoveBid(price string)

// 	AsksAscending() []OrderBasicI
// 	BidsDescending() []OrderBasicI
// }

// var _ OrderbookBasicI = (*orderbookplugindomain.OrdersResponse)(nil)
