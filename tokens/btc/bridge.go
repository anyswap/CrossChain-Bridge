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

// PairID unique btc pair ID
const PairID = "btc"

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

// SetChainAndGateway set chain and gateway config
func (b *Bridge) SetChainAndGateway(chainCfg *tokens.ChainConfig, gatewayCfg *tokens.GatewayConfig) {
	b.CrossChainBridgeBase.SetChainAndGateway(chainCfg, gatewayCfg)
	b.InitLatestBlockNumber()
}

// VerifyTokenConfig verify token config
func (b *Bridge) VerifyTokenConfig(tokenCfg *tokens.TokenConfig) {
	chainCfg := b.ChainConfig
	networkID := strings.ToLower(chainCfg.NetID)
	switch networkID {
	case netMainnet, netTestnet3:
	case netCustom:
		return
	default:
		log.Fatal("unsupported bitcoin network", "netID", chainCfg.NetID)
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
