package worker

import (
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/tokens/btc"
)

// StartScanJob scan job
func StartScanJob(isServer bool) {
	go tokens.SrcBridge.StartSwapinScanJob(isServer)
	go tokens.DstBridge.StartSwapinResultScanJob(isServer)
	go tokens.DstBridge.StartSwapoutScanJob(isServer)
	go tokens.SrcBridge.StartSwapoutResultScanJob(isServer)

	if btcBridge, ok := tokens.SrcBridge.(*btc.Bridge); ok {
		go btcBridge.StartP2shSwapinScanJob(isServer)
	}
}
