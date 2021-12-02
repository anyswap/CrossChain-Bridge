package worker

import (
	"time"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

var (
	adjustGatewayOrderInterval = 60 * time.Second
)

// StartUpdateLatestBlockHeightJob update latest block height job
func StartUpdateLatestBlockHeightJob() {
	go doUpdateLatestBlockHeightJob()
}

func doUpdateLatestBlockHeightJob() {
	for {
		if utils.IsCleanuping() {
			return
		}
		logWorker("adjustGatewayOrder", "adjust gateway api adddress order")
		tools.AdjustGatewayOrder(true)
		tools.AdjustGatewayOrder(false)
		time.Sleep(adjustGatewayOrderInterval)
	}
}
