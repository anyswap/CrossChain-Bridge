package electrs

type Tx struct {
	Txid     *string   `json:"txid"`
	Version  *uint32   `json:"version"`
	Locktime *uint32   `json:"locktime"`
	Size     *uint32   `json:"size"`
	Weight   *uint32   `json:"weight"`
	Fee      *uint64   `json:"fee"`
	Vin      []*TxIn   `json:"vin"`
	Vout     []*TxOut  `json:"vout"`
	Status   *TxStatus `json:"status,omitempty"`
}

type TxIn struct {
	Txid                   *string `json:"txid"`
	Vout                   *uint32 `json:"vout"`
	Scriptsig              *string `json:"scriptsig"`
	Scriptsig_asm          *string `json:"scriptsig_asm"`
	Is_coinbase            *bool   `json:"is_coinbase"`
	Sequence               *uint32 `json:"sequence"`
	Inner_redeemscript_asm *string `json:"inner_redeemscript_asm"`
	Prevout                *TxOut  `json:"prevout"`
}

type TxOut struct {
	Scriptpubkey         *string `json:"scriptpubkey"`
	Scriptpubkey_asm     *string `json:"scriptpubkey_asm"`
	Scriptpubkey_type    *string `json:"scriptpubkey_type"`
	Scriptpubkey_address *string `json:"scriptpubkey_address"`
	Value                *uint64 `json:"value"`
}

type TxStatus struct {
	Confirmed    *bool   `json:"confirmed"`
	Block_height *uint64 `json:"block_height"`
	Block_hash   *string `json:"block_hash"`
	Block_time   *uint64 `json:"block_time"`
}

type Utxo struct {
	Txid   *string   `json:"txid"`
	Vout   *uint32   `json:"vout"`
	Value  *uint64   `json:"value"`
	Status *TxStatus `json:"status"`
}
