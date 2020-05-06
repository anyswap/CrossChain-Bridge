package worker

import (
	"container/ring"
	"sync"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/dcrm"
	"github.com/fsn-dev/crossChain-Bridge/log"
)

var (
	acceptSignStarter sync.Once

	acceptRing        *ring.Ring
	acceptRingLock    sync.RWMutex
	acceptRingMaxSize = 500

	retryInterval = 3 * time.Second
	waitInterval  = 20 * time.Second
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
			history := getAcceptSignHistory(keyID)
			if history != nil {
				logWorker("accept", "ignore accepted sign", "keyID", keyID, "result", history.result)
				continue
			}
			agreeResult := "AGREE"
			res, err := dcrm.DoAcceptSign(keyID, agreeResult)
			if err != nil {
				logWorkerError("accept", "accept sign job failed", err, "keyID", keyID)
			} else {
				logWorker("accept", "accept sign job success", "keyID", keyID, "result", res)
				addAcceptSignHistory(keyID, agreeResult)
			}
		}
		time.Sleep(waitInterval)
	}
	return nil
}

type acceptSignInfo struct {
	keyID  string
	result string
}

func addAcceptSignHistory(keyID, result string) {
	// Create the new item as its own ring
	item := ring.New(1)
	item.Value = &acceptSignInfo{
		keyID:  keyID,
		result: result,
	}

	acceptRingLock.Lock()
	defer acceptRingLock.Unlock()

	if acceptRing == nil {
		acceptRing = item
	} else {
		if acceptRing.Len() == acceptRingMaxSize {
			// Drop the block out of the ring
			acceptRing = acceptRing.Move(-1)
			acceptRing.Unlink(1)
			acceptRing = acceptRing.Move(1)
		}
		acceptRing.Move(-1).Link(item)
	}
}

func getAcceptSignHistory(keyID string) *acceptSignInfo {
	acceptRingLock.RLock()
	defer acceptRingLock.RUnlock()

	if acceptRing == nil {
		return nil
	}

	r := acceptRing
	for i := 0; i < r.Len(); i++ {
		item := r.Value.(*acceptSignInfo)
		if item.keyID == keyID {
			return item
		}
		r = r.Prev()
	}

	return nil
}
