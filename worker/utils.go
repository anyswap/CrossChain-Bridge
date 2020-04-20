package worker

import (
	"log"
	"time"
)

var (
	maxRecallLifetime       = int64(10 * 24 * 3600)
	restIntervalInRecallJob = 60 * time.Second

	maxVerifyLifetime       = int64(7 * 24 * 3600)
	restIntervalInVerifyJob = 3 * time.Second

	maxDoSwapLifetime       = int64(7 * 24 * 3600)
	restIntervalInDoSwapJob = 3 * time.Second

	maxStableLifetime       = int64(7 * 24 * 3600)
	restIntervalInStableJob = 3 * time.Second
)

func now() int64 {
	return time.Now().Unix()
}

func logWorker(job, subject string) {
	log.Printf("[%v] %v\n", job, subject)
}

func logWorkerError(job, subject string, err error) {
	log.Printf("[%v] %v, err=%v\n", job, subject, err)
}

func getSepTimeInFind(dist int64) int64 {
	return now() - dist
}

func restInJob(duration time.Duration) {
	time.Sleep(duration)
}
