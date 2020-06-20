package worker

import (
	"time"

	"github.com/fsn-dev/crossChain-Bridge/rpc/client"
	"github.com/fsn-dev/crossChain-Bridge/tokens/bridge"
)

const interval = 10 * time.Millisecond

// StartWork start swap server work
func StartWork(isServer bool) {
	logWorker("worker", "start server worker")

	client.InitHTTPClient()
	bridge.InitCrossChainBridge(isServer)

	go StartScanJob(isServer)
	time.Sleep(interval)

	if !isServer {
		go StartAcceptSignJob()
		return
	}

	go StartUpdateLatestBlockHeightJob()
	time.Sleep(interval)

	go StartVerifyJob()
	time.Sleep(interval)

	go StartSwapJob()
	time.Sleep(interval)

	go StartStableJob()
	time.Sleep(interval)

	go StartAggregateJob()
}
