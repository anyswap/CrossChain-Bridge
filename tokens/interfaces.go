package tokens

import (
	"errors"
	"math"
	"math/big"
	"strings"
)

// transaction memo prefix
const (
	LockMemoPrefix   = "SWAPTO:"
	UnlockMemoPrefix = "SWAPTX:"
)

// common variables
var (
	SrcBridge CrossChainBridge
	DstBridge CrossChainBridge

	SrcLatestBlockHeight uint64
	DstLatestBlockHeight uint64
)

// common errors
var (
	ErrSwapTypeNotSupported          = errors.New("swap type not supported in this endpoint")
	ErrBridgeSourceNotSupported      = errors.New("bridge source not supported")
	ErrBridgeDestinationNotSupported = errors.New("bridge destination not supported")
	ErrUnknownSwapType               = errors.New("unknown swap type")
	ErrMsgHashMismatch               = errors.New("message hash mismatch")
	ErrWrongCountOfMsgHashes         = errors.New("wrong count of msg hashed")
	ErrWrongRawTx                    = errors.New("wrong raw tx")
	ErrWrongExtraArgs                = errors.New("wrong extra args")
	ErrNoBtcBridge                   = errors.New("no btc bridge exist")
	ErrWrongSwapinTxType             = errors.New("wrong swapin tx type")
	ErrBuildSwapTxInWrongEndpoint    = errors.New("build swap in/out tx in wrong endpoint")
	ErrTxBeforeInitialHeight         = errors.New("transaction before initial block height")
	ErrAddressIsInBlacklist          = errors.New("address is in black list")

	ErrTodo = errors.New("developing: TODO")

	ErrTxNotFound           = errors.New("tx not found")
	ErrTxNotStable          = errors.New("tx not stable")
	ErrTxWithWrongReceiver  = errors.New("tx with wrong receiver")
	ErrTxWithWrongContract  = errors.New("tx with wrong contract")
	ErrTxWithWrongInput     = errors.New("tx with wrong input data")
	ErrTxWithWrongLogData   = errors.New("tx with wrong log data")
	ErrTxIsAggregateTx      = errors.New("tx is aggregate tx")
	ErrWrongP2shBindAddress = errors.New("wrong p2sh bind address")
	ErrTxFuncHashMismatch   = errors.New("tx func hash mismatch")
	ErrDepositLogNotFound   = errors.New("deposit log not found or removed")
	ErrSwapoutLogNotFound   = errors.New("swapout log not found or removed")
	ErrUnknownPairID        = errors.New("unknown pair ID")

	// errors should register
	ErrTxWithWrongMemo          = errors.New("tx with wrong memo")
	ErrTxWithWrongValue         = errors.New("tx with wrong value")
	ErrTxWithWrongReceipt       = errors.New("tx with wrong receipt")
	ErrTxWithWrongSender        = errors.New("tx with wrong sender")
	ErrTxSenderNotRegistered    = errors.New("tx sender not registered")
	ErrTxIncompatible           = errors.New("tx incompatible")
	ErrBindAddrIsContract       = errors.New("bind address is contract")
	ErrRPCQueryError            = errors.New("rpc query error")
	ErrTxWithLockTimeOrSequence = errors.New("tx with lock time or sequenece")
)

// ShouldRegisterSwapForError return true if this error should record in database
func ShouldRegisterSwapForError(err error) bool {
	switch err {
	case nil,
		ErrTxWithWrongMemo,
		ErrTxWithWrongValue,
		ErrTxWithWrongReceipt,
		ErrTxWithWrongSender,
		ErrTxSenderNotRegistered,
		ErrTxIncompatible,
		ErrBindAddrIsContract,
		ErrRPCQueryError,
		ErrTxWithLockTimeOrSequence:
		return true
	}
	return false
}

// NonceGetter interface
type NonceGetter interface {
	GetPoolNonce(address, height string) (uint64, error)
}

// CrossChainBridge interface
type CrossChainBridge interface {
	IsSrcEndpoint() bool

	SetChainAndGateway(*ChainConfig, *GatewayConfig)

	GetChainConfig() *ChainConfig
	GetGatewayConfig() *GatewayConfig
	GetTokenConfig(pairID string) *TokenConfig

	IsValidAddress(address string) bool

	GetTransaction(txHash string) (interface{}, error)
	GetTransactionStatus(txHash string) *TxStatus
	VerifyTransaction(txHash string, allowUnstable bool) ([]*TxSwapInfo, []error)
	VerifyTransactionWithPairID(pairID, txHash string) (*TxSwapInfo, error)
	VerifyMsgHash(rawTx interface{}, msgHash []string) error

	BuildRawTransaction(args *BuildTxArgs) (rawTx interface{}, err error)
	DcrmSignTransaction(rawTx interface{}, args *BuildTxArgs) (signedTx interface{}, txHash string, err error)
	SendTransaction(signedTx interface{}) (txHash string, err error)

	GetLatestBlockNumber() (uint64, error)

	StartPoolTransactionScanJob()
	StartChainTransactionScanJob()

	AdjustNonce(pairID string, value uint64) (nonce uint64)
	IncreaseNonce(pairID string, value uint64)

	VerifyTokenConfig(*TokenConfig)

	GetBalance(accountAddress string) (*big.Int, error)
	GetTokenBalance(tokenType, tokenAddress, accountAddress string) (*big.Int, error)
	GetTokenSupply(tokenType, tokenAddress string) (*big.Int, error)
}

// SetLatestBlockHeight set latest block height
func SetLatestBlockHeight(latest uint64, isSrc bool) {
	if isSrc {
		SrcLatestBlockHeight = latest
	} else {
		DstLatestBlockHeight = latest
	}
}

// CrossChainBridgeBase base bridge
type CrossChainBridgeBase struct {
	ChainConfig   *ChainConfig
	GatewayConfig *GatewayConfig
	IsSrc         bool
	SwapinNonce   map[string]uint64
	SwapoutNonce  map[string]uint64
}

// NewCrossChainBridgeBase new base bridge
func NewCrossChainBridgeBase(isSrc bool) *CrossChainBridgeBase {
	return &CrossChainBridgeBase{IsSrc: isSrc}
}

// AdjustNonce adjust account nonce (eth like chain)
func (b *CrossChainBridgeBase) AdjustNonce(pairID string, value uint64) (nonce uint64) {
	tokenCfg := b.GetTokenConfig(pairID)
	account := strings.ToLower(tokenCfg.DcrmAddress)
	nonce = value
	if b.IsSrcEndpoint() {
		if b.SwapoutNonce[account] > value {
			nonce = b.SwapoutNonce[account]
		} else {
			b.SwapoutNonce[account] = value
		}
	} else {
		if b.SwapinNonce[account] > value {
			nonce = b.SwapinNonce[account]
		} else {
			b.SwapinNonce[account] = value
		}
	}
	return nonce
}

// IncreaseNonce decrease account nonce (eth like chain)
func (b *CrossChainBridgeBase) IncreaseNonce(pairID string, value uint64) {
	tokenCfg := b.GetTokenConfig(pairID)
	account := strings.ToLower(tokenCfg.DcrmAddress)
	if b.IsSrcEndpoint() {
		b.SwapoutNonce[account] += value
	} else {
		b.SwapinNonce[account] += value
	}
}

// IsSrcEndpoint returns if bridge is at the source endpoint
func (b *CrossChainBridgeBase) IsSrcEndpoint() bool {
	return b.IsSrc
}

// SetChainAndGateway set chain and gateway config
func (b *CrossChainBridgeBase) SetChainAndGateway(chainCfg *ChainConfig, gatewayCfg *GatewayConfig) {
	b.ChainConfig = chainCfg
	b.GatewayConfig = gatewayCfg
}

// GetChainConfig get chain config
func (b *CrossChainBridgeBase) GetChainConfig() *ChainConfig {
	return b.ChainConfig
}

// GetGatewayConfig get gateway config
func (b *CrossChainBridgeBase) GetGatewayConfig() *GatewayConfig {
	return b.GatewayConfig
}

// GetTokenConfig get token config
func (b *CrossChainBridgeBase) GetTokenConfig(pairID string) *TokenConfig {
	return GetTokenConfig(pairID, b.IsSrcEndpoint())
}

// GetCrossChainBridge get bridge of specified endpoint
func GetCrossChainBridge(isSrc bool) CrossChainBridge {
	if isSrc {
		return SrcBridge
	}
	return DstBridge
}

// FromBits convert from bits
func FromBits(value *big.Int, decimals uint8) float64 {
	oneToken := math.Pow(10, float64(decimals))
	fOneToken := new(big.Float).SetFloat64(oneToken)
	fValue := new(big.Float).SetInt(value)
	fTokens := new(big.Float).Quo(fValue, fOneToken)
	result, _ := fTokens.Float64()
	return result
}

// ToBits convert to bits
func ToBits(value float64, decimals uint8) *big.Int {
	oneToken := math.Pow(10, float64(decimals))
	fOneToken := new(big.Float).SetFloat64(oneToken)
	fValue := new(big.Float).SetFloat64(value)
	fBits := new(big.Float).Mul(fValue, fOneToken)

	result := big.NewInt(0)
	fBits.Int(result)
	return result
}

// GetBigValueThreshold get big value threshold
func GetBigValueThreshold(pairID string, isSrc bool) *big.Int {
	token := GetTokenConfig(pairID, isSrc)
	return token.bigValThreshhold
}

// CheckSwapValue check swap value is in right range
func CheckSwapValue(pairID string, value *big.Int, isSrc bool) bool {
	token := GetTokenConfig(pairID, isSrc)
	if value.Cmp(token.minSwap) < 0 {
		return false
	}
	if value.Cmp(token.maxSwap) > 0 {
		return false
	}
	swappedValue := CalcSwappedValue(pairID, value, isSrc)
	return swappedValue.Sign() > 0
}

// CalcSwappedValue calc swapped value (get rid of fee)
func CalcSwappedValue(pairID string, value *big.Int, isSrc bool) *big.Int {
	token := GetTokenConfig(pairID, isSrc)

	if *token.SwapFeeRate == 0.0 {
		return value
	}

	swapValue := new(big.Float).SetInt(value)
	swapFeeRate := new(big.Float).SetFloat64(*token.SwapFeeRate)
	swapFeeFloat := new(big.Float).Mul(swapValue, swapFeeRate)

	swapFee := big.NewInt(0)
	swapFeeFloat.Int(swapFee)

	if swapFee.Cmp(token.minSwapFee) < 0 {
		swapFee = token.minSwapFee
	} else if swapFee.Cmp(token.maxSwapFee) > 0 {
		swapFee = token.maxSwapFee
	}

	if value.Cmp(swapFee) > 0 {
		return new(big.Int).Sub(value, swapFee)
	}
	return big.NewInt(0)
}
