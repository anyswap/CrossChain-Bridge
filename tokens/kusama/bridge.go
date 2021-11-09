// Package kusama implements the bridge interfaces for kusama ecosystem blockchain.
package kusama

import (
	"math/big"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/eth"
)

// Bridge kusama bridge inherit from eth bridge
type Bridge struct {
	*eth.Bridge
}

// NewCrossChainBridge new kusama bridge
func NewCrossChainBridge(isSrc bool) *Bridge {
	bridge := &Bridge{Bridge: eth.NewCrossChainBridge(isSrc)}
	bridge.Inherit = bridge
	return bridge
}

// SetChainAndGateway set token and gateway config
func (b *Bridge) SetChainAndGateway(chainCfg *tokens.ChainConfig, gatewayCfg *tokens.GatewayConfig) {
	b.CrossChainBridgeBase.SetChainAndGateway(chainCfg, gatewayCfg)
	b.VerifyChainID()
	b.Init()
}

// VerifyChainID verify chain id
func (b *Bridge) VerifyChainID() {
	networkID := strings.ToLower(b.ChainConfig.NetID)
	targetChainID := eth.GetChainIDOfNetwork(eth.KusamaNetworkAndChainIDMap, networkID)
	isCustom := eth.IsCustomNetwork(networkID)
	if !isCustom && targetChainID == nil {
		log.Fatalf("unsupported kusama network: %v", b.ChainConfig.NetID)
	}

	var (
		chainID *big.Int
		err     error
	)

	for {
		chainID, err = b.GetSignerChainID()
		if err == nil {
			break
		}
		log.Errorf("can not get gateway chainID. %v", err)
		log.Println("retry query gateway", b.GatewayConfig.APIAddress)
		time.Sleep(3 * time.Second)
	}

	if !isCustom && chainID.Cmp(targetChainID) != 0 {
		log.Fatalf("gateway chainID '%v' is not '%v'", chainID, b.ChainConfig.NetID)
	}

	b.MakeSigner(chainID)

	log.Info("VerifyChainID succeed", "networkID", networkID, "chainID", chainID)
}
