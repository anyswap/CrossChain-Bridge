package worker

import (
	"time"

	"github.com/fsn-dev/crossChain-Bridge/tokens/bridge"
)

const interval = 10 * time.Millisecond

// StartWork start swap server work
func StartWork(isServer bool) {
	logWorker("worker", "start server worker")

	bridge.InitCrossChainBridge(true)

	go StartUpdateLatestBlockHeightJob()
	time.Sleep(interval)

	go StartScanJob(true)
	time.Sleep(interval)

	go StartAcceptSignJob()
	time.Sleep(interval)

	if !isServer {
		return
	}

	go StartVerifyJob()
	time.Sleep(interval)

	go StartSwapJob()
	time.Sleep(interval)

	go StartStableJob()
	time.Sleep(interval)

	//go StartRecallJob()
}
