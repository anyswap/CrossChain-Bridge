package tokens

import (
	"errors"
	"math"
	"math/big"
)

// transaction memo prefix
const (
	LockMemoPrefix   = "SWAPTO:"
	UnlockMemoPrefix = "SWAPTX:"
	RecallMemoPrefix = "RECALL:"
)

// common variables
var (
	SrcBridge CrossChainBridge
	DstBridge CrossChainBridge

	SrcLatestBlockHeight uint64
	DstLatestBlockHeight uint64

	// first 4 bytes of `Keccak256Hash([]byte("Swapin(bytes32,address,uint256)"))`
	SwapinFuncHash = [4]byte{0xec, 0x12, 0x6c, 0x77}
	LogSwapinTopic = "0x05d0634fe981be85c22e2942a880821b70095d84e152c3ea3c17a4e4250d9d61"

	// first 4 bytes of `Keccak256Hash([]byte("Swapout(uint256,string)"))`
	SwapoutFuncHash = [4]byte{0xad, 0x54, 0x05, 0x6d}
	LogSwapoutTopic = "0x9c92ad817e5474d30a4378deface765150479363a897b0590fbb12ae9d89396b"
)

// common errors
var (
	ErrSwapTypeNotSupported          = errors.New("swap type not supported in this endpoint")
	ErrBridgeSourceNotSupported      = errors.New("bridge source not supported")
	ErrBridgeDestinationNotSupported = errors.New("bridge destination not supported")
	ErrUnknownSwapType               = errors.New("unknown swap type")
	ErrMsgHashMismatch               = errors.New("message hash mismatch")
	ErrWrongRawTx                    = errors.New("wrong raw tx")
	ErrWrongExtraArgs                = errors.New("wrong extra args")
	ErrWrongP2shSwapin               = errors.New("wrong p2sh swapin (only btc supports)")
	ErrWrongSwapinTxType             = errors.New("wrong swapin tx type")

	ErrTodo = errors.New("developing: TODO")

	ErrTxNotFound          = errors.New("tx not found")
	ErrTxNotStable         = errors.New("tx not stable")
	ErrTxWithWrongReceiver = errors.New("tx with wrong receiver")
	ErrTxWithWrongContract = errors.New("tx with wrong contract")

	// errors should register
	ErrTxWithWrongMemo    = errors.New("tx with wrong memo")
	ErrTxWithWrongValue   = errors.New("tx with wrong value")
	ErrTxWithWrongReceipt = errors.New("tx with wrong receipt")
	ErrTxWithWrongSender  = errors.New("tx with wrong sender")
	ErrTxWithWrongInput   = errors.New("tx with wrong input data")
)

// ShouldRegisterSwapForError return true if this error should record in database
func ShouldRegisterSwapForError(err error) bool {
	switch err {
	case nil,
		ErrTxWithWrongMemo,
		ErrTxWithWrongValue,
		ErrTxWithWrongReceipt,
		ErrTxWithWrongSender,
		ErrTxWithWrongInput:
		return true
	}
	return false
}

// CrossChainBridge interface
type CrossChainBridge interface {
	IsSrcEndpoint() bool
	GetTokenAndGateway() (*TokenConfig, *GatewayConfig)
	SetTokenAndGateway(*TokenConfig, *GatewayConfig)

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
}

// NewCrossChainBridgeBase new base bridge
func NewCrossChainBridgeBase(isSrc bool) *CrossChainBridgeBase {
	return &CrossChainBridgeBase{IsSrc: isSrc}
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
		panic(err)
	}
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

// CheckSwapValue check swap value is in right range
func CheckSwapValue(value *big.Int, isSrc bool) bool {
	token := GetTokenConfig(isSrc)
	decimals := *token.Decimals
	minValue := ToBits(*token.MinimumSwap, decimals)
	toleranceBits := big.NewInt(100)
	if new(big.Int).Add(value, toleranceBits).Cmp(minValue) < 0 {
		return false
	}
	maxValue := ToBits(*token.MaximumSwap, decimals)
	return new(big.Int).Sub(value, toleranceBits).Cmp(maxValue) <= 0
}

// CalcSwappedValue calc swapped value (get rid of fee)
func CalcSwappedValue(value *big.Int, isSrc bool) *big.Int {
	token := GetTokenConfig(isSrc)

	swapFeeRate := new(big.Float).SetFloat64(*token.SwapFeeRate)
	swapValue := new(big.Float).SetInt(value)
	swapFee := new(big.Float).Mul(swapValue, swapFeeRate)

	swappedValue := new(big.Float).Sub(swapValue, swapFee)

	result := big.NewInt(0)
	swappedValue.Int(result)
	return result
}
