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

// Bridge struct
type Bridge struct {
	*cosmos.Bridge
}

// InitSDK init cosmos sdk
func InitSDK() {
	config := sdk.GetConfig()
	config.SetCoinType(core.CoinType)
	config.SetFullFundraiserPath(core.FullFundraiserPath)
	config.SetBech32PrefixForAccount(core.Bech32PrefixAccAddr, core.Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(core.Bech32PrefixValAddr, core.Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(core.Bech32PrefixConsAddr, core.Bech32PrefixConsPub)
	config.Seal()
}

// BeforeConfig run before loading bridge and token config
func (b *Bridge) BeforeConfig() {
	cyptes.RegisterAmino(cosmos.CDC)
	sdk.RegisterCodec(cosmos.CDC)
	cosmos.CDC.RegisterConcrete(cosmos.MsgSend{}, "bank/MsgSend", nil)
	cosmos.CDC.RegisterConcrete(cosmos.MsgMultiSend{}, "bank/MsgMultiSend", nil)
	cosmos.CDC.RegisterConcrete(authtypes.StdTx{}, "core/StdTx", nil)
	InitSDK()
	cosmos.ChainIDs["columbus-4"] = true
	cosmos.ChainIDs["tequila-0004"] = true
	/*b.SupportedCoins["LUNA"] = cosmos.CosmosCoin{"uluna", 6}
	b.SupportedCoins["USD"] = cosmos.CosmosCoin{"uusd", 6}
	b.SupportedCoins["KRW"] = cosmos.CosmosCoin{"ukrw", 6}
	b.SupportedCoins["SDR"] = cosmos.CosmosCoin{"usdr", 6}
	b.SupportedCoins["CNY"] = cosmos.CosmosCoin{"ucny", 6}
	b.SupportedCoins["JPY"] = cosmos.CosmosCoin{"ujpy", 6}
	b.SupportedCoins["EUR"] = cosmos.CosmosCoin{"ueur", 6}
	b.SupportedCoins["GBP"] = cosmos.CosmosCoin{"ugbp", 6}
	b.SupportedCoins["UMNT"] = cosmos.CosmosCoin{"umnt", 6}*/
	tokens.IsSwapoutToStringAddress = true
}

// AfterConfig run after loading bridge and token config
func (b *Bridge) AfterConfig() {
	cosmos.GetFeeAmount = b.FeeGetter()
	b.Bridge.InitLatestBlockNumber()
	b.LoadCoins()
	log.Info("111111", "coins", b.SupportedCoins)
	if luna, ok := b.SupportedCoins["LUNA"]; ok == false || luna.Denom != "uluna" || luna.Decimal != 6 {
		log.Info("222222", "luna", luna, "ok", ok, "check denom", (luna.Denom != "uluna"), "check decimal", luna.Decimal != 6)
		log.Fatalf("Terra bridge must have Luna token config")
	}
	b.MainCoin = b.SupportedCoins["LUNA"]
	log.Info("Terra bridge init success", "coins", b.SupportedCoins)
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

// DefaultSwapoutGas is terra default gas
var DefaultSwapoutGas uint64 = 300000

// FeeGetter returns terra fee getter
func (b *Bridge) FeeGetter() func() authtypes.StdFee {
	return func() authtypes.StdFee {
		// TODO
		feeAmount := sdk.Coins{sdk.Coin{"uluna", sdk.NewInt(1000)}}
		return authtypes.NewStdFee(DefaultSwapoutGas, feeAmount)
	}
}
