package mongodb

import (
	"fmt"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"gopkg.in/mgo.v2"
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

// PassSwapinBigValue pass swapin big value
func PassSwapinBigValue(txid string) error {
	return passBigValue(txid, true)
}

// PassSwapoutBigValue pass swapout big value
func PassSwapoutBigValue(txid string) error {
	return passBigValue(txid, false)
}

func passBigValue(txid string, isSwapin bool) error {
	var coll *mgo.Collection
	if isSwapin {
		coll = collSwapin
	} else {
		coll = collSwapout
	}
	swap, err := findSwap(coll, txid)
	if err != nil {
		return err
	}
	if swap == nil {
		return ErrSwapNotFound
	}
	if swap.Status != TxWithBigValue {
		return fmt.Errorf("swap status is %v, not big value status %v", swap.Status.String(), TxWithBigValue.String())
	}
	return updateSwapStatus(coll, txid, TxNotSwapped, time.Now().Unix(), "")
}
