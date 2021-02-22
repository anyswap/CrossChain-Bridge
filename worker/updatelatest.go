package worker

import (
	"sync"
	"time"

	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

var (
	updateLatestBlockHeightStarter  sync.Once
	updateLatestBlockHeightInterval = 5 * time.Second
	adjustGatewayOrderInterval      = 10 * time.Minute
)

// StartUpdateLatestBlockHeightJob update latest block height job
func StartUpdateLatestBlockHeightJob() {
	updateLatestBlockHeightStarter.Do(func() {
		logWorker("updatelatest", "start update latest block height job")
		go adjustGatewayOrder()
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

func adjustGatewayOrder() {
	for {
		time.Sleep(adjustGatewayOrderInterval)
		logWorker("adjustGatewayOrder", "adjust gateway api adddress order")
		adjustGatewayOrderImpl(true)
		adjustGatewayOrderImpl(false)
	}
}

func adjustGatewayOrderImpl(isSrc bool) {
	// use block number as weight
	var weightedAPIs tools.WeightedStringSlice
	var bridge tokens.CrossChainBridge
	if isSrc {
		bridge = tokens.SrcBridge
	} else {
		bridge = tokens.DstBridge
	}
	gateway := bridge.GetGatewayConfig()
	length := len(gateway.APIAddress)
	if length < 2 {
		return
	}
	for i := length; i > 0; i-- { // query in reverse order
		apiAddress := gateway.APIAddress[i-1]
		height, _ := bridge.GetLatestBlockNumberOf(apiAddress)
		weightedAPIs = weightedAPIs.Add(apiAddress, height+uint64((length-i)*10))
	}

	weightedAPIs = weightedAPIs.Sort()
	gateway.APIAddress = weightedAPIs.GetStrings()
	logWorkerTrace("gateway", "adjustGatewayOrder", "isSrc", isSrc, "result", gateway.APIAddress)
}
