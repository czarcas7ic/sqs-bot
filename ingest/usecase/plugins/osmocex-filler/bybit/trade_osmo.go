package bybit

import (
	"context"
	"fmt"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	gogogrpc "github.com/cosmos/gogoproto/grpc"
	"github.com/cosmos/ibc-go/v7/testing/simapp"
	"github.com/osmosis-labs/osmosis/osmomath"
	poolmanagertypes "github.com/osmosis-labs/osmosis/v25/x/poolmanager/types"
	"go.uber.org/zap"
)

// osmosis variables
var (
	chainID = "osmosis-1"

	RPC       = "http://127.0.0.1:26657"
	LCD       = "http://127.0.0.1:1317"
	Denom     = "uosmo"
	NobleUSDC = "ibc/498A0751C798A0D9A389AA3691123DADA57DAA4FE165D5C75894505B876BA6E4"

	encodingConfig = simapp.MakeTestEncodingConfig()

	defaultGasPrice = osmomath.MustNewBigDecFromStr("0.1")
)

// executes osmo trade
// TODO: move somewhere else
func (be *BybitExchange) tradeOsmosis(coin sdk.Coin, denom string, orderbookPoolId uint64) {
	swapMsg := &poolmanagertypes.MsgSwapExactAmountIn{
		Sender: (*be.osmoKeyring).GetAddress().String(),
		Routes: []poolmanagertypes.SwapAmountInRoute{
			{
				PoolId:        orderbookPoolId,
				TokenOutDenom: denom,
			},
		},
		TokenIn:           coin,
		TokenOutMinAmount: sdk.NewInt(1), // TODO: set this to a reasonable value
	}

	_, adjustedGasUsed, err := be.simulateOsmoMsg(be.ctx, swapMsg)
	if err != nil {
		be.logger.Error("failed to simulate osmo msg", zap.Error(err))
		return
	}

	_, _, err = be.executeOsmoMsg(swapMsg, adjustedGasUsed)
	if err != nil {
		be.logger.Error("failed to execute osmo msg", zap.Error(err))
		return
	}

	be.logger.Info("executed swap on OSMOSIS", zap.String("coin", coin.String()), zap.String("denom", denom))
}

func (be *BybitExchange) executeOsmoMsg(msg sdk.Msg, adjustedGasUsed uint64) (*coretypes.ResultBroadcastTx, string, error) {
	key := (*be.osmoKeyring).GetKey()
	keyBytes := key.Bytes()

	privKey := &secp256k1.PrivKey{Key: keyBytes}
	txBuilder := encodingConfig.TxConfig.NewTxBuilder()

	txFeeUosmo := defaultGasPrice.Dec().Mul(osmomath.NewIntFromUint64(adjustedGasUsed).ToLegacyDec()).Ceil().TruncateInt()
	feecoin := sdk.NewCoin(Denom, txFeeUosmo)

	err := txBuilder.SetMsgs(msg)
	if err != nil {
		return nil, "", err
	}

	txBuilder.SetGasLimit(adjustedGasUsed * 12 / 10)
	txBuilder.SetFeeAmount(sdk.NewCoins(feecoin))
	txBuilder.SetTimeoutHeight(0)

	// First round: we gather all the signer infos. We use the "set empty
	// signature" hack to do that.
	accSequence, accNumber := getInitialSequence(be.ctx, (*be.osmoKeyring).GetAddress().String())
	sigV2 := signing.SignatureV2{
		PubKey: privKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  encodingConfig.TxConfig.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: accSequence,
	}

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		fmt.Println("error setting signatures")
		return nil, "", err
	}

	signerData := authsigning.SignerData{
		ChainID:       chainID,
		AccountNumber: accNumber,
		Sequence:      accSequence,
	}

	signed, err := tx.SignWithPrivKey(
		encodingConfig.TxConfig.SignModeHandler().DefaultMode(), signerData,
		txBuilder, privKey, encodingConfig.TxConfig, accSequence)
	if err != nil {
		fmt.Println("couldn't sign")
		return nil, "", err
	}

	err = txBuilder.SetSignatures(signed)
	if err != nil {
		return nil, "", err
	}

	// Generate a JSON string.
	txJSONBytes, err := encodingConfig.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		fmt.Println(err)
		return nil, "", err
	}

	resp, err := broadcastTransaction(be.ctx, txJSONBytes, RPC)
	if err != nil {
		return nil, "", err
	}

	if resp.Code != 0 {
		return nil, "", err
	}

	be.logger.Info("executed transaction: ", zap.Uint32("code", resp.Code), zap.String("hash", string(resp.Hash)), zap.String("log", resp.Log), zap.String("codespace", resp.Codespace))

	return resp, string(txJSONBytes), nil
}

func (be *BybitExchange) simulateOsmoMsg(ctx context.Context, msg sdk.Msg) (*txtypes.SimulateResponse, uint64, error) {
	accSeq, accNum := getInitialSequence(ctx, (*be.osmoKeyring).GetAddress().String())

	txFactory := tx.Factory{}
	txFactory = txFactory.WithTxConfig(encodingConfig.TxConfig)
	txFactory = txFactory.WithAccountNumber(accNum)
	txFactory = txFactory.WithSequence(accSeq)
	txFactory = txFactory.WithChainID(chainID)
	txFactory = txFactory.WithGasAdjustment(1.02)

	// Estimate transaction
	gasResult, adjustedGasUsed, err := CalculateGas(ctx, (*be.osmoPassthroughGRPCClient).GetChainGRPCClient(), txFactory, msg)
	if err != nil {
		return nil, adjustedGasUsed, err
	}

	return gasResult, adjustedGasUsed, nil
}

// CalculateGas simulates the execution of a transaction and returns the
// simulation response obtained by the query and the adjusted gas amount.
func CalculateGas(
	ctx context.Context,
	clientCtx gogogrpc.ClientConn, txf tx.Factory, msgs ...sdk.Msg,
) (*txtypes.SimulateResponse, uint64, error) {
	txBytes, err := txf.BuildSimTx(msgs...)
	if err != nil {
		return nil, 0, err
	}

	txSvcClient := txtypes.NewServiceClient(clientCtx)
	simRes, err := txSvcClient.Simulate(ctx, &txtypes.SimulateRequest{
		TxBytes: txBytes,
	})
	if err != nil {
		return nil, 0, err
	}

	return simRes, uint64(txf.GasAdjustment() * float64(simRes.GasInfo.GasUsed)), nil
}
