package swapapi

import (
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// SwapStatus type alias
type SwapStatus = mongodb.SwapStatus

// Swap type alias
type Swap = mongodb.MgoSwap

// SwapResult type alias
type SwapResult = mongodb.MgoSwapResult

// SwapStatistics type alias
type SwapStatistics = mongodb.SwapStatistics

// LatestScanInfo type alias
type LatestScanInfo = mongodb.MgoLatestScanInfo

// RegisteredAddress type alias
type RegisteredAddress = mongodb.MgoRegisteredAddress

// ServerInfo server info
type ServerInfo struct {
	Identifier string
	SrcToken   *tokens.TokenConfig
	DestToken  *tokens.TokenConfig
	Version    string
}

// PostResult post result
type PostResult string

// SuccessPostResult success post result
var SuccessPostResult PostResult = "Success"

// SwapInfo swap info
type SwapInfo struct {
	TxID          string     `json:"txid"`
	TxHeight      uint64     `json:"txheight"`
	TxTime        uint64     `json:"txtime"`
	From          string     `json:"from"`
	To            string     `json:"to"`
	Bind          string     `json:"bind"`
	Value         string     `json:"value"`
	SwapTx        string     `json:"swaptx"`
	SwapHeight    uint64     `json:"swapheight"`
	SwapTime      uint64     `json:"swaptime"`
	SwapValue     string     `json:"swapvalue"`
	SwapType      uint32     `json:"swaptype"`
	SwapNonce     uint64     `json:"swapnonce"`
	Status        SwapStatus `json:"status"`
	Timestamp     int64      `json:"timestamp"`
	Memo          string     `json:"memo"`
	Confirmations uint64     `json:"confirmations"`
}
