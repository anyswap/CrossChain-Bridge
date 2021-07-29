package mongodb

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	maxCountOfResults = 5000
	allPairs          = "all"
	allAddresses      = "all"
)

var (
	retryLock        sync.Mutex
	updateResultLock sync.Mutex
)

// --------------- swapin and swapout uniform --------------------------------

// UpdateSwapStatus update swap status
func UpdateSwapStatus(isSwapin bool, txid, pairID, bind string, status SwapStatus, timestamp int64, memo string) error {
	if isSwapin {
		return updateSwapStatus(collSwapin, txid, pairID, bind, status, timestamp, memo)
	}
	return updateSwapStatus(collSwapout, txid, pairID, bind, status, timestamp, memo)
}

// UpdateSwapResultStatus update swap result status
func UpdateSwapResultStatus(isSwapin bool, txid, pairID, bind string, status SwapStatus, timestamp int64, memo string) error {
	if isSwapin {
		return updateSwapResultStatus(collSwapinResult, txid, pairID, bind, status, timestamp, memo)
	}
	return updateSwapResultStatus(collSwapoutResult, txid, pairID, bind, status, timestamp, memo)
}

// FindSwapResult find swap result
func FindSwapResult(isSwapin bool, txid, pairID, bind string) (*MgoSwapResult, error) {
	if isSwapin {
		return findSwapResult(collSwapinResult, txid, pairID, bind)
	}
	return findSwapResult(collSwapoutResult, txid, pairID, bind)
}

// FindSwap find swap
func FindSwap(isSwapin bool, txid, pairID, bind string) (*MgoSwap, error) {
	if isSwapin {
		return findSwap(collSwapin, txid, pairID, bind)
	}
	return findSwap(collSwapout, txid, pairID, bind)
}

// --------------- swapin --------------------------------

// AddSwapin add swapin
func AddSwapin(ms *MgoSwap) error {
	return addSwap(collSwapin, ms)
}

// UpdateSwapinStatus update swapin status
func UpdateSwapinStatus(txid, pairID, bind string, status SwapStatus, timestamp int64, memo string) error {
	return updateSwapStatus(collSwapin, txid, pairID, bind, status, timestamp, memo)
}

// FindSwapin find swapin
func FindSwapin(txid, pairID, bind string) (*MgoSwap, error) {
	return findSwap(collSwapin, txid, pairID, bind)
}

// FindSwapinsWithStatus find swapin with status in the past septime
func FindSwapinsWithStatus(status SwapStatus, septime int64) ([]*MgoSwap, error) {
	return findSwapsWithStatus(collSwapin, status, septime)
}

// FindSwapinsWithPairIDAndStatus find swapin with pairID and status in the past septime
func FindSwapinsWithPairIDAndStatus(pairID string, status SwapStatus, septime int64) ([]*MgoSwap, error) {
	return findSwapsWithPairIDAndStatus(pairID, collSwapin, status, septime)
}

// GetCountOfSwapinsWithStatus get count of swapins with status
func GetCountOfSwapinsWithStatus(pairID string, status SwapStatus) (int, error) {
	return getCountWithStatus(collSwapin, pairID, status)
}

// --------------- swapout --------------------------------

// AddSwapout add swapout
func AddSwapout(ms *MgoSwap) error {
	return addSwap(collSwapout, ms)
}

// UpdateSwapoutStatus update swapout status
func UpdateSwapoutStatus(txid, pairID, bind string, status SwapStatus, timestamp int64, memo string) error {
	return updateSwapStatus(collSwapout, txid, pairID, bind, status, timestamp, memo)
}

// FindSwapout find swapout
func FindSwapout(txid, pairID, bind string) (*MgoSwap, error) {
	return findSwap(collSwapout, txid, pairID, bind)
}

// FindSwapoutsWithStatus find swapout with status
func FindSwapoutsWithStatus(status SwapStatus, septime int64) ([]*MgoSwap, error) {
	return findSwapsWithStatus(collSwapout, status, septime)
}

// FindSwapoutsWithPairIDAndStatus find swapout with pairID and status in the past septime
func FindSwapoutsWithPairIDAndStatus(pairID string, status SwapStatus, septime int64) ([]*MgoSwap, error) {
	return findSwapsWithPairIDAndStatus(pairID, collSwapout, status, septime)
}

// GetCountOfSwapoutsWithStatus get count of swapout with status
func GetCountOfSwapoutsWithStatus(pairID string, status SwapStatus) (int, error) {
	return getCountWithStatus(collSwapout, pairID, status)
}

// ------------------ swapin / swapout common ------------------------

func addSwap(collection *mgo.Collection, ms *MgoSwap) error {
	if ms.TxID == "" || ms.PairID == "" || ms.Bind == "" {
		log.Error("mongodb add swap with wrong key", "txid", ms.TxID, "pairID", ms.PairID, "bind", ms.Bind, "isSwapin", isSwapin(collection))
		return ErrWrongKey
	}
	ms.PairID = strings.ToLower(ms.PairID)
	ms.Key = GetSwapKey(ms.TxID, ms.PairID, ms.Bind)
	ms.InitTime = common.NowMilli()
	err := collection.Insert(ms)
	if err == nil {
		log.Info("mongodb add swap", "txid", ms.TxID, "pairID", ms.PairID, "bind", ms.Bind, "isSwapin", isSwapin(collection))
	} else {
		log.Debug("mongodb add swap", "txid", ms.TxID, "pairID", ms.PairID, "bind", ms.Bind, "isSwapin", isSwapin(collection), "err", err)
	}
	return mgoError(err)
}

func updateSwapStatus(collection *mgo.Collection, txid, pairID, bind string, status SwapStatus, timestamp int64, memo string) error {
	pairID = strings.ToLower(pairID)
	updates := bson.M{"status": status, "timestamp": timestamp}
	if memo != "" {
		updates["memo"] = memo
	} else if status == TxNotSwapped || status == TxNotStable {
		updates["memo"] = ""
	}
	if status == TxNotStable {
		retryLock.Lock()
		defer retryLock.Unlock()
		swap, _ := findSwap(collection, txid, pairID, bind)
		if !(swap.Status.CanRetry() || swap.Status.CanReverify()) {
			return nil
		}
	}
	err := collection.UpdateId(GetSwapKey(txid, pairID, bind), bson.M{"$set": updates})
	if err == nil {
		printLog := log.Info
		switch status {
		case TxVerifyFailed, TxSwapFailed:
			printLog = log.Warn
		default:
		}
		printLog("mongodb update swap status", "txid", txid, "pairID", pairID, "bind", bind, "status", status, "isSwapin", isSwapin(collection))
	} else {
		log.Debug("mongodb update swap status", "txid", txid, "pairID", pairID, "bind", bind, "status", status, "isSwapin", isSwapin(collection), "err", err)
	}
	return mgoError(err)
}

// GetSwapKey txid + pairID + bind
func GetSwapKey(txid, pairID, bind string) string {
	return strings.ToLower(txid + ":" + pairID + ":" + bind)
}

func findSwap(collection *mgo.Collection, txid, pairID, bind string) (*MgoSwap, error) {
	result := &MgoSwap{}
	err := findSwapOrSwapResult(result, collection, txid, pairID, bind)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func findSwapOrSwapResult(result interface{}, collection *mgo.Collection, txid, pairID, bind string) (err error) {
	if bind != "" {
		err = collection.FindId(GetSwapKey(txid, pairID, bind)).One(result)
	} else {
		qtxid := bson.M{"txid": txid}
		qpair := bson.M{"pairid": strings.ToLower(pairID)}
		queries := []bson.M{qtxid, qpair}
		err = collection.Find(bson.M{"$and": queries}).One(result)
	}
	return mgoError(err)
}

func findSwapsWithStatus(collection *mgo.Collection, status SwapStatus, septime int64) (result []*MgoSwap, err error) {
	err = findSwapsOrSwapResultsWithStatus(&result, collection, status, septime)
	return result, err
}

func findSwapsOrSwapResultsWithStatus(result interface{}, collection *mgo.Collection, status SwapStatus, septime int64) error {
	qtime := bson.M{"timestamp": bson.M{"$gte": septime}}
	qstatus := bson.M{"status": status}
	queries := []bson.M{qtime, qstatus}
	q := collection.Find(bson.M{"$and": queries}).Sort("inittime").Limit(maxCountOfResults)
	return mgoError(q.All(result))
}

func findSwapsWithPairIDAndStatus(pairID string, collection *mgo.Collection, status SwapStatus, septime int64) (result []*MgoSwap, err error) {
	err = findSwapsOrSwapResultsWithPairIDAndStatus(&result, pairID, collection, status, septime)
	return result, err
}

func findSwapsOrSwapResultsWithPairIDAndStatus(result interface{}, pairID string, collection *mgo.Collection, status SwapStatus, septime int64) error {
	pairID = strings.ToLower(pairID)
	qpair := bson.M{"pairid": pairID}
	qtime := bson.M{"timestamp": bson.M{"$gte": septime}}
	qstatus := bson.M{"status": status}
	queries := []bson.M{qpair, qtime, qstatus}
	q := collection.Find(bson.M{"$and": queries}).Sort("inittime").Limit(maxCountOfResults)
	return mgoError(q.All(result))
}

// --------------- swapin result --------------------------------

// AddSwapinResult add swapin result
func AddSwapinResult(mr *MgoSwapResult) error {
	return addSwapResult(collSwapinResult, mr)
}

// UpdateSwapinResult update swapin result
func UpdateSwapinResult(txid, pairID, bind string, items *SwapResultUpdateItems) error {
	return updateSwapResult(collSwapinResult, txid, pairID, bind, items)
}

// UpdateSwapinResultStatus update swapin result status
func UpdateSwapinResultStatus(txid, pairID, bind string, status SwapStatus, timestamp int64, memo string) error {
	return updateSwapResultStatus(collSwapinResult, txid, pairID, bind, status, timestamp, memo)
}

// FindSwapinResult find swapin result
func FindSwapinResult(txid, pairID, bind string) (*MgoSwapResult, error) {
	return findSwapResult(collSwapinResult, txid, pairID, bind)
}

// FindSwapinResultsWithStatus find swapin result with status
func FindSwapinResultsWithStatus(status SwapStatus, septime int64) ([]*MgoSwapResult, error) {
	return findSwapResultsWithStatus(collSwapinResult, status, septime)
}

// FindSwapinResults find swapin history results
func FindSwapinResults(address, pairID string, offset, limit int, status string) ([]*MgoSwapResult, error) {
	return findSwapResults(collSwapinResult, address, pairID, offset, limit, status)
}

// FindSwapResultsToReplace find swap results to replace
func FindSwapResultsToReplace(status SwapStatus, septime int64, isSwapin bool) ([]*MgoSwapResult, error) {
	qstatus := bson.M{"status": status}
	qheight := bson.M{"swapheight": 0}
	qtime := bson.M{"timestamp": bson.M{"$gte": septime}}
	queries := []bson.M{qstatus, qheight, qtime}
	var collection *mgo.Collection
	if isSwapin {
		collection = collSwapinResult
	} else {
		collection = collSwapoutResult
	}
	result := make([]*MgoSwapResult, 0, 20)
	q := collection.Find(bson.M{"$and": queries}).Sort("inittime").Limit(5)
	err := q.All(&result)
	return result, mgoError(err)
}

// GetCountOfSwapinResults get count of swapin results
func GetCountOfSwapinResults(pairID string) (int, error) {
	return getCount(collSwapinResult, pairID)
}

// GetCountOfSwapinResultsWithStatus get count of swapin results with status
func GetCountOfSwapinResultsWithStatus(pairID string, status SwapStatus) (int, error) {
	return getCountWithStatus(collSwapinResult, pairID, status)
}

// --------------- swapout result --------------------------------

// AddSwapoutResult add swapout result
func AddSwapoutResult(mr *MgoSwapResult) error {
	return addSwapResult(collSwapoutResult, mr)
}

// UpdateSwapoutResult update swapout result
func UpdateSwapoutResult(txid, pairID, bind string, items *SwapResultUpdateItems) error {
	return updateSwapResult(collSwapoutResult, txid, pairID, bind, items)
}

// UpdateSwapoutResultStatus update swapout result status
func UpdateSwapoutResultStatus(txid, pairID, bind string, status SwapStatus, timestamp int64, memo string) error {
	return updateSwapResultStatus(collSwapoutResult, txid, pairID, bind, status, timestamp, memo)
}

// FindSwapoutResult find swapout result
func FindSwapoutResult(txid, pairID, bind string) (*MgoSwapResult, error) {
	return findSwapResult(collSwapoutResult, txid, pairID, bind)
}

// FindSwapoutResultsWithStatus find swapout result with status
func FindSwapoutResultsWithStatus(status SwapStatus, septime int64) ([]*MgoSwapResult, error) {
	return findSwapResultsWithStatus(collSwapoutResult, status, septime)
}

// FindSwapoutResults find swapout history results
func FindSwapoutResults(address, pairID string, offset, limit int, status string) ([]*MgoSwapResult, error) {
	return findSwapResults(collSwapoutResult, address, pairID, offset, limit, status)
}

// GetCountOfSwapoutResults get count of swapout results
func GetCountOfSwapoutResults(pairID string) (int, error) {
	return getCount(collSwapoutResult, pairID)
}

// GetCountOfSwapoutResultsWithStatus get count of swapout results with status
func GetCountOfSwapoutResultsWithStatus(pairID string, status SwapStatus) (int, error) {
	return getCountWithStatus(collSwapoutResult, pairID, status)
}

// ------------------ swapin / swapout result common ------------------------

func addSwapResult(collection *mgo.Collection, ms *MgoSwapResult) error {
	if ms.TxID == "" || ms.PairID == "" || ms.Bind == "" {
		log.Error("mongodb add swap result with wrong key", "txid", ms.TxID, "pairID", ms.PairID, "bind", ms.Bind, "swaptype", ms.SwapType, "isSwapin", isSwapin(collection))
		return ErrWrongKey
	}
	ms.PairID = strings.ToLower(ms.PairID)
	ms.Key = GetSwapKey(ms.TxID, ms.PairID, ms.Bind)
	ms.InitTime = common.NowMilli()
	err := collection.Insert(ms)
	if err == nil {
		log.Info("mongodb add swap result", "txid", ms.TxID, "pairID", ms.PairID, "bind", ms.Bind, "swaptype", ms.SwapType, "value", ms.Value, "isSwapin", isSwapin(collection))
	} else {
		log.Debug("mongodb add swap result", "txid", ms.TxID, "pairID", ms.PairID, "bind", ms.Bind, "swaptype", ms.SwapType, "value", ms.Value, "isSwapin", isSwapin(collection), "err", err)
	}
	return mgoError(err)
}

func updateSwapResult(collection *mgo.Collection, txid, pairID, bind string, items *SwapResultUpdateItems) error {
	pairID = strings.ToLower(pairID)
	updates := bson.M{
		"timestamp": items.Timestamp,
	}
	if items.Status != KeepStatus {
		updates["status"] = items.Status
	}
	if items.SwapTx != "" {
		updates["swaptx"] = items.SwapTx
	}
	if len(items.OldSwapTxs) != 0 {
		updates["oldswaptxs"] = items.OldSwapTxs
	}
	if len(items.OldSwapVals) != 0 {
		updates["oldswapvals"] = items.OldSwapVals
	}
	if items.SwapHeight != 0 {
		updates["swapheight"] = items.SwapHeight
	}
	if items.SwapTime != 0 {
		updates["swaptime"] = items.SwapTime
	}
	if items.SwapValue != "" {
		updates["swapvalue"] = items.SwapValue
	}
	if items.SwapType != 0 {
		updates["swaptype"] = items.SwapType
	}
	if items.Memo != "" {
		updates["memo"] = items.Memo
	} else if items.Status == MatchTxNotStable {
		updates["memo"] = ""
	}
	if items.SwapNonce != 0 || items.Status == MatchTxNotStable {
		updateResultLock.Lock()
		defer updateResultLock.Unlock()
		swapRes, err := findSwapResult(collection, txid, pairID, bind)
		if err != nil {
			return err
		}
		if swapRes.SwapNonce != 0 {
			log.Error("forbid update swap nonce again", "old", swapRes.SwapNonce, "new", items.SwapNonce)
			return ErrForbidUpdateNonce
		}
		if swapRes.SwapTx != "" {
			log.Error("forbid update swap tx again", "old", swapRes.SwapTx, "new", items.SwapTx)
			return ErrForbidUpdateSwapTx
		}
		if items.SwapNonce != 0 {
			updates["swapnonce"] = items.SwapNonce
		}
	}
	err := collection.UpdateId(GetSwapKey(txid, pairID, bind), bson.M{"$set": updates})
	if err == nil {
		log.Info("mongodb update swap result", "txid", txid, "pairID", pairID, "bind", bind, "updates", updates, "isSwapin", isSwapin(collection))
	} else {
		log.Debug("mongodb update swap result", "txid", txid, "pairID", pairID, "bind", bind, "updates", updates, "isSwapin", isSwapin(collection), "err", err)
	}
	return mgoError(err)
}

func updateSwapResultStatus(collection *mgo.Collection, txid, pairID, bind string, status SwapStatus, timestamp int64, memo string) error {
	pairID = strings.ToLower(pairID)
	updates := bson.M{"status": status, "timestamp": timestamp}
	if memo != "" {
		updates["memo"] = memo
	}
	if status == Reswapping {
		updates["memo"] = ""
		updates["swaptx"] = ""
		updates["oldswaptxs"] = nil
		updates["swapheight"] = 0
		updates["swaptime"] = 0
		updates["swapnonce"] = 0
	}
	err := collection.UpdateId(GetSwapKey(txid, pairID, bind), bson.M{"$set": updates})
	isSwapin := isSwapin(collection)
	if err == nil {
		log.Info("mongodb update swap result status", "txid", txid, "pairID", pairID, "bind", bind, "status", status, "isSwapin", isSwapin)
	} else {
		log.Debug("mongodb update swap result status", "txid", txid, "pairID", pairID, "bind", bind, "status", status, "isSwapin", isSwapin, "err", err)
	}
	if status == MatchTxStable {
		if swapResult, errq := findSwapResult(collection, txid, pairID, bind); errq == nil {
			_ = updateSwapStatistics(pairID, swapResult.Value, swapResult.SwapValue, isSwapin)
		}
	}
	return mgoError(err)
}

func findSwapResult(collection *mgo.Collection, txid, pairID, bind string) (*MgoSwapResult, error) {
	result := &MgoSwapResult{}
	err := findSwapOrSwapResult(result, collection, txid, pairID, bind)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func findSwapResultsWithStatus(collection *mgo.Collection, status SwapStatus, septime int64) (result []*MgoSwapResult, err error) {
	err = findSwapsOrSwapResultsWithStatus(&result, collection, status, septime)
	return result, err
}

func getStatusesFromStr(status string) []SwapStatus {
	parts := strings.Split(status, ",")
	result := make([]SwapStatus, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		num, err := common.GetUint64FromStr(part)
		if err == nil {
			result = append(result, SwapStatus(num))
		}
	}
	return result
}

func findSwapResults(collection *mgo.Collection, address, pairID string, offset, limit int, status string) ([]*MgoSwapResult, error) {
	pairID = strings.ToLower(pairID)
	result := make([]*MgoSwapResult, 0, 20)

	var queries []bson.M

	if pairID != "" && pairID != allPairs {
		queries = append(queries, bson.M{"pairid": pairID})
	}

	if address != "" && address != allAddresses {
		if common.IsHexAddress(address) {
			address = strings.ToLower(address)
		}
		queries = append(queries, bson.M{"from": address})
	}

	filterStatuses := getStatusesFromStr(status)
	if len(filterStatuses) > 0 {
		if len(filterStatuses) == 1 {
			queries = append(queries, bson.M{"status": filterStatuses[0]})
		} else {
			qstatus := bson.M{"status": bson.M{"$in": filterStatuses}}
			queries = append(queries, qstatus)
		}
	}

	var q *mgo.Query
	switch len(queries) {
	case 0:
		q = collection.Find(nil)
	case 1:
		q = collection.Find(queries[0])
	default:
		q = collection.Find(bson.M{"$and": queries})
	}
	if limit >= 0 {
		q = q.Skip(offset).Limit(limit)
	} else {
		q = q.Sort("-inittime").Skip(offset).Limit(-limit)
	}
	err := q.All(&result)
	if err != nil {
		return nil, mgoError(err)
	}
	return result, nil
}

func getCount(collection *mgo.Collection, pairID string) (int, error) {
	pairID = strings.ToLower(pairID)
	return collection.Find(bson.M{"pairid": pairID}).Count()
}

func getCountWithStatus(collection *mgo.Collection, pairID string, status SwapStatus) (int, error) {
	pairID = strings.ToLower(pairID)
	qpair := bson.M{"pairid": pairID}
	qstatus := bson.M{"status": status}
	queries := []bson.M{qpair, qstatus}
	return collection.Find(bson.M{"$and": queries}).Count()
}

// ------------------ statistics ------------------------

func updateSwapStatistics(pairID, value, swapValue string, isSwapin bool) error {
	pairID = strings.ToLower(pairID)
	curr, _ := FindSwapStatistics(pairID)
	if curr == nil {
		curr = &MgoSwapStatistics{
			Key:                pairID,
			PairID:             pairID,
			StableSwapinCount:  0,
			TotalSwapinValue:   "0",
			TotalSwapinFee:     "0",
			StableSwapoutCount: 0,
			TotalSwapoutValue:  "0",
			TotalSwapoutFee:    "0",
		}
		_ = collSwapStatistics.Insert(curr)
	}

	addVal, _ := new(big.Int).SetString(value, 0)
	addSwapVal, _ := new(big.Int).SetString(swapValue, 0)
	addSwapFee := new(big.Int).Sub(addVal, addSwapVal)

	curVal := big.NewInt(0)
	curFee := big.NewInt(0)

	updates := bson.M{}
	if isSwapin {
		curVal.SetString(curr.TotalSwapinValue, 0)
		curFee.SetString(curr.TotalSwapinFee, 0)
		curVal.Add(curVal, addSwapVal)
		curFee.Add(curFee, addSwapFee)
		updates["swapincount"] = curr.StableSwapinCount + 1
		updates["totalswapinvalue"] = curVal.String()
		updates["totalswapinfee"] = curFee.String()
	} else {
		curVal.SetString(curr.TotalSwapoutValue, 0)
		curFee.SetString(curr.TotalSwapoutFee, 0)
		curVal.Add(curVal, addSwapVal)
		curFee.Add(curFee, addSwapFee)
		updates["swapoutcount"] = curr.StableSwapoutCount + 1
		updates["totalswapoutvalue"] = curVal.String()
		updates["totalswapoutfee"] = curFee.String()
	}
	err := collSwapStatistics.UpdateId(pairID, bson.M{"$set": updates})
	if err == nil {
		log.Info("mongodb update swap statistics", "updates", updates)
	} else {
		log.Debug("mongodb update swap statistics", "updates", updates, "err", err)
	}
	return mgoError(err)
}

// FindSwapStatistics find swap statistics
func FindSwapStatistics(pairID string) (*MgoSwapStatistics, error) {
	pairID = strings.ToLower(pairID)
	var result MgoSwapStatistics
	err := collSwapStatistics.FindId(pairID).One(&result)
	return &result, mgoError(err)
}

// SwapStatistics rpc return struct
type SwapStatistics struct {
	PairID              string
	TotalSwapinCount    int
	TotalSwapoutCount   int
	PendingSwapinCount  int
	PendingSwapoutCount int
	StableSwapinCount   int
	TotalSwapinValue    string
	TotalSwapinFee      string
	StableSwapoutCount  int
	TotalSwapoutValue   string
	TotalSwapoutFee     string
}

// GetSwapStatistics get swap statistics
func GetSwapStatistics(pairID string) (*SwapStatistics, error) {
	pairID = strings.ToLower(pairID)
	stat := &SwapStatistics{PairID: pairID}

	if curr, _ := FindSwapStatistics(pairID); curr != nil {
		stat.StableSwapinCount = curr.StableSwapinCount
		stat.TotalSwapinValue = curr.TotalSwapinValue
		stat.TotalSwapinFee = curr.TotalSwapinFee
		stat.StableSwapoutCount = curr.StableSwapoutCount
		stat.TotalSwapoutValue = curr.TotalSwapoutValue
		stat.TotalSwapoutFee = curr.TotalSwapoutFee
	}

	stat.TotalSwapinCount, _ = GetCountOfSwapinResults(pairID)
	stat.TotalSwapoutCount, _ = GetCountOfSwapoutResults(pairID)
	stat.PendingSwapinCount, _ = GetCountOfSwapinResultsWithStatus(pairID, MatchTxEmpty)
	stat.PendingSwapoutCount, _ = GetCountOfSwapoutResultsWithStatus(pairID, MatchTxEmpty)

	return stat, nil
}

// ------------------ p2sh address ------------------------

// AddP2shAddress add p2sh address
func AddP2shAddress(ma *MgoP2shAddress) error {
	err := collP2shAddress.Insert(ma)
	if err == nil {
		log.Info("mongodb add p2sh address", "key", ma.Key, "p2shaddress", ma.P2shAddress)
	} else {
		log.Debug("mongodb add p2sh address", "key", ma.Key, "p2shaddress", ma.P2shAddress, "err", err)
	}
	return mgoError(err)
}

// FindP2shAddress find p2sh addrss through bind address
func FindP2shAddress(key string) (*MgoP2shAddress, error) {
	var result MgoP2shAddress
	err := collP2shAddress.FindId(key).One(&result)
	if err != nil {
		return nil, mgoError(err)
	}
	return &result, nil
}

// FindP2shBindAddress find bind address through p2sh address
func FindP2shBindAddress(p2shAddress string) (string, error) {
	var result MgoP2shAddress
	err := collP2shAddress.Find(bson.M{"p2shaddress": p2shAddress}).One(&result)
	if err != nil {
		return "", mgoError(err)
	}
	return result.Key, nil
}

// FindP2shAddresses find p2sh address
func FindP2shAddresses(offset, limit int) ([]*MgoP2shAddress, error) {
	result := make([]*MgoP2shAddress, 0, limit)
	q := collP2shAddress.Find(nil).Skip(offset).Limit(limit)
	err := q.All(&result)
	if err != nil {
		return nil, mgoError(err)
	}
	return result, nil
}

// ------------------ latest scan info ------------------------

// UpdateLatestScanInfo update latest scan info
func UpdateLatestScanInfo(isSrc bool, blockHeight uint64) error {
	oldInfo, _ := FindLatestScanInfo(isSrc)
	if oldInfo != nil {
		oldHeight := oldInfo.BlockHeight
		if blockHeight <= oldHeight {
			return nil
		}
	}
	var key string
	if isSrc {
		key = keyOfSrcLatestScanInfo
	} else {
		key = keyOfDstLatestScanInfo
	}
	updates := bson.M{
		"blockheight": blockHeight,
		"timestamp":   time.Now().Unix(),
	}
	err := collLatestScanInfo.UpdateId(key, bson.M{"$set": updates})
	if err == nil {
		log.Info("mongodb update lastest scan info", "isSrc", isSrc, "updates", updates)
	} else {
		log.Debug("mongodb update latest scan info", "isSrc", isSrc, "updates", updates, "err", err)
	}
	return mgoError(err)
}

// FindLatestScanInfo find latest scan info
func FindLatestScanInfo(isSrc bool) (*MgoLatestScanInfo, error) {
	var result MgoLatestScanInfo
	var key string
	if isSrc {
		key = keyOfSrcLatestScanInfo
	} else {
		key = keyOfDstLatestScanInfo
	}
	err := collLatestScanInfo.FindId(key).One(&result)
	return &result, mgoError(err)
}

// ------------------------ register address ------------------------------

// AddRegisteredAddress add register address
func AddRegisteredAddress(address string) error {
	ma := &MgoRegisteredAddress{
		Key:       address,
		Timestamp: time.Now().Unix(),
	}
	err := collRegisteredAddress.Insert(ma)
	if err == nil {
		log.Info("mongodb add register address", "key", ma.Key)
	} else {
		log.Debug("mongodb add register address", "key", ma.Key, "err", err)
	}
	return mgoError(err)
}

// FindRegisteredAddress find register address
func FindRegisteredAddress(key string) (*MgoRegisteredAddress, error) {
	var result MgoRegisteredAddress
	err := collRegisteredAddress.FindId(key).One(&result)
	if err != nil {
		return nil, mgoError(err)
	}
	return &result, nil
}

// AddSwapAgreement add swapin agreement
func AddSwapAgreement(ptype, pkey, pvalue string) error {
	mp := &MgoSwapAgreement{
		Type:      ptype,
		Key:       pkey,
		Value:     pvalue,
		Cancelled: false,
	}
	err := collSwapAgreement.Insert(mp)
	if err == nil {
		log.Info("mongodb add swapin agreement", "key", mp.Key)
	} else {
		log.Debug("mongodb add swapin agreement", "key", mp.Key, "err", err)
		// ---------------------- latest swap nonces -----------------------------
	}
	return err
}

func getSwapNonceKey(address string, isSwapin bool) string {
	return strings.ToLower(fmt.Sprintf("%v:%v", address, isSwapin))
}

// UpdateLatestSwapinNonce update
func UpdateLatestSwapinNonce(address string, nonce uint64) error {
	return UpdateLatestSwapNonce(address, true, nonce)
}

// UpdateLatestSwapoutNonce update
func UpdateLatestSwapoutNonce(address string, nonce uint64) error {
	return UpdateLatestSwapNonce(address, false, nonce)
}

// UpdateLatestSwapNonce update
func UpdateLatestSwapNonce(address string, isSwapin bool, nonce uint64) (err error) {
	key := getSwapNonceKey(address, isSwapin)
	oldItem, _ := FindLatestSwapNonce(key)
	if oldItem != nil && oldItem.SwapNonce >= nonce {
		return nil // only increase
	}
	if oldItem == nil {
		ma := &MgoLatestSwapNonce{
			Key:       key,
			Address:   strings.ToLower(address),
			IsSwapin:  isSwapin,
			SwapNonce: nonce,
			Timestamp: time.Now().Unix(),
		}
		err = collLatestSwapNonces.Insert(ma)
	} else {
		updates := bson.M{
			"address":   strings.ToLower(address),
			"isswapin":  isSwapin,
			"swapnonce": nonce,
			"timestamp": time.Now().Unix(),
		}
		err = collLatestSwapNonces.UpdateId(key, bson.M{"$set": updates})
	}
	if err == nil {
		log.Info("mongodb update swap nonce success", "address", address, "nonce", nonce, "isSwapin", isSwapin)
	} else {
		log.Warn("mongodb update swap nonce failed", "address", address, "nonce", nonce, "isSwapin", isSwapin, "err", err)
	}
	return mgoError(err)
}

// CancelSwapAgreement cancels a swapin agreement
func CancelSwapAgreement(key string) error {
	err := collSwapAgreement.UpdateId(key, bson.D{{"cancelled", true}})
	if err != nil {
		log.Debug("mongodb cancelled swapin agreement", "key", key, "err", err)
		return mgoError(err)
	}
	log.Debug("mongodb cancelled swapin agreement", "key", key)
	return nil
}

// UpdateSwapAgreement update swapin agreement
func UpdateSwapAgreement(ptype, pkey, pvalue string) error {
	mp := &MgoSwapAgreement{
		Type:      ptype,
		Key:       pkey,
		Value:     pvalue,
		Cancelled: false,
	}
	err := collSwapAgreement.UpdateId(pkey, bson.M{"$set": mp})
	if err == nil {
		log.Info("mongodb update swapin agreement", "key", mp.Key)
	} else {
		log.Debug("mongodb update swapin agreement", "key", mp.Key, "err", err)
	}
	return err
}

// FindLatestSwapNonce find
func FindLatestSwapNonce(key string) (*MgoLatestSwapNonce, error) {
	var result MgoLatestSwapNonce
	err := collLatestSwapNonces.FindId(key).One(&result)
	if err != nil {
		return nil, mgoError(err)
	}
	return &result, nil
}

// LoadAllSwapNonces load
func LoadAllSwapNonces() (swapinNonces, swapoutNonces map[string]uint64) {
	swapinNonces = make(map[string]uint64)
	swapoutNonces = make(map[string]uint64)
	var result MgoLatestSwapNonce
	iter := collLatestSwapNonces.Find(nil).Iter()
	for iter.Next(&result) {
		address := result.Address
		if address == "" {
			continue
		}
		if result.IsSwapin {
			swapinNonces[address] = result.SwapNonce
		} else {
			swapoutNonces[address] = result.SwapNonce
		}
	}
	log.Info("load swap nonces finished", "swapinNonces", swapinNonces, "swapoutNonces", swapoutNonces)
	return swapinNonces, swapoutNonces
}

// ---------------------- swap hisitory -----------------------------

// AddSwapHistory add
func AddSwapHistory(isSwapin bool, txid, bind, swaptx string) error {
	item := &MgoSwapHistory{
		Key:      bson.NewObjectId(),
		IsSwapin: isSwapin,
		TxID:     txid,
		Bind:     bind,
		SwapTx:   swaptx,
	}
	err := collSwapHistory.Insert(item)
	if err == nil {
		log.Info("mongodb add swap history success", "txid", txid, "bind", bind, "isSwapin", isSwapin)
	} else {
		log.Debug("mongodb add swap history failed", "txid", txid, "bind", bind, "isSwapin", isSwapin, "err", err)
	}
	return mgoError(err)
}

// FindSwapAgreement finds swapin agreement
func FindSwapAgreement(key string) (*MgoSwapAgreement, error) {
	var result MgoSwapAgreement
	err := collSwapAgreement.FindId(key).One(&result)
	if err != nil {
		return nil, mgoError(err)
	}
	if result.Cancelled {
		return nil, mgoError(errors.New("Swapin agreement is cancelled"))
	}
	return &result, nil
}

// FindLatestSolanaTxid finds latest solana txid
func FindLatestSolanaTxid(address string) string {
	key := strings.ToLower(address)
	var result MgoSolanaScannedTx
	err := collSolanaScannedTx.FindId(key).One(&result)
	if err != nil {
		return ""
	}
	return result.Txid
}

func UpdateLatestSolanaTxid(addess, txid string) error {
	key := strings.ToLower(addess)
	_, err := collSolanaScannedTx.UpsertId(key, MgoSolanaScannedTx{key, txid})
	return err
}

// GetSwapHistory get
func GetSwapHistory(isSwapin bool, txid, bind string) ([]*MgoSwapHistory, error) {
	qtxid := bson.M{"txid": txid}
	qbind := bson.M{"bind": bind}
	qisswapin := bson.M{"isswapin": isSwapin}
	queries := []bson.M{qtxid, qbind, qisswapin}
	result := make([]*MgoSwapHistory, 0, 20)
	err := collSwapHistory.Find(bson.M{"$and": queries}).All(&result)
	return result, mgoError(err)
}

// ---------------------- used rvalue -----------------------------

// AddUsedRValue add used r, if error mean already exist
func AddUsedRValue(pubkey, r string) error {
	key := strings.ToLower(r + ":" + pubkey)
	mr := &MgoUsedRValue{
		Key:       key,
		Timestamp: common.NowMilli(),
	}
	err := collUsedRValue.Insert(mr)
	switch {
	case err == nil:
		log.Info("mongodb add used r success", "pubkey", pubkey, "r", r)
		return nil
	case mgo.IsDup(err):
		log.Warn("mongodb add used r failed", "pubkey", pubkey, "r", r, "err", err)
		return ErrItemIsDup
	default:
		old := &MgoUsedRValue{}
		if collUsedRValue.FindId(key).One(old) == nil {
			log.Warn("mongodb add used r failed", "pubkey", pubkey, "r", r, "err", ErrItemIsDup)
			return ErrItemIsDup
		}

		err = collUsedRValue.Insert(mr) // retry once
		if err != nil {
			log.Warn("mongodb add used r failed in retry", "pubkey", pubkey, "r", r, "err", err)
		}
		return mgoError(err)
	}
}
