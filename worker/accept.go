package worker

import (
	"sync"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/dcrm"
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
		for _, info := range signInfo {
			keyID := info.Key
			agreeResult := "AGREE"
			res, err := dcrm.DoAcceptSign(keyID, agreeResult)
			logWorker("accept", "start accept sign job", "result", res, "err", err)
		}
		time.Sleep(waitInterval)
	}
	return nil
}
