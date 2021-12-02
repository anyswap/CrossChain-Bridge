// Package colx implements the bridge interfaces for colx blockchain.
package colx

import (
	"fmt"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc"
)

const (
	netMainnet  = "mainnet"
	netTestnet4 = "testnet4"
	netCustom   = "custom"
)

// PairID unique colx pair ID
var PairID = "colx"

// Bridge colx bridge
type Bridge struct {
	*tokens.CrossChainBridgeBase
}

var instance *Bridge

// NewCrossChainBridge new colx bridge
func NewCrossChainBridge(isSrc bool) *Bridge {
	if !isSrc {
		log.Fatalf("colx::NewCrossChainBridge error %v", tokens.ErrBridgeDestinationNotSupported)
	}
	btc.PairID = PairID
	instance = &Bridge{tokens.NewCrossChainBridgeBase(isSrc)}
	btc.BridgeInstance = instance
	return instance
}

// SetChainAndGateway set chain and gateway config
func (b *Bridge) SetChainAndGateway(chainCfg *tokens.ChainConfig, gatewayCfg *tokens.GatewayConfig) {
	b.CrossChainBridgeBase.SetChainAndGateway(chainCfg, gatewayCfg)
	b.VerifyChainConfig()
	b.InitLatestBlockNumber()
	go b.StartMonitLockedUtxo()
}

// VerifyChainConfig verify chain config
func (b *Bridge) VerifyChainConfig() {
	chainCfg := b.ChainConfig
	networkID := strings.ToLower(chainCfg.NetID)
	switch networkID {
	case netMainnet, netTestnet4:
	case netCustom:
		return
	default:
		log.Fatal("unsupported colossus network", "netID", chainCfg.NetID)
	}
}

// VerifyTokenConfig verify token config
func (b *Bridge) VerifyTokenConfig(tokenCfg *tokens.TokenConfig) error {
	if !b.IsP2pkhAddress(tokenCfg.DcrmAddress) {
		return fmt.Errorf("invalid dcrm address (not p2pkh): %v", tokenCfg.DcrmAddress)
	}
	if !b.IsValidAddress(tokenCfg.DepositAddress) {
		return fmt.Errorf("invalid deposit address: %v", tokenCfg.DepositAddress)
	}
	if strings.EqualFold(tokenCfg.Symbol, "COLX") && *tokenCfg.Decimals != 8 {
		return fmt.Errorf("invalid decimals for COLX: want 8 but have %v", *tokenCfg.Decimals)
	}
	return nil
}

// InitLatestBlockNumber init latest block number
func (b *Bridge) InitLatestBlockNumber() {
	chainCfg := b.ChainConfig
	gatewayCfg := b.GatewayConfig
	var latest uint64
	var err error
	for {
		latest, err = b.GetLatestBlockNumber()
		if err == nil {
			tokens.SetLatestBlockHeight(latest, b.IsSrc)
			log.Info("get latst block number succeed.", "number", latest, "BlockChain", chainCfg.BlockChain, "NetID", chainCfg.NetID)
			break
		}
		log.Error("get latst block number failed.", "BlockChain", chainCfg.BlockChain, "NetID", chainCfg.NetID, "err", err)
		log.Println("retry query gateway", gatewayCfg.APIAddress)
		time.Sleep(3 * time.Second)
	}
}
