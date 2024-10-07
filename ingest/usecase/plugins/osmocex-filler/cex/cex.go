package cex

import "github.com/osmosis-labs/sqs/domain"

type CExchangeI interface {
	// ProcessOrderbook processes a specific orderbook from this centralized exchange against the osmosis's orderbook
	ProcessOrderbook(osmoData domain.CanonicalOrderBooksResult) error
	// RegisterPair adds a pair to the list of arb-able pairs
	RegisterPair(pair Pair) error

	SupportedPair(pair Pair) bool
}

type Pair struct {
	Base  string
	Quote string
}

func (p Pair) String() string {
	return p.Base + p.Quote
}
