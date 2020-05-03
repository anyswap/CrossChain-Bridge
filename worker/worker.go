package worker

import (
	"time"

	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/tokens/bridge"
)

const interval = 10 * time.Millisecond

func StartServerWork() {
	log.Println("start server worker")

	bridge.InitCrossChainBridge()

	go StartVerifyJob()
	time.Sleep(interval)

	go StartSwapJob()
	time.Sleep(interval)

	go StartStableJob()
	time.Sleep(interval)

	go StartRecallJob()
}

func StartOracleWork() {
	log.Println("start oracle worker")

	bridge.InitCrossChainBridge()

	go StartAcceptSignJob()
}
