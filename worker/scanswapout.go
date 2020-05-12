package worker

import (
	"sync"
)

var (
	swapoutScanStarter sync.Once
)

func StartSwapoutScanJob(isServer bool) error {
	swapoutScanStarter.Do(func() {
		logWorker("scanswapout", "start scan swapout job")
	})
	return nil
}
