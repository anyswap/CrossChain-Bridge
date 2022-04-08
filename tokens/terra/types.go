package terra

import (
	"encoding/json"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
)

// GetBlockResponse get block response
type GetBlockResponse struct {
	Block *Block `json:"block"`
}

// Block block
type Block struct {
	Header *Header `json:"header"`
}

// Header header
type Header struct {
	ChainID string    `json:"chain_id"`
	Height  int64     `json:"height"`
	Time    time.Time `json:"time"`
}

func (h *Header) UnmarshalJSON(data []byte) error {
	var head struct {
		ChainID string    `json:"chain_id"`
		Height  string    `json:"height"`
		Time    time.Time `json:"time"`
	}
	err := json.Unmarshal(data, &head)
	if err != nil {
		return err
	}
	h.ChainID = head.ChainID
	h.Time = head.Time
	biHeight, err := common.GetBigIntFromStr(head.Height)
	h.Height = biHeight.Int64()
	return err
}

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
