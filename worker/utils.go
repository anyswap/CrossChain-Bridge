package worker

import (
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
)

var (
	maxRecallLifetime       = int64(10 * 24 * 3600)
	restIntervalInRecallJob = 3 * time.Second

	maxVerifyLifetime       = int64(7 * 24 * 3600)
	restIntervalInVerifyJob = 3 * time.Second

	maxDoSwapLifetime       = int64(7 * 24 * 3600)
	restIntervalInDoSwapJob = 3 * time.Second

	maxStableLifetime       = int64(7 * 24 * 3600)
	restIntervalInStableJob = 3 * time.Second

	retrySendTxCount    = 3
	retrySendTxInterval = 1 * time.Second
)

func now() int64 {
	return time.Now().Unix()
}

func logWorker(job, subject string, context ...interface{}) {
	log.Info("["+job+"] "+subject, context...)
}

func logWorkerError(job, subject string, err error, context ...interface{}) {
	fields := []interface{}{"err", err}
	fields = append(fields, context...)
	log.Error("["+job+"] "+subject, fields...)
}

func logWorkerTrace(job, subject string, context ...interface{}) {
	log.Trace("["+job+"] "+subject, context...)
}

func getSepTimeInFind(dist int64) int64 {
	return now() - dist
}

func restInJob(duration time.Duration) {
	time.Sleep(duration)
}
