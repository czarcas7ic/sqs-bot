package bybit

import osmocexfillertypes "github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/types"

var (
	ArbPairs = []osmocexfillertypes.Pair{
		{Base: "BTC", Quote: "USDC"},
	}

	SymbolToChainDenom = map[string]string{
		"BTC":  "factory/osmo1z6r6qdknhgsc0zeracktgpcxf43j6sekq07nw8sxduc9lg0qjjlqfu25e3/alloyed/allBTC", // alloyed bitcoin
		"USDC": "ibc/498A0751C798A0D9A389AA3691123DADA57DAA4FE165D5C75894505B876BA6E4",                   // USDC
	}
)
