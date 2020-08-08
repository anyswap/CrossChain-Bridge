package mongodb

import (
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
)

// --------------- blacklist --------------------------------

// AddToBlacklist add to blacklist
func AddToBlacklist(address string) error {
	mb := &MgoBlackAccount{
		Key:       address,
		Timestamp: time.Now().Unix(),
	}
	err := collBlacklist.Insert(mb)
	if err == nil {
		log.Info("mongodb add to black list success", "address", address)
	} else {
		log.Info("mongodb add to black list failed", "address", address, "err", err)
	}
	return mgoError(err)
}

// RemoveFromBlacklist remove from blacklist
func RemoveFromBlacklist(address string) error {
	err := collBlacklist.RemoveId(address)
	if err == nil {
		log.Info("mongodb remove from black list success", "address", address)
	} else {
		log.Info("mongodb remove from black list failed", "address", address, "err", err)
	}
	return mgoError(err)
}

// QueryBlacklist query if is blacked
func QueryBlacklist(address string) (isBlacked bool, err error) {
	var result MgoBlackAccount
	err = collBlacklist.FindId(address).One(&result)
	if err == nil {
		return true, nil
	}
	return false, err
}
