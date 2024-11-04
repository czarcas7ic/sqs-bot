package bybit

import "github.com/osmosis-labs/osmosis/osmomath"

type TradeType int

const (
	BUY TradeType = iota
	SELL
)

type ExchangeType int

const (
	OSMO ExchangeType = iota
	BYBIT
)

type ArbitrageDirection struct {
	BuyFrom ExchangeType
	SellTo  ExchangeType

	FillAmountBase  osmomath.BigDec // in base tokens
	FillAmountQuote osmomath.BigDec // in quote tokens
}

func NewArbitrageDirection(buyFrom, sellTo ExchangeType, fillAmountBase, fillAmountQuote osmomath.BigDec) *ArbitrageDirection {
	return &ArbitrageDirection{
		BuyFrom: buyFrom,
		SellTo:  sellTo,

		FillAmountBase:  fillAmountBase,
		FillAmountQuote: fillAmountQuote,
	}
}

func (ad *ArbitrageDirection) String() string {
	return string(rune(ad.BuyFrom)) + " -> " + string(rune(ad.SellTo)) + " : " + ad.FillAmountBase.String() + " | " + ad.FillAmountQuote.String()
}
