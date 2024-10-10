package osmocexfillertypes

type OrderbookData struct {
	Symbol string
	Bids   []OrderbookEntry
	Asks   []OrderbookEntry
}

type OrderbookEntry struct {
	Price string
	Size  string
}

type Pair struct {
	Base  string
	Quote string
}

func (p Pair) String() string {
	return p.Base + p.Quote
}
