package swapapi

import (
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

type SwapStatus = mongodb.SwapStatus
type Swap = mongodb.MgoSwap
type SwapResult = mongodb.MgoSwapResult
type SwapStatistics = mongodb.SwapStatistics

type ServerInfo struct {
	Identifier string
	SrcToken   *tokens.TokenConfig
	DestToken  *tokens.TokenConfig
	Version    string
}

type PostResult string

var SuccessPostResult PostResult = "Success"

type SwapInfo struct {
	TxId          string     `json:"txid"`
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
	Status        SwapStatus `json:"status"`
	Timestamp     int64      `json:"timestamp"`
	Memo          string     `json:"memo"`
	Confirmations uint64     `json:"confirmations"`
}
