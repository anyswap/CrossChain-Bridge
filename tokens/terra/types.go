package terra

import (
	"time"
)

// GetBlockResult get block result
type GetBlockResult struct {
	Block *Block `json:"block"`
}

// Block block
type Block struct {
	Header *Header `json:"header"`
}

// Header header
type Header struct {
	ChainID string    `json:"chain_id"`
	Height  string    `json:"height"`
	Time    time.Time `json:"time"`
}

// GetTxResult gettx result
type GetTxResult struct {
	Tx         Tx         `json:"tx"`
	TxResponse TxResponse `json:"tx_response"`
}

// Tx tx
type Tx struct {
	Body TxBody `protobuf:"bytes,1,opt,name=body,proto3" json:"body,omitempty"`
}

// TxResponse tx response
type TxResponse struct {
	// The block height
	Height string `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
	// The transaction hash.
	TxHash string `protobuf:"bytes,2,opt,name=txhash,proto3" json:"txhash,omitempty"`
	// Response code.
	Code uint32 `protobuf:"varint,4,opt,name=code,proto3" json:"code,omitempty"`
	// The output of the application's logger (typed). May be non-deterministic.
	Logs ABCIMessageLogs `protobuf:"bytes,7,rep,name=logs,proto3,castrepeated=ABCIMessageLogs" json:"logs"`
	// the timestamps of the valid votes in the block.LastCommit. For height == 1,
	// it's genesis time.
	// Timestamp string `protobuf:"bytes,12,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
}


type TxBody struct {
	// memo is any arbitrary note/comment to be added to the transaction.
	// WARNING: in clients, any publicly exposed text should not be called memo,
	// but should be called `note` instead (see https://github.com/cosmos/cosmos-sdk/issues/9122).
	Memo string `protobuf:"bytes,2,opt,name=memo,proto3" json:"memo,omitempty"`
}

type ABCIMessageLogs []ABCIMessageLog

type ABCIMessageLog struct {
	Events StringEvents `protobuf:"bytes,3,rep,name=events,proto3,castrepeated=StringEvents" json:"events"`
}

type StringEvents []StringEvent

type StringEvent struct {
	Type       string      `protobuf:"bytes,1,opt,name=type,proto3" json:"type,omitempty"`
	Attributes []Attribute `protobuf:"bytes,2,rep,name=attributes,proto3" json:"attributes"`
}

type Attribute struct {
	Key   string `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value string `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

// BroadcastTxRequest broadcat tx request
type BroadcastTxRequest struct {
	TxBytes string `json:"tx_bytes"`
	Mode    string `json:"mode"`
}

// BroadcastTxResult broadcast tx result
type BroadcastTxResult struct {
	TxResponse BroadcastTxResponse `json:"tx_response"`
}

// BroadcastTxResponse broadcast tx response
type BroadcastTxResponse struct {
	TxHash string `json:"txhash"`
	Code   int64  `json:"code"`
}

// SimulateRequest simulate request
type SimulateRequest struct {
}

// SimulateResponse simulate responce
type SimulateResponse struct {
}

// GetBaseAccountResult get base account result
type GetBaseAccountResult struct {
	Account *BaseAccount `json:"account"`
}

// BaseAccount base account
type BaseAccount struct {
	TypeURL       string  `json:"@type"`
	Address       string  `json:"address"`
	Pubkey        *Pubkey `json:"pub_key,omitempty"`
	AccountNumber string  `json:"account_number"`
	Sequence      string  `json:"sequence"`
	Value         string  `json:"value,omitempty"`
}

// Pubkey public key
type Pubkey struct {
	TypeURL string `json:"@type"`
	Key     string `json:"key"`
}
