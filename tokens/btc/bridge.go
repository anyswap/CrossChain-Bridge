package btc

import (
	"fmt"
	"strings"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/log"
	. "github.com/fsn-dev/crossChain-Bridge/tokens"
)

type BtcBridge struct {
	CrossChainBridgeBase
	IsSrc bool
}

func NewCrossChainBridge(isSrc bool) CrossChainBridge {
	if !isSrc {
		panic(ErrBridgeDestinationNotSupported)
	}
	return &BtcBridge{
		IsSrc: isSrc,
	}
}

func (b *BtcBridge) SetTokenAndGateway(tokenCfg *TokenConfig, gatewayCfg *GatewayConfig) {
	b.CrossChainBridgeBase.SetTokenAndGateway(tokenCfg, gatewayCfg)

	networkID := strings.ToLower(*tokenCfg.NetID)
	switch networkID {
	case "mainnet", "testnet3":
	case "custom":
		return
	default:
		panic(fmt.Sprintf("unsupported bitcoin network: %v", *tokenCfg.NetID))
	}

	var latest uint64
	var err error
	for {
		latest, err = b.GetLatestBlockNumber()
		if err == nil {
			log.Info("get latst block number succeed.", "number", latest, "BlockChain", *tokenCfg.BlockChain, "NetID", *tokenCfg.NetID)
			break
		}
		log.Error("get latst block number failed.", "BlockChain", *tokenCfg.BlockChain, "NetID", *tokenCfg.NetID, "err", err)
		log.Println("retry query gateway", gatewayCfg.ApiAddress)
		time.Sleep(3 * time.Second)
	}
}
