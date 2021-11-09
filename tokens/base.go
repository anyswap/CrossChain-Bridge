package tokens

import (
	"math"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/types"
)

// transaction memo prefix
const (
	LockMemoPrefix   = "SWAPTO:"
	UnlockMemoPrefix = "SWAPTX:"
	AggregateMemo    = "aggregate"

	MaxPlusGasPricePercentage = uint64(100)
)

// common variables
var (
	AggregateIdentifier = "aggregate"

	SrcBridge CrossChainBridge
	DstBridge CrossChainBridge

	SrcNonceSetter NonceSetter
	DstNonceSetter NonceSetter

	SrcForkChecker ForkChecker
	DstForkChecker ForkChecker

	SrcLatestBlockHeight uint64
	DstLatestBlockHeight uint64

	SrcStableConfirmations uint64
	DstStableConfirmations uint64

	IsDcrmDisabled bool

	TokenPriceCfg *TokenPriceConfig
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

// IsSwapTxOnChainAndFailed to make failed of swaptx
func (s *TxStatus) IsSwapTxOnChainAndFailed(token *TokenConfig) bool {
	if s == nil || s.BlockHeight == 0 {
		return false // not on chain
	}
	if s.Receipt != nil { // for eth-like blockchain
		receipt, ok := s.Receipt.(*types.RPCTxReceipt)
		if !ok || !receipt.IsStatusOk() {
			return true
		}
		if token != nil && token.ContractAddress != "" && len(receipt.Logs) == 0 {
			return true
		}
	}
	return false
}

// GetCrossChainBridge get bridge of specified endpoint
func GetCrossChainBridge(isSrc bool) CrossChainBridge {
	if isSrc {
		return SrcBridge
	}
	return DstBridge
}

// GetNonceSetter get nonce setter of specified endpoint
func GetNonceSetter(isSrc bool) NonceSetter {
	if isSrc {
		return SrcNonceSetter
	}
	return DstNonceSetter
}

// GetForkChecker get fork checker of specified endpoint
func GetForkChecker(isSrc bool) ForkChecker {
	if isSrc {
		return SrcForkChecker
	}
	return DstForkChecker
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
	swappedValue := CalcSwappedValue(pairID, value, isSrc)
	return swappedValue.Sign() > 0
}

// CalcSwappedValue calc swapped value (get rid of fee)
func CalcSwappedValue(pairID string, value *big.Int, isSrc bool) *big.Int {
	if value == nil || value.Sign() <= 0 {
		return big.NewInt(0)
	}

	token := GetTokenConfig(pairID, isSrc)

	if value.Cmp(token.minSwap) < 0 {
		return big.NewInt(0)
	}
	if value.Cmp(token.maxSwap) > 0 {
		return big.NewInt(0)
	}

	if *token.SwapFeeRate == 0.0 {
		return value
	}

	feeRateMul1e18 := new(big.Int).SetUint64(uint64(*token.SwapFeeRate * 1e18))
	swapFee := new(big.Int).Mul(value, feeRateMul1e18)
	swapFee.Div(swapFee, big.NewInt(1e18))

	if swapFee.Cmp(token.minSwapFee) < 0 {
		swapFee = token.minSwapFee
	} else if swapFee.Cmp(token.maxSwapFee) > 0 {
		swapFee = token.maxSwapFee
	}

	var adjustBaseFee *big.Int
	if GetNonceSetter(!isSrc) != nil { // eth-like
		chainCfg := GetCrossChainBridge(!isSrc).GetChainConfig()
		if chainCfg.BaseFeePercent != 0 && token.minSwapFee.Sign() > 0 {
			adjustBaseFee = new(big.Int).Set(token.minSwapFee)
			adjustBaseFee.Mul(adjustBaseFee, big.NewInt(chainCfg.BaseFeePercent))
			adjustBaseFee.Div(adjustBaseFee, big.NewInt(100))
			swapFee = new(big.Int).Add(swapFee, adjustBaseFee)
			if swapFee.Sign() < 0 {
				swapFee = big.NewInt(0)
			}
		}
	}

	if value.Cmp(swapFee) <= 0 {
		log.Warn("check swap value failed", "pairID", pairID, "value", value, "isSrc", isSrc,
			"minSwapFee", token.minSwapFee, "adjustBaseFee", adjustBaseFee, "swapFee", swapFee)
		return big.NewInt(0)
	}

	swappedValue := new(big.Int).Sub(value, swapFee)
	// recheck swap value range
	if swappedValue.Cmp(value) > 0 || swappedValue.Cmp(token.maxSwap) > 0 {
		return big.NewInt(0)
	}
	return swappedValue
}

// SetLatestBlockHeight set latest block height
func SetLatestBlockHeight(latest uint64, isSrc bool) {
	if isSrc {
		SrcLatestBlockHeight = latest
	} else {
		DstLatestBlockHeight = latest
	}
}

// CmpAndSetLatestBlockHeight cmp and set latest block height
func CmpAndSetLatestBlockHeight(latest uint64, isSrc bool) {
	if isSrc {
		if latest > SrcLatestBlockHeight {
			SrcLatestBlockHeight = latest
		}
	} else {
		if latest > DstLatestBlockHeight {
			DstLatestBlockHeight = latest
		}
	}
}

// GetStableConfirmations get stable confirmations
func GetStableConfirmations(isSrc bool) uint64 {
	if isSrc {
		return SrcStableConfirmations
	}
	return DstStableConfirmations
}
