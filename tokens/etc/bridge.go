package etc

import (
	"math/big"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/eth"
	"github.com/anyswap/CrossChain-Bridge/types"
)

const (
	netMainnet = "mainnet"
	netKotti   = "kotti"
	netMordor  = "mordor"
)

// Bridge etc bridge inherit from eth bridge
type Bridge struct {
	*eth.Bridge
}

// NewCrossChainBridge new etc bridge
func NewCrossChainBridge(isSrc bool) *Bridge {
	return &Bridge{Bridge: eth.NewCrossChainBridge(isSrc)}
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
	switch networkID {
	case netMainnet:
	default:
		log.Fatalf("unsupported etc network: %v", b.ChainConfig.NetID)
	}

	var (
		chainID *big.Int
		err     error
	)

	for {
		// call NetworkID instead of ChainID as ChainID may return 0x0 wrongly
		chainID, err = b.NetworkID() // network id
		if err == nil {
			break
		}
		log.Errorf("can not get gateway chainID. %v", err)
		log.Println("retry query gateway", b.GatewayConfig.APIAddress)
		time.Sleep(3 * time.Second)
	}

	panicMismatchChainID := func() {
		log.Fatalf("gateway chainID %v is not %v", chainID, b.ChainConfig.NetID)
	}

	switch networkID {
	case netMainnet:
		if chainID.Uint64() != 1 {
			panicMismatchChainID()
		}
		chainID = big.NewInt(61)
	case netKotti:
		if chainID.Uint64() != 6 {
			panicMismatchChainID()
		}
	case netMordor:
		if chainID.Uint64() != 7 {
			panicMismatchChainID()
		}
		chainID = big.NewInt(63)
	default:
		log.Fatalf("unsupported etc network %v", networkID)
	}

	b.Signer = types.MakeSigner("EIP155", chainID)

	log.Info("VerifyChainID succeed", "networkID", networkID, "chainID", chainID)
}
