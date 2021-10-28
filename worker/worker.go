package worker

import (
	"time"

	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens/bridge"
)

const interval = 10 * time.Millisecond

// StartWork start swap server work
func StartWork(isServer bool) {
	if isServer {
		logWorker("worker", "start server worker")
	} else {
		logWorker("worker", "start oracle worker")
	}

	client.InitHTTPClient()
	bridge.InitCrossChainBridge(isServer)

	StartScanJob(isServer)
	time.Sleep(interval)

	StartUpdateLatestBlockHeightJob()
	time.Sleep(interval)

	if !isServer {
		StartAcceptSignJob()
		time.Sleep(interval)
		AddTokenPairDynamically()
		time.Sleep(interval)
		StartReportStatJob()
		return
	}

	StartSwapJob()
	time.Sleep(interval)

	StartVerifyJob()
	time.Sleep(interval)

	StartStableJob()
	time.Sleep(interval)

	StartReplaceJob()
	time.Sleep(interval)

	StartPassBigValueJob()
	time.Sleep(interval)

	StartAggregateJob()
	time.Sleep(interval)

	StartCheckFailedSwapJob()
}
