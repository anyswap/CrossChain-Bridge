package eth

import (
	"fmt"
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

	minReserveFee  *big.Int
	latestGasPrice *big.Int
)

// BuildRawTransaction build raw tx
func (b *Bridge) BuildRawTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	var input []byte
	var tokenCfg *tokens.TokenConfig
	if args.Input == nil {
		if args.SwapType != tokens.NoSwapType {
			pairID := args.PairID
			tokenCfg = b.GetTokenConfig(pairID)
			if tokenCfg == nil {
				return nil, tokens.ErrUnknownPairID
			}
			if args.From == "" {
				args.From = tokenCfg.DcrmAddress // from
			}
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
				args.To = args.Bind
				input = b.getUnlockCoinMemo(args)
			}
		default:
		}
	} else {
		input = *args.Input
		if args.SwapType != tokens.NoSwapType {
			return nil, fmt.Errorf("forbid build raw swap tx with input data")
		}
	}

	extra, err := b.setDefaults(args, input)
	if err != nil {
		return nil, err
	}

	return b.buildTx(args, extra, input)
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
		if tokenCfg == nil {
			return nil, tokens.ErrUnknownPairID
		}
		if !tokenCfg.IsErc20() {
			value = tokens.CalcSwappedValue(pairID, args.OriginValue, false, args.OriginFrom, args.OriginTxTo)
		}
	}

	if args.SwapType != tokens.NoSwapType {
		args.Identifier = params.GetIdentifier()
	}

	needValue := big.NewInt(0)
	if value != nil && value.Sign() > 0 {
		needValue = value
	}
	if args.SwapType != tokens.NoSwapType {
		needValue = new(big.Int).Add(needValue, b.getMinReserveFee())
	} else {
		gasFee := new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(gasLimit))
		needValue = new(big.Int).Add(needValue, gasFee)
	}
	err = b.checkBalance("", args.From, needValue)
	if err != nil {
		return nil, err
	}

	rawTx = types.NewTransaction(nonce, to, value, gasLimit, gasPrice, input)

	log.Info("build raw tx", "pairID", args.PairID, "identifier", args.Identifier,
		"swapID", args.SwapID, "swapType", args.SwapType,
		"bind", args.Bind, "originValue", args.OriginValue,
		"from", args.From, "to", to.String(), "value", value, "nonce", nonce,
		"gasLimit", gasLimit, "gasPrice", gasPrice, "data", common.ToHex(input),
		"replaceNum", args.GetReplaceNum(),
	)

	return rawTx, nil
}

func (b *Bridge) getMinReserveFee() *big.Int {
	if minReserveFee != nil {
		return minReserveFee
	}
	minReserveFee = b.ChainConfig.GetMinReserveFee()
	if minReserveFee == nil {
		minReserveFee = big.NewInt(1e17) // default 0.1 ETH
	}
	return minReserveFee
}

func (b *Bridge) setDefaults(args *tokens.BuildTxArgs, input []byte) (extra *tokens.EthExtraArgs, err error) {
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
		extra.GasPrice, err = b.getGasPrice(args)
		if err != nil {
			return nil, err
		}
	}
	if extra.Nonce == nil {
		extra.Nonce, err = b.getAccountNonce(args.PairID, args.From, args.SwapType)
		if err != nil {
			return nil, err
		}
	}
	if extra.Gas == nil {
		esGasLimit, errf := b.EstimateGas(args.From, args.To, args.Value, input)
		if errf != nil {
			log.Error(fmt.Sprintf("build %s tx estimate gas failed", args.SwapType.String()),
				"swapID", args.SwapID, "from", args.From, "to", args.To,
				"value", args.Value, "data", common.ToHex(input), "err", errf)
			return nil, tokens.ErrEstimateGasFailed
		}

		esGasLimit += esGasLimit * 30 / 100
		defGasLimit := b.getDefaultGasLimit(args.PairID)
		if esGasLimit < defGasLimit {
			esGasLimit = defGasLimit
		}
		extra.Gas = new(uint64)
		*extra.Gas = esGasLimit
	}
	return extra, nil
}

func (b *Bridge) getDefaultGasLimit(pairID string) (gasLimit uint64) {
	tokenCfg := b.GetTokenConfig(pairID)
	if tokenCfg != nil {
		gasLimit = tokenCfg.DefaultGasLimit
	}
	if gasLimit == 0 {
		gasLimit = 90000
	}
	return gasLimit
}

func (b *Bridge) getGasPrice(args *tokens.BuildTxArgs) (price *big.Int, err error) {
	fixedGasPrice := b.ChainConfig.GetFixedGasPrice()
	if fixedGasPrice != nil {
		price = fixedGasPrice
		if args.GetReplaceNum() == 0 {
			return price, nil
		}
	} else {
		for i := 0; i < retryRPCCount; i++ {
			price, err = b.SuggestPrice()
			if err == nil {
				break
			}
			time.Sleep(retryRPCInterval)
		}
		if err != nil {
			return nil, err
		}

		minGasPrice := b.ChainConfig.GetMinGasPrice()
		if minGasPrice != nil && price.Cmp(minGasPrice) < 0 {
			price = minGasPrice
		}
	}

	if args != nil && args.SwapType != tokens.NoSwapType {
		price, err = b.adjustSwapGasPrice(args, price)
		if err != nil {
			return nil, err
		}
	}

	maxGasPrice := b.ChainConfig.GetMaxGasPrice()
	if maxGasPrice != nil && price.Cmp(maxGasPrice) > 0 {
		return nil, fmt.Errorf("gas price %v exceeded maximum limit", price)
	}

	return price, err
}

// args and oldGasPrice should be read only
func (b *Bridge) adjustSwapGasPrice(args *tokens.BuildTxArgs, oldGasPrice *big.Int) (newGasPrice *big.Int, err error) {
	tokenCfg := b.GetTokenConfig(args.PairID)
	if tokenCfg == nil {
		return nil, tokens.ErrUnknownPairID
	}
	addPercent := uint64(0)
	if !b.ChainConfig.IsFixedGasPrice() {
		addPercent = tokenCfg.PlusGasPricePercentage
	}
	replaceNum := args.GetReplaceNum()
	if replaceNum > 0 {
		addPercent += replaceNum * b.ChainConfig.ReplacePlusGasPricePercent
	}
	if addPercent > tokens.MaxPlusGasPricePercentage {
		addPercent = tokens.MaxPlusGasPricePercentage
	}
	newGasPrice = new(big.Int).Set(oldGasPrice) // clone from old
	if addPercent > 0 {
		newGasPrice.Mul(newGasPrice, big.NewInt(int64(100+addPercent)))
		newGasPrice.Div(newGasPrice, big.NewInt(100))
	}
	maxGasPriceFluctPercent := b.ChainConfig.MaxGasPriceFluctPercent
	if maxGasPriceFluctPercent > 0 {
		if latestGasPrice != nil && newGasPrice.Cmp(latestGasPrice) < 0 {
			maxFluct := new(big.Int).Set(latestGasPrice)
			maxFluct.Mul(maxFluct, new(big.Int).SetUint64(maxGasPriceFluctPercent))
			maxFluct.Div(maxFluct, big.NewInt(100))
			minGasPrice := new(big.Int).Sub(latestGasPrice, maxFluct)
			if newGasPrice.Cmp(minGasPrice) < 0 {
				newGasPrice = minGasPrice
			}
		}
		if replaceNum == 0 { // exclude replace situation
			latestGasPrice = newGasPrice
		}
	}
	return newGasPrice, nil
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
	if swapType != tokens.NoSwapType {
		tokenCfg := b.GetTokenConfig(pairID)
		if tokenCfg != nil && from == tokenCfg.DcrmAddress {
			nonce = b.AdjustNonce(pairID, nonce)
		}
	}
	return &nonce, nil
}

func (b *Bridge) checkBalance(token, account string, amount *big.Int) (err error) {
	var balance *big.Int
	for i := 0; i < retryRPCCount; i++ {
		if token != "" {
			balance, err = b.GetErc20Balance(token, account)
		} else {
			balance, err = b.GetBalance(account)
		}
		if err == nil {
			break
		}
		time.Sleep(retryRPCInterval)
	}
	if err == nil && balance.Cmp(amount) < 0 {
		return fmt.Errorf("not enough %v balance. %v < %v", token, balance, amount)
	}
	if err != nil {
		log.Warn("get balance error", "token", token, "account", account, "err", err)
	}
	return err
}
