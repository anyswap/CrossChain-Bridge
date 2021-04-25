package worker

import (
	"time"

	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
)

var (
	adjustGatewayOrderInterval = 60 * time.Second
)

// StartUpdateLatestBlockHeightJob update latest block height job
func StartUpdateLatestBlockHeightJob() {
	adjustGatewayOrder()
}

func adjustGatewayOrder() {
	for {
		logWorker("adjustGatewayOrder", "adjust gateway api adddress order")
		adjustGatewayOrderImpl(true)
		adjustGatewayOrderImpl(false)
		time.Sleep(adjustGatewayOrderInterval)
	}
}

func adjustGatewayOrderImpl(isSrc bool) {
	// use block number as weight
	var weightedAPIs tools.WeightedStringSlice
	bridge := tokens.GetCrossChainBridge(isSrc)
	gateway := bridge.GetGatewayConfig()
	length := len(gateway.APIAddress)
	if length < 2 {
		return
	}
	maxHeight := uint64(0)
	for i := length; i > 0; i-- { // query in reverse order
		apiAddress := gateway.APIAddress[i-1]
		height, _ := bridge.GetLatestBlockNumberOf(apiAddress)
		weightedAPIs = weightedAPIs.Add(apiAddress, height)
		if height > maxHeight {
			maxHeight = height
		}
	}
	weightedAPIs.Reverse() // reverse as iter in reverse order in the above
	weightedAPIs = weightedAPIs.Sort()
	gateway.APIAddress = weightedAPIs.GetStrings()
	tokens.SetLatestBlockHeight(maxHeight, isSrc)
	if isSrc {
		logWorker("gateway", "adjust source gateways", "result", weightedAPIs)
	} else {
		logWorker("gateway", "adjust dest gateways", "result", weightedAPIs)
	}
}
