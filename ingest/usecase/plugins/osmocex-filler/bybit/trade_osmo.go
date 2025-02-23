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
	"github.com/osmosis-labs/sqs/domain"
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

	// (1 - slippageTolerance) * 100
	// slippage tolerance = 0.5%
	valueFraction = osmomath.MustNewBigDecFromStr("0.995")
)

// simulates osmo trade. Returns expected amount out and gas used
// TODO: move somewhere else
func (be *BybitExchange) simulateOsmoTradeOutGivenIn(coinIn sdk.Coin, denomOut string) (func(), osmomath.BigDec, uint64, error) {
	quote, err := (*be.osmoRouterUsecase).GetOptimalQuote(be.ctx, coinIn, denomOut, domain.WithDisableSplitRoutes())
	if err != nil {
		be.logger.Error("failed to get simple quote", zap.Error(err))
		return nil, osmomath.ZeroBigDec(), 0, err
	}

	amountOutExpected := quote.GetAmountOut()
	amountOutExpectedBigDec := osmomath.NewBigDecFromBigInt(amountOutExpected.BigInt())
	amountOutExpectedBigDec.MulMut(valueFraction)

	splitRoute := quote.GetRoute()
	if len(splitRoute) != 1 { // disabled splits
		be.logger.Error("split route should have 1 route")
		return nil, osmomath.ZeroBigDec(), 0, fmt.Errorf("split route should have 1 route")
	}

	route := splitRoute[0].GetPools()
	poolmanagerRoute := make([]poolmanagertypes.SwapAmountInRoute, 0, len(route))
	for _, pool := range route {
		poolmanagerRoute = append(poolmanagerRoute, poolmanagertypes.SwapAmountInRoute{
			PoolId:        pool.GetId(),
			TokenOutDenom: pool.GetTokenOutDenom(),
		})
	}

	swapMsg := &poolmanagertypes.MsgSwapExactAmountIn{
		Sender:            (*be.osmoKeyring).GetAddress().String(),
		Routes:            poolmanagerRoute,
		TokenIn:           coinIn,
		TokenOutMinAmount: amountOutExpectedBigDec.Dec().RoundInt(),
	}

	msgs := []sdk.Msg{swapMsg}

	_, adjustedGasUsed, err := be.simulateOsmoMsgs(be.ctx, msgs)
	if err != nil {
		be.logger.Error("failed to simulate osmo msg", zap.Error(err))
		return nil, osmomath.ZeroBigDec(), 0, err
	}

	// function to perform this trade upstream
	tradeFunction := func() {
		localAdjustedGasUsed := adjustedGasUsed
		_, _, err := be.executeOsmoMsgs(msgs, localAdjustedGasUsed)
		if err != nil {
			be.logger.Error("failed to execute osmo msg", zap.Error(err))
			return
		}
	}

	return tradeFunction, amountOutExpectedBigDec, adjustedGasUsed, nil
}

// func (be *BybitExchange) simulateOsmoTradeInGivenOut(coinOut sdk.Coin, denomIn string) (func(), osmomath.BigDec, uint64, error) {
// 	quote, err := (*be.osmoRouterUsecase).GetOptimalQuoteInGivenOut(be.ctx, coinOut, denomIn, domain.WithDisableSplitRoutes())
// 	if err != nil {
// 		be.logger.Error("failed to get simple quote", zap.Error(err))
// 		return nil, osmomath.ZeroBigDec(), 0, err
// 	}

// 	amountInExpected := osmomath.NewBigDecFromBigInt(quote.GetAmountIn().Amount.BigInt())
// 	amountInExpected = amountInExpected.Quo(osmomath.NewBigDecFromBigInt(valueFraction.BigInt()))

// 	splitRoute := quote.GetRoute()
// 	if len(splitRoute) != 1 { // disabled splits
// 		be.logger.Error("route should have 1 route")
// 		return nil, osmomath.ZeroBigDec(), 0, fmt.Errorf("route should have 1 route")
// 	}

// 	route := splitRoute[0].GetPools()
// 	poolmanagerRoute := make([]poolmanagertypes.SwapAmountOutRoute, 0, len(route))
// 	for _, pool := range route {
// 		poolmanagerRoute = append(poolmanagerRoute, poolmanagertypes.SwapAmountOutRoute{
// 			PoolId:       pool.GetId(),
// 			TokenInDenom: pool.GetTokenInDenom(),
// 		})
// 	}

// 	swapMsg := &poolmanagertypes.MsgSwapExactAmountOut{
// 		Sender:           (*be.osmoKeyring).GetAddress().String(),
// 		Routes:           poolmanagerRoute,
// 		TokenOut:         coinOut,
// 		TokenInMaxAmount: amountInExpected.Dec().RoundInt(),
// 	}

// 	msgs := []sdk.Msg{swapMsg}

// 	_, adjustedGasUsed, err := be.simulateOsmoMsgs(be.ctx, msgs)
// 	if err != nil {
// 		be.logger.Error("failed to simulate osmo msg", zap.Error(err))
// 		return nil, osmomath.ZeroBigDec(), 0, err
// 	}

// 	// function to perform this trade upstream
// 	tradeFunction := func() {
// 		localAdjustedGasUsed := adjustedGasUsed
// 		_, _, err := be.executeOsmoMsgs(msgs, localAdjustedGasUsed)
// 		if err != nil {
// 			be.logger.Error("failed to execute osmo msg", zap.Error(err))
// 			return
// 		}
// 	}

// 	return tradeFunction, amountInExpected, adjustedGasUsed, nil
// }

func (be *BybitExchange) executeOsmoMsgs(msgs []sdk.Msg, adjustedGasUsed uint64) (*coretypes.ResultBroadcastTx, string, error) {
	key := (*be.osmoKeyring).GetKey()
	keyBytes := key.Bytes()

	privKey := &secp256k1.PrivKey{Key: keyBytes}
	txBuilder := encodingConfig.TxConfig.NewTxBuilder()

	txFeeUosmo := defaultGasPrice.Dec().Mul(osmomath.NewIntFromUint64(adjustedGasUsed).ToLegacyDec()).Ceil().TruncateInt()
	feecoin := sdk.NewCoin(Denom, txFeeUosmo)

	err := txBuilder.SetMsgs(msgs...)
	if err != nil {
		return nil, "", err
	}

	txBuilder.SetGasLimit(adjustedGasUsed)
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

func (be *BybitExchange) simulateOsmoMsgs(ctx context.Context, msgs []sdk.Msg) (*txtypes.SimulateResponse, uint64, error) {
	accSeq, accNum := getInitialSequence(ctx, (*be.osmoKeyring).GetAddress().String())

	txFactory := tx.Factory{}
	txFactory = txFactory.WithTxConfig(encodingConfig.TxConfig)
	txFactory = txFactory.WithAccountNumber(accNum)
	txFactory = txFactory.WithSequence(accSeq)
	txFactory = txFactory.WithChainID(chainID)
	txFactory = txFactory.WithGasAdjustment(1.02)

	// Estimate transaction
	gasResult, adjustedGasUsed, err := CalculateGas(ctx, (*be.osmoPassthroughGRPCClient).GetChainGRPCClient(), txFactory, msgs...)
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
