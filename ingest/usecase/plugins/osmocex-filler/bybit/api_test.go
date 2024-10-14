package bybit_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/joho/godotenv"
	bybit "github.com/wuhewuhe/bybit.go.api"
)

func Test(t *testing.T) {
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

	// if true { // prevent accidental trades
	// 	panic("BYBIT_API_KEY or BYBIT_API_SECRET not set")
	// }

	client := bybit.NewBybitHttpClient(os.Getenv("BYBIT_API_KEY"), os.Getenv("BYBIT_API_SECRET"), bybit.WithBaseURL(bybit.MAINNET))
	params := map[string]interface{}{"accountType": "UNIFIED"}
	accountResult, err := client.NewUtaBybitServiceWithParams(params).GetAccountWallet(context.Background())
	if err != nil {
		fmt.Println(err)
	}

	result := accountResult.Result.(map[string]interface{})

	fmt.Println(result)
}
