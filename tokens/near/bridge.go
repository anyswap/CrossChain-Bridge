// Package terra implements the bridge interfaces for terra blockchain.
package near

import (
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/base"
)

var (
	// ensure Bridge impl tokens.CrossChainBridge
	_ tokens.CrossChainBridge = &Bridge{}
	// ensure Bridge impl tokens.NonceSetter
	_ tokens.NonceSetter = &Bridge{}

	// SupportedChains supported chains, key is chainID or netID
	SupportedChains = make(map[string]bool)
	// SupportedCoinDecimals supported coins and decimals, key is denom symbol
	SupportedCoinDecimals = make(map[string]uint8)
)

// Bridge eth bridge
type Bridge struct {
	*base.NonceSetterBase
}

// InitSDK init cosmos sdk
func InitSDK() {

}

// NewCrossChainBridge new bridge
func NewCrossChainBridge(isSrc bool) *Bridge {
	Init()
	if !isSrc {
		log.Fatalf("terra::NewCrossChainBridge error %v", tokens.ErrBridgeDestinationNotSupported)
	}
	return &Bridge{
		NonceSetterBase: base.NewNonceSetterBase(isSrc),
	}
}

// Init run before loading bridge and token config
func Init() {
	InitSDK()

	tokens.IsSwapoutToStringAddress = true

}

// InitAfterConfig init and verify after loading config
func (b *Bridge) InitAfterConfig() {
	if b.ChainConfig == nil {
		log.Fatal("chain config is nil")
	}
}

// SetChainAndGateway set chain and gateway config
func (b *Bridge) SetChainAndGateway(chainCfg *tokens.ChainConfig, gatewayCfg *tokens.GatewayConfig) {
	b.CrossChainBridgeBase.SetChainAndGateway(chainCfg, gatewayCfg)

	err := b.VerifyChainConfig()
	if err != nil {
		log.Fatal("verify chain config failed", "err", err)
	}
}

// VerifyChainConfig verify chain config
func (b *Bridge) VerifyChainConfig() (err error) {

	return nil
}

// VerifyTokenConfig verify token config
//nolint:gocyclo // verify token config together
func (b *Bridge) VerifyTokenConfig(c *tokens.TokenConfig) (err error) {

	return nil
}
