package cosmos

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/tendermint/tendermint/crypto/tmhash"
)

// StdSignContent saves all tx components required to build SignBytes
type StdSignContent struct {
	AccountNumber uint64
	ChainID       string
	Fee           authtypes.StdFee
	Memo          string
	Msgs          []sdk.Msg
	Sequence      uint64
}

// HashableStdTx saves all data of a signed tx
type HashableStdTx struct {
	StdSignContent
	Signatures []authtypes.StdSignature
}

// SignBytes returns sign bytes
func (tx StdSignContent) SignBytes() []byte {
	signBytes := StdSignBytes(tx.ChainID, tx.AccountNumber, tx.Sequence, tx.Fee, tx.Msgs, tx.Memo)
	return SignBytesModifier(signBytes) // ugly but works
}

// Hash returns tx sign bytes hash string
// not the tx hash
func (tx StdSignContent) Hash() string {
	signBytes := tx.SignBytes()
	txHash := fmt.Sprintf("%X", tmhash.Sum(signBytes))
	return txHash
}

// ToStdTx converts HashableStdTx to authtypes.StdTx
func (tx HashableStdTx) ToStdTx() authtypes.StdTx {
	return authtypes.StdTx{
		Msgs:       tx.Msgs,
		Fee:        tx.Fee,
		Signatures: tx.Signatures,
		Memo:       tx.Memo,
	}
}

// StdSignBytes returns signing bytes
func StdSignBytes(chainID string, accnum uint64, sequence uint64, fee authtypes.StdFee, msgs []sdk.Msg, memo string) []byte {
	msgsBytes := make([]json.RawMessage, 0, len(msgs))
	for _, msg := range msgs {
		msgsBytes = append(msgsBytes, json.RawMessage(msg.GetSignBytes()))
	}
	bz, err := CDC.MarshalJSON(authtypes.StdSignDoc{
		AccountNumber: accnum,
		ChainID:       chainID,
		Fee:           json.RawMessage(fee.Bytes()),
		Memo:          memo,
		Msgs:          msgsBytes,
		Sequence:      sequence,
	})
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(bz)
}

// SignBytesModifier modifies sign bytes
var SignBytesModifier (func([]byte) []byte) = func(bz []byte) []byte {
	return bz
}
