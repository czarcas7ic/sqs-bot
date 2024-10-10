package osmocexfillertypes

type ExchangeI interface {
	// ProcessOrderbook acknowledges the osmosis orderbook
	// Each exchange implements this acknowledgement differently
	// ProcessOrderbook(osmoData domain.CanonicalOrderBooksResult) error

	// RegisterPair adds a pair of tokens to the list of arb-able pairs
	RegisterPair(pair Pair) error

	// SupportedPair returns true if the pair is supported by the exchange
	// SupportedPair(pair Pair) bool

	// Signal signals the websocket callback to start matching orderbooks
	// Called at the beginning of each block
	Signal()
}
