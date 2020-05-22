package worker

import (
	"time"

	"github.com/fsn-dev/crossChain-Bridge/tokens/bridge"
)

const interval = 10 * time.Millisecond

func StartServerWork() {
	logWorker("worker", "start server worker")

	bridge.InitCrossChainBridge(true)

	go StartScanJob(true)
	time.Sleep(interval)

	go StartVerifyJob()
	time.Sleep(interval)

	go StartSwapJob()
	time.Sleep(interval)

	go StartStableJob()
	time.Sleep(interval)

	go StartUpdateLatestBlockHeightJob()
	time.Sleep(interval)

	//go StartRecallJob()
}

func StartOracleWork() {
	logWorker("worker", "start oracle worker")

	bridge.InitCrossChainBridge(false)

	go StartScanJob(false)
	time.Sleep(interval)

	go StartAcceptSignJob()
}
