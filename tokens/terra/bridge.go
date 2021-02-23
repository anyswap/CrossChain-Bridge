package terra

import (
	"github.com/anyswap/CrossChain-Bridge/tokens/cosmos"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	core "github.com/terra-project/core/types"
)

type Bridge struct {
	*cosmos.Bridge
}

func initSDK() {
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
	initSDK()
	b.InitCoins()
	b.Bridge.InitLatestBlockNumber()
	cosmos.GetFeeAmount = TerraGetFeeAmount
}

// InitChains init chains
func InitChains() {
	cosmos.ChainIDs["columbus-4"] = true
	cosmos.ChainIDs["tequila-0004"] = true
}

// InitCoins init coins
func (b *Bridge) InitCoins() {
	cosmos.Coins["LUNA"] = cosmos.CosmosCoin{"uluna", 6}
	cosmos.Coins["USD"] = cosmos.CosmosCoin{"uusd", 6}
	cosmos.Coins["KRW"] = cosmos.CosmosCoin{"ukrw", 6}
	cosmos.Coins["SDR"] = cosmos.CosmosCoin{"usdr", 6}
	cosmos.Coins["CNY"] = cosmos.CosmosCoin{"ucny", 6}
	cosmos.Coins["JPY"] = cosmos.CosmosCoin{"ujpy", 6}
	cosmos.Coins["EUR"] = cosmos.CosmosCoin{"ueur", 6}
	cosmos.Coins["GBP"] = cosmos.CosmosCoin{"ugbp", 6}
	cosmos.Coins["UMNT"] = cosmos.CosmosCoin{"umnt", 6}
}

var TerraGetFeeAmount = func() authtypes.StdFee {
	// TODO
	return sdk.Coins{sdk.Coin{"uluna", sdk.NewInt(1000)}}
}
