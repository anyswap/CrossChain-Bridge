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

// LatestScanInfo type alias
type LatestScanInfo = mongodb.MgoLatestScanInfo

// RegisteredAddress type alias
type RegisteredAddress = mongodb.MgoRegisteredAddress

// ServerInfo server info
type ServerInfo struct {
	Identifier          string
	MustRegisterAccount bool
	SrcChain            *tokens.ChainConfig
	DestChain           *tokens.ChainConfig
	PairIDs             []string
	Version             string
}

// PostResult post result
type PostResult string

// SuccessPostResult success post result
var SuccessPostResult PostResult = "Success"

// SwapInfo swap info
type SwapInfo struct {
	PairID        string     `json:"pairid"`
	TxID          string     `json:"txid"`
	TxTo          string     `json:"txto"`
	TxHeight      uint64     `json:"txheight"`
	From          string     `json:"from"`
	To            string     `json:"to"`
	Bind          string     `json:"bind"`
	Value         string     `json:"value"`
	SwapTx        string     `json:"swaptx"`
	SwapHeight    uint64     `json:"swapheight"`
	SwapValue     string     `json:"swapvalue"`
	SwapType      uint32     `json:"swaptype"`
	SwapNonce     uint64     `json:"swapnonce"`
	Status        SwapStatus `json:"status"`
	StatusMsg     string     `json:"statusmsg"`
	InitTime      int64      `json:"inittime"`
	Timestamp     int64      `json:"timestamp"`
	Memo          string     `json:"memo"`
	ReplaceCount  int        `json:"replaceCount"`
	Confirmations uint64     `json:"confirmations"`
}

// SwapNonceInfo swap nonce info
type SwapNonceInfo struct {
	SwapinNonces  map[string]uint64 `json:"swapinNonces"`
	SwapoutNonces map[string]uint64 `json:"swapoutNonces"`
}
