// Package terra implements the bridge interfaces for terra blockchain.
package terra

import (
	"fmt"

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
	InitSDK()
	tokens.IsSwapoutToStringAddress = true
	if !isSrc {
		log.Fatalf("terra::NewCrossChainBridge error %v", tokens.ErrBridgeDestinationNotSupported)
	}
	return &Bridge{
		NonceSetterBase: base.NewNonceSetterBase(isSrc),
	}
}

// InitAfterConfig init and verify after loading config
func (b *Bridge) InitAfterConfig() {
}

// SetChainAndGateway set chain and gateway config
func (b *Bridge) SetChainAndGateway(chainCfg *tokens.ChainConfig, gatewayCfg *tokens.GatewayConfig) {
	b.CrossChainBridgeBase.SetChainAndGateway(chainCfg, gatewayCfg)
}

// VerifyTokenConfig verify token config
func (b *Bridge) VerifyTokenConfig(token *tokens.TokenConfig) (err error) {
	if token.DcrmAccountNumber == 0 {
		token.DcrmAccountNumber, err = b.GetAccountNumber(token.DcrmAddress)
		if err != nil {
			return err
		}
	}
	if token.TaxCap <= 0 {
		return fmt.Errorf("invalid tax cap: %v", token.TaxCap)
	}
	if token.TaxRate <= 0 || token.TaxRate >= 0.01 {
		return fmt.Errorf("invalid tax tax rate: %v", token.TaxRate)
	}
	if token.GasRate <= 0 {
		return fmt.Errorf("invalid tax gas rate: %v", token.GasRate)
	}
	return nil
}
