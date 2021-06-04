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

	go StartScanJob(isServer)
	time.Sleep(interval)

	go StartUpdateLatestBlockHeightJob()
	time.Sleep(interval)

	if !isServer {
		StartAcceptSignJob()
		time.Sleep(interval)
		AddTokenPairDynamically()
		return
	}

	StartSwapJob()
	time.Sleep(interval)

	go StartVerifyJob()
	time.Sleep(interval)

	go StartStableJob()
	time.Sleep(interval)

	go StartReplaceJob()
	time.Sleep(interval)

	go StartPassBigValueJob()
	time.Sleep(interval)

	go StartAggregateJob()
}
