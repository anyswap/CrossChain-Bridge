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
		adjustSrcGatewayOrder()
		adjustDstGatewayOrder()
	}
}

func adjustSrcGatewayOrder() {
	// use block number as weight
	var weightedAPIs tools.WeightedStringSlice

	gateway := tokens.SrcBridge.GetGatewayConfig()
	if len(gateway.APIAddress) < 2 {
		return
	}
	for _, apiAddress := range gateway.APIAddress {
		height, _ := tokens.SrcBridge.GetLatestBlockNumberOf(apiAddress)
		weightedAPIs = weightedAPIs.Add(apiAddress, height)
	}

	weightedAPIs = weightedAPIs.Sort()
	gateway.APIAddress = weightedAPIs.GetStrings()
}

func adjustDstGatewayOrder() {
	// use block number as weight
	var weightedAPIs tools.WeightedStringSlice

	gateway := tokens.DstBridge.GetGatewayConfig()
	if len(gateway.APIAddress) < 2 {
		return
	}
	for _, apiAddress := range gateway.APIAddress {
		height, _ := tokens.DstBridge.GetLatestBlockNumberOf(apiAddress)
		weightedAPIs = weightedAPIs.Add(apiAddress, height)
	}

	weightedAPIs = weightedAPIs.Sort()
	gateway.APIAddress = weightedAPIs.GetStrings()
}
