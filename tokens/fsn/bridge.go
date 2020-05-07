package fsn

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

type FsnBridge struct {
	*tokens.CrossChainBridgeBase
	Signer types.Signer
}

func NewCrossChainBridge(isSrc bool) *FsnBridge {
	if isSrc {
		panic(tokens.ErrTodo)
	}
	return &FsnBridge{CrossChainBridgeBase: tokens.NewCrossChainBridgeBase(isSrc)}
}

func (b *FsnBridge) SetTokenAndGateway(tokenCfg *tokens.TokenConfig, gatewayCfg *tokens.GatewayConfig) {
	b.CrossChainBridgeBase.SetTokenAndGateway(tokenCfg, gatewayCfg)

	networkID := strings.ToLower(*tokenCfg.NetID)

	switch networkID {
	case "mainnet", "testnet", "devnet":
	case "custom":
		return
	default:
		panic(fmt.Sprintf("unsupported fusion network: %v", *tokenCfg.NetID))
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

	panicMismatchChainID := func() {
		panic(fmt.Sprintf("gateway chainID %v is not %v", chainID, *tokenCfg.NetID))
	}

	switch networkID {
	case "mainnet":
		if chainID.Uint64() != 32659 {
			panicMismatchChainID()
		}
	case "testnet":
		if chainID.Uint64() != 46688 {
			panicMismatchChainID()
		}
	case "devnet":
		if chainID.Uint64() != 55555 {
			panicMismatchChainID()
		}
	default:
		panic("unsupported ethereum network")
	}

	b.Signer = types.MakeSigner("EIP155", chainID)
}
