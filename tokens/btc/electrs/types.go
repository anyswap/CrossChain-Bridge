package electrs

import (
	"fmt"
)

// ElectBlock struct
type ElectBlock struct {
	Hash         *string `json:"id"`
	Height       *uint32 `json:"height"`
	Version      *uint32 `json:"version"`
	Timestamp    *uint32 `json:"timestamp"`
	TxCount      *uint32 `json:"tx_count"`
	Size         *uint32 `json:"size"`
	Weight       *uint32 `json:"weight"`
	MerkleRoot   *string `json:"merkle_root"`
	PreviousHash *string `json:"previousblockhash"`
	Nonce        *uint32 `json:"nonce"`
	Bits         *uint32 `json:"bits"`
	Difficulty   *uint64 `json:"difficulty"`
}

// ElectTx struct
type ElectTx struct {
	Txid     *string        `json:"txid"`
	Version  *uint32        `json:"version"`
	Locktime *uint32        `json:"locktime"`
	Size     *uint32        `json:"size"`
	Weight   *uint32        `json:"weight"`
	Fee      *uint64        `json:"fee"`
	Vin      []*ElectTxin   `json:"vin"`
	Vout     []*ElectTxOut  `json:"vout"`
	Status   *ElectTxStatus `json:"status,omitempty"`
}

// ElectTxin struct
type ElectTxin struct {
	Txid                 *string     `json:"txid"`
	Vout                 *uint32     `json:"vout"`
	Scriptsig            *string     `json:"scriptsig"`
	ScriptsigAsm         *string     `json:"scriptsig_asm"`
	IsCoinbase           *bool       `json:"is_coinbase"`
	Sequence             *uint32     `json:"sequence"`
	InnerRedeemscriptAsm *string     `json:"inner_redeemscript_asm"`
	Prevout              *ElectTxOut `json:"prevout"`
}

// ElectTxOut struct
type ElectTxOut struct {
	Scriptpubkey        *string `json:"scriptpubkey"`
	ScriptpubkeyAsm     *string `json:"scriptpubkey_asm"`
	ScriptpubkeyType    *string `json:"scriptpubkey_type"`
	ScriptpubkeyAddress *string `json:"scriptpubkey_address"`
	Value               *uint64 `json:"value"`
}

// ElectOutspend struct
type ElectOutspend struct {
	Spent  *bool          `json:"spent"`
	Txid   *string        `json:"txid"`
	Vin    *uint32        `json:"vin"`
	Status *ElectTxStatus `json:"status,omitempty"`
}

func (outspend *ElectOutspend) String() string {
	if outspend.Txid != nil && outspend.Vin != nil {
		return fmt.Sprintf("spent %v txid %v vin %v status %v", *outspend.Spent, *outspend.Txid, *outspend.Vin, outspend.Status.String())
	}
	return fmt.Sprintf("spent %v status %v", *outspend.Spent, outspend.Status.String())
}

// ElectTxStatus struct
type ElectTxStatus struct {
	Confirmed   *bool   `json:"confirmed"`
	BlockHeight *uint64 `json:"block_height"`
	BlockHash   *string `json:"block_hash"`
	BlockTime   *uint64 `json:"block_time"`
}

func (status *ElectTxStatus) String() string {
	if status == nil {
		return "<nil>"
	}
	if status.BlockHeight != nil && status.BlockHash != nil && status.BlockTime != nil {
		return fmt.Sprintf("confirmed %v blockHeight %v blockHash %v blockTime %v", *status.Confirmed, *status.BlockHeight, *status.BlockHash, *status.BlockTime)
	}
	return fmt.Sprintf("confirmed %v", *status.Confirmed)
}

// ElectUtxo struct
type ElectUtxo struct {
	Txid   *string        `json:"txid"`
	Vout   *uint32        `json:"vout"`
	Value  *uint64        `json:"value"`
	Status *ElectTxStatus `json:"status"`
}

func (utxo *ElectUtxo) String() string {
	return fmt.Sprintf("txid %v vout %v value %v status %v", *utxo.Txid, *utxo.Vout, *utxo.Value, utxo.Status.String())
}

// SortableElectUtxoSlice sortable
type SortableElectUtxoSlice []*ElectUtxo

// Len impl Sortable
func (s SortableElectUtxoSlice) Len() int {
	return len(s)
}

// Swap impl Sortable
func (s SortableElectUtxoSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less impl Sortable
// sort utxos
// 1. confirmed fisrt
// 2. value first
func (s SortableElectUtxoSlice) Less(i, j int) bool {
	confirmed1 := *s[i].Status.Confirmed
	confirmed2 := *s[j].Status.Confirmed
	if confirmed1 != confirmed2 {
		return confirmed1
	}
	return *s[i].Value > *s[j].Value
}
