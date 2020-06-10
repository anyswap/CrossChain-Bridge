package eth

import (
	"math/big"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/params"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

var (
	swapinNonce uint64

	retryRPCCount    = 3
	retryRPCInterval = 1 * time.Second
)

// BuildRawTransaction build raw tx
func (b *Bridge) BuildRawTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	isSwapin := args.SwapType == tokens.SwapinType
	if isSwapin && args.Input == nil {
		b.buildSwapinTxInput(args)
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
		input    []byte
	)
	if args.Input != nil {
		input = *args.Input
	}

	switch args.SwapType {
	case tokens.SwapoutType, tokens.SwapRecallType:
		value = tokens.CalcSwappedValue(value, b.IsSrc)
	}

	if args.SwapType != tokens.NoSwapType {
		args.Identifier = params.GetIdentifier()
	}

	return types.NewTransaction(nonce, to, value, gasLimit, gasPrice, input), nil
}

func (b *Bridge) setDefaults(args *tokens.BuildTxArgs) (*tokens.EthExtraArgs, error) {
	if args.Value == nil {
		args.Value = new(big.Int)
	}
	var extra *tokens.EthExtraArgs
	if args.Extra == nil || args.Extra.EthExtra == nil {
		extra = &tokens.EthExtraArgs{}
		args.Extra = &tokens.AllExtras{EthExtra: extra}
	} else {
		extra = args.Extra.EthExtra
	}
	var err error
	if extra.GasPrice == nil {
		var price *big.Int
		for i := 0; i < retryRPCCount; i++ {
			price, err = b.SuggestPrice()
			if err == nil {
				break
			}
			if i+1 == retryRPCCount {
				return nil, err
			}
			time.Sleep(retryRPCInterval)
		}
		extra.GasPrice = price
	}
	if extra.Nonce == nil {
		var nonce uint64
		for i := 0; i < retryRPCCount; i++ {
			nonce, err = b.GetPoolNonce(args.From)
			if err == nil {
				break
			}
			if i+1 == retryRPCCount {
				return nil, err
			}
			time.Sleep(retryRPCInterval)
		}
		if args.SwapType == tokens.SwapinType &&
			args.From == b.TokenConfig.DcrmAddress {
			if swapinNonce >= nonce {
				swapinNonce++
				nonce = swapinNonce
			} else {
				swapinNonce = nonce
			}
		}
		extra.Nonce = &nonce
	}
	if extra.Gas == nil {
		extra.Gas = new(uint64)
		*extra.Gas = 90000
	}
	return extra, nil
}

// build input for calling `Swapin(bytes32 txhash, address account, uint256 amount)`
func (b *Bridge) buildSwapinTxInput(args *tokens.BuildTxArgs) {
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
	args.From = token.DcrmAddress   // from
	args.To = token.ContractAddress // to
	args.Value = big.NewInt(0)      // value
}
