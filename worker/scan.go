package worker

import (
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

func StartScanJob(isServer bool) error {
	go tokens.SrcBridge.StartSwapinScanJob(isServer)
	go tokens.DstBridge.StartSwapinResultScanJob(isServer)
	go tokens.DstBridge.StartSwapoutScanJob(isServer)
	go tokens.SrcBridge.StartSwapoutResultScanJob(isServer)
	return nil
}
