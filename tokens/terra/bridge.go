package terra

import (
	"fmt"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/cosmos"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	cyptes "github.com/tendermint/tendermint/rpc/core/types"
	core "github.com/terra-project/core/types"
)

type Bridge struct {
	*cosmos.Bridge
}

func InitSDK() {
	config := sdk.GetConfig()
	config.SetCoinType(core.CoinType)
	config.SetFullFundraiserPath(core.FullFundraiserPath)
	config.SetBech32PrefixForAccount(core.Bech32PrefixAccAddr, core.Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(core.Bech32PrefixValAddr, core.Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(core.Bech32PrefixConsAddr, core.Bech32PrefixConsPub)
	config.Seal()
}

var PairID = "TERRA"

func (b *Bridge) BeforeConfig() {
	cosmos.PairID = PairID
	cyptes.RegisterAmino(cosmos.CDC)
	sdk.RegisterCodec(cosmos.CDC)
	InitSDK()
	cosmos.GetFeeAmount = TerraGetFeeAmount
	b.InitChains()
}

func (b *Bridge) AfterConfig() {
	b.InitCoins()
	b.Bridge.InitLatestBlockNumber()
}

// InitChains init chains
func (b *Bridge) InitChains() {
	cosmos.ChainIDs["columbus-4"] = true
	cosmos.ChainIDs["tequila-0004"] = true
	cosmos.ChainIDs["mytestnet"] = true
}

// InitCoins init coins
func (b *Bridge) InitCoins() {
	cosmos.SupportedCoins["LUNA"] = cosmos.CosmosCoin{"uluna", 6}
	cosmos.SupportedCoins["USD"] = cosmos.CosmosCoin{"uusd", 6}
	cosmos.SupportedCoins["KRW"] = cosmos.CosmosCoin{"ukrw", 6}
	cosmos.SupportedCoins["SDR"] = cosmos.CosmosCoin{"usdr", 6}
	cosmos.SupportedCoins["CNY"] = cosmos.CosmosCoin{"ucny", 6}
	cosmos.SupportedCoins["JPY"] = cosmos.CosmosCoin{"ujpy", 6}
	cosmos.SupportedCoins["EUR"] = cosmos.CosmosCoin{"ueur", 6}
	cosmos.SupportedCoins["GBP"] = cosmos.CosmosCoin{"ugbp", 6}
	cosmos.SupportedCoins["UMNT"] = cosmos.CosmosCoin{"umnt", 6}

	tokenCfg := b.GetTokenConfig(PairID)
	symbol := strings.ToUpper(tokenCfg.Symbol)
	cosmos.TheCoin = cosmos.SupportedCoins[symbol]
}

// NewCrossChainBridge new bridge
func NewCrossChainBridge(isSrc bool) *Bridge {
	return &Bridge{
		Bridge: cosmos.NewCrossChainBridge(isSrc),
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
	if cosmos.ChainIDs[chainID] == false {
		log.Fatalf("unsupported cosmos network: %v", b.ChainConfig.NetID)
	}
}

// VerifyTokenConfig verify token config
func (b *Bridge) VerifyTokenConfig(tokenCfg *tokens.TokenConfig) error {
	if !b.IsValidAddress(tokenCfg.DepositAddress) {
		return fmt.Errorf("invalid deposit address: %v", tokenCfg.DepositAddress)
	}
	symbol := strings.ToUpper(tokenCfg.Symbol)
	if coin, ok := cosmos.SupportedCoins[symbol]; ok {
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

var DefaultSwapoutGas uint64 = 300000

var TerraGetFeeAmount = func() authtypes.StdFee {
	// TODO
	feeAmount := sdk.Coins{sdk.Coin{"uluna", sdk.NewInt(1000)}}
	return authtypes.NewStdFee(DefaultSwapoutGas, feeAmount)
}
