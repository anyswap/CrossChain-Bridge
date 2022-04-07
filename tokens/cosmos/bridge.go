package cosmos

import (
	"fmt"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/base"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	cyptes "github.com/tendermint/tendermint/rpc/core/types"
)

var (
	// ChainIDs saves supported chain ids
	ChainIDs = make(map[string]bool)

	// ensure Bridge impl tokens.CrossChainBridge
	_ tokens.CrossChainBridge = &Bridge{}
	// ensure Bridge impl tokens.NonceSetter
	_ tokens.NonceSetter = &Bridge{}
)

// BridgeInterface interface
type BridgeInterface interface {
	BeforeConfig()
	AfterConfig()
}

// BeforeConfig run before loading bridge and token config
func (b *Bridge) BeforeConfig() {
	cyptes.RegisterAmino(CDC)
	sdk.RegisterCodec(CDC)
	RegisterCodec(CDC)
	CDC.RegisterConcrete(&authtypes.BaseAccount{}, "cosmos-sdk/Account", nil)
	initTxHashCdc()
	ChainIDs["cosmos-hub4"] = true
	ChainIDs["stargate-final"] = true
	// SupportedCoins["ATOM"] = Coin{"uatom", 9}
}

// AfterConfig run after loading bridge and token config
func (b *Bridge) AfterConfig() {
	GetFeeAmount = b.FeeGetter()
	b.InitLatestBlockNumber()
	b.LoadCoins()

	switch b.ChainConfig.NetID {
	case "stargate-final":
		if umuon, ok := b.SupportedCoins["MUON"]; !ok || umuon.Denom != "umuon" || umuon.Decimal != 6 {
			log.Fatalf("Cosmos post-stargate bridge must have MUON token config")
		}
		b.MainCoin = b.SupportedCoins["MUON"]
	case "cosmos-hub4":
		if atom, ok := b.SupportedCoins["ATOM"]; !ok || atom.Denom != "uatom" || atom.Decimal != 9 {
			log.Fatalf("Cosmos pre-stargate bridge must have Atom token config")
		}
		b.MainCoin = b.SupportedCoins["ATOM"]
	default:
		if atom, ok := b.SupportedCoins["ATOM"]; !ok || atom.Denom != "uatom" || atom.Decimal != 9 {
			if umuon, ok := b.SupportedCoins["MUON"]; !ok || umuon.Denom != "umuon" || umuon.Decimal != 9 {
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
		name := strings.ToUpper(tokenCfg.PairID)
		unit := tokenCfg.SrcToken.Unit
		decimal := *(tokenCfg.SrcToken.Decimals)
		b.SupportedCoins[name] = Coin{unit, decimal}
	}
}

// GetCoin returns supported coin by name
func (b *Bridge) GetCoin(name string) (Coin, bool) {
	name = strings.ToUpper(name)
	coin, ok := b.SupportedCoins[name]
	if !ok {
		b.LoadCoins()
		coin, ok = b.SupportedCoins[name]
	}
	return coin, ok
}

// Coin struct
type Coin struct {
	Denom   string
	Decimal uint8
}

// Bridge cosmos bridge
type Bridge struct {
	*base.NonceSetterBase
	// MainCoin is the gas coin
	MainCoin Coin
	// SupportedCoins save cosmos coins
	SupportedCoins map[string]Coin
}

// NewCrossChainBridge new bridge
func NewCrossChainBridge(isSrc bool) *Bridge {
	return &Bridge{
		NonceSetterBase: base.NewNonceSetterBase(isSrc),
		SupportedCoins:  make(map[string]Coin),
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
	if !ChainIDs[chainID] {
		log.Fatalf("unsupported cosmos network: %v", b.ChainConfig.NetID)
	}
}

// VerifyTokenConfig verify token config
func (b *Bridge) VerifyTokenConfig(tokenCfg *tokens.TokenConfig) error {
	if !b.IsValidAddress(tokenCfg.DepositAddress) {
		return fmt.Errorf("invalid deposit address: %v", tokenCfg.DepositAddress)
	}
	if tokenCfg.Unit == "" {
		return fmt.Errorf("empty 'Unit' in token config")
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
var GetFeeAmount func(string, *StdSignContent) authtypes.StdFee

// FeeGetter returns a cosmos fee getter
func (b *Bridge) FeeGetter() func(pairID string, tx *StdSignContent) authtypes.StdFee {
	switch b.ChainConfig.NetID {
	case "stargate-final":
		return func(pairID string, tx *StdSignContent) authtypes.StdFee {
			feeAmount := sdk.Coins{sdk.Coin{Denom: "umuon", Amount: sdk.NewInt(3000)}}
			return authtypes.NewStdFee(DefaultSwapoutGas, feeAmount)
		}
	case "cosmos-hub4":
		return func(pairID string, tx *StdSignContent) authtypes.StdFee {
			feeAmount := sdk.Coins{sdk.Coin{Denom: "uatom", Amount: sdk.NewInt(3000)}}
			return authtypes.NewStdFee(DefaultSwapoutGas, feeAmount)
		}
	default:
		return func(pairID string, tx *StdSignContent) authtypes.StdFee {
			feeAmount := sdk.Coins{sdk.Coin{Denom: "uatom", Amount: sdk.NewInt(3000)}}
			return authtypes.NewStdFee(DefaultSwapoutGas, feeAmount)
		}
	}
}

var txhashcdc *codec.Codec

func initTxHashCdc() {
	txhashcdc = codec.New()
	codec.RegisterCrypto(txhashcdc)
	RegisterCodec(txhashcdc)
	txhashcdc.RegisterConcrete(authtypes.StdTx{}, "auth/StdTx", nil)
	txhashcdc.RegisterInterface((*sdk.Msg)(nil), nil)
}

// CaluculateTxHash calculate tx hash
var CaluculateTxHash = func(signedTx HashableStdTx) (string, error) {
	return tokens.StubSignedTxHash, nil // TODO
}
