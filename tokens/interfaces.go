package tokens

import (
	"errors"
	"math"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/log"
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

	// errors should register
	ErrTxWithWrongMemo       = errors.New("tx with wrong memo")
	ErrTxWithWrongValue      = errors.New("tx with wrong value")
	ErrTxWithWrongReceipt    = errors.New("tx with wrong receipt")
	ErrTxWithWrongSender     = errors.New("tx with wrong sender")
	ErrTxSenderNotRegistered = errors.New("tx sender not registered")
	ErrTxIncompatible        = errors.New("tx incompatible")
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
		ErrTxIncompatible:
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
	GetTokenAndGateway() (*TokenConfig, *GatewayConfig)
	SetTokenAndGateway(*TokenConfig, *GatewayConfig)
	SetTokenAndGatewayWithoutCheck(*TokenConfig, *GatewayConfig)

	IsValidAddress(address string) bool

	GetTransaction(txHash string) (interface{}, error)
	GetTransactionStatus(txHash string) *TxStatus
	VerifyTransaction(txHash string, allowUnstable bool) (*TxSwapInfo, error)
	VerifyMsgHash(rawTx interface{}, msgHash []string, extra interface{}) error

	BuildRawTransaction(args *BuildTxArgs) (rawTx interface{}, err error)
	DcrmSignTransaction(rawTx interface{}, args *BuildTxArgs) (signedTx interface{}, txHash string, err error)
	SendTransaction(signedTx interface{}) (txHash string, err error)

	GetLatestBlockNumber() (uint64, error)

	StartPoolTransactionScanJob()
	StartChainTransactionScanJob()
	StartSwapHistoryScanJob()

	AdjustNonce(value uint64) (nonce uint64)
	IncreaseNonce(value uint64)
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
	TokenConfig   *TokenConfig
	GatewayConfig *GatewayConfig
	IsSrc         bool
	SwapinNonce   uint64
	SwapoutNonce  uint64
}

// NewCrossChainBridgeBase new base bridge
func NewCrossChainBridgeBase(isSrc bool) *CrossChainBridgeBase {
	return &CrossChainBridgeBase{IsSrc: isSrc}
}

// AdjustNonce adjust account nonce (eth like chain)
func (b *CrossChainBridgeBase) AdjustNonce(value uint64) (nonce uint64) {
	nonce = value
	if b.IsSrcEndpoint() {
		if b.SwapoutNonce > value {
			nonce = b.SwapoutNonce
		} else {
			b.SwapoutNonce = value
		}
	} else {
		if b.SwapinNonce > value {
			nonce = b.SwapinNonce
		} else {
			b.SwapinNonce = value
		}
	}
	return nonce
}

// IncreaseNonce decrease account nonce (eth like chain)
func (b *CrossChainBridgeBase) IncreaseNonce(value uint64) {
	if b.IsSrcEndpoint() {
		b.SwapoutNonce += value
	} else {
		b.SwapinNonce += value
	}
}

// IsSrcEndpoint returns if bridge is at the source endpoint
func (b *CrossChainBridgeBase) IsSrcEndpoint() bool {
	return b.IsSrc
}

// GetTokenAndGateway get token and gateway config
func (b *CrossChainBridgeBase) GetTokenAndGateway() (*TokenConfig, *GatewayConfig) {
	return b.TokenConfig, b.GatewayConfig
}

// SetTokenAndGateway set token and gateway config
func (b *CrossChainBridgeBase) SetTokenAndGateway(tokenCfg *TokenConfig, gatewayCfg *GatewayConfig) {
	b.TokenConfig = tokenCfg
	b.GatewayConfig = gatewayCfg
	err := tokenCfg.CheckConfig(b.IsSrc)
	if err != nil {
		log.Fatalf("set token and gateway error %v", err)
	}
}

// SetTokenAndGatewayWithoutCheck set token and gateway config without check
func (b *CrossChainBridgeBase) SetTokenAndGatewayWithoutCheck(tokenCfg *TokenConfig, gatewayCfg *GatewayConfig) {
	b.TokenConfig = tokenCfg
	b.GatewayConfig = gatewayCfg
}

// GetCrossChainBridge get bridge of specified endpoint
func GetCrossChainBridge(isSrc bool) CrossChainBridge {
	if isSrc {
		return SrcBridge
	}
	return DstBridge
}

// GetTokenConfig get token config of specified endpoint
func GetTokenConfig(isSrc bool) *TokenConfig {
	var token *TokenConfig
	if isSrc {
		token, _ = SrcBridge.GetTokenAndGateway()
	} else {
		token, _ = DstBridge.GetTokenAndGateway()
	}
	return token
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
func GetBigValueThreshold(isSrc bool) *big.Int {
	token := GetTokenConfig(isSrc)
	return token.bigValThreshhold
}

// CheckSwapValue check swap value is in right range
func CheckSwapValue(value *big.Int, isSrc bool) bool {
	token := GetTokenConfig(isSrc)
	if value.Cmp(token.minSwap) < 0 {
		return false
	}
	if value.Cmp(token.maxSwap) > 0 {
		return false
	}
	swappedValue := CalcSwappedValue(value, isSrc)
	return swappedValue.Sign() > 0
}

// CalcSwappedValue calc swapped value (get rid of fee)
func CalcSwappedValue(value *big.Int, isSrc bool) *big.Int {
	token := GetTokenConfig(isSrc)

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
