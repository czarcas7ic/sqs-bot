package bybit

import (
	"os"
	"testing"

	wsbybit "github.com/hirokisan/bybit/v2"
)

func TestWs(t *testing.T) {
	wsClient := wsbybit.NewTestWebsocketClient().WithAuth(os.Getenv("BYBIT_API_KEY"), os.Getenv("BYBIT_API_SECRET"))
	svc, err := wsClient.V5().Private()
	if err != nil {
		t.Fatal(err)
	}

	svc.SubscribeOrder(func(resp wsbybit.V5WebsocketPrivateOrderResponse) error {
		return nil
	})

	select {}
}
