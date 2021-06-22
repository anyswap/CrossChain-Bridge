package worker

import (
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc"
)

// StartScanJob scan job
func StartScanJob(isServer bool) {
	srcChainCfg := tokens.SrcBridge.GetChainConfig()
	if srcChainCfg.EnableScan {
		go tokens.SrcBridge.StartChainTransactionScanJob()
		if srcChainCfg.EnableScanPool {
			go tokens.SrcBridge.StartPoolTransactionScanJob()
		}
		if btc.BridgeInstance != nil {
			go btc.BridgeInstance.StartSwapHistoryScanJob()
		}
	}

	dstChainCfg := tokens.DstBridge.GetChainConfig()
	if dstChainCfg.EnableScan {
		go tokens.DstBridge.StartChainTransactionScanJob()
		if dstChainCfg.EnableScanPool {
			go tokens.DstBridge.StartPoolTransactionScanJob()
		}
	}
}
