package eth

import (
	"math/big"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

func (b *EthBridge) BuildRawTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	isSwapin := args.SwapType == tokens.Swap_Swapin
	if isSwapin && args.Input == nil {
		b.buildSwapinTxInput(args)
	}
	err = b.setDefaults(args)
	if err != nil {
		return nil, err
	}
	var (
		to       = common.HexToAddress(args.To)
		value    = args.Value
		nonce    = *args.Nonce
		gasLimit = *args.Gas
		gasPrice = args.GasPrice
		input    []byte
	)
	if args.Input != nil {
		input = *args.Input
	}

	if !isSwapin {
		value = tokens.CalcSwappedValue(value, b.IsSrc)
	}

	return types.NewTransaction(nonce, to, value, gasLimit, gasPrice, input), nil
}

func (b *EthBridge) setDefaults(args *tokens.BuildTxArgs) error {
	if args.GasPrice == nil {
		price, err := b.SuggestPrice()
		if err != nil {
			return err
		}
		args.GasPrice = price
	}
	if args.Value == nil {
		args.Value = new(big.Int)
	}
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(args.From)
		if err != nil {
			return err
		}
		args.Nonce = &nonce
	}
	if args.Gas == nil {
		args.Gas = new(uint64)
		*args.Gas = 90000
	}
	return nil
}

// build input for calling `Swapin(bytes32 txhash, address account, uint256 amount)`
func (b *EthBridge) buildSwapinTxInput(args *tokens.BuildTxArgs) {
	funcHash := tokens.SwapinFuncHash[:]
	txHash := common.HexToHash(args.SwapID).Bytes()
	address := common.LeftPadBytes(common.HexToAddress(args.To).Bytes(), 32)
	amount := common.LeftPadBytes(tokens.CalcSwappedValue(args.Value, b.IsSrc).Bytes(), 32)
	input := make([]byte, 100)
	copy(input[:4], funcHash)
	copy(input[4:36], txHash)
	copy(input[36:68], address)
	copy(input[68:100], amount)
	args.Input = &input // input

	token := b.TokenConfig
	args.From = *token.DcrmAddress   // from
	args.To = *token.ContractAddress // to
	args.Value = big.NewInt(0)       // value
}
