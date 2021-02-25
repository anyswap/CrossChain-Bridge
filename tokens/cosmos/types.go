package cosmos

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/tendermint/tendermint/crypto/tmhash"
)

type StdSignContent struct {
	AccountNumber uint64
	ChainID       string
	Fee           authtypes.StdFee
	Memo          string
	Msgs          []sdk.Msg
	Sequence      uint64
}

type HashableStdTx struct {
	StdSignContent
	Signatures []authtypes.StdSignature
}

func (tx StdSignContent) SignBytes() []byte {
	return authtypes.StdSignBytes(tx.ChainID, tx.AccountNumber, tx.Sequence, tx.Fee, tx.Msgs, tx.Memo)
}

func (tx StdSignContent) Hash() string {
	signBytes := authtypes.StdSignBytes(tx.ChainID, tx.AccountNumber, tx.Sequence, tx.Fee, tx.Msgs, tx.Memo)
	txHash := fmt.Sprintf("%X", tmhash.Sum(signBytes))
	return txHash
}

func (tx HashableStdTx) ToStdTx() authtypes.StdTx {
	return authtypes.StdTx{
		Msgs:       tx.Msgs,
		Fee:        tx.Fee,
		Signatures: tx.Signatures,
		Memo:       tx.Memo,
	}
}
