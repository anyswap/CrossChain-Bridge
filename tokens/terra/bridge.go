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
	// SupportedCoinDecimals supported coins and decimals, key is denom symbol
	SupportedCoinDecimals = make(map[string]uint8)
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

	tokens.IsSwapoutToStringAddress = true

	SupportedChains[core.ColumbusChainID] = true
	SupportedChains[core.BombayChainID] = true
	SupportedChains["custom"] = true

	SupportedCoinDecimals[core.MicroLunaDenom] = 6
	SupportedCoinDecimals[core.MicroUSDDenom] = 6
	SupportedCoinDecimals[core.MicroKRWDenom] = 6
	SupportedCoinDecimals[core.MicroSDRDenom] = 6
	SupportedCoinDecimals[core.MicroCNYDenom] = 6
	SupportedCoinDecimals[core.MicroJPYDenom] = 6
	SupportedCoinDecimals[core.MicroEURDenom] = 6
	SupportedCoinDecimals[core.MicroGBPDenom] = 6
	SupportedCoinDecimals[core.MicroMNTDenom] = 6
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

	if !SupportedChains[c.NetID] {
		return fmt.Errorf("unsupported terra network: %v", c.NetID)
	}

	coin := c.MetaCoin
	if coin == nil {
		return fmt.Errorf("chain must config 'MetaCoin'")
	}
	if coin.Unit == "" {
		return fmt.Errorf("chain meta coin symbol is empty")
	}
	decimals, exist := SupportedCoinDecimals[coin.Unit]
	if !exist {
		return fmt.Errorf("meta coin '%v' is not supported", coin.Unit)
	}
	if coin.Decimals != decimals {
		return fmt.Errorf("chain meta coin %v decimals mismatch, have %v want %v", coin.Unit, coin.Decimals, decimals)
	}

	return nil
}

// VerifyTokenConfig verify token config
//nolint:gocyclo // verify token config together
func (b *Bridge) VerifyTokenConfig(c *tokens.TokenConfig) (err error) {
	// try init dcrm account number
	if c.DcrmAccountNumber == 0 {
		c.DcrmAccountNumber, err = b.GetAccountNumber(c.DcrmAddress)
		log.Error("get dcrm account number failed", "err", err)
	}

	// verify addresses
	_, err = sdk.AccAddressFromBech32(c.DcrmAddress)
	if err != nil {
		return fmt.Errorf("wrong dcrm address: %w", err)
	}
	if c.DepositAddress != c.DcrmAddress {
		_, err = sdk.AccAddressFromBech32(c.DepositAddress)
		if err != nil {
			return fmt.Errorf("wrong deposit address: %w", err)
		}
	}
	if c.ContractAddress != "" {
		_, err = sdk.AccAddressFromBech32(c.ContractAddress)
		if err != nil {
			return fmt.Errorf("wrong contract address: %w", err)
		}
		if c.Unit != "" {
			return fmt.Errorf("only meta coin (empty contract address) have unit")
		}
	} else {
		if c.Unit == "" {
			return fmt.Errorf("meta coin (empty contract address) must config 'Unit'")
		}
		decimals, exist := SupportedCoinDecimals[c.Unit]
		if !exist {
			return fmt.Errorf("meta coin '%v' is not supported", c.Unit)
		}
		if *c.Decimals != decimals {
			return fmt.Errorf("meta coin %v decimals mismatch, have %v want %v", c.Unit, c.Decimals, decimals)
		}
		err = sdk.ValidateDenom(c.Unit)
		if err != nil {
			return fmt.Errorf("wrong denom: %v %w", c.Unit, err)
		}
	}

	// verify public key
	pubAddr, err := PublicKeyToAddress(c.DcrmPubkey)
	if err != nil {
		return err
	}
	if !strings.EqualFold(pubAddr, c.DcrmAddress) {
		return fmt.Errorf("dcrm address %v and public key address %v is not match", c.DcrmAddress, pubAddr)
	}

	// check  tax config
	if c.TaxCap < 0 {
		return fmt.Errorf("invalid tax cap: %v", c.TaxCap)
	}
	if c.TaxRate < 0 || c.TaxRate > 0.01 {
		return fmt.Errorf("invalid tax tax rate: %v", c.TaxRate)
	}

	// verify fees config
	_, err = sdk.ParseCoinsNormalized(c.DefaultFees)
	if err != nil {
		return fmt.Errorf("parse coin error: %w", err)
	}

	return nil
}
