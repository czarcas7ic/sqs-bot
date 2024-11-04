package bybit

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	cometrpc "github.com/cometbft/cometbft/rpc/client/http"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
	wsbybit "github.com/hirokisan/bybit/v2"
	"github.com/osmosis-labs/osmosis/osmomath"
	osmocexfillertypes "github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/types"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"
)

func parseBybitOrderbook(data wsbybit.V5WebsocketPublicOrderBookData) *osmocexfillertypes.OrderbookData {
	bids := make(map[string]string)
	asks := make(map[string]string)

	for _, bid := range data.Bids {
		bids[bid.Price] = bid.Size
	}

	for _, ask := range data.Asks {
		asks[ask.Price] = ask.Size
	}

	return osmocexfillertypes.NewOrderbookData(string(data.Symbol), bids, asks)
}

func (be *BybitExchange) updateBybitOrderbook(data wsbybit.V5WebsocketPublicOrderBookData) {
	orderbookAny, ok := be.orderbooks.Load(data.Symbol)
	if !ok {
		be.logger.Error("orderbook not found", zap.String("symbol", string(data.Symbol)))
		return
	}

	orderbook := orderbookAny.(*osmocexfillertypes.OrderbookData)

	for _, bid := range data.Bids {
		if bid.Size == "0" {
			orderbook.RemoveBid(bid.Price)
		} else {
			orderbook.SetBid(bid.Price, bid.Size)
		}
	}

	for _, ask := range data.Asks {
		if ask.Size == "0" {
			orderbook.RemoveAsk(ask.Price)
		} else {
			orderbook.SetAsk(ask.Price, ask.Size)
		}
	}

	be.orderbooks.Store(string(data.Symbol), orderbook)
}

// adjustFillAmount adjusts fill amount based on available balances (effectively, sets it to min of three)
// also returns final 1/fillAmount
func (be *BybitExchange) adjustFillAmount(fillAmount, balance osmomath.BigDec) osmomath.BigDec {
	if balance.LT(fillAmount) {
		fillAmount = balance
	}

	return fillAmount
}

func (be *BybitExchange) getBybitOrderbookForPair(pair osmocexfillertypes.Pair) (*osmocexfillertypes.OrderbookData, error) {
	orderbookAny, ok := be.orderbooks.Load(pair.String())
	if !ok {
		be.logger.Error("orderbook not found", zap.String("pair", pair.String()))
		return nil, errors.New("orderbook not found")
	}

	orderbook := orderbookAny.(*osmocexfillertypes.OrderbookData)

	return orderbook, nil
}

// adds decimals extra decimals to a bigdec
// (10.201, 2) -> 1020.1
func addBigDecDecimals(bd *osmomath.BigDec, decimals int) {
	base := osmomath.NewBigDec(10)
	multiplier := base.Power(osmomath.NewBigDec(int64(decimals)))
	bd.MulMut(multiplier)
}

// removes decimals from a bigdec
// (1020.1, 2) -> 10.201
// func removeBigDecDecimals(bd *osmomath.BigDec, decimals int) {
// 	base := osmomath.NewBigDec(10)
// 	divisor := base.Power(osmomath.NewBigDec(int64(decimals)))
// 	bd.QuoMut(divisor)
// }

// scaleBigDecDecimals adjusts the number of decimals in a big dec. Positive decimals means appending more decimals, negative means removing decimals
func scaleBigDecDecimals(bd *osmomath.BigDec, decimals int) {
	base := osmomath.NewBigDec(10)
	if decimals < 0 {
		divisor := base.Power(osmomath.NewBigDec(int64(-decimals)))
		bd.QuoMut(divisor)
	} else {
		multiplier := base.Power(osmomath.NewBigDec(int64(decimals)))
		bd.MulMut(multiplier)
	}
}

// returns the precision of a token in the interchain ecosystem
func (be *BybitExchange) getInterchainDenomDecimals(denom string) (int, error) {
	denomMetadata, err := (*be.osmoTokensUsecase).GetMetadataByChainDenom(denom)
	if err != nil {
		be.logger.Error("failed to get token metadata", zap.Error(err))
		return 0, err
	}

	return denomMetadata.Precision, nil
}

type AccountInfo struct {
	Sequence      string `json:"sequence"`
	AccountNumber string `json:"account_number"`
}

type AccountResult struct {
	Account AccountInfo `json:"account"`
}

func getInitialSequence(ctx context.Context, address string) (uint64, uint64) {
	resp, err := httpGet(ctx, LCD+"/cosmos/auth/v1beta1/accounts/"+address)
	if err != nil {
		log.Printf("Failed to get initial sequence: %v", err)
		return 0, 0
	}

	var accountRes AccountResult
	err = json.Unmarshal(resp, &accountRes)
	if err != nil {
		log.Printf("Failed to unmarshal account result: %v", err)
		return 0, 0
	}

	seqint, err := strconv.ParseUint(accountRes.Account.Sequence, 10, 64)
	if err != nil {
		log.Printf("Failed to convert sequence to int: %v", err)
		return 0, 0
	}

	accnum, err := strconv.ParseUint(accountRes.Account.AccountNumber, 10, 64)
	if err != nil {
		log.Printf("Failed to convert account number to int: %v", err)
		return 0, 0
	}

	return seqint, accnum
}

var client = &http.Client{
	Timeout:   10 * time.Second, // Adjusted timeout to 10 seconds
	Transport: otelhttp.NewTransport(http.DefaultTransport),
}

func httpGet(ctx context.Context, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		netErr, ok := err.(net.Error)
		if ok && netErr.Timeout() {
			log.Printf("Request to %s timed out, continuing...", url)
			return nil, nil
		}
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// broadcastTransaction broadcasts a transaction to the chain.
// Returning the result and error.
func broadcastTransaction(ctx context.Context, txBytes []byte, rpcEndpoint string) (*coretypes.ResultBroadcastTx, error) {
	cmtCli, err := cometrpc.New(rpcEndpoint, "/websocket")
	if err != nil {
		return nil, err
	}

	t := tmtypes.Tx(txBytes)

	res, err := cmtCli.BroadcastTxSync(ctx, t)
	if err != nil {
		return nil, err
	}

	return res, nil
}
