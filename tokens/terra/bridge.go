// Package terra implements the bridge interfaces for terra blockchain.
package terra

import (
	"fmt"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/base"
	sdk "github.com/cosmos/cosmos-sdk/types"
	core "github.com/terra-money/core/types"
)

var (
	// ensure Bridge impl tokens.CrossChainBridge
	_ tokens.CrossChainBridge = &Bridge{}
	// ensure Bridge impl tokens.NonceSetter
	_ tokens.NonceSetter = &Bridge{}

	// SupportedChains supported chains, key is chainID or netID
	SupportedChains = make(map[string]bool)
)

// Bridge eth bridge
type Bridge struct {
	*base.NonceSetterBase
}

// InitSDK init cosmos sdk
func InitSDK() {
	config := sdk.GetConfig()
	config.SetPurpose(44)
	config.SetCoinType(core.CoinType)
	config.SetBech32PrefixForAccount(core.Bech32PrefixAccAddr, core.Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(core.Bech32PrefixValAddr, core.Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(core.Bech32PrefixConsAddr, core.Bech32PrefixConsPub)
	config.SetAddressVerifier(core.AddressVerifier)
	config.Seal()
}

// NewCrossChainBridge new bridge
func NewCrossChainBridge(isSrc bool) *Bridge {
	Init()
	if !isSrc {
		log.Fatalf("terra::NewCrossChainBridge error %v", tokens.ErrBridgeDestinationNotSupported)
	}
	return &Bridge{
		NonceSetterBase: base.NewNonceSetterBase(isSrc),
	}
}

// Init run before loading bridge and token config
func Init() {
	InitSDK()

	SupportedChains["columbus-5"] = true
	SupportedChains["tequila-0004"] = true

	tokens.IsSwapoutToStringAddress = true
}

// InitAfterConfig init and verify after loading config
func (b *Bridge) InitAfterConfig() {
	if b.ChainConfig == nil {
		log.Fatal("chain config is nil")
	}
}

// SetChainAndGateway set chain and gateway config
func (b *Bridge) SetChainAndGateway(chainCfg *tokens.ChainConfig, gatewayCfg *tokens.GatewayConfig) {
	b.CrossChainBridgeBase.SetChainAndGateway(chainCfg, gatewayCfg)

	err := b.VerifyChainConfig()
	if err != nil {
		log.Fatal("verify chain config failed", "err", err)
	}
}

// VerifyChainConfig verify chain config
func (b *Bridge) VerifyChainConfig() (err error) {
	c := b.ChainConfig
	// possible nil in testing code
	if c == nil {
		return nil
	}

	netID := strings.ToLower(c.NetID)
	if !SupportedChains[netID] {
		return fmt.Errorf("unsupported terra network: %v", c.NetID)
	}

	if c.MetaCoin == nil {
		return fmt.Errorf("chain must config 'MetaCoin'")
	}
	if c.MetaCoin.Symbol == "" {
		return fmt.Errorf("chain meta coin symbol is empty")
	}

	return nil
}

// VerifyTokenConfig verify token config
func (b *Bridge) VerifyTokenConfig(token *tokens.TokenConfig) (err error) {
	if token.DcrmAccountNumber == 0 {
		token.DcrmAccountNumber, err = b.GetAccountNumber(token.DcrmAddress)
		if err != nil {
			return err
		}
	}
	if token.TaxCap < 0 {
		return fmt.Errorf("invalid tax cap: %v", token.TaxCap)
	}
	if token.TaxRate < 0 || token.TaxRate > 0.01 {
		return fmt.Errorf("invalid tax tax rate: %v", token.TaxRate)
	}
	return nil
}
