package osmocexfillertypes

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/osmosis-labs/osmosis/osmomath"
)

type ExchangeI interface {
	// RegisterPair adds a pair of tokens to the list of arb-able pairs
	RegisterPairs(ctx context.Context) error

	// Signal starts the arb process at the beginning of each block
	Signal()

	// GetBotBalances returns the balances of the bot on exchange and on osmosis
	GetBotBalances() (map[string]CoinBalanceI, sdk.Coins, error)
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

// CoinBalanceI is an interface for a coin returned when querying exchange balance
type CoinBalanceI interface {
	Balance() string
	BigDecBalance(precision int) osmomath.BigDec

	Token() string
}
