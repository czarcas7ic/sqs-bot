package osmocexfillertypes

import "context"

type ExchangeI interface {
	// RegisterPair adds a pair of tokens to the list of arb-able pairs
	RegisterPairs(ctx context.Context) error

	// Signal signals the websocket callback to start matching orderbooks
	// Called at the beginning of each block
	Signal()
}
