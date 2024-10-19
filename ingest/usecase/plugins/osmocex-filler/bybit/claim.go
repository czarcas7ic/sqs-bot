package bybit

import (
	"encoding/json"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// claim.go has the necessary code to claim filled orders on osmosis
// claim runs when an order is filled fully on osmosis
// This is done to assert that a correct state of an orderbook is maintained after arb fills
// Otherwise, the bot may find the same arbitrage opportunity that in reality does not exist anymore
// TODO: move this upstream, this should not by in the bybit package

type possibleClaim struct {
	TickId  int64
	OrderID int64
}

type batchClaim struct {
	Orders [][]int `json:"orders"` // [[tickId, orderId], [tickId, orderId], ...]
}

type batchClaimRequest struct {
	batchClaim `json:"batch_claim"`
}

func (be *BybitExchange) batchClaimMsg(contractAddress string) (sdk.Msg, error) {
	batchClaim := batchClaim{
		Orders: make([][]int, 0, len(be.claims)),
	}

	for _, claim := range be.claims {
		batchClaim.Orders = append(batchClaim.Orders, []int{int(claim.TickId), int(claim.OrderID)})
	}

	request := batchClaimRequest{batchClaim}

	// marshal request
	bz, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	claimMsg := &wasmtypes.MsgExecuteContract{
		Sender:   (*be.osmoKeyring).GetAddress().String(),
		Contract: contractAddress,
		Msg:      bz,
	}

	return claimMsg, nil
}

func (be *BybitExchange) addClaim(claim possibleClaim) {
	be.claims = append(be.claims, claim)
}

func (be *BybitExchange) clearClaims() {
	be.claims = make([]possibleClaim, 0)
}
