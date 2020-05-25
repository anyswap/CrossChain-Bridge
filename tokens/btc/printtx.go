package btc

import (
	"bytes"
	"encoding/json"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/wallet/txauthor"
	"github.com/fsn-dev/crossChain-Bridge/common/hexutil"
)

func MarshalToJson(obj interface{}, pretty bool) string {
	var jsdata []byte
	if pretty {
		jsdata, _ = json.MarshalIndent(obj, "", "  ")
	} else {
		jsdata, _ = json.Marshal(obj)
	}
	return string(jsdata)
}

func AuthoredTxToString(authtx interface{}, pretty bool) string {
	authoredTx, ok := authtx.(*txauthor.AuthoredTx)
	if !ok {
		return MarshalToJson(authtx, pretty)
	}

	var encAuthTx EncAuthoredTx

	encAuthTx.ChangeIndex = authoredTx.ChangeIndex
	encAuthTx.TotalInput = authoredTx.TotalInput

	tx := authoredTx.Tx
	if tx == nil {
		return MarshalToJson(encAuthTx, pretty)
	}

	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	tx.Serialize(buf)
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
		encTx.TxOut[i].PkScript, _ = txscript.DisasmString(txOut.PkScript)
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
		encTx.TxIn[i].SignatureScript, _ = txscript.DisasmString(txIn.SignatureScript)
		encTx.TxIn[i].Witness = make([]hexutil.Bytes, len(txIn.Witness))
		for j, witness := range txIn.Witness {
			encTx.TxIn[i].Witness[j] = (hexutil.Bytes)(witness)
		}
	}

	encAuthTx.Tx = &encTx
	return MarshalToJson(encAuthTx, pretty)
}

type EncAuthoredTx struct {
	Tx          *EncMsgTx
	TotalInput  btcutil.Amount
	ChangeIndex int
}

type EncMsgTx struct {
	Txid     string
	Version  int32
	TxIn     []*EncTxIn
	TxOut    []*EncTxOut
	LockTime uint32
}

type EncTxOut struct {
	PkScript string
	Value    int64
}

type EncOutPoint struct {
	Hash  string
	Index uint32
}

type EncTxIn struct {
	PreviousOutPoint EncOutPoint
	SignatureScript  string
	Witness          []hexutil.Bytes
	Sequence         uint32
	Value            btcutil.Amount
}
