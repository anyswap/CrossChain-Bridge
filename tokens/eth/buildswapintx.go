package eth

import (
	"errors"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// build input for calling `Swapin(bytes32 txhash, address account, uint256 amount)`
func (b *Bridge) buildSwapinTxInput(args *tokens.BuildTxArgs) (err error) {
	pairID := args.PairID
	token := b.GetTokenConfig(pairID)
	if token == nil {
		return tokens.ErrUnknownPairID
	}
	funcHash := getSwapinFuncHash()
	txHash := common.HexToHash(args.SwapID)
	address := common.HexToAddress(args.Bind)
	if address == (common.Address{}) || !common.IsHexAddress(args.Bind) {
		log.Warn("swapin to wrong address", "address", args.Bind)
		return errors.New("can not swapin to empty or invalid address")
	}
	amount := tokens.CalcSwappedValue(pairID, args.OriginValue, true)
	amount, err = b.adjustSwapValue(args, amount)
	if err != nil {
		return err
	}
	args.SwapValue = amount

	input := PackDataWithFuncHash(funcHash, txHash, address, amount)
	args.Input = &input // input

	args.To = token.ContractAddress // to

	if token.IsDelegateContract && !token.IsAnyswapAdapter {
		return b.checkBalance(token.DelegateToken, token.ContractAddress, amount)
	}
	return nil
}

func (b *Bridge) adjustSwapValue(args *tokens.BuildTxArgs, swapValue *big.Int) (*big.Int, error) {
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
	log.Info("adjust swap value", "isSrc", b.IsSrc, "chainID", b.SignerChainID,
		"pairID", args.PairID, "txid", args.SwapID, "bind", args.Bind, "swapType", args.SwapType.String(),
		"originValue", args.OriginValue, "oldSwapValue", swapValue, "newSwapValue", newSwapValue,
		"oldFee", fee, "extraFee", extraFee, "baseGasPrice", baseGasPrice, "gasPrice", gasPrice, "extraGasPrice", extraGasPrice)
	if newSwapValue.Sign() <= 0 {
		return nil, tokens.ErrWrongSwapValue
	}
	return newSwapValue, nil
}
