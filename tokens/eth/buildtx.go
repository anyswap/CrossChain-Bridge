package eth

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/types"
)

var (
	retryRPCCount    = 3
	retryRPCInterval = 1 * time.Second

	minReserveFee  *big.Int
	latestGasPrice *big.Int
	baseGasPrice   *big.Int

	errEmptyIdentifier       = errors.New("build swaptx without identifier")
	errNonEmptyInputData     = errors.New("build swap tx with non-empty input data")
	errNoSenderSpecified     = errors.New("build swaptx without specify sender")
	errNonzeroValueSpecified = errors.New("build swap tx with non-zero value")
)

func (b *Bridge) buildNonswapTx(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	err = b.setDefaults(args)
	if err != nil {
		return nil, err
	}
	return b.buildTx(args)
}

// BuildRawTransaction build raw tx
func (b *Bridge) BuildRawTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	if args.SwapType == tokens.NoSwapType {
		return b.buildNonswapTx(args)
	}

	err = b.checkBuildTxArgs(args)
	if err != nil {
		return nil, err
	}

	err = b.setDefaults(args)
	if err != nil {
		return nil, err
	}

	switch args.SwapType {
	case tokens.SwapinType:
		err = b.buildSwapinTxInput(args)
	case tokens.SwapoutType:
		err = b.buildSwapoutTxInput(args)
	default:
		return nil, tokens.ErrUnknownSwapType
	}

	if err != nil {
		return nil, err
	}

	return b.buildTx(args)
}

func (b *Bridge) checkBuildTxArgs(args *tokens.BuildTxArgs) error {
	if args.Identifier == "" {
		return errEmptyIdentifier
	}
	if args.Input != nil {
		return errNonEmptyInputData
	}
	if args.From == "" {
		return errNoSenderSpecified
	}
	if args.Value != nil && args.Value.Sign() != 0 {
		return errNonzeroValueSpecified
	}

	switch args.SwapType {
	case tokens.SwapinType:
		if b.IsSrc {
			return tokens.ErrBuildSwapTxInWrongEndpoint
		}
	case tokens.SwapoutType:
		if !b.IsSrc {
			return tokens.ErrBuildSwapTxInWrongEndpoint
		}
	default:
		return tokens.ErrUnknownSwapType
	}

	return nil
}

func (b *Bridge) buildTx(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	var (
		to        = common.HexToAddress(args.To)
		value     = args.Value
		extra     = args.Extra.EthExtra
		nonce     = *extra.Nonce
		gasLimit  = *extra.Gas
		gasPrice  = extra.GasPrice
		gasTipCap = extra.GasTipCap
		gasFeeCap = extra.GasFeeCap

		isDynamicFeeTx = b.ChainConfig.EnableDynamicFeeTx
	)

	var input []byte
	if args.Input != nil {
		input = *args.Input
	}

	needValue := big.NewInt(0)
	if value != nil && value.Sign() > 0 {
		needValue = value
	}
	if args.SwapType != tokens.NoSwapType {
		needValue = new(big.Int).Add(needValue, b.getMinReserveFee())
	} else if isDynamicFeeTx {
		gasFee := new(big.Int).Mul(gasFeeCap, new(big.Int).SetUint64(gasLimit))
		needValue = new(big.Int).Add(needValue, gasFee)
	} else {
		gasFee := new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(gasLimit))
		needValue = new(big.Int).Add(needValue, gasFee)
	}
	err = b.checkBalance("", args.From, needValue)
	if err != nil {
		log.Warn("check balance failed", "account", args.From, "needValue", needValue, "err", err)
		return nil, err
	}

	if isDynamicFeeTx {
		rawTx = types.NewDynamicFeeTx(b.SignerChainID, nonce, &to, value, gasLimit, gasTipCap, gasFeeCap, input, nil)
	} else {
		rawTx = types.NewTransaction(nonce, to, value, gasLimit, gasPrice, input)
	}

	ctx := []interface{}{"identifier", args.Identifier,
		"chainID", b.SignerChainID, "pairID", args.PairID, "swapID", args.SwapID,
		"from", args.From, "to", to.String(), "nonce", nonce, "bind", args.Bind,
		"originValue", args.OriginValue, "swapValue", args.SwapValue,
		"gasLimit", gasLimit, "data", common.ToHex(input),
		"replaceNum", args.GetReplaceNum(),
	}
	if gasTipCap != nil || gasFeeCap != nil {
		ctx = append(ctx, "gasTipCap", gasTipCap, "gasFeeCap", gasFeeCap)
	} else {
		ctx = append(ctx, "gasPrice", gasPrice)
	}
	log.Info(fmt.Sprintf("build %s raw tx", args.SwapType.String()), ctx...)

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

func (b *Bridge) setDefaults(args *tokens.BuildTxArgs) (err error) {
	if args.Value == nil {
		args.Value = new(big.Int)
	}
	if args.Extra == nil || args.Extra.EthExtra == nil {
		args.Extra = &tokens.AllExtras{EthExtra: &tokens.EthExtraArgs{}}
	}
	extra := args.Extra.EthExtra

	if b.ChainConfig.EnableDynamicFeeTx {
		if extra.GasTipCap == nil {
			extra.GasTipCap, err = b.getGasTipCap(args)
			if err != nil {
				return err
			}
		}
		if extra.GasFeeCap == nil {
			extra.GasFeeCap, err = b.getGasFeeCap(args, extra.GasTipCap)
			if err != nil {
				return err
			}
		}
		extra.GasPrice = nil
	} else if extra.GasPrice == nil {
		extra.GasPrice, err = b.getGasPrice(args)
		if err != nil {
			return err
		}
		extra.GasTipCap = nil
		extra.GasFeeCap = nil
	}
	if extra.Nonce == nil {
		extra.Nonce, err = b.getAccountNonce(args.PairID, args.From, args.SwapType)
		if err != nil {
			return err
		}
	}
	if extra.Gas == nil {
		var input []byte
		if args.Input != nil {
			input = *args.Input
		}

		esGasLimit, errf := b.EstimateGas(args.From, args.To, args.Value, input)
		if errf != nil {
			log.Error(fmt.Sprintf("build %s tx estimate gas failed", args.SwapType.String()),
				"swapID", args.SwapID, "from", args.From, "to", args.To,
				"value", args.Value, "data", common.ToHex(input), "err", errf)
			return tokens.ErrEstimateGasFailed
		}

		esGasLimit += esGasLimit * 30 / 100
		defGasLimit := b.getDefaultGasLimit(args.PairID)
		if esGasLimit < defGasLimit {
			esGasLimit = defGasLimit
		}
		extra.Gas = new(uint64)
		*extra.Gas = esGasLimit
	}
	return nil
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
		return fixedGasPrice, nil
	}

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

	if args != nil && args.SwapType != tokens.NoSwapType {
		price, err = b.adjustSwapGasPrice(args, price)
		if err != nil {
			return nil, err
		}
	}
	if baseGasPrice != nil {
		maxGasPrice := new(big.Int).Mul(baseGasPrice, big.NewInt(10))
		if price.Cmp(maxGasPrice) > 0 {
			log.Info("gas price exceeds upper bound", "baseGasPrice", baseGasPrice, "maxGasPrice", maxGasPrice, "price", price)
			price = maxGasPrice
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
	addPercent := tokenCfg.PlusGasPricePercentage
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

func (b *Bridge) getGasTipCap(args *tokens.BuildTxArgs) (gasTipCap *big.Int, err error) {
	for i := 0; i < retryRPCCount; i++ {
		gasTipCap, err = b.SuggestGasTipCap()
		if err == nil {
			break
		}
		time.Sleep(retryRPCInterval)
	}
	if err != nil {
		return nil, err
	}
	if args == nil || args.SwapType == tokens.NoSwapType {
		return gasTipCap, err
	}

	addPercent := b.ChainConfig.PlusGasTipCapPercent
	replaceNum := args.GetReplaceNum()
	if replaceNum > 0 {
		addPercent += replaceNum * b.ChainConfig.ReplacePlusGasPricePercent
	}
	if addPercent > tokens.MaxPlusGasPricePercentage {
		addPercent = tokens.MaxPlusGasPricePercentage
	}
	if addPercent > 0 {
		gasTipCap.Mul(gasTipCap, big.NewInt(int64(100+addPercent)))
		gasTipCap.Div(gasTipCap, big.NewInt(100))
	}

	maxGasTipCap := b.ChainConfig.GetMaxGasTipCap()
	if maxGasTipCap != nil && gasTipCap.Cmp(maxGasTipCap) > 0 {
		gasTipCap = maxGasTipCap
	}
	return gasTipCap, nil
}

func (b *Bridge) getGasFeeCap(args *tokens.BuildTxArgs, gasTipCap *big.Int) (gasFeeCap *big.Int, err error) {
	blockCount := b.ChainConfig.BlockCountFeeHistory
	var baseFee *big.Int
	for i := 0; i < retryRPCCount; i++ {
		baseFee, err = b.GetBaseFee(blockCount)
		if err == nil {
			break
		}
		time.Sleep(retryRPCInterval)
	}
	if err != nil {
		return nil, err
	}

	newGasFeeCap := new(big.Int).Set(gasTipCap) // copy
	newGasFeeCap.Add(newGasFeeCap, baseFee.Mul(baseFee, big.NewInt(2)))
	if args == nil || args.SwapType == tokens.NoSwapType {
		return newGasFeeCap, err
	}

	newGasFeeCap.Mul(newGasFeeCap, big.NewInt(int64(100+b.ChainConfig.PlusGasFeeCapPercent)))
	newGasFeeCap.Div(newGasFeeCap, big.NewInt(100))

	maxGasFeeCap := b.ChainConfig.GetMaxGasFeeCap()
	if maxGasFeeCap != nil && newGasFeeCap.Cmp(maxGasFeeCap) > 0 {
		newGasFeeCap = maxGasFeeCap
	}
	return newGasFeeCap, nil
}
