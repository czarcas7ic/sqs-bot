package bybit

import osmocexfillertypes "github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/types"

var (
	ArbPairs = []osmocexfillertypes.Pair{
		{Base: "BTC", Quote: "USDC"},
	}

	SymbolToChainDenom = map[string]string{
		"BTC":  "factory/osmo1z0qrq605sjgcqpylfl4aa6s90x738j7m58wyatt0tdzflg2ha26q67k743/wbtc", // WBTC
		"USDC": "ibc/498A0751C798A0D9A389AA3691123DADA57DAA4FE165D5C75894505B876BA6E4",         // USDC
	}
)
