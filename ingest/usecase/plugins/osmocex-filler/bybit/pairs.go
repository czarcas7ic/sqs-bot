package bybit

import osmocexfillertypes "github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/types"

var (
	pairs = []osmocexfillertypes.Pair{
		{Base: "BTC", Quote: "USDC"},
		{Base: "ETH", Quote: "USDC"},
		{Base: "ATOM", Quote: "USDC"},
		{Base: "TIA", Quote: "USDC"},
		{Base: "SOL", Quote: "USDC"},
		{Base: "INJ", Quote: "USDC"},
		// {Base: "OSMO", Quote: "USDT"}, // perp contract
	}
)
