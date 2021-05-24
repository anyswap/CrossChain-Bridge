package eth

import (
	"errors"
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
	baseGasPrice   *big.Int

	errEmptyIdentifier       = errors.New("build swaptx without identifier")
	errNonEmptyInputData     = errors.New("build swap tx with non-empty input data")
	errNoSenderSpecified     = errors.New("build swaptx without specify sender")
	errNonzeroValueSpecified = errors.New("build swap tx with non-zero value")
)

func (b *Bridge) buildNonswapTx(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	extra, err := b.setDefaults(args)
	if err != nil {
		return nil, err
	}
	var input []byte
	if args.Input != nil {
		input = *args.Input
	}
	return b.buildTx(args, extra, input)
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

	extra, err := b.setDefaults(args)
	if err != nil {
		return nil, err
	}

	var input []byte

	switch args.SwapType {
	case tokens.SwapinType:
		err = b.buildSwapinTxInput(args)
		if err != nil {
			return nil, err
		}
		input = *args.Input
	case tokens.SwapoutType:
		err = b.buildSwapoutTxInput(args)
		if err != nil {
			return nil, err
		}
		input = *args.Input
	default:
		return nil, tokens.ErrUnknownSwapType
	}

	return b.buildTx(args, extra, input)
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

func (b *Bridge) buildTx(args *tokens.BuildTxArgs, extra *tokens.EthExtraArgs, input []byte) (rawTx interface{}, err error) {
	var (
		to       = common.HexToAddress(args.To)
		value    = args.Value
		nonce    = *extra.Nonce
		gasLimit = *extra.Gas
		gasPrice = extra.GasPrice
	)

	needValue := big.NewInt(0)
	if value != nil && value.Sign() > 0 {
		needValue = value
	}
	if args.SwapType != tokens.NoSwapType {
		needValue = new(big.Int).Add(needValue, getMinReserveFee())
	} else {
		gasFee := new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(gasLimit))
		needValue = new(big.Int).Add(needValue, gasFee)
	}
	err = b.checkBalance("", args.From, needValue)
	if err != nil {
		log.Warn("check balance failed", "account", args.From, "needValue", needValue, "err", err)
		return nil, err
	}

	rawTx = types.NewTransaction(nonce, to, value, gasLimit, gasPrice, input)

	log.Info("build raw tx", "pairID", args.PairID, "identifier", args.Identifier,
		"swapID", args.SwapID, "swapType", args.SwapType,
		"bind", args.Bind, "originValue", args.OriginValue,
		"from", args.From, "to", to.String(), "value", value, "nonce", nonce,
		"gasLimit", gasLimit, "gasPrice", gasPrice, "data", common.ToHex(input))

	return rawTx, nil
}

func getMinReserveFee() *big.Int {
	if minReserveFee != nil {
		return minReserveFee
	}
	if params.GetExtraConfig() == nil || params.GetExtraConfig().MinReserveFee == "" {
		minReserveFee = big.NewInt(1e16) // default 0.01 ETH
	} else {
		minReserveFee, _ = new(big.Int).SetString(params.GetExtraConfig().MinReserveFee, 10)
	}
	return minReserveFee
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
		extra.Gas = new(uint64)
		*extra.Gas = b.getDefaultGasLimit(args.PairID)
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
	return price, err
}

// args and oldGasPrice should be read only
func (b *Bridge) adjustSwapGasPrice(args *tokens.BuildTxArgs, oldGasPrice *big.Int) (newGasPrice *big.Int, err error) {
	tokenCfg := b.GetTokenConfig(args.PairID)
	if tokenCfg == nil {
		return nil, tokens.ErrUnknownPairID
	}
	addPercent := tokenCfg.PlusGasPricePercentage
	if args.ReplaceNum > 0 {
		addPercent += args.ReplaceNum * b.ChainConfig.ReplacePlusGasPricePercent
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
		if args.ReplaceNum == 0 { // exclude replace situation
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
