// Package eth implements the bridge interfaces for eth-like blockchain.
package eth

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/types"
)

var (
	// ensure Bridge impl tokens.CrossChainBridge
	_ tokens.CrossChainBridge = &Bridge{}
	// ensure Bridge impl tokens.NonceSetter
	_ tokens.NonceSetter = &Bridge{}
	// ensure Bridge impl InheritInterface
	_ InheritInterface = &Bridge{}
)

// InheritInterface inherit interface
type InheritInterface interface {
	GetLatestBlockNumberOf(apiAddress string) (uint64, error)
}

// Bridge eth bridge
type Bridge struct {
	Inherit InheritInterface
	*tokens.CrossChainBridgeBase
	*NonceSetterBase
	Signer        types.Signer
	SignerChainID *big.Int
}

// NewCrossChainBridge new bridge
func NewCrossChainBridge(isSrc bool) *Bridge {
	bridge := &Bridge{
		CrossChainBridgeBase: tokens.NewCrossChainBridgeBase(isSrc),
		NonceSetterBase:      NewNonceSetterBase(),
	}
	bridge.Inherit = bridge
	return bridge
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

	if b.ChainConfig.BaseGasPrice != "" {
		gasPrice, err := common.GetBigIntFromStr(b.ChainConfig.BaseGasPrice)
		if err != nil {
			log.Crit("wrong chain config 'BaseGasPrice'", "BaseGasPrice", b.ChainConfig.BaseGasPrice, "err", err)
		}
		baseGasPrice = gasPrice
	}
	log.Info("init base gas price", "baseGasPrice", baseGasPrice, "isSrc", b.IsSrc, "chainID", b.SignerChainID)
}

// VerifyChainID verify chain id
func (b *Bridge) VerifyChainID() {
	networkID := strings.ToLower(b.ChainConfig.NetID)
	targetChainID := GetChainIDOfNetwork(EthNetworkAndChainIDMap, networkID)
	isCustom := IsCustomNetwork(networkID)
	if !isCustom && targetChainID == nil {
		log.Fatalf("unsupported ethereum network: %v", b.ChainConfig.NetID)
	}

	var (
		chainID *big.Int
		err     error
	)

	for i := 0; i < 5; i++ {
		chainID, err = b.GetSignerChainID()
		if err == nil {
			break
		}
		log.Errorf("can not get gateway chainID. %v", err)
		log.Println("retry query gateway", b.GatewayConfig.APIAddress)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatal("get chain ID failed", "err", err)
	}

	if !isCustom && chainID.Cmp(targetChainID) != 0 {
		log.Fatalf("gateway chainID '%v' is not '%v'", chainID, b.ChainConfig.NetID)
	}

	b.MakeSigner(chainID)

	log.Info("VerifyChainID succeed", "networkID", networkID, "chainID", chainID)
}

// MakeSigner make signer
func (b *Bridge) MakeSigner(chainID *big.Int) {
	b.SignerChainID = chainID
	b.ChainConfig.SetChainID(chainID)
	if b.ChainConfig.EnableDynamicFeeTx {
		b.Signer = types.MakeSigner("London", chainID)
	}
	b.Signer = types.MakeSigner("EIP155", chainID)
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
	checkToken := tokenCfg.ContractAddress
	if tokenCfg.IsDelegateContract {
		checkToken = tokenCfg.DelegateToken
	}
	switch strings.ToUpper(tokenCfg.Symbol) {
	case "ETH", "FSN":
		if configedDecimals != 18 {
			return fmt.Errorf("invalid decimals: want 18 but have %v", configedDecimals)
		}
		log.Info(tokenCfg.Symbol+" verify decimals success", "decimals", configedDecimals)
	}

	if checkToken != "" {
		decimals, err := b.GetErc20Decimals(checkToken)
		if err != nil {
			log.Error("get erc20 decimals failed", "address", checkToken, "err", err)
			return err
		}
		if decimals != configedDecimals {
			return fmt.Errorf("invalid decimals for %v, want %v but configed %v", tokenCfg.Symbol, decimals, configedDecimals)
		}
		log.Info(tokenCfg.Symbol+" verify decimals success", "address", checkToken, "decimals", configedDecimals)

		if !tokenCfg.IsMappingTokenProxy {
			if err := b.VerifyErc20ContractAddress(checkToken, tokenCfg.ContractCodeHash, tokenCfg.IsProxyErc20()); err != nil {
				return fmt.Errorf("wrong token address: %v, %w", checkToken, err)
			}
			log.Info("verify token address pass", "address", checkToken)
		}
	}
	return nil
}

func (b *Bridge) verifyContractAddress(tokenCfg *tokens.TokenConfig) error {
	contractAddr := tokenCfg.ContractAddress
	if contractAddr == "" {
		return nil
	}
	if !b.IsValidAddress(contractAddr) {
		return fmt.Errorf("invalid contract address: %v", contractAddr)
	}
	if b.IsSrc && !(tokenCfg.IsErc20() || tokenCfg.IsProxyErc20() || tokenCfg.IsDelegateContract || tokenCfg.IsMappingTokenProxy) {
		return fmt.Errorf("source token %v is not ERC20, ProxyERC20 or delegated or MappingTokenProxy", contractAddr)
	}
	if tokenCfg.IsDelegateContract && !tokenCfg.IsAnyswapAdapter && !b.IsSrc {
		// keccak256 'proxyToken()' is '0x4faaefae'
		res, err := b.CallContract(contractAddr, common.FromHex("0x4faaefae"), "latest")
		if err != nil {
			return fmt.Errorf("get proxyToken of %v failed, %w", contractAddr, err)
		}
		proxyToken := common.HexToAddress(res)
		if common.HexToAddress(tokenCfg.DelegateToken) != proxyToken {
			return fmt.Errorf("mismatch 'DelegateToken', has %v, want %v", tokenCfg.DelegateToken, proxyToken.String())
		}
	}
	if !b.IsSrc && !tokenCfg.IsMappingTokenProxy {
		err := b.VerifyAnyswapContractAddress(contractAddr)
		if err != nil {
			return fmt.Errorf("wrong anyswap contract address: %v, %w", contractAddr, err)
		}
	}
	log.Info("verify contract address pass", "address", contractAddr)
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
