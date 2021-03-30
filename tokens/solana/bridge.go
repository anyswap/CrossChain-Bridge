package solana

import (
	"fmt"
	"strings"
	"time"

	"github.com/dfuse-io/solana-go"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var PairID = "sol"

// Bridge solana bridge
type Bridge struct {
	*tokens.CrossChainBridgeBase
}

func (b *Bridge) RegisterCDC(isSrc bool) {
	if isSrc {
		tokens.TokenCDC.RegisterConcrete(&Solana2ETHSwapinAgreement{}, Solana2ETHSwapinAgreementType, nil)
	} else {
		tokens.TokenCDC.RegisterConcrete(&ETH2SolanaSwapinAgreement{}, ETH2SolanaSwapinAgreementType, nil)
		tokens.TokenCDC.RegisterConcrete(&ETH2SolanaSwapoutAgreement{}, ETH2SolanaSwapoutAgreementType, nil)
	}
}

// NewCrossChainBridge new bridge
func NewCrossChainBridge(isSrc bool) *Bridge {
	tokens.IsSwapoutToStringAddress = true
	b := &Bridge{
		CrossChainBridgeBase: tokens.NewCrossChainBridgeBase(isSrc),
	}
	b.RegisterCDC(isSrc)
	return b
}

// SetChainAndGateway set chain and gateway config
func (b *Bridge) SetChainAndGateway(chainCfg *tokens.ChainConfig, gatewayCfg *tokens.GatewayConfig) {
	b.CrossChainBridgeBase.SetChainAndGateway(chainCfg, gatewayCfg)
	b.InitLatestBlockNumber()
	b.VerifyChainID()
}

// VerifyChainID verify chain id
func (b *Bridge) VerifyChainID() {
	networkID := strings.ToLower(b.ChainConfig.NetID)
	switch networkID {
	case "mainnet", "testnet", "devnet":
	default:
		log.Fatalf("unsupported solana network: %v", b.ChainConfig.NetID)
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

// VerifyTokenConfig verify token config
func (b *Bridge) VerifyTokenConfig(tokenCfg *tokens.TokenConfig) error {
	if tokenCfg.PrivateKeyType != tokens.ED25519KeyType {
		return fmt.Errorf("solana private key type must be ed25519")
	}
	if !b.IsValidAddress(tokenCfg.DcrmAddress) {
		return fmt.Errorf("invalid dcrm address (not p2pkh): %v", tokenCfg.DcrmAddress)
	}
	if !b.IsValidAddress(tokenCfg.DepositAddress) {
		return fmt.Errorf("invalid deposit address: %v", tokenCfg.DepositAddress)
	}
	if strings.EqualFold(tokenCfg.Symbol, "SOL") && *tokenCfg.Decimals != 9 {
		return fmt.Errorf("invalid decimals for SOL: want 9 but have %v", *tokenCfg.Decimals)
	}
	if _, err := solana.PublicKeyFromBase58(tokenCfg.ContractAddress); err != nil {
		return fmt.Errorf("invalid solana program id (contract address)")
	}
	return nil
}
