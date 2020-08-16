package btc

import (
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

const (
	netMainnet  = "mainnet"
	netTestnet3 = "testnet3"
	netCustom   = "custom"
)

// BridgeInstance btc bridge instance
var BridgeInstance *Bridge

// Bridge btc bridge
type Bridge struct {
	*tokens.CrossChainBridgeBase
}

// NewCrossChainBridge new btc bridge
func NewCrossChainBridge(isSrc bool) *Bridge {
	if !isSrc {
		log.Fatalf("btc::NewCrossChainBridge error %v", tokens.ErrBridgeDestinationNotSupported)
	}
	BridgeInstance = &Bridge{tokens.NewCrossChainBridgeBase(isSrc)}
	return BridgeInstance
}

// SetTokenAndGateway set token and gateway config
func (b *Bridge) SetTokenAndGateway(tokenCfg *tokens.TokenConfig, gatewayCfg *tokens.GatewayConfig, check bool) {
	b.CrossChainBridgeBase.SetTokenAndGateway(tokenCfg, gatewayCfg, check)
	b.VerifyConfig()
	b.InitLatestBlockNumber()
}

// VerifyConfig verify config
func (b *Bridge) VerifyConfig() {
	tokenCfg := b.TokenConfig
	networkID := strings.ToLower(tokenCfg.NetID)
	switch networkID {
	case netMainnet, netTestnet3:
	case netCustom:
		return
	default:
		log.Fatal("unsupported bitcoin network", "netID", tokenCfg.NetID)
	}

	if !b.IsP2pkhAddress(tokenCfg.DcrmAddress) {
		log.Fatal("invalid dcrm address (not p2pkh)", "address", tokenCfg.DcrmAddress)
	}
	if !b.IsValidAddress(tokenCfg.DepositAddress) {
		log.Fatal("invalid deposit address", "address", tokenCfg.DepositAddress)
	}

	if strings.EqualFold(tokenCfg.Symbol, "BTC") && *tokenCfg.Decimals != 8 {
		log.Fatal("invalid decimals for BTC", "configed", *tokenCfg.Decimals, "want", 8)
	}
}

// InitLatestBlockNumber init latest block number
func (b *Bridge) InitLatestBlockNumber() {
	tokenCfg := b.TokenConfig
	gatewayCfg := b.GatewayConfig
	var latest uint64
	var err error
	for {
		latest, err = b.GetLatestBlockNumber()
		if err == nil {
			tokens.SetLatestBlockHeight(latest, b.IsSrc)
			log.Info("get latst block number succeed.", "number", latest, "BlockChain", tokenCfg.BlockChain, "NetID", tokenCfg.NetID)
			break
		}
		log.Error("get latst block number failed.", "BlockChain", tokenCfg.BlockChain, "NetID", tokenCfg.NetID, "err", err)
		log.Println("retry query gateway", gatewayCfg.APIAddress)
		time.Sleep(3 * time.Second)
	}
}
