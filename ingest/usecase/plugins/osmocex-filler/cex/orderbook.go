package cex

type OrderbookData struct {
	Symbol string
	Bids   []OrderbookEntry
	Asks   []OrderbookEntry
}

type OrderbookEntry struct {
	Price  string
	Amount string
}
