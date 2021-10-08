package worker

import (
	"sync"
	"time"

	"github.com/anyswap/CrossChain-Bridge/dcrm"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
)

var (
	reportStatStarter sync.Once

	reportInterval = 120 * time.Second
)

// StartReportStatJob report stat job
func StartReportStatJob() {
	if params.ServerAPIAddress == "" {
		return
	}
	reportStatStarter.Do(func() {
		logWorker("reportstat", "start report stat job")
		go reportStat()
	})
}

func reportStat() {
	for {
		updateHeartbeat()

		time.Sleep(reportInterval)
	}
}

func updateHeartbeat() {
	method := "swap.UpdateOracleHeartbeat"
	timestamp := time.Now().Unix()
	args := map[string]interface{}{
		"enode":     dcrm.GetSelfEnode(),
		"timestamp": timestamp,
	}
	var result string
	var err error
	for i := 0; i < 3; i++ {
		err = client.RPCPostWithTimeout(20, &result, params.ServerAPIAddress, method, args)
		if err == nil {
			break
		}
	}
	if err != nil {
		logWorkerWarn("reportstat", "report stat failed", "err", err)
	} else {
		logWorker("reportstat", "report stat success", "timestamp", timestamp)
	}
}
