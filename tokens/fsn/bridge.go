package fsn

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/tokens/eth"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

// Bridge fsn bridge inherit from eth bridge
type Bridge struct {
	*eth.Bridge
}

// NewCrossChainBridge new fsn bridge
func NewCrossChainBridge(isSrc bool) *Bridge {
	return &Bridge{Bridge: eth.NewCrossChainBridge(isSrc)}
}

// SetTokenAndGateway set token and gateway config
func (b *Bridge) SetTokenAndGateway(tokenCfg *tokens.TokenConfig, gatewayCfg *tokens.GatewayConfig) {
	b.CrossChainBridgeBase.SetTokenAndGateway(tokenCfg, gatewayCfg)
	b.VerifyChainID()
	b.VerifyTokenCofig()
	b.InitLatestBlockNumber()
}

// VerifyChainID verify chain id
func (b *Bridge) VerifyChainID() {
	tokenCfg := b.TokenConfig
	gatewayCfg := b.GatewayConfig

	networkID := strings.ToLower(tokenCfg.NetID)

	switch networkID {
	case "mainnet", "testnet", "devnet":
	case "custom":
		return
	default:
		panic(fmt.Sprintf("unsupported fusion network: %v", tokenCfg.NetID))
	}

	var (
		chainID *big.Int
		err     error
	)

	for {
		chainID, err = b.ChainID()
		if err == nil {
			break
		}
		log.Errorf("can not get gateway chainID. %v", err)
		log.Println("retry query gateway", gatewayCfg.APIAddress)
		time.Sleep(3 * time.Second)
	}

	panicMismatchChainID := func() {
		panic(fmt.Sprintf("gateway chainID %v is not %v", chainID, tokenCfg.NetID))
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
		panic("unsupported fusion network")
	}

	b.Signer = types.MakeSigner("EIP155", chainID)

	log.Info("VerifyChainID succeed", "networkID", networkID, "chainID", chainID)
}
