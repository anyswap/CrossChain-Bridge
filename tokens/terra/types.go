package terra

// GetTxResponse gettx response
type GetTxResponse struct {
	Tx         Tx         `json:"tx"`
	TxResponse TxResponse `json:"tx_response"`
}

// Tx tx
type Tx struct {
}

// TxResponse tx response
type TxResponse struct {
}
