package bybit

import (
	"context"
	"fmt"

	"github.com/osmosis-labs/osmosis/osmomath"
	"go.uber.org/zap"
)

const (
	MINIMUM_FILL_VALUE = 10
)

var (
	DEFAULT_MINIMUM_PRICE_DEVIATION_BPS = osmomath.NewBigDec(60) // 0.60%
)

type arbitrageConfig struct {
	// minFillAmounts is a map from a pair of tokens to the minimum fill amount in base tokens
	minFillAmounts map[string]osmomath.BigDec
	// denomToDecimals is a map from a token to the number of decimals bybit's api expects (maximum)
	denomToDecimals map[string]int
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
	denomToDecimals := make(map[string]int)
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
				listdata := list[0].(map[string]interface{})
				if listdata["lotSizeFilter"] != nil {
					// set min fill amount
					lotSizeFilter := listdata["lotSizeFilter"].(map[string]interface{})

					// for base
					minQty := osmomath.MustNewBigDecFromStr(lotSizeFilter["minOrderQty"].(string))
					baseDecimals, err := be.getInterchainDenomDecimals(pair.BaseInterchainDenom())
					if err != nil {
						be.logger.Error("failed to get interchain denom decimals", zap.Error(err))
						panic(err)
					}

					addBigDecDecimals(&minQty, baseDecimals)

					// for quote
					minAmt := osmomath.MustNewBigDecFromStr(lotSizeFilter["minOrderAmt"].(string))
					quoteDecimals, err := be.getInterchainDenomDecimals(pair.QuoteInterchainDenom())
					if err != nil {
						be.logger.Error("failed to get interchain denom decimals", zap.Error(err))
						panic(err)
					}

					addBigDecDecimals(&minAmt, quoteDecimals)

					minFillAmounts[pair.BaseInterchainDenom()] = minQty
					minFillAmounts[pair.QuoteInterchainDenom()] = minAmt

					// set decimals

					// for base token
					basePrecisionField := osmomath.MustNewBigDecFromStr(lotSizeFilter["basePrecision"].(string)) // ex: 0.00001
					basePrecisionFieldInverse := osmomath.NewBigDec(1).Quo(basePrecisionField)                   // ex: 100000
					bDecimals := len(basePrecisionFieldInverse.String()) - 1 - osmomath.BigDecPrecision - 1      // ex: len(100000) - 1 = 5

					quotePrecisionField := osmomath.MustNewBigDecFromStr(lotSizeFilter["quotePrecision"].(string)) // ex: 0.0000001
					quotePrecisionFieldInverse := osmomath.NewBigDec(1).Quo(quotePrecisionField)                   // ex: 10000000.(36 zeros)
					qDecimals := len(quotePrecisionFieldInverse.String()) - 1 - osmomath.BigDecPrecision - 1       // -2 for "1" and "." // ex: len(10000000) - 1 = 7

					denomToDecimals[pair.Base] = bDecimals
					denomToDecimals[pair.Quote] = qDecimals

					continue
				}
			}
		}

		panic(fmt.Sprintf("failed to get min order qty for pair {%s}", pair.String()))
	}

	be.arbitrageConfig = &arbitrageConfig{
		minFillAmounts:  minFillAmounts,
		denomToDecimals: denomToDecimals,
	}
}
