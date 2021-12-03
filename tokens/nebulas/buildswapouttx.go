package nebulas

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	errInvalidReceiverAddress = errors.New("invalid receiver address")
)

func (b *Bridge) buildSwapoutTxInput(args *tokens.BuildTxArgs) (err error) {
	token := b.GetTokenConfig(args.PairID)
	if token == nil {
		return tokens.ErrUnknownPairID
	}

	_, err = AddressParse(args.Bind)
	if err != nil {
		log.Warn("swapout to wrong address", "receiver", args.Bind)
		return errInvalidReceiverAddress
	}

	swapValue := tokens.CalcSwappedValue(args.PairID, args.OriginValue, false)
	swapValue, err = b.adjustSwapValue(args, swapValue)
	if err != nil {
		return err
	}
	args.SwapValue = swapValue // swap value

	if token.ContractAddress == "" {
		input := b.getUnlockCoinMemo(args)
		args.Input = &input    // input
		args.To = args.Bind    // to
		args.Value = swapValue // value
		return nil
	} else {
		callArgs := fmt.Sprintf("[\"%s\",\"%s\"]", args.Bind, swapValue.String())
		payload, err := NewCallPayload("transfer", callArgs)
		if err != nil {
			return err
		}
		pbytes, err := payload.ToBytes()
		if err != nil {
			return err
		}
		args.Input = &pbytes            // input
		args.To = token.ContractAddress // to
	}

	return b.checkBalance(token.ContractAddress, token.DcrmAddress, swapValue)
}

func (b *Bridge) getUnlockCoinMemo(args *tokens.BuildTxArgs) (input []byte) {
	if params.IsNullSwapoutNativeMemo() {
		return input
	}
	isContract, err := b.IsContractAddress(args.Bind)
	if err == nil && !isContract {
		input = []byte(tokens.UnlockMemoPrefix + args.SwapID)
	}
	return input
}

func (b *Bridge) adjustSwapValue(args *tokens.BuildTxArgs, swapValue *big.Int) (*big.Int, error) {
	isDynamicFeeTx := b.ChainConfig.EnableDynamicFeeTx
	if isDynamicFeeTx {
		return swapValue, nil
	}

	if baseGasPrice == nil {
		return swapValue, nil
	}

	gasPrice := args.GetTxGasPrice()
	if gasPrice.Cmp(baseGasPrice) <= 0 {
		return swapValue, nil
	}

	fee := new(big.Int).Sub(args.OriginValue, swapValue)
	if fee.Sign() == 0 {
		return swapValue, nil
	}
	if fee.Sign() < 0 {
		return nil, tokens.ErrWrongSwapValue
	}

	extraGasPrice := new(big.Int).Sub(gasPrice, baseGasPrice)
	extraFee := new(big.Int).Mul(fee, extraGasPrice)
	extraFee.Div(extraFee, baseGasPrice)

	newSwapValue := new(big.Int).Sub(swapValue, extraFee)
	log.Info("adjust swap value", "isSrc", b.IsSrc, "chainID", b.ChainConfig.GetChainID(),
		"pairID", args.PairID, "txid", args.SwapID, "bind", args.Bind, "swapType", args.SwapType.String(),
		"originValue", args.OriginValue, "oldSwapValue", swapValue, "newSwapValue", newSwapValue,
		"oldFee", fee, "extraFee", extraFee, "baseGasPrice", baseGasPrice, "gasPrice", gasPrice, "extraGasPrice", extraGasPrice)
	if newSwapValue.Sign() <= 0 {
		return nil, tokens.ErrWrongSwapValue
	}
	return newSwapValue, nil
}
