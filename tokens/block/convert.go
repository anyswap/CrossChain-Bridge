package block

import (
	"bytes"
	"encoding/hex"
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
	*etx.Locktime = uint32(tx.LockTime)
	*etx.Size = uint32(tx.Size)
	*etx.Weight = uint32(tx.Weight)
	for _, vin := range tx.Vin {
		evin := ConvertVin(vin)
		etx.Vin = append(etx.Vin, evin)
	}
	for _, vout := range tx.Vout {
		evout := ConvertVout(vout)
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
func TxOutspend(tx *btcjson.TxRawResult, vout uint32) *electrs.ElectOutspend {
	/*for _, txout := range tx.Vout{
		if txout.N == vout {
			outspend := &electrs.TxOutspend {
				Spent: new(bool),
				Txid: new(string),
				Vin: new(uint32),
				Status: TxStatus(tx),
			}
			*outspend.Txid = tx.Txid
			*outspend.Vin = txout.Value
			return outspend
		}
	}
	return nil*/
	return nil
}

// DecodeTxHex decode tx hex to msgTx
func DecodeTxHex(txHex string, protocolversion uint32, isWitness bool) *wire.MsgTx {
	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		return nil
	}

	msgtx := new(wire.MsgTx)

	if isWitness {
		msgtx.BtcDecode(bytes.NewReader(txBytes), protocolversion, wire.WitnessEncoding)
	} else {
		msgtx.BtcDecode(bytes.NewReader(txBytes), protocolversion, wire.BaseEncoding)
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
func ConvertVin(vin btcjson.Vin) *electrs.ElectTxin {
	evin := &electrs.ElectTxin{
		Txid:         &vin.Txid,
		Vout:         &vin.Vout,
		Scriptsig:    &vin.ScriptSig.Hex,
		ScriptsigAsm: &vin.ScriptSig.Asm,
		IsCoinbase:   new(bool),
		Sequence:     &vin.Sequence,
	}
	*evin.IsCoinbase = (vin.Coinbase != "")
	return evin
}

// ConvertVout converts btcjson vout to elect vout
func ConvertVout(vout btcjson.Vout) *electrs.ElectTxOut {
	evout := &electrs.ElectTxOut{
		Scriptpubkey:        &vout.ScriptPubKey.Hex,
		ScriptpubkeyAsm:     &vout.ScriptPubKey.Asm,
		ScriptpubkeyType:    &vout.ScriptPubKey.Type,
		ScriptpubkeyAddress: &vout.ScriptPubKey.Addresses[0],
		Value:               new(uint64),
	}
	*evout.Value = uint64(vout.Value * 1e8)
	return evout
}
