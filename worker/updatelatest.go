package worker

import (
	"sync"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

var (
	updateLatestBlockHeightStarter  sync.Once
	updateLatestBlockHeightInterval = 5 * time.Second
)

// StartUpdateLatestBlockHeightJob update latest block height job
func StartUpdateLatestBlockHeightJob() error {
	updateLatestBlockHeightStarter.Do(func() {
		logWorker("updatelatest", "start update latest block height job")
		for {
			updateSrcLatestBlockHeight()
			updateDstLatestBlockHeight()
			time.Sleep(updateLatestBlockHeightInterval)
		}
	})
	return nil
}

func updateSrcLatestBlockHeight() error {
	srcLatest, err := tokens.SrcBridge.GetLatestBlockNumber()
	if err != nil {
		logWorkerError("updatelatest", "get src latest block number error", err)
		return err
	}
	if tokens.SrcLatestBlockHeight != srcLatest {
		tokens.SrcLatestBlockHeight = srcLatest
		logWorker("updatelatest", "update src latest block number", "latest", srcLatest)
	}
	return nil
}

func updateDstLatestBlockHeight() error {
	dstLatest, err := tokens.DstBridge.GetLatestBlockNumber()
	if err != nil {
		logWorkerError("updatelatest", "get dest latest block number error", err)
		return err
	}
	if tokens.DstLatestBlockHeight != dstLatest {
		tokens.DstLatestBlockHeight = dstLatest
		logWorker("updatelatest", "update dest latest block number", "latest", dstLatest)
	}
	return nil
}
