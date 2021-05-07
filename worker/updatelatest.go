package worker

import (
	"time"

	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

var (
	adjustGatewayOrderInterval = 60 * time.Second
)

// StartUpdateLatestBlockHeightJob update latest block height job
func StartUpdateLatestBlockHeightJob() {
	for {
		logWorker("adjustGatewayOrder", "adjust gateway api adddress order")
		tools.AdjustGatewayOrder(true)
		tools.AdjustGatewayOrder(false)
		time.Sleep(adjustGatewayOrderInterval)
	}
}
