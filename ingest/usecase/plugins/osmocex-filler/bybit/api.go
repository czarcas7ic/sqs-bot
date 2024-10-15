package bybit

import (
	"encoding/json"
	"errors"

	"github.com/osmosis-labs/osmosis/osmomath"
	osmocexfillertypes "github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/types"
)

var _ osmocexfillertypes.CoinBalanceI = (*Coin)(nil)

// Coin is a json object that gets returned when calling the bybit balance endpoint
type Coin struct {
	AccruedInterest     string `json:"accruedInterest"`
	AvailableToBorrow   string `json:"availableToBorrow"`
	AvailableToWithdraw string `json:"availableToWithdraw"`
	Bonus               string `json:"bonus"`
	BorrowAmount        string `json:"borrowAmount"`
	Coin                string `json:"coin"`
	CollateralSwitch    bool   `json:"collateralSwitch"`
	CumRealisedPnl      string `json:"cumRealisedPnl"`
	Equity              string `json:"equity"`
	Locked              string `json:"locked"`
	MarginCollateral    bool   `json:"marginCollateral"`
	SpotHedgingQty      string `json:"spotHedgingQty"`
	TotalOrderIM        string `json:"totalOrderIM"`
	TotalPositionIM     string `json:"totalPositionIM"`
	TotalPositionMM     string `json:"totalPositionMM"`
	UnrealisedPnl       string `json:"unrealisedPnl"`
	UsdValue            string `json:"usdValue"`
	WalletBalance       string `json:"walletBalance"`
}

func (c Coin) Token() string {
	return c.Coin
}

func (c Coin) Balance() string {
	return c.WalletBalance
}

func (c Coin) BigDecBalance(precision int) osmomath.BigDec {
	unscaledBalance := osmomath.MustNewBigDecFromStr(c.WalletBalance)
	return unscaledBalance.Mul(osmomath.NewBigDec(10).Power(osmomath.NewBigDec(int64(precision))))
}

func (be *BybitExchange) getBybitBalances() (map[string]osmocexfillertypes.CoinBalanceI, error) {
	params := map[string]interface{}{"accountType": "UNIFIED"}
	accountResult, err := be.httpclient.NewUtaBybitServiceWithParams(params).GetAccountWallet(be.ctx)
	if err != nil {
		return nil, err
	}

	result := accountResult.Result.(map[string]interface{})
	/*
		Schema:
			result {
				list {[
					accountType: "UNIFIED"
					coin: []
				]}
			}
	*/
	if result["list"] != nil {
		list := result["list"].([]interface{})
		for _, account := range list {
			accountJson := account.(map[string]interface{})
			if accountJson["accountType"] == "UNIFIED" {
				coin := accountJson["coin"].([]interface{})
				coinMap := make(map[string]osmocexfillertypes.CoinBalanceI)
				for _, c := range coin {
					var coin Coin
					coinBytes, _ := json.Marshal(c)
					if err := json.Unmarshal(coinBytes, &coin); err != nil {
						return nil, err
					}
					coinMap[coin.Coin] = coin
				}

				return coinMap, nil
			}
		}
	}

	be.logger.Error("failed to parse json when querying bot balance")
	return nil, errors.New("json parsing error")
}
