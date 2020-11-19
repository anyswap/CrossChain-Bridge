package tokens

import (
	"math"
	"math/big"
)

// transaction memo prefix
const (
	LockMemoPrefix      = "SWAPTO:"
	UnlockMemoPrefix    = "SWAPTX:"
	AggregateIdentifier = "aggregate"
	AggregateMemo       = "aggregate"
)

// common variables
var (
	SrcBridge CrossChainBridge
	DstBridge CrossChainBridge

	SrcLatestBlockHeight uint64
	DstLatestBlockHeight uint64

	IsDcrmDisabled bool
)

// CrossChainBridgeBase base bridge
type CrossChainBridgeBase struct {
	ChainConfig   *ChainConfig
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

// GetDcrmPublicKey get dcrm address's public key
func (b *CrossChainBridgeBase) GetDcrmPublicKey(pairID string) string {
	tokenCfg := GetTokenConfig(pairID, b.IsSrcEndpoint())
	if tokenCfg != nil {
		return tokenCfg.DcrmPubkey
	}
	return ""
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

// SetLatestBlockHeight set latest block height
func SetLatestBlockHeight(latest uint64, isSrc bool) {
	if isSrc {
		SrcLatestBlockHeight = latest
	} else {
		DstLatestBlockHeight = latest
	}
}
