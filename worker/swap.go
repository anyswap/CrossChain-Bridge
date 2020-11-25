package worker

import (
	"container/ring"
	"errors"
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
	swapinTaskChanMap  = make(map[string]chan *tokens.BuildTxArgs)
	swapoutTaskChanMap = make(map[string]chan *tokens.BuildTxArgs)

	errAlreadySwapped = errors.New("already swapped")
)

// StartSwapJob swap job
func StartSwapJob() {
	for _, pairCfg := range tokens.GetTokenPairsConfig() {
		AddSwapJob(pairCfg)
	}
}

// AddSwapJob add swap job
func AddSwapJob(pairCfg *tokens.TokenPairConfig) {
	pairID := strings.ToLower(pairCfg.PairID)
	swapinDcrmAddr := strings.ToLower(pairCfg.DestToken.DcrmAddress)
	if _, exist := swapinTaskChanMap[swapinDcrmAddr]; !exist {
		swapinTaskChanMap[swapinDcrmAddr] = make(chan *tokens.BuildTxArgs, swapChanSize)
		go processSwapTask(swapinTaskChanMap[swapinDcrmAddr])
	}
	swapoutDcrmAddr := strings.ToLower(pairCfg.SrcToken.DcrmAddress)
	if _, exist := swapoutTaskChanMap[swapoutDcrmAddr]; !exist {
		swapoutTaskChanMap[swapoutDcrmAddr] = make(chan *tokens.BuildTxArgs, swapChanSize)
		go processSwapTask(swapoutTaskChanMap[swapoutDcrmAddr])
	}

	go startSwapinSwapJob(pairID)
	go startSwapoutSwapJob(pairID)
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
			switch err {
			case nil, errAlreadySwapped:
			default:
				logWorkerError("swapin", "process swapin swap error", err, "pairID", swap.PairID, "txid", swap.TxID, "bind", swap.Bind)
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
			switch err {
			case nil, errAlreadySwapped:
			default:
				logWorkerError("swapout", "process swapout swap error", err, "pairID", swap.PairID, "txid", swap.TxID, "bind", swap.Bind)
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
	bind := swap.Bind

	res, err := mongodb.FindSwapResult(isSwapin, txid, pairID, bind)
	if err != nil {
		return err
	}

	logWorker("swap", "start process swap", "pairID", pairID, "txid", txid, "bind", bind, "status", swap.Status, "isSwapin", isSwapin, "value", res.Value)

	fromTokenCfg, toTokenCfg := tokens.GetTokenConfigsByDirection(pairID, isSwapin)
	if fromTokenCfg == nil || toTokenCfg == nil {
		logWorkerTrace("swap", "swap is not configed", "pairID", pairID, "isSwapin", isSwapin)
		return nil
	}
	if fromTokenCfg.DisableSwap {
		logWorkerTrace("swap", "swap is disabled", "pairID", pairID, "isSwapin", isSwapin)
		return nil
	}
	isBlacked, err := isSwapInBlacklist(res)
	if err != nil {
		return err
	}
	if isBlacked {
		logWorkerTrace("swap", "address is in blacklist", "txid", txid, "bind", bind, "isSwapin", isSwapin)
		err = tokens.ErrAddressIsInBlacklist
		_ = mongodb.UpdateSwapStatus(isSwapin, txid, pairID, bind, mongodb.SwapInBlacklist, now(), err.Error())
		return nil
	}

	err = preventReswap(res, isSwapin)
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
			TxType:   tokens.SwapTxType(swap.TxType),
			Bind:     bind,
		},
		From:        toTokenCfg.DcrmAddress,
		OriginValue: value,
	}

	return dispatchSwapTask(args)
}

func preventReswap(res *mongodb.MgoSwapResult, isSwapin bool) (err error) {
	err = processNonEmptySwapResult(res, isSwapin)
	if err != nil {
		return err
	}
	return processHistory(res, isSwapin)
}

func getSwapType(isSwapin bool) tokens.SwapType {
	if isSwapin {
		return tokens.SwapinType
	}
	return tokens.SwapoutType
}

func processNonEmptySwapResult(res *mongodb.MgoSwapResult, isSwapin bool) error {
	if res.SwapTx == "" {
		return nil
	}
	txid := res.TxID
	pairID := res.PairID
	bind := res.Bind
	_ = mongodb.UpdateSwapStatus(isSwapin, txid, pairID, bind, mongodb.TxProcessed, now(), "")
	if res.Status != mongodb.MatchTxEmpty {
		return errAlreadySwapped
	}
	resBridge := tokens.GetCrossChainBridge(!isSwapin)
	if _, err := resBridge.GetTransaction(res.SwapTx); err == nil {
		return errAlreadySwapped
	}
	return nil
}

func processHistory(res *mongodb.MgoSwapResult, isSwapin bool) error {
	pairID, txid, bind := res.PairID, res.TxID, res.Bind
	history := getSwapHistory(txid, bind, isSwapin)
	if history == nil {
		return nil
	}
	if res.Status == mongodb.MatchTxFailed {
		history.txid = "" // mark ineffective
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
		_ = updateSwapResult(txid, pairID, bind, matchTx)
		logWorker("swap", "ignore swapped swap", "txid", txid, "pairID", pairID, "bind", bind, "matchTx", history.matchTx, "isSwapin", isSwapin)
		return errAlreadySwapped
	}
	return nil
}

func dispatchSwapTask(args *tokens.BuildTxArgs) error {
	from := strings.ToLower(args.From)
	switch args.SwapType {
	case tokens.SwapinType:
		swapChan, exist := swapinTaskChanMap[from]
		if !exist {
			return fmt.Errorf("no swapin task channel for dcrm address '%v'", args.From)
		}
		swapChan <- args
	case tokens.SwapoutType:
		swapChan, exist := swapoutTaskChanMap[from]
		if !exist {
			return fmt.Errorf("no swapout task channel for dcrm address '%v'", args.From)
		}
		swapChan <- args
	default:
		return fmt.Errorf("wrong swap type '%v'", args.SwapType.String())
	}
	logWorker("doSwap", "dispatch swap task", "pairID", args.PairID, "txid", args.SwapID, "bind", args.Bind, "swapType", args.SwapType.String(), "value", args.OriginValue)
	return nil
}

func processSwapTask(swapChan <-chan *tokens.BuildTxArgs) {
	for {
		args := <-swapChan
		err := doSwap(args)
		switch err {
		case nil, errAlreadySwapped:
		default:
			logWorkerError("doSwap", "process failed", err, "pairID", args.PairID, "txid", args.SwapID, "swapType", args.SwapType.String(), "value", args.OriginValue)
		}
	}
}

func doSwap(args *tokens.BuildTxArgs) (err error) {
	pairID := args.PairID
	txid := args.SwapID
	bind := args.Bind
	swapType := args.SwapType
	originValue := args.OriginValue

	isSwapin := swapType == tokens.SwapinType
	resBridge := tokens.GetCrossChainBridge(!isSwapin)

	res, err := mongodb.FindSwapResult(isSwapin, txid, pairID, bind)
	if err != nil {
		return err
	}
	err = preventReswap(res, isSwapin)
	if err != nil {
		return err
	}

	logWorker("doSwap", "start to process", "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin, "value", originValue)

	rawTx, err := resBridge.BuildRawTransaction(args)
	if err != nil {
		logWorkerError("doSwap", "build tx failed", err, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		return err
	}

	var signedTx interface{}
	var txHash string
	tokenCfg := resBridge.GetTokenConfig(pairID)
	if tokenCfg.GetDcrmAddressPrivateKey() != nil {
		signedTx, txHash, err = resBridge.SignTransaction(rawTx, pairID)
	} else {
		signedTx, txHash, err = dcrmSignTransaction(resBridge, rawTx, args.GetExtraArgs())
	}
	if err != nil {
		logWorkerError("doSwap", "sign tx failed", err, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		return err
	}

	swapTxNonce := args.GetTxNonce()

	// update database before sending transaction
	addSwapHistory(txid, bind, originValue, txHash, swapTxNonce, isSwapin)
	matchTx := &MatchTx{
		SwapTx:    txHash,
		SwapValue: tokens.CalcSwappedValue(pairID, originValue, isSwapin).String(),
		SwapType:  swapType,
		SwapNonce: swapTxNonce,
	}
	err = updateSwapResult(txid, pairID, bind, matchTx)
	if err != nil {
		logWorkerError("doSwap", "update swap result failed", err, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		return err
	}

	err = mongodb.UpdateSwapStatus(isSwapin, txid, pairID, bind, mongodb.TxProcessed, now(), "")
	if err != nil {
		logWorkerError("doSwap", "update swap status failed", err, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		return err
	}

	return sendSignedTransaction(resBridge, signedTx, txid, pairID, bind, isSwapin)
}

type swapInfo struct {
	txid     string
	bind     string
	value    *big.Int
	matchTx  string
	nonce    uint64
	isSwapin bool
}

func addSwapHistory(txid, bind string, value *big.Int, matchTx string, nonce uint64, isSwapin bool) {
	// Create the new item as its own ring
	item := ring.New(1)
	item.Value = &swapInfo{
		txid:     txid,
		bind:     bind,
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

func getSwapHistory(txid, bind string, isSwapin bool) *swapInfo {
	swapRingLock.RLock()
	defer swapRingLock.RUnlock()

	if swapRing == nil {
		return nil
	}

	r := swapRing
	for i := 0; i < r.Len(); i++ {
		item := r.Value.(*swapInfo)
		if item.txid == txid && item.bind == bind && item.isSwapin == isSwapin {
			return item
		}
		r = r.Prev()
	}

	return nil
}
