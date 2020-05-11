package electrs

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

type ElectTxin struct {
	Txid                   *string     `json:"txid"`
	Vout                   *uint32     `json:"vout"`
	Scriptsig              *string     `json:"scriptsig"`
	Scriptsig_asm          *string     `json:"scriptsig_asm"`
	Is_coinbase            *bool       `json:"is_coinbase"`
	Sequence               *uint32     `json:"sequence"`
	Inner_redeemscript_asm *string     `json:"inner_redeemscript_asm"`
	Prevout                *ElectTxOut `json:"prevout"`
}

type ElectTxOut struct {
	Scriptpubkey         *string `json:"scriptpubkey"`
	Scriptpubkey_asm     *string `json:"scriptpubkey_asm"`
	Scriptpubkey_type    *string `json:"scriptpubkey_type"`
	Scriptpubkey_address *string `json:"scriptpubkey_address"`
	Value                *uint64 `json:"value"`
}

type ElectTxStatus struct {
	Confirmed    *bool   `json:"confirmed"`
	Block_height *uint64 `json:"block_height"`
	Block_hash   *string `json:"block_hash"`
	Block_time   *uint64 `json:"block_time"`
}

type ElectUtxo struct {
	Txid   *string        `json:"txid"`
	Vout   *uint32        `json:"vout"`
	Value  *uint64        `json:"value"`
	Status *ElectTxStatus `json:"status"`
}

type SortableElectUtxoSlice []*ElectUtxo

func (s SortableElectUtxoSlice) Len() int {
	return len(s)
}

func (s SortableElectUtxoSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s SortableElectUtxoSlice) Less(i, j int) bool {
	return *s[i].Value > *s[j].Value
}
