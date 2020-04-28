package eth

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/log"
	. "github.com/fsn-dev/crossChain-Bridge/tokens"
)

type EthBridge struct {
	CrossChainBridgeBase
	IsSrc bool
}

func NewCrossChainBridge(isSrc bool) CrossChainBridge {
	if isSrc {
		panic(ErrTodo)
	}
	return &EthBridge{
		IsSrc: isSrc,
	}
}

func (b *EthBridge) SetTokenAndGateway(tokenCfg *TokenConfig, gatewayCfg *GatewayConfig) {
	b.CrossChainBridgeBase.SetTokenAndGateway(tokenCfg, gatewayCfg)

	networkID := strings.ToLower(*tokenCfg.NetID)

	switch networkID {
	case "mainnet", "rinkeby":
	case "custom":
		return
	default:
		panic(fmt.Sprintf("unsupported ethereum network: %v", *tokenCfg.NetID))
	}

	var (
		latest  uint64
		chainID *big.Int
		err     error
	)

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

	for {
		chainID, err = b.ChainID()
		if err == nil {
			break
		}
		log.Errorf("can not get gateway chainID. %v", err)
		log.Println("retry query gateway", gatewayCfg.ApiAddress)
		time.Sleep(3 * time.Second)
	}

	switch networkID {
	case "mainnet":
		if chainID.Uint64() != 1 {
			panic(fmt.Sprintf("gateway chainID %v is not %v", chainID, *tokenCfg.NetID))
		}
	case "rinkeby":
		if chainID.Uint64() != 4 {
			panic(fmt.Sprintf("gateway chainID %v is not %v", chainID, *tokenCfg.NetID))
		}
	default:
		panic("unsupported ethereum network")
	}
}
