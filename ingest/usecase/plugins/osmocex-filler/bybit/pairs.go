package bybit

import osmocexfillertypes "github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/types"

var (
	ArbPairs = []osmocexfillertypes.Pair{
		{Base: "BTC", Quote: "USDC"},
		// {Base: "ETH", Quote: "USDC"},
		// {Base: "ATOM", Quote: "USDC"},
	}

	SymbolToChainDenom = map[string]string{
		"USDC": "ibc/498A0751C798A0D9A389AA3691123DADA57DAA4FE165D5C75894505B876BA6E4",                    // USDC
		"BTC":  "factory/osmo1z6r6qdknhgsc0zeracktgpcxf43j6sekq07nw8sxduc9lg0qjjlqfu25e3/alloyed/allBTC",  // alloyed bitcoin
		"USDT": "factory/osmo1em6xs47hd82806f5cxgyufguxrrc7l0aqx7nzzptjuqgswczk8csavdxek/alloyed/allUSDT", // alloyed USDT
		"ETH":  "factory/osmo1k6c8jln7ejuqwtqmay3yvzrg3kueaczl96pk067ldg8u835w0yhsw27twm/alloyed/allETH",  // alloyed ethereum
		"ATOM": "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2",                    // ATOM
	}
)
