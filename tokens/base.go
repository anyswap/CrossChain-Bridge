package tokens

import (
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

// SetNonce set nonce directly
func (b *CrossChainBridgeBase) SetNonce(value uint64) {
	if b.IsSrcEndpoint() {
		b.SwapoutNonce = value
	} else {
		b.SwapinNonce = value
	}
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
func (b *CrossChainBridgeBase) SetTokenAndGateway(tokenCfg *TokenConfig, gatewayCfg *GatewayConfig, check bool) {
	b.TokenConfig = tokenCfg
	b.GatewayConfig = gatewayCfg
	if !check {
		return
	}
	err := tokenCfg.CheckConfig(b.IsSrc)
	if err != nil {
		log.Fatalf("set token and gateway error %v", err)
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

// SetLatestBlockHeight set latest block height
func SetLatestBlockHeight(latest uint64, isSrc bool) {
	if isSrc {
		SrcLatestBlockHeight = latest
	} else {
		DstLatestBlockHeight = latest
	}
}
