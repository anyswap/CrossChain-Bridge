package mongodb

import (
	"math/big"
	"strings"
	"sync"
	"time"

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
	retryLock sync.Mutex
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
	q := collection.Find(bson.M{"$and": queries}).Sort("timestamp").Limit(maxCountOfResults)
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
	q := collection.Find(bson.M{"$and": queries}).Sort("timestamp").Limit(maxCountOfResults)
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
func FindSwapinResults(address, pairID string, offset, limit int) ([]*MgoSwapResult, error) {
	return findSwapResults(collSwapinResult, address, pairID, offset, limit)
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
func FindSwapoutResults(address, pairID string, offset, limit int) ([]*MgoSwapResult, error) {
	return findSwapResults(collSwapoutResult, address, pairID, offset, limit)
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
		"status":    items.Status,
		"timestamp": items.Timestamp,
	}
	if items.SwapTx != "" {
		updates["swaptx"] = items.SwapTx
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
	if items.SwapNonce != 0 {
		updates["swapnonce"] = items.SwapNonce
	}
	if items.Memo != "" {
		updates["memo"] = items.Memo
	} else if items.Status == MatchTxNotStable {
		updates["memo"] = ""
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
	} else if status == MatchTxEmpty {
		updates["memo"] = ""
		updates["swaptx"] = ""
		updates["swapheight"] = 0
		updates["swaptime"] = 0
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

func findSwapResults(collection *mgo.Collection, address, pairID string, offset, limit int) ([]*MgoSwapResult, error) {
	pairID = strings.ToLower(pairID)
	result := make([]*MgoSwapResult, 0, 20)

	var queries []bson.M

	if pairID != "" && pairID != allPairs {
		queries = append(queries, bson.M{"pairid": pairID})
	}

	if address != "" && address != allAddresses {
		queries = append(queries, bson.M{"from": address})
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
	q = q.Skip(offset).Limit(limit)
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
	curr, err := FindSwapStatistics(pairID)
	if err != nil {
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
	_, err = collSwapStatistics.UpsertId(pairID, bson.M{"$set": updates})
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
