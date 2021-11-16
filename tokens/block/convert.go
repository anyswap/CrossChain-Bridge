package block

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/wire"
)

// ConvertTx converts btcjson raw tx result to elect tx
func ConvertTx(tx *btcjson.TxRawResult) *electrs.ElectTx {
	etx := &electrs.ElectTx{
		Txid:     &tx.Txid,
		Version:  new(uint32),
		Locktime: new(uint32),
		Size:     new(uint32),
		Weight:   new(uint32),
		Fee:      new(uint64),
		Vin:      make([]*electrs.ElectTxin, 0),
		Vout:     make([]*electrs.ElectTxOut, 0),
		Status:   TxStatus(tx),
	}
	*etx.Version = uint32(tx.Version)
	*etx.Locktime = tx.LockTime
	*etx.Size = uint32(tx.Size)
	*etx.Weight = uint32(tx.Weight)
	for i := 0; i < len(tx.Vin); i++ {
		evin := ConvertVin(&tx.Vin[i])
		etx.Vin = append(etx.Vin, evin)
	}
	for j := 0; j < len(tx.Vout); j++ {
		evout := ConvertVout(&tx.Vout[j])
		etx.Vout = append(etx.Vout, evout)
	}
	return etx
}

// TxStatus make elect tx status from btcjson tx raw result
func TxStatus(tx *btcjson.TxRawResult) *electrs.ElectTxStatus {
	status := &electrs.ElectTxStatus{
		Confirmed:   new(bool),
		BlockHeight: new(uint64),
		BlockHash:   new(string),
		BlockTime:   new(uint64),
	}
	*status.Confirmed = tx.Confirmations > 6
	*status.BlockHash = tx.BlockHash
	*status.BlockTime = uint64(tx.Blocktime)
	return status
}

// TxOutspend make elect outspend from btcjson tx raw result
func TxOutspend(txout *btcjson.GetTxOutResult) *electrs.ElectOutspend {
	outspend := &electrs.ElectOutspend{
		Spent: new(bool),
	}
	if txout == nil {
		*outspend.Spent = true
	}
	return outspend
}

// DecodeTxHex decode tx hex to msgTx
func DecodeTxHex(txHex string, protocolversion uint32, isWitness bool) *wire.MsgTx {
	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		return nil
	}

	msgtx := new(wire.MsgTx)

	if isWitness {
		_ = msgtx.BtcDecode(bytes.NewReader(txBytes), protocolversion, wire.WitnessEncoding)
	} else {
		_ = msgtx.BtcDecode(bytes.NewReader(txBytes), protocolversion, wire.BaseEncoding)
	}

	return msgtx
}

// ConvertBlock converts btcjson block verbose result to elect block
func ConvertBlock(blk *btcjson.GetBlockVerboseResult) *electrs.ElectBlock {
	eblk := &electrs.ElectBlock{
		Hash:         new(string),
		Height:       new(uint32),
		Version:      new(uint32),
		Timestamp:    new(uint32),
		TxCount:      new(uint32),
		Size:         new(uint32),
		Weight:       new(uint32),
		MerkleRoot:   new(string),
		PreviousHash: new(string),
		Nonce:        new(uint32),
		Bits:         new(uint32),
		Difficulty:   new(uint64),
	}
	*eblk.Hash = blk.Hash
	*eblk.Height = uint32(blk.Height)
	*eblk.Version = uint32(blk.Version)
	*eblk.Timestamp = uint32(blk.Time)
	*eblk.TxCount = uint32(len(blk.Tx))
	*eblk.Size = uint32(blk.Size)
	*eblk.Weight = uint32(blk.Weight)
	*eblk.MerkleRoot = blk.MerkleRoot
	*eblk.PreviousHash = blk.PreviousHash
	*eblk.Nonce = blk.Nonce
	if bits, err := strconv.ParseUint(blk.Bits, 16, 32); err == nil {
		*eblk.Bits = uint32(bits)
	}
	*eblk.Difficulty = uint64(blk.Difficulty)
	return eblk
}

// ConvertVin converts btcjson vin to elect vin
func ConvertVin(vin *btcjson.Vin) *electrs.ElectTxin {
	evin := &electrs.ElectTxin{
		Txid:         &vin.Txid,
		Vout:         &vin.Vout,
		Scriptsig:    new(string),
		ScriptsigAsm: new(string),
		IsCoinbase:   new(bool),
		Sequence:     &vin.Sequence,
	}
	if vin.ScriptSig != nil {
		*evin.Scriptsig = vin.ScriptSig.Hex
		*evin.ScriptsigAsm = vin.ScriptSig.Asm
	}
	*evin.IsCoinbase = (vin.Coinbase != "")
	return evin
}

// ConvertVout converts btcjson vout to elect vout
func ConvertVout(vout *btcjson.Vout) *electrs.ElectTxOut {
	evout := &electrs.ElectTxOut{
		Scriptpubkey:        &vout.ScriptPubKey.Hex,
		ScriptpubkeyAsm:     &vout.ScriptPubKey.Asm,
		ScriptpubkeyType:    new(string),
		ScriptpubkeyAddress: new(string),
		Value:               new(uint64),
	}
	switch vout.ScriptPubKey.Type {
	case "pubkeyhash":
		*evout.ScriptpubkeyType = p2pkhType
	case "scripthash":
		*evout.ScriptpubkeyType = p2shType
	default:
		*evout.ScriptpubkeyType = opReturnType
	}
	if len(vout.ScriptPubKey.Addresses) == 1 {
		*evout.ScriptpubkeyAddress = vout.ScriptPubKey.Addresses[0]
	}
	if len(vout.ScriptPubKey.Addresses) > 1 {
		*evout.ScriptpubkeyAddress = fmt.Sprintf("%+v", vout.ScriptPubKey.Addresses)
	}
	*evout.Value = uint64(vout.Value * 1e8)
	return evout
}
