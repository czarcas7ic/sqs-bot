package bybit

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	wsbybit "github.com/hirokisan/bybit/v2"
)

func TestWs(t *testing.T) {
	wsClient := wsbybit.NewWebsocketClient().WithAuth(os.Getenv("BYBIT_API_KEY"), os.Getenv("BYBIT_API_SECRET"))
	svc, err := wsClient.V5().Public(wsbybit.CategoryV5Spot)
	if err != nil {
		t.Fatal(err)
	}

	// svc.SubscribeOrderBook(
	// 	wsbybit.V5WebsocketPublicOrderBookParamKey{
	// 		Depth:  1,
	// 		Symbol: wsbybit.SymbolV5("BTCUSDT"),
	// 	},
	// 	func(resp wsbybit.V5WebsocketPublicOrderBookResponse) error {
	// 		wg := &sync.WaitGroup{}
	// 		wg.Add(1)
	// 		go func() {
	// 			defer wg.Done()
	// 			fmt.Println(1, resp)
	// 			time.Sleep(5 * time.Second)
	// 		}()

	// 		wg.Wait()
	// 		return nil
	// 	},
	// )

	svc.SubscribeOrderBook(
		wsbybit.V5WebsocketPublicOrderBookParamKey{
			Depth:  50,
			Symbol: wsbybit.SymbolV5("BTCUSDC"),
		},
		func(resp wsbybit.V5WebsocketPublicOrderBookResponse) error {
			fmt.Println("BIDS: ", resp.Data.Bids, resp.TimeStamp, resp.Type)
			fmt.Println("ASKS: ", resp.Data.Asks, resp.TimeStamp, resp.Type)

			time.Sleep(1 * time.Second)
			return nil
		},
	)

	go svc.Start(context.Background(), nil)

	fmt.Println("test")

	select {}
}
