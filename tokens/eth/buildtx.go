package eth

import (
	"math/big"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/types"
)

var (
	retryRPCCount    = 3
	retryRPCInterval = 1 * time.Second
)

// BuildRawTransaction build raw tx
func (b *Bridge) BuildRawTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	var input []byte
	if args.Input == nil {
		if args.SwapType != tokens.NoSwapType {
			args.From = b.TokenConfig.DcrmAddress // from
		}
		switch args.SwapType {
		case tokens.SwapinType:
			if b.IsSrc {
				return nil, tokens.ErrBuildSwapTxInWrongEndpoint
			}
			b.buildSwapinTxInput(args)
			input = *args.Input
		case tokens.SwapoutType, tokens.SwapRecallType:
			if !b.IsSrc {
				return nil, tokens.ErrBuildSwapTxInWrongEndpoint
			}
			switch {
			case b.TokenConfig.IsErc20():
				b.buildErc20SwapoutTxInput(args)
				input = *args.Input
			case args.SwapType == tokens.SwapoutType:
				input = []byte(tokens.UnlockMemoPrefix + args.SwapID)
			default:
				input = []byte(tokens.RecallMemoPrefix + args.SwapID)
			}
		}
	} else {
		input = *args.Input
	}

	extra, err := b.setDefaults(args)
	if err != nil {
		return nil, err
	}
	var (
		to       = common.HexToAddress(args.To)
		value    = args.Value
		nonce    = *extra.Nonce
		gasLimit = *extra.Gas
		gasPrice = extra.GasPrice
	)

	switch args.SwapType {
	case tokens.SwapoutType, tokens.SwapRecallType:
		if !b.TokenConfig.IsErc20() {
			value = tokens.CalcSwappedValue(value, false)
		}
	}

	if args.SwapType != tokens.NoSwapType {
		args.Identifier = params.GetIdentifier()
	}

	return types.NewTransaction(nonce, to, value, gasLimit, gasPrice, input), nil
}

func (b *Bridge) setDefaults(args *tokens.BuildTxArgs) (extra *tokens.EthExtraArgs, err error) {
	if args.Value == nil {
		args.Value = new(big.Int)
	}
	if args.Extra == nil || args.Extra.EthExtra == nil {
		extra = &tokens.EthExtraArgs{}
		args.Extra = &tokens.AllExtras{EthExtra: extra}
	} else {
		extra = args.Extra.EthExtra
	}
	if extra.GasPrice == nil {
		extra.GasPrice, err = b.getGasPrice()
		if err != nil {
			return nil, err
		}
	}
	if extra.Nonce == nil {
		extra.Nonce, err = b.getAccountNonce(args.From, args.SwapType)
		if err != nil {
			return nil, err
		}
	}
	if extra.Gas == nil {
		extra.Gas = new(uint64)
		*extra.Gas = 90000
	}
	return extra, nil
}

func (b *Bridge) getGasPrice() (price *big.Int, err error) {
	for i := 0; i < retryRPCCount; i++ {
		price, err = b.SuggestPrice()
		if err == nil {
			return price, nil
		}
		time.Sleep(retryRPCInterval)
	}
	return nil, err
}

func (b *Bridge) getAccountNonce(from string, swapType tokens.SwapType) (nonceptr *uint64, err error) {
	var nonce uint64
	for i := 0; i < retryRPCCount; i++ {
		nonce, err = b.GetPoolNonce(from, "pending")
		if err == nil {
			break
		}
		time.Sleep(retryRPCInterval)
	}
	if err != nil {
		return nil, err
	}
	if from == b.TokenConfig.DcrmAddress {
		if swapType != tokens.NoSwapType {
			nonce = b.AdjustNonce(nonce)
		}
	}
	return &nonce, nil
}

// build input for calling `Swapin(bytes32 txhash, address account, uint256 amount)`
func (b *Bridge) buildSwapinTxInput(args *tokens.BuildTxArgs) {
	funcHash := getSwapinFuncHash()
	txHash := common.HexToHash(args.SwapID)
	address := common.HexToAddress(args.To)
	amount := tokens.CalcSwappedValue(args.Value, true)

	input := PackDataWithFuncHash(funcHash, txHash, address, amount)
	args.Input = &input // input

	token := b.TokenConfig
	args.To = token.ContractAddress // to
	args.Value = big.NewInt(0)      // value
}

func (b *Bridge) buildErc20SwapoutTxInput(args *tokens.BuildTxArgs) {
	funcHash := erc20CodeParts["transfer"]
	address := common.HexToAddress(args.To)
	amount := tokens.CalcSwappedValue(args.Value, false)

	input := PackDataWithFuncHash(funcHash, address, amount)
	args.Input = &input // input

	token := b.TokenConfig
	args.To = token.ContractAddress // to
	args.Value = big.NewInt(0)      // value
}
