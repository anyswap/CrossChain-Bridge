package cosmos

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type StdSignContent struct {
	AccountNumber uint64
	ChainID       string
	Fee           authtypes.StdFee
	Memo          string
	Msgs          []sdk.Msg
	Sequence      uint64
}
