package bybit_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	wsbybit "github.com/hirokisan/bybit/v2"
	"github.com/joho/godotenv"
	bybit "github.com/wuhewuhe/bybit.go.api"
)

// NOTE: Running this test buys 40$ worth of BTC on bybit
func TestBuy40(t *testing.T) {
	// Get the path of the current file
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("No caller information")
	}

	// Get the directory of the current file
	currentDir := filepath.Dir(currentFile)

	err := godotenv.Load(currentDir + "/.env")
	if err != nil {
		panic(err)
	}

	if os.Getenv("BYBIT_API_KEY") == "" || os.Getenv("BYBIT_API_SECRET") == "" {
		fmt.Println(currentDir+"/.env", os.Getenv("BYBIT_API_KEY") == "")
	}

	client := bybit.NewBybitHttpClient(os.Getenv("BYBIT_API_KEY"), os.Getenv("BYBIT_API_SECRET"), bybit.WithBaseURL(bybit.MAINNET))
	accountResult, err := client.NewPlaceOrderService("spot", "BTCUSDC", "Buy", "Market", "40").Do(context.Background())
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(bybit.PrettyPrint(accountResult))

	params := map[string]interface{}{"category": "spot", "symbol": "BTCUSDC"}
	res, _ := client.NewUtaBybitServiceWithParams(params).GetOrderBookInfo(context.Background())
	fmt.Println(bybit.PrettyPrint(res))
}

func TestWebsocket(t *testing.T) {
	// Get the path of the current file
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("No caller information")
	}

	// Get the directory of the current file
	currentDir := filepath.Dir(currentFile)

	err := godotenv.Load(currentDir + "/.env")
	if err != nil {
		panic(err)
	}

	if os.Getenv("BYBIT_API_KEY") == "" || os.Getenv("BYBIT_API_SECRET") == "" {
		fmt.Println(currentDir+"/.env", os.Getenv("BYBIT_API_KEY") == "")
	}

	wsclient := wsbybit.NewWebsocketClient().WithAuth(os.Getenv("BYBIT_API_KEY"), os.Getenv("BYBIT_API_SECRET"))
	svc, _ := wsclient.V5().Public(wsbybit.CategoryV5Spot)
	svc.SubscribeOrderBook(wsbybit.V5WebsocketPublicOrderBookParamKey{
		Symbol: "BTCUSDC",
		Depth:  50,
	}, func(resp wsbybit.V5WebsocketPublicOrderBookResponse) error {
		if len(resp.Data.Asks) > 0 {
			fmt.Println(resp.Data.Asks[0].Price)
		}
		return nil
	},
	)

	svc.Start(context.Background(), nil)
}

func TestAny(t *testing.T) {

	type batchClaim struct {
		Orders [][]int `json:"orders"` // [[tickId, orderId], [tickId, orderId], ...]
	}

	type batchClaimRequest struct {
		batchClaim `json:"batch_claim"`
	}

	orders := [][]int{{1, 2}, {3, 4}}

	bc := batchClaim{
		Orders: orders,
	}

	req := batchClaimRequest{batchClaim: bc}

	bz, _ := json.Marshal(req)
	fmt.Println(string(bz))
}
