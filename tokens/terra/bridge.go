// Package terra implements the bridge interfaces for terra blockchain.
package terra

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
)

// Bridge eth bridge
type Bridge struct {
	*base.NonceSetterBase
}

// NewCrossChainBridge new bridge
func NewCrossChainBridge(isSrc bool) *Bridge {
	tokens.IsSwapoutToStringAddress = true
	if !isSrc {
		log.Fatalf("terra::NewCrossChainBridge error %v", tokens.ErrBridgeDestinationNotSupported)
	}
	return &Bridge{
		NonceSetterBase: base.NewNonceSetterBase(isSrc),
	}
}

// InitAfterConfig init and verify after loading config
func (b *Bridge) InitAfterConfig() {
}

// SetChainAndGateway set chain and gateway config
func (b *Bridge) SetChainAndGateway(chainCfg *tokens.ChainConfig, gatewayCfg *tokens.GatewayConfig) {
	b.CrossChainBridgeBase.SetChainAndGateway(chainCfg, gatewayCfg)
}

// VerifyTokenConfig verify token config
func (b *Bridge) VerifyTokenConfig(tokenCfg *tokens.TokenConfig) error {
	return tokens.ErrTodo
}
