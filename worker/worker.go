package worker

import (
	"time"

	"github.com/fsn-dev/crossChain-Bridge/log"
)

const interval = 10 * time.Millisecond

func StartWork() {
	log.Println("start worker")
	go StartVerifyJob()
	time.Sleep(interval)

	go StartSwapJob()
	time.Sleep(interval)

	go StartStableJob()
	time.Sleep(interval)

	go StartRecallJob()
}
