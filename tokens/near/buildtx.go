package near

import (
	"fmt"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// default values to calc tx gas and fees
var (
	DefaultFees   = "3000uluna"
	DefaultFeeCap = uint64(10000)

	DefaultGasLimit          = uint64(130000)
	DefaultPlusGasPercentage = uint64(20)

	DefaultGasPrice = 0.02
)

// GetDefaultExtras get default extras
func (b *Bridge) GetDefaultExtras() *tokens.AllExtras {
	return &tokens.AllExtras{TerraExtra: &tokens.TerraExtra{}}
}

// BuildRawTransaction build raw tx
func (b *Bridge) BuildRawTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	tokenCfg, err := b.getAndInitTokenConfig(args.PairID)
	if err != nil {
		return nil, err
	}

	switch args.SwapType {
	case tokens.SwapinType:
		return nil, tokens.ErrSwapTypeNotSupported
	case tokens.SwapoutType:
		return b.buildSwapoutTx(args, tokenCfg)
	default:
		return nil, tokens.ErrUnknownSwapType
	}
}

func (b *Bridge) getAndInitTokenConfig(pairID string) (tokenCfg *tokens.TokenConfig, err error) {
	tokenCfg = b.GetTokenConfig(pairID)
	if tokenCfg == nil {
		return nil, fmt.Errorf("swap pair '%v' is not configed", pairID)
	}
	if tokenCfg.DcrmAccountNumber == 0 {
		tokenCfg.DcrmAccountNumber, err = b.GetAccountNumber(tokenCfg.DcrmAddress)
		if err != nil {
			return nil, fmt.Errorf("init dcrm account number failed: %w", err)
		}
	}
	return tokenCfg, nil
}

func (b *Bridge) buildSwapoutTx(args *tokens.BuildTxArgs, tokenCfg *tokens.TokenConfig) (txb *TxBuilder, err error) {
	return nil, nil
}

// BuildTx build tx
func (b *Bridge) BuildTx(
	from, to, memo string,
	amount *big.Int,
	extra *tokens.TerraExtra,
	tokenCfg *tokens.TokenConfig,
) (*TxBuilder, error) {
	return nil, nil
}

func (b *Bridge) adjustFees(txb *TxBuilder, extra *tokens.TerraExtra, tokenCfg *tokens.TokenConfig) error {
	return nil
}

func (b *Bridge) simulateTx(txb *TxBuilder) (gasUsed uint64, err error) {
	return 0, nil
}

func (b *Bridge) initExtra(args *tokens.BuildTxArgs, tokenCfg *tokens.TokenConfig) (extra *tokens.TerraExtra, err error) {
	return nil, nil
}

func (b *Bridge) getMinReserveFee() *big.Int {
	minReserveFee := b.ChainConfig.GetMinReserveFee()
	if minReserveFee == nil {
		minReserveFee = big.NewInt(0)
	}
	return minReserveFee
}

func (b *Bridge) getSequence(args *tokens.BuildTxArgs) (uint64, error) {
	return 0, nil
}

func getOrInitExtra(args *tokens.BuildTxArgs) *tokens.TerraExtra {
	if args.Extra == nil || args.Extra.TerraExtra == nil {
		args.Extra = &tokens.AllExtras{TerraExtra: &tokens.TerraExtra{}}
	}
	return args.Extra.TerraExtra
}

// GetPoolNonce impl NonceSetter interface
func (b *Bridge) GetPoolNonce(address, _height string) (uint64, error) {
	return b.GetAccountSequence(address)
}

// GetAccountSequence get account sequence
func (b *Bridge) GetAccountSequence(address string) (uint64, error) {

	return 0, nil
}

// GetAccountNumber get account number
func (b *Bridge) GetAccountNumber(address string) (uint64, error) {

	return 0, nil
}

func (b *Bridge) checkCoinBalance(account, denom string, amount *big.Int) error {
	return nil
}

func (b *Bridge) checkTokenBalance(token, account string, amount *big.Int) error {
	return nil
}
