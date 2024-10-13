package osmocexfillertypes

import (
	"sort"
	"strconv"
	"sync"
)

type OrderbookData struct {
	mu sync.Mutex

	Symbol string
	bids   map[string]string // Price: Size
	asks   map[string]string // Price: Size
}

func NewOrderbookData(symbol string, bids, asks map[string]string) *OrderbookData {
	return &OrderbookData{
		mu: sync.Mutex{},

		Symbol: symbol,
		bids:   bids,
		asks:   asks,
	}
}

func (o *OrderbookData) Bids() []OrderBasicI {
	o.mu.Lock()
	defer o.mu.Unlock()

	bids := make([]OrderBasicI, 0, len(o.bids))
	for price, size := range o.bids {
		bids = append(bids, &OrderbookEntry{
			Direction: "bid",
			Price:     price,
			Size:      size,
		})
	}
	return bids
}

func (o *OrderbookData) Asks() []OrderBasicI {
	o.mu.Lock()
	defer o.mu.Unlock()

	asks := make([]OrderBasicI, 0, len(o.asks))
	for price, size := range o.asks {
		asks = append(asks, &OrderbookEntry{
			Direction: "ask",
			Price:     price,
			Size:      size,
		})
	}
	return asks
}

// func (o *OrderbookData) GetBid(price string) (string, bool) {
// 	o.mu.Lock()
// 	defer o.mu.Unlock()

// 	size, ok := o.bids[price]
// 	return size, ok
// }

// func (o *OrderbookData) GetAsk(price string) (string, bool) {
// 	o.mu.Lock()
// 	defer o.mu.Unlock()

// 	size, ok := o.asks[price]
// 	return size, ok
// }

func (o *OrderbookData) SetBid(price, size string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.bids[price] = size
}

func (o *OrderbookData) SetAsk(price, size string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.asks[price] = size
}

func (o *OrderbookData) RemoveBid(price string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	delete(o.bids, price)
}

func (o *OrderbookData) RemoveAsk(price string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	delete(o.asks, price)
}

func (o *OrderbookData) BidsDescending() []OrderBasicI {
	o.mu.Lock()
	defer o.mu.Unlock()

	bids := make([]OrderBasicI, 0, len(o.bids))
	for price, size := range o.bids {
		bids = append(bids, &OrderbookEntry{
			Direction: "bid",
			Price:     price,
			Size:      size,
		})
	}

	sort.Slice(bids, func(i, j int) bool {
		priceI, _ := strconv.ParseFloat(bids[i].GetPrice(), 64)
		priceJ, _ := strconv.ParseFloat(bids[j].GetPrice(), 64)
		return priceI > priceJ
	})
	return bids
}

func (o *OrderbookData) AsksAscending() []OrderBasicI {
	o.mu.Lock()
	defer o.mu.Unlock()

	asks := make([]OrderBasicI, 0, len(o.asks))
	for price, size := range o.asks {
		asks = append(asks, &OrderbookEntry{
			Direction: "ask",
			Price:     price,
			Size:      size,
		})
	}

	sort.Slice(asks, func(i, j int) bool {
		priceI, _ := strconv.ParseFloat(asks[i].GetPrice(), 64)
		priceJ, _ := strconv.ParseFloat(asks[j].GetPrice(), 64)
		return priceI < priceJ
	})
	return asks
}

type OrderbookEntry struct {
	Direction string
	Price     string
	Size      string
}

func (o OrderbookEntry) GetPrice() string {
	return o.Price
}

func (o OrderbookEntry) GetSize() string {
	return o.Size
}

func (o OrderbookEntry) GetDirection() string {
	return o.Direction
}

func (o *OrderbookEntry) SetSize(size string) {
	o.Size = size
}

var _ OrderBasicI = (*OrderbookEntry)(nil)

type Pair struct {
	Base  string
	Quote string
}

func (p Pair) String() string {
	return p.Base + p.Quote
}
