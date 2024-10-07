package cex

type CExchangeI interface {
	ProcessOrderbook(data OrderbookData) error
}
