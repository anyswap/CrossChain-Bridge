package eth

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

type EthBridge struct {
	*tokens.CrossChainBridgeBase
	Signer types.Signer
}

func NewCrossChainBridge(isSrc bool) *EthBridge {
	if isSrc {
		panic(tokens.ErrTodo)
	}
	return &EthBridge{CrossChainBridgeBase: tokens.NewCrossChainBridgeBase(isSrc)}
}

func (b *EthBridge) SetTokenAndGatewayWithoutCheck(tokenCfg *tokens.TokenConfig, gatewayCfg *tokens.GatewayConfig) {
	b.TokenConfig = tokenCfg
	b.GatewayConfig = gatewayCfg
}

func (b *EthBridge) SetTokenAndGateway(tokenCfg *tokens.TokenConfig, gatewayCfg *tokens.GatewayConfig) {
	b.CrossChainBridgeBase.SetTokenAndGateway(tokenCfg, gatewayCfg)

	networkID := strings.ToLower(tokenCfg.NetID)

	switch networkID {
	case "mainnet", "rinkeby":
	case "custom":
		return
	default:
		panic(fmt.Sprintf("unsupported ethereum network: %v", tokenCfg.NetID))
	}

	var (
		latest  uint64
		chainID *big.Int
		err     error
	)

	for {
		latest, err = b.GetLatestBlockNumber()
		if err == nil {
			tokens.SetLatestBlockHeight(latest, b.IsSrc)
			log.Info("get latst block number succeed.", "number", latest, "BlockChain", tokenCfg.BlockChain, "NetID", tokenCfg.NetID)
			break
		}
		log.Error("get latst block number failed.", "BlockChain", tokenCfg.BlockChain, "NetID", tokenCfg.NetID, "err", err)
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

	panicMismatchChainID := func() {
		panic(fmt.Sprintf("gateway chainID %v is not %v", chainID, tokenCfg.NetID))
	}

	switch networkID {
	case "mainnet":
		if chainID.Uint64() != 1 {
			panicMismatchChainID()
		}
	case "rinkeby":
		if chainID.Uint64() != 4 {
			panicMismatchChainID()
		}
	default:
		panic("unsupported ethereum network")
	}

	b.Signer = types.MakeSigner("EIP155", chainID)
}
