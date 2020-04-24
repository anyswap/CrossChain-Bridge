package swapapi

import (
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/params/server"
)

type SwapStatus = mongodb.SwapStatus

type ServerInfo = server.SwapServerConfig

type SwapStatistics struct {
	TotalValue string
}

type PostResult string

var SuccessPostResult PostResult = "Success"

type SwapInfo struct {
	TxId       string     `json:"txid"`
	TxHeight   uint64     `json:"txheight"`
	TxTime     uint64     `json:"txtime"`
	From       string     `json:"from"`
	To         string     `json:"to"`
	Bind       string     `json:"bind"`
	Value      string     `json:"value"`
	SwapTx     string     `json:"swaptx"`
	SwapHeight uint64     `json:"swapheight"`
	SwapTime   uint64     `json:"swaptime"`
	Status     SwapStatus `json:"status"`
	Timestamp  int64      `json:"timestamp"`
	Memo       string     `json:"memo"`
}
