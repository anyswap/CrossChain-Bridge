package worker

import (
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/tokens/btc"
)

func StartScanJob(isServer bool) error {
	go tokens.SrcBridge.StartSwapinScanJob(isServer)
	go tokens.DstBridge.StartSwapinResultScanJob(isServer)
	go tokens.DstBridge.StartSwapoutScanJob(isServer)
	go tokens.SrcBridge.StartSwapoutResultScanJob(isServer)

	if btcBridge, ok := tokens.SrcBridge.(*btc.BtcBridge); ok {
		go btcBridge.StartP2shSwapinScanJob(isServer)
	}

	return nil
}
