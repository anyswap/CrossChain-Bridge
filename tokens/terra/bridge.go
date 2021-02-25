package terra

import (
	"strings"

	"github.com/anyswap/CrossChain-Bridge/tokens/cosmos"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
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

// Init init after verify
func (b *Bridge) Init() {
	cosmos.PairID = PairID
	InitSDK()
	b.InitCoins()
	b.Bridge.InitLatestBlockNumber()
	cosmos.GetFeeAmount = TerraGetFeeAmount
}

// InitChains init chains
func (b *Bridge) InitChains() {
	cosmos.ChainIDs["columbus-4"] = true
	cosmos.ChainIDs["tequila-0004"] = true
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

var DefaultSwapoutGas uint64 = 300000

var TerraGetFeeAmount = func() authtypes.StdFee {
	// TODO
	feeAmount := sdk.Coins{sdk.Coin{"uluna", sdk.NewInt(1000)}}
	return authtypes.NewStdFee(DefaultSwapoutGas, feeAmount)
}
