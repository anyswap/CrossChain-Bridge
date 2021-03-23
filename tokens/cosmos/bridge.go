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
	RegisterCodec(CDC)
	CDC.RegisterConcrete(&authtypes.BaseAccount{}, "cosmos-sdk/Account", nil)
	ChainIDs["cosmos-hub4"] = true
	ChainIDs["stargate-final"] = true
	// SupportedCoins["ATOM"] = CosmosCoin{"uatom", 9}
	tokens.IsSwapoutToStringAddress = true
}

// AfterConfig run after loading bridge and token config
func (b *Bridge) AfterConfig() {
	GetFeeAmount = b.FeeGetter()
	b.InitLatestBlockNumber()
	b.LoadCoins()

	switch b.ChainConfig.NetID {
	case "stargate-final":
		if umuon, ok := b.SupportedCoins["MUON"]; ok == false || umuon.Denom != "umuon" || umuon.Decimal != 6 {
			log.Fatalf("Cosmos post-stargate bridge must have MUON token config")
		}
		b.MainCoin = b.SupportedCoins["MUON"]
	case "cosmos-hub4":
		if atom, ok := b.SupportedCoins["ATOM"]; ok == false || atom.Denom != "uatom" || atom.Decimal != 9 {
			log.Fatalf("Cosmos pre-stargate bridge must have Atom token config")
		}
		b.MainCoin = b.SupportedCoins["ATOM"]
	default:
		if atom, ok := b.SupportedCoins["ATOM"]; ok == false || atom.Denom != "uatom" || atom.Decimal != 9 {
			if umuon, ok := b.SupportedCoins["MUON"]; ok == false || umuon.Denom != "umuon" || umuon.Decimal != 9 {
				log.Fatalf("Cosmos bridge must have one of Atom or Muon token config")
			}
		}
		b.MainCoin = b.SupportedCoins["ATOM"]
	}
	log.Info("Cosmos bridge init success", "coins", b.SupportedCoins)
}

// LoadCoins read and check token pairs config
func (b *Bridge) LoadCoins() {
	pairs := tokens.GetTokenPairsConfig()
	for _, tokenCfg := range pairs {
		name := strings.ToUpper(tokenCfg.SrcToken.ID)
		unit := tokenCfg.SrcToken.Unit
		decimal := *(tokenCfg.SrcToken.Decimals)
		b.SupportedCoins[name] = CosmosCoin{unit, decimal}
	}
}

// GetCoin returns supported coin by name
func (b *Bridge) GetCoin(name string) (CosmosCoin, bool) {
	name = strings.ToUpper(name)
	coin, ok := b.SupportedCoins[name]
	if !ok {
		b.LoadCoins()
	}
	coin, ok = b.SupportedCoins[name]
	return coin, ok
}

// CosmosCoin struct
type CosmosCoin struct {
	Denom   string
	Decimal uint8
}

// Bridge cosmos bridge
type Bridge struct {
	*tokens.CrossChainBridgeBase
	*eth.NonceSetterBase
	// MainCoin is the gas coin
	MainCoin CosmosCoin
	// SupportedCoins save cosmos coins
	SupportedCoins map[string]CosmosCoin
}

// NewCrossChainBridge new bridge
func NewCrossChainBridge(isSrc bool) *Bridge {
	return &Bridge{
		CrossChainBridgeBase: tokens.NewCrossChainBridgeBase(isSrc),
		NonceSetterBase:      eth.NewNonceSetterBase(),
		SupportedCoins:       make(map[string]CosmosCoin),
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
	switch b.ChainConfig.NetID {
	case "stargate-final":
		return func() authtypes.StdFee {
			feeAmount := sdk.Coins{sdk.Coin{"umuon", sdk.NewInt(3000)}}
			return authtypes.NewStdFee(DefaultSwapoutGas, feeAmount)
		}
	case "cosmos-hub4":
		return func() authtypes.StdFee {
			feeAmount := sdk.Coins{sdk.Coin{"uatom", sdk.NewInt(3000)}}
			return authtypes.NewStdFee(DefaultSwapoutGas, feeAmount)
		}
	default:
		return func() authtypes.StdFee {
			feeAmount := sdk.Coins{sdk.Coin{"uatom", sdk.NewInt(3000)}}
			return authtypes.NewStdFee(DefaultSwapoutGas, feeAmount)
		}
	}
}
