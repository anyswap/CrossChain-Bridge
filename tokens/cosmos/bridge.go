package cosmos

import (
	"fmt"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/eth"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cyptes "github.com/tendermint/tendermint/rpc/core/types"
)

var ChainIDs = make(map[string]bool)

type CosmosBridgeInterface interface {
	BeforeConfig()
	AfterConfig()
}

func (b *Bridge) BeforeConfig() {
	cyptes.RegisterAmino(CDC)
	sdk.RegisterCodec(CDC)
	b.InitChains()
}

func (b *Bridge) AfterConfig() {
	b.InitCoins()
}

// PairID unique cosmos pair ID
var PairID = "cosmos"

// SupportedCoins save cosmos coins
var SupportedCoins = make(map[string]CosmosCoin)

var TheCoin CosmosCoin

type CosmosCoin struct {
	Denom   string
	Decimal uint8
}

// InitChains init chains
func (b *Bridge) InitChains() {
	ChainIDs["cosmos-hub4"] = true
}

// InitCoins init coins
func (b *Bridge) InitCoins() {
	SupportedCoins["ATOM"] = CosmosCoin{"uatom", 9}
	tokenCfg := b.GetTokenConfig(PairID)
	symbol := strings.ToUpper(tokenCfg.Symbol)
	TheCoin = SupportedCoins[symbol]
}

// Bridge btc bridge
type Bridge struct {
	*tokens.CrossChainBridgeBase
	*eth.NonceSetterBase
}

// NewCrossChainBridge new bridge
func NewCrossChainBridge(isSrc bool) *Bridge {
	return &Bridge{
		CrossChainBridgeBase: tokens.NewCrossChainBridgeBase(isSrc),
		NonceSetterBase:      eth.NewNonceSetterBase(),
	}
}

// SetChainAndGateway set chain and gateway config
func (b *Bridge) SetChainAndGateway(chainCfg *tokens.ChainConfig, gatewayCfg *tokens.GatewayConfig) {
	b.CrossChainBridgeBase.SetChainAndGateway(chainCfg, gatewayCfg)
	b.InitLatestBlockNumber()
	b.VerifyChainID()
}

// VerifyChainID verify chain id
func (b *Bridge) VerifyChainID() {
	chainID := strings.ToLower(b.ChainConfig.NetID)
	if ChainIDs[chainID] == false {
		log.Fatalf("unsupported cosmos network: %v", b.ChainConfig.NetID)
	}
}

// VerifyTokenConfig verify token config
func (b *Bridge) VerifyTokenConfig(tokenCfg *tokens.TokenConfig) error {
	if !b.IsValidAddress(tokenCfg.DepositAddress) {
		return fmt.Errorf("invalid deposit address: %v", tokenCfg.DepositAddress)
	}
	symbol := strings.ToUpper(tokenCfg.Symbol)
	if coin, ok := SupportedCoins[symbol]; ok {
		if coin.Decimal != *tokenCfg.Decimals {
			return fmt.Errorf("invalid decimals for %v: want %v but have %v", symbol, coin.Decimal, *tokenCfg.Decimals)
		}
	} else {
		return fmt.Errorf("Unsupported cosmos coin type")
	}
	return nil
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
