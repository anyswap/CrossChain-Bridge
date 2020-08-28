package worker

import (
	"container/ring"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	swapRing        *ring.Ring
	swapRingLock    sync.RWMutex
	swapRingMaxSize = 1000

	swapChanSize       = 10
	swapinTaskChanMap  map[string]chan *tokens.BuildTxArgs
	swapoutTaskChanMap map[string]chan *tokens.BuildTxArgs
)

// StartSwapJob swap job
func StartSwapJob() {
	for pairID, pairCfg := range tokens.GetTokenPairsConfig() {
		swapinDcrmAddr := strings.ToLower(pairCfg.SrcToken.DcrmAddress)
		if _, exist := swapinTaskChanMap[swapinDcrmAddr]; !exist {
			swapinTaskChanMap[swapinDcrmAddr] = make(chan *tokens.BuildTxArgs, swapChanSize)
			go processSwapTask(swapinTaskChanMap[swapinDcrmAddr])
		}
		swapoutDcrmAddr := strings.ToLower(pairCfg.DestToken.DcrmAddress)
		if _, exist := swapoutTaskChanMap[swapoutDcrmAddr]; !exist {
			swapoutTaskChanMap[swapoutDcrmAddr] = make(chan *tokens.BuildTxArgs, swapChanSize)
			go processSwapTask(swapoutTaskChanMap[swapoutDcrmAddr])
		}

		go startSwapinSwapJob(pairID)
		go startSwapoutSwapJob(pairID)
	}
}

func startSwapinSwapJob(pairID string) {
	logWorker("swap", "start swapin swap job")
	for {
		res, err := findSwapinsToSwap(pairID)
		if err != nil {
			logWorkerError("swapin", "find swapins error", err)
		}
		if len(res) > 0 {
			logWorker("swapin", "find swapins to swap", "count", len(res))
		}
		for _, swap := range res {
			err = processSwapinSwap(swap)
			if err != nil {
				logWorkerError("swapin", "process swapin swap error", err, "pairID", swap.PairID, "txid", swap.TxID)
			}
		}
		restInJob(restIntervalInDoSwapJob)
	}
}

func startSwapoutSwapJob(pairID string) {
	logWorker("swapout", "start swapout swap job")
	for {
		res, err := findSwapoutsToSwap(pairID)
		if err != nil {
			logWorkerError("swapout", "find swapouts error", err)
		}
		if len(res) > 0 {
			logWorker("swapout", "find swapouts to swap", "count", len(res))
		}
		for _, swap := range res {
			err = processSwapoutSwap(swap)
			if err != nil {
				logWorkerError("swapout", "process swapout swap error", err, "pairID", swap.PairID, "txid", swap.TxID)
			}
		}
		restInJob(restIntervalInDoSwapJob)
	}
}

func findSwapinsToSwap(pairID string) ([]*mongodb.MgoSwap, error) {
	status := mongodb.TxNotSwapped
	septime := getSepTimeInFind(maxDoSwapLifetime)
	return mongodb.FindSwapinsWithPairIDAndStatus(pairID, status, septime)
}

func findSwapoutsToSwap(pairID string) ([]*mongodb.MgoSwap, error) {
	status := mongodb.TxNotSwapped
	septime := getSepTimeInFind(maxDoSwapLifetime)
	return mongodb.FindSwapoutsWithPairIDAndStatus(pairID, status, septime)
}

func isSwapInBlacklist(swap *mongodb.MgoSwapResult) (isBlacked bool, err error) {
	isBlacked, err = mongodb.QueryBlacklist(swap.From, swap.PairID)
	if err != nil {
		return isBlacked, err
	}
	if !isBlacked && swap.Bind != swap.From {
		isBlacked, err = mongodb.QueryBlacklist(swap.Bind, swap.PairID)
		if err != nil {
			return isBlacked, err
		}
	}
	return isBlacked, nil
}

func processSwapinSwap(swap *mongodb.MgoSwap) (err error) {
	return processSwap(swap, true)
}

func processSwapoutSwap(swap *mongodb.MgoSwap) (err error) {
	return processSwap(swap, false)
}

func processSwap(swap *mongodb.MgoSwap, isSwapin bool) (err error) {
	pairID := swap.PairID
	txid := swap.TxID
	logWorker("swap", "start process swap", "pairID", pairID, "txid", txid, "status", swap.Status, "isSwapin", isSwapin)

	res, err := mongodb.FindSwapResult(isSwapin, txid)
	if err != nil {
		return err
	}
	tokenCfg := tokens.GetTokenConfig(pairID, isSwapin)
	if tokenCfg == nil {
		logWorkerTrace("swap", "swap is not configed", "pairID", pairID, "isSwapin", isSwapin)
		return nil
	}
	if tokenCfg.DisableSwap {
		logWorkerTrace("swap", "swap is disabled", "pairID", pairID, "isSwapin", isSwapin)
		return nil
	}
	isBlacked, err := isSwapInBlacklist(res)
	if err != nil {
		return err
	}
	if isBlacked {
		logWorkerTrace("swap", "address is in blacklist", "txid", txid, "isSwapin", isSwapin)
		err = tokens.ErrAddressIsInBlacklist
		_ = mongodb.UpdateSwapStatus(isSwapin, txid, mongodb.SwapInBlacklist, now(), err.Error())
		return nil
	}
	if res.SwapTx != "" {
		err = processNonEmptySwapResult(res, isSwapin)
		if err != nil {
			return err
		}
	}

	err = processHistory(pairID, txid, isSwapin)
	if err != nil {
		return err
	}

	value, err := common.GetBigIntFromStr(res.Value)
	if err != nil {
		return fmt.Errorf("wrong value %v", res.Value)
	}

	swapType := getSwapType(isSwapin)
	args := &tokens.BuildTxArgs{
		SwapInfo: tokens.SwapInfo{
			PairID:   pairID,
			SwapID:   txid,
			SwapType: swapType,
		},
		From:  tokenCfg.DcrmAddress,
		To:    res.Bind,
		Value: value,
	}
	if isSwapin {
		args.TxType = tokens.SwapTxType(swap.TxType)
		args.Bind = swap.Bind
	}

	return dispatchSwapTask(args)
}

func getSwapType(isSwapin bool) tokens.SwapType {
	if isSwapin {
		return tokens.SwapinType
	}
	return tokens.SwapoutType
}

func processNonEmptySwapResult(res *mongodb.MgoSwapResult, isSwapin bool) error {
	txid := res.TxID
	_ = mongodb.UpdateSwapStatus(isSwapin, txid, mongodb.TxProcessed, now(), "")
	if res.Status != mongodb.MatchTxEmpty {
		return fmt.Errorf("%v already swapped to %v with status %v", txid, res.SwapTx, res.Status)
	}
	resBridge := tokens.GetCrossChainBridge(!isSwapin)
	if _, err := resBridge.GetTransaction(res.SwapTx); err == nil {
		return fmt.Errorf("[warn] %v already swapped to %v but with status %v", txid, res.SwapTx, res.Status)
	}
	return nil
}

func processHistory(pairID, txid string, isSwapin bool) error {
	history := getSwapHistory(txid, isSwapin)
	if history == nil {
		return nil
	}
	resBridge := tokens.GetCrossChainBridge(!isSwapin)
	swapType := getSwapType(isSwapin)
	if _, err := resBridge.GetTransaction(history.matchTx); err == nil {
		matchTx := &MatchTx{
			SwapTx:    history.matchTx,
			SwapValue: tokens.CalcSwappedValue(pairID, history.value, isSwapin).String(),
			SwapType:  swapType,
			SwapNonce: history.nonce,
		}
		_ = updateSwapResult(txid, matchTx)
		logWorker("swap", "ignore swapped swap", "txid", txid, "matchTx", history.matchTx, "isSwapin", isSwapin)
		return fmt.Errorf("found swapped in history, txid=%v, matchTx=%v", txid, history.matchTx)
	}
	return nil
}

func dispatchSwapTask(args *tokens.BuildTxArgs) error {
	from := strings.ToLower(args.From)
	switch args.SwapType {
	case tokens.SwapinType:
		swapChan, exist := swapinTaskChanMap[from]
		if !exist {
			return fmt.Errorf("no swapin task channel for withdraw address '%v'", args.From)
		}
		swapChan <- args
	case tokens.SwapoutType:
		swapChan, exist := swapoutTaskChanMap[from]
		if !exist {
			return fmt.Errorf("no swapout task channel for withdraw address '%v'", args.From)
		}
		swapChan <- args
	default:
		return fmt.Errorf("wrong swap type '%v'", args.SwapType.String())
	}
	return nil
}

func processSwapTask(swapChan <-chan *tokens.BuildTxArgs) {
	args := <-swapChan
	err := doSwap(args)
	if err != nil {
		logWorkerError("doSwap", "process failed", err, "pairID", args.PairID, "txid", args.SwapID, "swapType", args.SwapType.String())
	}
}

func doSwap(args *tokens.BuildTxArgs) (err error) {
	pairID := args.PairID
	txid := args.SwapID
	swapType := args.SwapType
	originValue := args.Value

	isSwapin := swapType == tokens.SwapinType
	resBridge := tokens.GetCrossChainBridge(!isSwapin)

	rawTx, err := resBridge.BuildRawTransaction(args)
	if err != nil {
		logWorkerError("doSwap", "BuildRawTransaction failed", err, "txid", txid, "isSwapin", isSwapin)
		return err
	}

	signedTx, txHash, err := dcrmSignTransaction(resBridge, rawTx, args.GetExtraArgs())
	if err != nil {
		logWorkerError("doSwap", "DcrmSignTransaction failed", err, "txid", txid, "isSwapin", isSwapin)
		return err
	}

	swapTxNonce := args.GetTxNonce()

	// update database before sending transaction
	addSwapHistory(txid, originValue, txHash, swapTxNonce, isSwapin)
	matchTx := &MatchTx{
		SwapTx:    txHash,
		SwapValue: tokens.CalcSwappedValue(pairID, originValue, isSwapin).String(),
		SwapType:  swapType,
		SwapNonce: swapTxNonce,
	}
	err = updateSwapResult(txid, matchTx)
	if err != nil {
		logWorkerError("doSwap", "update swap result failed", err, "txid", txid, "isSwapin", isSwapin)
		return err
	}

	err = mongodb.UpdateSwapStatus(isSwapin, txid, mongodb.TxProcessed, now(), "")
	if err != nil {
		logWorkerError("doSwap", "update swap status failed", err, "txid", txid, "isSwapin", isSwapin)
		return err
	}

	return sendSignedTransaction(pairID, resBridge, signedTx, txid, isSwapin)
}

type swapInfo struct {
	txid     string
	value    *big.Int
	matchTx  string
	nonce    uint64
	isSwapin bool
}

func addSwapHistory(txid string, value *big.Int, matchTx string, nonce uint64, isSwapin bool) {
	// Create the new item as its own ring
	item := ring.New(1)
	item.Value = &swapInfo{
		txid:     txid,
		value:    value,
		matchTx:  matchTx,
		nonce:    nonce,
		isSwapin: isSwapin,
	}

	swapRingLock.Lock()
	defer swapRingLock.Unlock()

	if swapRing == nil {
		swapRing = item
	} else {
		if swapRing.Len() == swapRingMaxSize {
			swapRing = swapRing.Move(-1)
			swapRing.Unlink(1)
			swapRing = swapRing.Move(1)
		}
		swapRing.Move(-1).Link(item)
	}
}

func getSwapHistory(txid string, isSwapin bool) *swapInfo {
	swapRingLock.RLock()
	defer swapRingLock.RUnlock()

	if swapRing == nil {
		return nil
	}

	r := swapRing
	for i := 0; i < r.Len(); i++ {
		item := r.Value.(*swapInfo)
		if item.txid == txid && item.isSwapin == isSwapin {
			return item
		}
		r = r.Prev()
	}

	return nil
}
