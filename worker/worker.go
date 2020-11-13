package worker

import (
	"time"

	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/bridge"
)

const interval = 10 * time.Millisecond

// StartWork start swap server work
func StartWork(isServer bool) {
	logWorker("worker", "start server worker")

	client.InitHTTPClient()
	bridge.InitCrossChainBridge(isServer)

	go StartScanJob(isServer)
	time.Sleep(interval)

	go StartUpdateLatestBlockHeightJob()
	time.Sleep(interval)

	if !isServer {
		go StartAcceptSignJob()
		time.Sleep(interval)
		go AddTokenPairDynamically()
		return
	}

	go StartVerifyJob()
	time.Sleep(interval)

	go StartSwapJob()
	time.Sleep(interval)

	go StartStableJob()
	time.Sleep(interval)

	go tokens.SrcBridge.StartAggregateJob()
}
