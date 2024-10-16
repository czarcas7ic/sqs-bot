package bybit

import (
	"context"
	"fmt"

	"github.com/osmosis-labs/osmosis/osmomath"
	"go.uber.org/zap"
)

const (
	MINIMUM_FILL_VALUE          = 10
	MINIMUM_PRICE_DEVIATION_BPS = 1 // 0.01%
)

// custom min amounts
// https://www.bybit.com/en/announcement-info/spot-trading-rules/
// var (
// 	minBTCAmount = osmomath.MustNewBigDecFromStr("19800") // 15k satoshis
// )

// var (
// 	// TODO: make human-readable
// 	customMinFillAmounts = map[string]osmomath.BigDec{
// 		"factory/osmo1z6r6qdknhgsc0zeracktgpcxf43j6sekq07nw8sxduc9lg0qjjlqfu25e3/alloyed/allBTC": minBTCAmount, // allBTC -> 15k satoshis minimum for bybit api
// 	}
// )

type arbitrageConfig struct {
	// minFillAmounts is a map from a pair of tokens to the minimum fill amount in base tokens
	minFillAmounts map[string]osmomath.BigDec
	// minimumPriceDeviation is the minimum price deviation (in basis points) between the two exchanges for the bot to proceed with the trade
	minimumPriceDeviationBPS int
}

func (ac *arbitrageConfig) getMinFillAmount(denom string) (osmomath.BigDec, bool) {
	if amount, ok := ac.minFillAmounts[denom]; ok {
		return amount, true
	}

	return osmomath.NewBigDec(-1), false
}

// initConfig initializes the config for the bybit exchange
func (be *BybitExchange) initConfig() {
	if be.arbitrageConfig != nil {
		panic("config already initialized")
	}

	minFillAmounts := make(map[string]osmomath.BigDec)
	for _, pair := range pairs {
		params := map[string]interface{}{"category": "spot", "symbol": pair.String()}
		serverResult, err := be.httpclient.NewUtaBybitServiceWithParams(params).GetInstrumentInfo(context.Background())
		if err != nil {
			be.logger.Error("failed to get instrument info", zap.Error(err))
			panic(err)
		}

		result := serverResult.Result.(map[string]interface{})
		if result["list"] != nil {
			list := result["list"].([]interface{})
			if len(list) != 0 {
				base := list[0].(map[string]interface{})
				if base["lotSizeFilter"] != nil {
					lotSizeFilter := base["lotSizeFilter"].(map[string]interface{})
					minQty := osmomath.MustNewBigDecFromStr(lotSizeFilter["minOrderQty"].(string))
					baseDecimals, err := be.getInterchainDenomDecimals(pair.BaseInterchainDenom())
					if err != nil {
						be.logger.Error("failed to get interchain denom decimals", zap.Error(err))
						panic(err)
					}
					
					addBigDecPrecision(&minQty, baseDecimals)

					minFillAmounts[pair.BaseInterchainDenom()] = minQty
					continue
				}
			}
		}

		panic(fmt.Sprintf("failed to get min order qty for pair {%s}", pair.String()))
	}

	be.arbitrageConfig = &arbitrageConfig{
		minFillAmounts:           minFillAmounts,
		minimumPriceDeviationBPS: MINIMUM_PRICE_DEVIATION_BPS,
	}
}
