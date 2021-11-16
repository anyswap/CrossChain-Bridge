package eth

import (
	"errors"

	"github.com/anyswap/CrossChain-Bridge/common"
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

	receiver := common.HexToAddress(args.Bind)
	if receiver == (common.Address{}) || !common.IsHexAddress(args.Bind) {
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
	}

	funcHash := erc20CodeParts["transfer"]
	input := PackDataWithFuncHash(funcHash, receiver, swapValue)
	args.Input = &input             // input
	args.To = token.ContractAddress // to

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
