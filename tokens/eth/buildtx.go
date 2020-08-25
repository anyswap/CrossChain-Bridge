package eth

import (
	"errors"
	"math/big"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/types"
)

var (
	retryRPCCount    = 3
	retryRPCInterval = 1 * time.Second

	defPlusGasPricePercentage uint64 = 15 // 15%
)

// BuildRawTransaction build raw tx
func (b *Bridge) BuildRawTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	var input []byte
	if args.Input == nil {
		pairID := args.PairID
		tokenCfg := b.GetTokenConfig(pairID)
		if args.SwapType != tokens.NoSwapType {
			args.From = tokenCfg.DcrmAddress // from
		}
		switch args.SwapType {
		case tokens.SwapinType:
			if b.IsSrc {
				return nil, tokens.ErrBuildSwapTxInWrongEndpoint
			}
			err = b.buildSwapinTxInput(args)
			if err != nil {
				return nil, err
			}
			input = *args.Input
		case tokens.SwapoutType:
			if !b.IsSrc {
				return nil, tokens.ErrBuildSwapTxInWrongEndpoint
			}
			if tokenCfg.IsErc20() {
				err = b.buildErc20SwapoutTxInput(args)
				if err != nil {
					return nil, err
				}
				input = *args.Input
			} else {
				input = []byte(tokens.UnlockMemoPrefix + args.SwapID)
			}
		}
	} else {
		input = *args.Input
	}

	extra, err := b.setDefaults(args)
	if err != nil {
		return nil, err
	}

	return b.buildTx(args, extra, input)
}

func (b *Bridge) buildTx(args *tokens.BuildTxArgs, extra *tokens.EthExtraArgs, input []byte) (rawTx interface{}, err error) {
	var (
		to       = common.HexToAddress(args.To)
		value    = args.Value
		nonce    = *extra.Nonce
		gasLimit = *extra.Gas
		gasPrice = extra.GasPrice
	)

	if args.SwapType == tokens.SwapoutType {
		pairID := args.PairID
		tokenCfg := b.GetTokenConfig(pairID)
		if !tokenCfg.IsErc20() {
			value = tokens.CalcSwappedValue(pairID, value, false)
		}
	}

	if args.SwapType != tokens.NoSwapType {
		args.Identifier = params.GetIdentifier()
	}

	if value != nil && value.Sign() > 0 {
		var balance *big.Int
		for i := 0; i < retryRPCCount; i++ {
			balance, err = b.GetBalance(args.From)
			if err == nil {
				break
			}
			time.Sleep(retryRPCInterval)
		}
		if err == nil && balance.Cmp(value) < 0 {
			return nil, errors.New("not enough coin balance")
		}
		if err != nil {
			return nil, err
		}
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
		pairID := args.PairID
		tokenCfg := b.GetTokenConfig(pairID)
		addPercent := tokenCfg.PlusGasPricePercentage
		if addPercent == 0 {
			addPercent = defPlusGasPricePercentage
		}
		extra.GasPrice.Mul(extra.GasPrice, big.NewInt(int64(100+addPercent)))
		extra.GasPrice.Div(extra.GasPrice, big.NewInt(100))
	}
	if extra.Nonce == nil {
		extra.Nonce, err = b.getAccountNonce(args.PairID, args.From, args.SwapType)
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

func (b *Bridge) getAccountNonce(pairID, from string, swapType tokens.SwapType) (nonceptr *uint64, err error) {
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
	tokenCfg := b.GetTokenConfig(pairID)
	if tokenCfg != nil && from == tokenCfg.DcrmAddress {
		if swapType != tokens.NoSwapType {
			nonce = b.AdjustNonce(pairID, nonce)
		}
	}
	return &nonce, nil
}

// build input for calling `Swapin(bytes32 txhash, address account, uint256 amount)`
func (b *Bridge) buildSwapinTxInput(args *tokens.BuildTxArgs) error {
	pairID := args.PairID
	funcHash := getSwapinFuncHash()
	txHash := common.HexToHash(args.SwapID)
	address := common.HexToAddress(args.To)
	if address == (common.Address{}) || !common.IsHexAddress(args.To) {
		log.Warn("swapin to wrong address", "address", args.To)
		return errors.New("can not swapin to empty or invalid address")
	}
	amount := tokens.CalcSwappedValue(pairID, args.Value, true)

	input := PackDataWithFuncHash(funcHash, txHash, address, amount)
	args.Input = &input // input

	token := b.GetTokenConfig(pairID)
	args.To = token.ContractAddress // to
	args.Value = big.NewInt(0)      // value
	return nil
}

func (b *Bridge) buildErc20SwapoutTxInput(args *tokens.BuildTxArgs) (err error) {
	pairID := args.PairID
	funcHash := erc20CodeParts["transfer"]
	address := common.HexToAddress(args.To)
	if address == (common.Address{}) || !common.IsHexAddress(args.To) {
		log.Warn("swapout to wrong address", "address", args.To)
		return errors.New("can not swapout to empty or invalid address")
	}
	amount := tokens.CalcSwappedValue(pairID, args.Value, false)

	input := PackDataWithFuncHash(funcHash, address, amount)
	args.Input = &input // input

	token := b.GetTokenConfig(pairID)
	args.To = token.ContractAddress // to
	args.Value = big.NewInt(0)      // value

	var balance *big.Int
	for i := 0; i < retryRPCCount; i++ {
		balance, err = b.GetErc20Balance(token.ContractAddress, token.DcrmAddress)
		if err == nil {
			break
		}
		time.Sleep(retryRPCInterval)
	}
	if err == nil && balance.Cmp(amount) < 0 {
		return errors.New("not enough token balance to swapout")
	}
	return err
}
