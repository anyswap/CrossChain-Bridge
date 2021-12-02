package worker

import (
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
)

var (
	maxVerifyLifetime       = int64(7 * 24 * 3600)
	restIntervalInVerifyJob = 3 * time.Second

	maxDoSwapLifetime       = int64(7 * 24 * 3600)
	restIntervalInDoSwapJob = 10 * time.Second

	maxStableLifetime       = int64(7 * 24 * 3600)
	restIntervalInStableJob = 10 * time.Second

	maxReplaceSwapLifetime       = int64(7 * 24 * 3600)
	restIntervalInReplaceSwapJob = 60 * time.Second

	maxPassBigValueLifetime     = int64(7 * 24 * 3600)
	restIntervalInPassBigValJob = 300 * time.Second
	passBigValueTimeRequired    = int64(12 * 3600) // seconds

	maxCheckFailedSwapLifetime       = int64(2 * 24 * 3600)
	restIntervalInCheckFailedSwapJob = 60 * time.Second

	retrySignInterval = 3 * time.Second
)

func now() int64 {
	return time.Now().Unix()
}

func logWorker(job, subject string, context ...interface{}) {
	log.Info("["+job+"] "+subject, context...)
}

func logWorkerWarn(job, subject string, context ...interface{}) {
	log.Warn("["+job+"] "+subject, context...)
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
	nowTime := now()
	if nowTime > dist {
		return nowTime - dist
	}
	return 0
}

func restInJob(duration time.Duration) {
	time.Sleep(duration)
}
