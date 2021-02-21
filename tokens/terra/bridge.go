package terra

import (
	"github.com/anyswap/CrossChain-Bridge/tokens/cosmos"
	core "github.com/terra-project/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Bridge struct {
	*cosmos.Bridge
}

func init() {
	config := sdk.GetConfig()
	config.SetCoinType(core.CoinType)
	config.SetFullFundraiserPath(core.FullFundraiserPath)
	config.SetBech32PrefixForAccount(core.Bech32PrefixAccAddr, core.Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(core.Bech32PrefixValAddr, core.Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(core.Bech32PrefixConsAddr, core.Bech32PrefixConsPub)
	config.Seal()
}