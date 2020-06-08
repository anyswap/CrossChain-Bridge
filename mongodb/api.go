package mongodb

import (
	"math/big"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/log"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	collSwapin         *mgo.Collection
	collSwapout        *mgo.Collection
	collSwapinResult   *mgo.Collection
	collSwapoutResult  *mgo.Collection
	collP2shAddress    *mgo.Collection
	collSwapStatistics *mgo.Collection
)

const (
	maxCountOfResults = 5000
)

// do this when reconnect to the database
func deinintCollections() {
	collSwapin = nil
	collSwapout = nil
	collSwapinResult = nil
	collSwapoutResult = nil
	collP2shAddress = nil
	collSwapStatistics = nil
}

func getOrInitCollection(table string, collection **mgo.Collection, indexKey ...string) *mgo.Collection {
	if *collection == nil {
		*collection = database.C(table)
		if len(indexKey) != 0 {
			(*collection).EnsureIndexKey(indexKey...)
		}
	}
	return *collection
}

func getCollection(table string) *mgo.Collection {
	switch table {
	case tbSwapins:
		return getOrInitCollection(table, &collSwapin, "timestamp", "status")
	case tbSwapouts:
		return getOrInitCollection(table, &collSwapout, "timestamp", "status")
	case tbSwapinResults:
		return getOrInitCollection(table, &collSwapinResult, "from", "timestamp")
	case tbSwapoutResults:
		return getOrInitCollection(table, &collSwapoutResult, "from", "timestamp")
	case tbP2shAddresses:
		return getOrInitCollection(table, &collP2shAddress, "p2shaddress")
	case tbSwapStatistics:
		return getOrInitCollection(table, &collSwapStatistics)
	default:
		panic("unknown talbe " + table)
	}
	return nil
}

// --------------- swapin --------------------------------

func AddSwapin(ms *MgoSwap) error {
	return addSwap(tbSwapins, ms)
}

func RecallSwapin(txid string) error {
	swap, err := findSwap(tbSwapins, txid)
	if err != nil {
		return err
	}
	if swap == nil {
		return ErrSwapNotFound
	}
	switch swap.Status {
	case TxNotStable:
		return ErrSwapinTxNotStable
	case TxToBeRecall:
		return ErrSwapinRecallExist
	case TxCanRecall:
		return updateSwapStatus(tbSwapins, txid, TxToBeRecall, time.Now().Unix(), "")
	default:
		return ErrSwapinRecalledOrForbidden
	}
}

func UpdateSwapinStatus(txid string, status SwapStatus, timestamp int64, memo string) error {
	return updateSwapStatus(tbSwapins, txid, status, timestamp, memo)
}

func FindSwapin(txid string) (*MgoSwap, error) {
	return findSwap(tbSwapins, txid)
}

func FindSwapinsWithStatus(status SwapStatus, septime int64) ([]*MgoSwap, error) {
	return findSwapsWithStatus(tbSwapins, status, septime)
}

func GetCountOfSwapinsWithStatus(status SwapStatus) (int, error) {
	return getCountWithStatus(tbSwapins, status)
}

// --------------- swapout --------------------------------

func AddSwapout(ms *MgoSwap) error {
	return addSwap(tbSwapouts, ms)
}

func UpdateSwapoutStatus(txid string, status SwapStatus, timestamp int64, memo string) error {
	return updateSwapStatus(tbSwapouts, txid, status, timestamp, memo)
}

func FindSwapout(txid string) (*MgoSwap, error) {
	return findSwap(tbSwapouts, txid)
}

func FindSwapoutsWithStatus(status SwapStatus, septime int64) ([]*MgoSwap, error) {
	return findSwapsWithStatus(tbSwapouts, status, septime)
}

func GetCountOfSwapoutsWithStatus(status SwapStatus) (int, error) {
	return getCountWithStatus(tbSwapouts, status)
}

// ------------------ swapin / swapout common ------------------------

func addSwap(tbName string, ms *MgoSwap) error {
	err := getCollection(tbName).Insert(ms)
	if err == nil {
		log.Info("mongodb add swap", "txid", ms.TxId, "isSwapin", tbName == tbSwapins)
	} else {
		log.Debug("mongodb add swap", "txid", ms.TxId, "isSwapin", tbName == tbSwapins, "err", err)
	}
	return mgoError(err)
}

func updateSwapStatus(tbName string, txid string, status SwapStatus, timestamp int64, memo string) error {
	updates := bson.M{"status": status, "timestamp": timestamp}
	if memo != "" {
		updates["memo"] = memo
	} else if status == TxNotSwapped {
		updates["memo"] = ""
	}
	err := getCollection(tbName).UpdateId(txid, bson.M{"$set": updates})
	if err == nil {
		log.Info("mongodb update swap status", "txid", txid, "status", status, "isSwapin", tbName == tbSwapins)
	} else {
		log.Debug("mongodb update swap status", "txid", txid, "status", status, "isSwapin", tbName == tbSwapins, "err", err)
	}
	return mgoError(err)
}

func findSwap(tbName string, txid string) (*MgoSwap, error) {
	var result MgoSwap
	err := getCollection(tbName).FindId(txid).One(&result)
	if err != nil {
		return nil, mgoError(err)
	}
	return &result, nil
}

func findSwapsWithStatus(tbName string, status SwapStatus, septime int64) ([]*MgoSwap, error) {
	result := make([]*MgoSwap, 0, 10)
	qtime := bson.M{"timestamp": bson.M{"$gte": septime}}
	qstatus := bson.M{"status": status}
	queries := []bson.M{qtime, qstatus}
	q := getCollection(tbName).Find(bson.M{"$and": queries}).Limit(maxCountOfResults)
	err := q.All(&result)
	if err != nil {
		return nil, mgoError(err)
	}
	return result, nil
}

// --------------- swapin result --------------------------------

func AddSwapinResult(mr *MgoSwapResult) error {
	return addSwapResult(tbSwapinResults, mr)
}

func UpdateSwapinResult(txid string, items *SwapResultUpdateItems) error {
	return updateSwapResult(tbSwapinResults, txid, items)
}

func UpdateSwapinResultStatus(txid string, status SwapStatus, timestamp int64, memo string) error {
	return updateSwapResultStatus(tbSwapinResults, txid, status, timestamp, memo)
}

func FindSwapinResult(txid string) (*MgoSwapResult, error) {
	return findSwapResult(tbSwapinResults, txid)
}

func FindSwapinResultsWithStatus(status SwapStatus, septime int64) ([]*MgoSwapResult, error) {
	return findSwapResultsWithStatus(tbSwapinResults, status, septime)
}

func FindSwapinResults(address string, offset, limit int) ([]*MgoSwapResult, error) {
	return findSwapResults(tbSwapinResults, address, offset, limit)
}

func GetCountOfSwapinResults() (int, error) {
	return getCount(tbSwapinResults)
}

func GetCountOfSwapinResultsWithStatus(status SwapStatus) (int, error) {
	return getCountWithStatus(tbSwapinResults, status)
}

// --------------- swapout result --------------------------------

func AddSwapoutResult(mr *MgoSwapResult) error {
	return addSwapResult(tbSwapoutResults, mr)
}

func UpdateSwapoutResult(txid string, items *SwapResultUpdateItems) error {
	return updateSwapResult(tbSwapoutResults, txid, items)
}

func UpdateSwapoutResultStatus(txid string, status SwapStatus, timestamp int64, memo string) error {
	return updateSwapResultStatus(tbSwapoutResults, txid, status, timestamp, memo)
}

func FindSwapoutResult(txid string) (*MgoSwapResult, error) {
	return findSwapResult(tbSwapoutResults, txid)
}

func FindSwapoutResultsWithStatus(status SwapStatus, septime int64) ([]*MgoSwapResult, error) {
	return findSwapResultsWithStatus(tbSwapoutResults, status, septime)
}

func FindSwapoutResults(address string, offset, limit int) ([]*MgoSwapResult, error) {
	return findSwapResults(tbSwapoutResults, address, offset, limit)
}

func GetCountOfSwapoutResults() (int, error) {
	return getCount(tbSwapoutResults)
}

func GetCountOfSwapoutResultsWithStatus(status SwapStatus) (int, error) {
	return getCountWithStatus(tbSwapoutResults, status)
}

// ------------------ swapin / swapout result common ------------------------

func addSwapResult(tbName string, ms *MgoSwapResult) error {
	err := getCollection(tbName).Insert(ms)
	if err == nil {
		log.Info("mongodb add swap result", "txid", ms.TxId, "swaptype", ms.SwapType, "isSwapin", tbName == tbSwapinResults)
	} else {
		log.Debug("mongodb add swap result", "txid", ms.TxId, "swaptype", ms.SwapType, "isSwapin", tbName == tbSwapinResults, "err", err)
	}
	return mgoError(err)
}

func updateSwapResult(tbName string, txid string, items *SwapResultUpdateItems) error {
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
	if items.Memo != "" {
		updates["memo"] = items.Memo
	} else if items.Status == MatchTxNotStable {
		updates["memo"] = ""
	}
	err := getCollection(tbName).UpdateId(txid, bson.M{"$set": updates})
	if err == nil {
		log.Info("mongodb update swap result", "txid", txid, "updates", updates, "isSwapin", tbName == tbSwapinResults)
	} else {
		log.Debug("mongodb update swap result", "txid", txid, "updates", updates, "isSwapin", tbName == tbSwapinResults, "err", err)
	}
	return mgoError(err)
}

func updateSwapResultStatus(tbName string, txid string, status SwapStatus, timestamp int64, memo string) error {
	updates := bson.M{"status": status, "timestamp": timestamp}
	if memo != "" {
		updates["memo"] = memo
	}
	err := getCollection(tbName).UpdateId(txid, bson.M{"$set": updates})
	isSwapin := tbName == tbSwapinResults
	if err == nil {
		log.Info("mongodb update swap result status", "txid", txid, "status", status, "isSwapin", isSwapin)
	} else {
		log.Debug("mongodb update swap result status", "txid", txid, "status", status, "isSwapin", isSwapin, "err", err)
	}
	if status == MatchTxStable {
		swapResult, err := findSwapResult(tbName, txid)
		if err == nil {
			UpdateSwapStatistics(swapResult.Value, swapResult.SwapValue, isSwapin)
		}
	}
	return mgoError(err)
}

func findSwapResult(tbName string, txid string) (*MgoSwapResult, error) {
	var result MgoSwapResult
	err := getCollection(tbName).FindId(txid).One(&result)
	if err != nil {
		return nil, mgoError(err)
	}
	return &result, nil
}

func findSwapResultsWithStatus(tbName string, status SwapStatus, septime int64) ([]*MgoSwapResult, error) {
	result := make([]*MgoSwapResult, 0, 10)
	qtime := bson.M{"timestamp": bson.M{"$gte": septime}}
	qstatus := bson.M{"status": status}
	queries := []bson.M{qtime, qstatus}
	q := getCollection(tbName).Find(bson.M{"$and": queries}).Limit(maxCountOfResults)
	err := q.All(&result)
	if err != nil {
		return nil, mgoError(err)
	}
	return result, nil
}

func findSwapResults(tbName string, address string, offset, limit int) ([]*MgoSwapResult, error) {
	result := make([]*MgoSwapResult, 0, 20)
	var q *mgo.Query
	if address == "all" {
		q = getCollection(tbName).Find(nil).Skip(offset).Limit(limit)
	} else {
		q = getCollection(tbName).Find(bson.M{"from": address}).Skip(offset).Limit(limit)
	}
	err := q.All(&result)
	if err != nil {
		return nil, mgoError(err)
	}
	return result, nil
}

func getCount(tbName string) (int, error) {
	return getCollection(tbName).Find(nil).Count()
}

func getCountWithStatus(tbName string, status SwapStatus) (int, error) {
	return getCollection(tbName).Find(bson.M{"status": status}).Count()
}

// ------------------ statistics ------------------------

func AddSwapStatistics(ms *MgoSwapStatistics) error {
	return getCollection(tbSwapStatistics).Insert(ms)
}

func UpdateSwapStatistics(value string, swapValue string, isSwapin bool) error {
	curr, err := FindSwapStatistics()
	if err != nil {
		ms := &MgoSwapStatistics{
			Key: KeyOfSwapStatistics,
		}
		err = AddSwapStatistics(ms)
		if err != nil {
			return mgoError(err)
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
		curVal.SetString(curr.TotalSwapinValue, 0)
		curFee.SetString(curr.TotalSwapinFee, 0)
		curVal.Add(curVal, addSwapVal)
		curFee.Add(curFee, addSwapFee)
		updates["swapoutcount"] = curr.StableSwapoutCount + 1
		updates["totalswapoutvalue"] = curVal.String()
		updates["totalswapoutfee"] = curFee.String()
	}
	err = getCollection(tbSwapStatistics).UpdateId(KeyOfSwapStatistics, bson.M{"$set": updates})
	if err == nil {
		log.Info("mongodb update swap statistics", "updates", updates)
	} else {
		log.Debug("mongodb update swap statistics", "updates", updates, "err", err)
	}
	return mgoError(err)
}

func FindSwapStatistics() (*MgoSwapStatistics, error) {
	var result MgoSwapStatistics
	err := getCollection(tbSwapStatistics).FindId(KeyOfSwapStatistics).One(&result)
	return &result, mgoError(err)
}

type SwapStatistics struct {
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

func GetSwapStatistics() (*SwapStatistics, error) {
	stat := &SwapStatistics{}

	if curr, _ := FindSwapStatistics(); curr != nil {
		stat.StableSwapinCount = curr.StableSwapinCount
		stat.TotalSwapinValue = curr.TotalSwapinValue
		stat.TotalSwapinFee = curr.TotalSwapinFee
		stat.StableSwapoutCount = curr.StableSwapoutCount
		stat.TotalSwapoutValue = curr.TotalSwapoutValue
		stat.TotalSwapoutFee = curr.TotalSwapoutFee
	}

	stat.TotalSwapinCount, _ = GetCountOfSwapinResults()
	stat.TotalSwapoutCount, _ = GetCountOfSwapoutResults()
	stat.PendingSwapinCount, _ = GetCountOfSwapinResultsWithStatus(MatchTxEmpty)
	stat.PendingSwapoutCount, _ = GetCountOfSwapoutResultsWithStatus(MatchTxEmpty)

	return stat, nil
}

// ------------------ p2sh address ------------------------

func AddP2shAddress(ma *MgoP2shAddress) error {
	err := getCollection(tbP2shAddresses).Insert(ma)
	if err == nil {
		log.Info("mongodb add p2sh address", "key", ma.Key, "p2shaddress", ma.P2shAddress)
	} else {
		log.Debug("mongodb add p2sh address", "key", ma.Key, "p2shaddress", ma.P2shAddress, "err", err)
	}
	return mgoError(err)
}

func FindP2shAddress(key string) (*MgoP2shAddress, error) {
	var result MgoP2shAddress
	err := getCollection(tbP2shAddresses).FindId(key).One(&result)
	if err != nil {
		return nil, mgoError(err)
	}
	return &result, nil
}

func FindP2shBindAddress(p2shAddress string) (string, error) {
	var result MgoP2shAddress
	err := getCollection(tbP2shAddresses).Find(bson.M{"p2shaddress": p2shAddress}).One(&result)
	if err != nil {
		return "", mgoError(err)
	}
	return result.Key, nil
}
