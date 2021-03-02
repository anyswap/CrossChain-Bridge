package cosmos

import (
	"fmt"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/eth"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	cyptes "github.com/tendermint/tendermint/rpc/core/types"
)

var (
	// ChainIDs saves supported chain ids
	ChainIDs = make(map[string]bool)
	// MainCoin is the gas coin
	MainCoin CosmosCoin
	// SupportedCoins save cosmos coins
	SupportedCoins = make(map[string]CosmosCoin)
)

// CosmosBridgeInterface interface
type CosmosBridgeInterface interface {
	BeforeConfig()
	AfterConfig()
}

// BeforeConfig run before loading bridge and token config
func (b *Bridge) BeforeConfig() {
	cyptes.RegisterAmino(CDC)
	sdk.RegisterCodec(CDC)
	ChainIDs["cosmos-hub4"] = true
	SupportedCoins["ATOM"] = CosmosCoin{"uatom", 9}
	MainCoin = SupportedCoins["ATOM"]
	tokens.IsSwapoutToStringAddress = true
}

// AfterConfig run after loading bridge and token config
func (b *Bridge) AfterConfig() {
	GetFeeAmount = b.FeeGetter()
	b.InitLatestBlockNumber()
}

// CosmosCoin struct
type CosmosCoin struct {
	Denom   string
	Decimal uint8
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

// DefaultSwapoutGas is default cosmos tx gas
var DefaultSwapoutGas uint64 = 300000

// GetFeeAmount returns StdFee
var GetFeeAmount func() authtypes.StdFee

// FeeGetter returns a cosmos fee getter
func (b *Bridge) FeeGetter() func() authtypes.StdFee {
	return func() authtypes.StdFee {
		// TODO
		feeAmount := sdk.Coins{sdk.Coin{"uatom", sdk.NewInt(3000)}}
		return authtypes.NewStdFee(DefaultSwapoutGas, feeAmount)
	}
}
