package worker

import (
	"sync"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/dcrm"
	"github.com/fsn-dev/crossChain-Bridge/log"
)

var (
	acceptSignStarter sync.Once

	retryInterval = 3 * time.Second
	waitInterval  = 3 * time.Second
)

func StartAcceptSignJob() error {
	acceptSignStarter.Do(func() {
		logWorker("accept", "start accept sign job")
		acceptSign()
	})
	return nil
}

func acceptSign() error {
	for {
		signInfo, err := dcrm.GetCurNodeSignInfo()
		if err != nil {
			time.Sleep(retryInterval)
			continue
		}
		log.Debug("acceptSign", "count", len(signInfo))
		for _, info := range signInfo {
			keyID := info.Key
			agreeResult := "AGREE"
			res, err := dcrm.DoAcceptSign(keyID, agreeResult)
			if err != nil {
				logWorkerError("accept", "accept sign job failed", err, "keyID", keyID)
			} else {
				logWorker("accept", "accept sign job success", "keyID", keyID, "result", res)
			}
		}
		time.Sleep(waitInterval)
	}
	return nil
}
