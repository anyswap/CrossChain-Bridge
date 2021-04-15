package tron

import (
	"fmt"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

type Bridge struct {
	*tokens.CrossChainBridgeBase
}

const (
	PairID         = "TRX"
	TRC10TokenType = "TRC10"
	TRC20TokenType = "TRC20"
)

// NewCrossChainBridge new bridge
/*
	1. tron -> eth/fusion/bsc/heco
		TRX -> anyTRX (ERC20)
		TRC20 tokens -> anyTokens (ERC20)
	2. eth/fusion/bsc/heco -> tron
		ETH/FSN/BNB/HT -> anyETH/anyFSN/anyBNB/anyHT (TRC20)
		USDT/Anyswap/BUSD/HUSD (ERC20) -> anyTokens (TRC20)
	3. btc/ltc -> tron
		BTC/LTC -> anyBTC/anyLTC (TRC20)
*/
func NewCrossChainBridge(isSrc bool) *Bridge {
	tokens.IsSwapoutToStringAddress = false
	InitExtCodeParts()
	return &Bridge{
		CrossChainBridgeBase: tokens.NewCrossChainBridgeBase(isSrc),
	}
}

// VerifyChainID verify chain id
func (b *Bridge) VerifyChainID() {
	networkID := strings.ToLower(b.ChainConfig.NetID)
	switch networkID {
	case "mainnet", "shasta":
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
	if tokenCfg.ContractAddress != "" {
		if !b.IsValidAddress(tokenCfg.ContractAddress) {
			return fmt.Errorf("invalid contract address: %v", tokenCfg.ContractAddress)
		}
		switch {
		case !b.IsSrc:
			if err := b.VerifyMbtcContractAddress(tokenCfg.ContractAddress); err != nil {
				return fmt.Errorf("wrong contract address: %v, %v", tokenCfg.ContractAddress, err)
			}
		case tokenCfg.IsTrc20():
			if err := b.VerifyTrc20ContractAddress(tokenCfg.ContractAddress, tokenCfg.ContractCodeHash, tokenCfg.IsProxyErc20()); err != nil {
				return fmt.Errorf("wrong contract address: %v, %v", tokenCfg.ContractAddress, err)
			}
		default:
			return fmt.Errorf("unsupported type of contract address '%v' in source chain, please assign SrcToken.ID (eg. ERC20) in config file", tokenCfg.ContractAddress)
		}
		log.Info("verify contract address pass", "address", tokenCfg.ContractAddress)
	} else if tokenCfg.ID != "TRX" {
		return fmt.Errorf("token ID is not TRX and contract address is not given")
	}
	return nil
}
