package eth

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/types"
)

const (
	netMainnet = "mainnet"
	netRinkeby = "rinkeby"
	netCustom  = "custom"
)

// Bridge eth bridge
type Bridge struct {
	*tokens.CrossChainBridgeBase
	*NonceSetterBase
	Signer types.Signer
}

// NewCrossChainBridge new bridge
func NewCrossChainBridge(isSrc bool) *Bridge {
	return &Bridge{
		CrossChainBridgeBase: tokens.NewCrossChainBridgeBase(isSrc),
		NonceSetterBase:      NewNonceSetterBase(),
	}
}

// SetChainAndGateway set chain and gateway config
func (b *Bridge) SetChainAndGateway(chainCfg *tokens.ChainConfig, gatewayCfg *tokens.GatewayConfig) {
	b.CrossChainBridgeBase.SetChainAndGateway(chainCfg, gatewayCfg)
	b.VerifyChainID()
	b.Init()
}

// Init init after verify
func (b *Bridge) Init() {
	InitExtCodeParts()
	b.InitLatestBlockNumber()
}

// VerifyChainID verify chain id
func (b *Bridge) VerifyChainID() {
	networkID := strings.ToLower(b.ChainConfig.NetID)
	switch networkID {
	case netMainnet, netRinkeby:
	case netCustom:
	default:
		log.Fatalf("unsupported ethereum network: %v", b.ChainConfig.NetID)
	}

	var (
		chainID *big.Int
		err     error
	)

	for {
		// call NetworkID instead of ChainID as ChainID may return 0x0 wrongly
		chainID, err = b.NetworkID()
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
	case netRinkeby:
		if chainID.Uint64() != 4 {
			panicMismatchChainID()
		}
	case netCustom:
	default:
		log.Fatalf("unsupported ethereum network %v", networkID)
	}

	b.Signer = types.MakeSigner("EIP155", chainID)

	log.Info("VerifyChainID succeed", "networkID", networkID, "chainID", chainID)
}

// VerifyTokenConfig verify token config
func (b *Bridge) VerifyTokenConfig(tokenCfg *tokens.TokenConfig) (err error) {
	if !b.IsValidAddress(tokenCfg.DcrmAddress) {
		return fmt.Errorf("invalid dcrm address: %v", tokenCfg.DcrmAddress)
	}
	if b.IsSrc && !b.IsValidAddress(tokenCfg.DepositAddress) {
		return fmt.Errorf("invalid deposit address: %v", tokenCfg.DepositAddress)
	}

	err = b.verifyDecimals(tokenCfg)
	if err != nil {
		return err
	}

	err = b.verifyContractAddress(tokenCfg)
	if err != nil {
		return err
	}

	return nil
}

func (b *Bridge) verifyDecimals(tokenCfg *tokens.TokenConfig) error {
	configedDecimals := *tokenCfg.Decimals
	switch strings.ToUpper(tokenCfg.Symbol) {
	case "ETH", "FSN":
		if configedDecimals != 18 {
			return fmt.Errorf("invalid decimals: want 18 but have %v", configedDecimals)
		}
		log.Info(tokenCfg.Symbol+" verify decimals success", "decimals", configedDecimals)
	}

	if tokenCfg.IsErc20() {
		for {
			decimals, err := b.GetErc20Decimals(tokenCfg.ContractAddress)
			if err == nil {
				if decimals != configedDecimals {
					return fmt.Errorf("invalid decimals for %v, want %v but configed %v", tokenCfg.Symbol, decimals, configedDecimals)
				}
				log.Info(tokenCfg.Symbol+" verify decimals success", "decimals", configedDecimals)
				break
			}
			log.Error("get erc20 decimals failed", "err", err)
			time.Sleep(3 * time.Second)
		}
	}
	return nil
}

func (b *Bridge) verifyContractAddress(tokenCfg *tokens.TokenConfig) error {
	if tokenCfg.ContractAddress != "" {
		if !b.IsValidAddress(tokenCfg.ContractAddress) {
			return fmt.Errorf("invalid contract address: %v", tokenCfg.ContractAddress)
		}
		switch {
		case !b.IsSrc:
			if err := b.VerifyMbtcContractAddress(tokenCfg.ContractAddress); err != nil {
				return fmt.Errorf("wrong contract address: %v, %v", tokenCfg.ContractAddress, err)
			}
		case tokenCfg.IsErc20():
			if err := b.VerifyErc20ContractAddress(tokenCfg.ContractAddress, tokenCfg.ContractCodeHash, tokenCfg.IsProxyErc20()); err != nil {
				return fmt.Errorf("wrong contract address: %v, %v", tokenCfg.ContractAddress, err)
			}
		default:
			return fmt.Errorf("unsupported type of contract address '%v' in source chain, please assign SrcToken.ID (eg. ERC20) in config file", tokenCfg.ContractAddress)
		}
		log.Info("verify contract address pass", "address", tokenCfg.ContractAddress)
	}
	return nil
}

// InitLatestBlockNumber init latest block number
func (b *Bridge) InitLatestBlockNumber() {
	var (
		latest uint64
		err    error
	)

	for {
		latest, err = b.GetLatestBlockNumber()
		if err == nil {
			tokens.SetLatestBlockHeight(latest, b.IsSrc)
			log.Info("get latst block number succeed.", "number", latest, "BlockChain", b.ChainConfig.BlockChain, "NetID", b.ChainConfig.NetID)
			break
		}
		log.Error("get latst block number failed.", "BlockChain", b.ChainConfig.BlockChain, "NetID", b.ChainConfig.NetID, "err", err)
		log.Println("retry query gateway", b.GatewayConfig.APIAddress)
		time.Sleep(3 * time.Second)
	}
}
