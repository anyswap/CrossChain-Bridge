package ltc

import (
	"bytes"
	"encoding/json"

	"github.com/anyswap/CrossChain-Bridge/common/hexutil"
	"github.com/ltcsuite/ltcwallet/wallet/txauthor"
)

// MarshalToJSON marshal to json
func MarshalToJSON(obj interface{}, pretty bool) string {
	var jsdata []byte
	if pretty {
		jsdata, _ = json.MarshalIndent(obj, "", "  ")
	} else {
		jsdata, _ = json.Marshal(obj)
	}
	return string(jsdata)
}

// AuthoredTxToString AuthoredTx to string
func AuthoredTxToString(authtx interface{}, pretty bool) string {
	authoredTx, ok := authtx.(*txauthor.AuthoredTx)
	if !ok {
		return MarshalToJSON(authtx, pretty)
	}

	var encAuthTx EncAuthoredTx

	encAuthTx.ChangeIndex = authoredTx.ChangeIndex
	encAuthTx.TotalInput = authoredTx.TotalInput

	tx := authoredTx.Tx
	if tx == nil {
		return MarshalToJSON(encAuthTx, pretty)
	}

	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	_ = tx.Serialize(buf)
	txid := tx.TxHash().String()

	var encTx EncMsgTx

	encTx.Txid = txid
	encTx.Version = tx.Version
	encTx.LockTime = tx.LockTime

	encTx.TxOut = make([]*EncTxOut, len(tx.TxOut))
	for i, txOut := range tx.TxOut {
		encTx.TxOut[i] = &EncTxOut{
			Value: txOut.Value,
		}
		encTx.TxOut[i].PkScript, _ = disasmScriptToString(txOut.PkScript)
	}

	encTx.TxIn = make([]*EncTxIn, len(tx.TxIn))
	for i, txIn := range tx.TxIn {
		encTx.TxIn[i] = &EncTxIn{
			PreviousOutPoint: EncOutPoint{
				Hash:  txIn.PreviousOutPoint.Hash.String(),
				Index: txIn.PreviousOutPoint.Index,
			},
			Sequence: txIn.Sequence,
			Value:    authoredTx.PrevInputValues[i],
		}
		encTx.TxIn[i].SignatureScript, _ = disasmScriptToString(txIn.SignatureScript)
		encTx.TxIn[i].Witness = make([]hexutil.Bytes, len(txIn.Witness))
		for j, witness := range txIn.Witness {
			encTx.TxIn[i].Witness[j] = hexutil.Bytes(witness)
		}
	}

	encAuthTx.Tx = &encTx
	return MarshalToJSON(encAuthTx, pretty)
}

// EncAuthoredTx stuct
type EncAuthoredTx struct {
	Tx          *EncMsgTx
	TotalInput  ltcAmountType
	ChangeIndex int
}

// EncMsgTx struct
type EncMsgTx struct {
	Txid     string
	Version  int32
	TxIn     []*EncTxIn
	TxOut    []*EncTxOut
	LockTime uint32
}

// EncTxOut struct
type EncTxOut struct {
	PkScript string
	Value    int64
}

// EncOutPoint struct
type EncOutPoint struct {
	Hash  string
	Index uint32
}

// EncTxIn struct
type EncTxIn struct {
	PreviousOutPoint EncOutPoint
	SignatureScript  string
	Witness          []hexutil.Bytes
	Sequence         uint32
	Value            ltcAmountType
}
