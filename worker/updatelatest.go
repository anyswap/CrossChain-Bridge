package worker

import (
	"sync"
	"time"

	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	updateLatestBlockHeightStarter  sync.Once
	updateLatestBlockHeightInterval = 5 * time.Second
)

// StartUpdateLatestBlockHeightJob update latest block height job
func StartUpdateLatestBlockHeightJob() {
	updateLatestBlockHeightStarter.Do(func() {
		logWorker("updatelatest", "start update latest block height job")
		for {
			updateSrcLatestBlockHeight()
			updateDstLatestBlockHeight()
			time.Sleep(updateLatestBlockHeightInterval)
		}
	})
}

func updateSrcLatestBlockHeight() {
	srcLatest, err := tokens.SrcBridge.GetLatestBlockNumber()
	if err != nil {
		logWorkerError("updatelatest", "get src latest block number error", err)
		return
	}
	if tokens.SrcLatestBlockHeight != srcLatest {
		tokens.SrcLatestBlockHeight = srcLatest
		logWorker("updatelatest", "update src latest block number", "latest", srcLatest)
	}
}

func updateDstLatestBlockHeight() {
	dstLatest, err := tokens.DstBridge.GetLatestBlockNumber()
	if err != nil {
		logWorkerError("updatelatest", "get dest latest block number error", err)
		return
	}
	if tokens.DstLatestBlockHeight != dstLatest {
		tokens.DstLatestBlockHeight = dstLatest
		logWorker("updatelatest", "update dest latest block number", "latest", dstLatest)
	}
}
