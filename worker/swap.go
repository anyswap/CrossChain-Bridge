package worker

import (
	"container/ring"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/types"
)

var (
	swapRing        *ring.Ring
	swapRingLock    sync.RWMutex
	swapRingMaxSize = 1000

	swapChanSize       = 10
	swapinTaskChanMap  = make(map[string]chan *tokens.BuildTxArgs)
	swapoutTaskChanMap = make(map[string]chan *tokens.BuildTxArgs)

	errAlreadySwapped = errors.New("already swapped")
	errDBError        = errors.New("database error")
)

// StartSwapJob swap job
func StartSwapJob() {
	swapinNonces, swapoutNonces := mongodb.LoadAllSwapNonces()
	if dstNonceSetter != nil {
		dstNonceSetter.InitNonces(swapinNonces)
	}
	if srcNonceSetter != nil {
		srcNonceSetter.InitNonces(swapoutNonces)
	}
	for _, pairCfg := range tokens.GetTokenPairsConfig() {
		AddSwapJob(pairCfg)
	}

	go startSwapinSwapJob()
	go startSwapoutSwapJob()
}

// AddSwapJob add swap job
func AddSwapJob(pairCfg *tokens.TokenPairConfig) {
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
}

func startSwapinSwapJob() {
	logWorker("swap", "start swapin swap job")
	for {
		processSwapins(mongodb.TxNotSwapped)
		restInJob(restIntervalInDoSwapJob)
	}
}

func startSwapoutSwapJob() {
	logWorker("swap", "start swapout swap job")
	for {
		processSwapouts(mongodb.TxNotSwapped)
		restInJob(restIntervalInDoSwapJob)
	}
}

func processSwapins(status mongodb.SwapStatus) {
	swapins, err := findSwapinsToSwap(status)
	if err != nil {
		logWorkerError("swapin", "find swapins error", err, "status", status)
		return
	}
	if len(swapins) == 0 {
		return
	}
	logWorker("swapin", "find swapins to swap", "status", status, "count", len(swapins))
	for _, swap := range swapins {
		err := processSwapinSwap(swap)
		switch err {
		case nil,
			errAlreadySwapped,
			errDBError,
			tokens.ErrUnknownPairID,
			tokens.ErrAddressIsInBlacklist,
			tokens.ErrSwapIsClosed:
		default:
			logWorkerError("swapin", "process swapin swap error", err, "pairID", swap.PairID, "txid", swap.TxID, "bind", swap.Bind)
		}
	}
}

func processSwapouts(status mongodb.SwapStatus) {
	swapouts, err := findSwapoutsToSwap(status)
	if err != nil {
		logWorkerError("swapout", "find swapouts error", err, "status", status)
	}
	if len(swapouts) == 0 {
		return
	}
	logWorker("swapout", "find swapouts to swap", "status", status, "count", len(swapouts))
	for _, swap := range swapouts {
		err := processSwapoutSwap(swap)
		switch err {
		case nil,
			errAlreadySwapped,
			errDBError,
			tokens.ErrUnknownPairID,
			tokens.ErrAddressIsInBlacklist,
			tokens.ErrSwapIsClosed:
		default:
			logWorkerError("swapout", "process swapout swap error", err, "pairID", swap.PairID, "txid", swap.TxID, "bind", swap.Bind)
		}
	}
}

func findSwapinsToSwap(status mongodb.SwapStatus) ([]*mongodb.MgoSwap, error) {
	septime := getSepTimeInFind(maxDoSwapLifetime)
	return mongodb.FindSwapinsWithStatus(status, septime)
}

func findSwapoutsToSwap(status mongodb.SwapStatus) ([]*mongodb.MgoSwap, error) {
	septime := getSepTimeInFind(maxDoSwapLifetime)
	return mongodb.FindSwapoutsWithStatus(status, septime)
}

func isSwapInBlacklist(swap *mongodb.MgoSwapResult) (isBlacked bool, err error) {
	isBlacked, err = mongodb.QueryBlacklist(swap.From, swap.PairID)
	if err != nil {
		logWorkerTrace("swap", "query blacklist failed", "err", err)
		return isBlacked, err
	}
	if !isBlacked && swap.Bind != swap.From {
		isBlacked, err = mongodb.QueryBlacklist(swap.Bind, swap.PairID)
		if err != nil {
			logWorkerTrace("swap", "query blacklist failed", "err", err)
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

	err = preventReswap(res, isSwapin)
	if err != nil {
		return err
	}

	dcrmAddress, err := checkSwapResult(res, isSwapin)
	if err != nil {
		return err
	}

	logWorker("swap", "start process swap", "pairID", pairID, "txid", txid, "bind", bind, "status", swap.Status, "isSwapin", isSwapin, "value", res.Value)

	srcBridge := tokens.GetCrossChainBridge(isSwapin)
	swapInfo, err := verifySwapTransaction(srcBridge, pairID, txid, bind, tokens.SwapTxType(swap.TxType))
	if err != nil {
		return fmt.Errorf("[doSwap] reverify swap failed, %w", err)
	}
	if swapInfo.Value.String() != res.Value {
		return fmt.Errorf("[doSwap] reverify swap value mismatch, in db %v != %v", res.Value, swapInfo.Value)
	}
	if !strings.EqualFold(swapInfo.Bind, bind) {
		return fmt.Errorf("[doSwap] reverify swap bind address mismatch, in db %v != %v", bind, swapInfo.Bind)
	}

	swapType := getSwapType(isSwapin)
	args := &tokens.BuildTxArgs{
		SwapInfo: tokens.SwapInfo{
			Identifier: params.GetIdentifier(),
			PairID:     pairID,
			SwapID:     txid,
			SwapType:   swapType,
			TxType:     tokens.SwapTxType(swap.TxType),
			Bind:       bind,
		},
		From:        dcrmAddress,
		OriginValue: swapInfo.Value,
	}

	return dispatchSwapTask(args)
}

func checkSwapResult(res *mongodb.MgoSwapResult, isSwapin bool) (dcrmAddress string, err error) {
	pairID := res.PairID
	txid := res.TxID
	bind := res.Bind

	fromTokenCfg, toTokenCfg := tokens.GetTokenConfigsByDirection(pairID, isSwapin)
	if fromTokenCfg == nil || toTokenCfg == nil {
		logWorkerTrace("swap", "swap is not configed", "pairID", pairID, "isSwapin", isSwapin)
		return "", tokens.ErrUnknownPairID
	}
	if fromTokenCfg.DisableSwap {
		logWorkerTrace("swap", "swap is disabled", "pairID", pairID, "isSwapin", isSwapin)
		return "", tokens.ErrSwapIsClosed
	}
	isBlacked, err := isSwapInBlacklist(res)
	if err != nil {
		return "", errDBError
	}
	if isBlacked {
		logWorkerTrace("swap", "address is in blacklist", "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		err = tokens.ErrAddressIsInBlacklist
		_ = mongodb.UpdateSwapStatus(isSwapin, txid, pairID, bind, mongodb.SwapInBlacklist, now(), err.Error())
		return "", err
	}

	return toTokenCfg.DcrmAddress, nil
}

func preventReswap(res *mongodb.MgoSwapResult, isSwapin bool) error {
	if res.SwapNonce > 0 || res.SwapTx != "" || res.SwapHeight != 0 || len(res.OldSwapTxs) > 0 {
		_ = mongodb.UpdateSwapStatus(isSwapin, res.TxID, res.PairID, res.Bind, mongodb.TxProcessed, now(), "")
		return errAlreadySwapped
	}
	switch res.Status {
	case mongodb.TxWithBigValue,
		mongodb.TxWithWrongMemo,
		mongodb.BindAddrIsContract,
		mongodb.TxWithWrongValue:
		_ = mongodb.UpdateSwapStatus(isSwapin, res.TxID, res.PairID, res.Bind, res.Status, now(), "")
		return fmt.Errorf("forbid doswap for swap with status %v", res.Status.String())
	}
	if res.Status != mongodb.Reswapping {
		history := getSwapHistory(isSwapin, res.TxID, res.Bind)
		if history != nil {
			logWorkerError("[doSwap]", "forbid reswap by cache", errAlreadySwapped,
				"isSwapin", history.isSwapin, "txid", history.txid, "bind", history.bind, "swaptx", history.matchTx)
			_ = mongodb.UpdateSwapStatus(isSwapin, res.TxID, res.PairID, res.Bind, mongodb.TxProcessed, now(), "")
			return errAlreadySwapped
		}
	}
	return preventReswapByHistory(res, isSwapin)
}

func preventReswapByHistory(res *mongodb.MgoSwapResult, isSwapin bool) error {
	swapHistories, _ := mongodb.GetSwapHistory(isSwapin, res.TxID, res.Bind)
	if len(swapHistories) == 0 {
		return nil
	}
	var alreadySwapped bool
	if res.Status != mongodb.Reswapping {
		alreadySwapped = true
	} else {
		resBridge := tokens.GetCrossChainBridge(!isSwapin)
		for _, swaphist := range swapHistories {
			txStatus := resBridge.GetTransactionStatus(swaphist.SwapTx)
			if txStatus.Receipt != nil {
				receipt, ok := txStatus.Receipt.(*types.RPCTxReceipt)
				if ok && *receipt.Status == 1 {
					alreadySwapped = true
					break
				}
			} else if txStatus != nil && txStatus.BlockHeight > 0 {
				alreadySwapped = true
				break
			}
		}
	}
	if alreadySwapped {
		logWorkerError("[doSwap]", "forbid reswap by history", errAlreadySwapped,
			"isSwapin", isSwapin, "txid", res.TxID, "bind", res.Bind, "history", swapHistories)
		_ = mongodb.UpdateSwapStatus(isSwapin, res.TxID, res.PairID, res.Bind, mongodb.TxProcessed, now(), "")
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
		logWorkerError("doSwap", "build tx failed", err, "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		return err
	}

	swapNonce := args.GetTxNonce()

	var signedTx interface{}
	var txHash string
	tokenCfg := resBridge.GetTokenConfig(pairID)
	for i := 1; i <= 3; i++ { // with retry
		if tokenCfg.GetDcrmAddressPrivateKey() != nil {
			signedTx, txHash, err = resBridge.SignTransaction(rawTx, pairID)
		} else {
			signedTx, txHash, err = resBridge.DcrmSignTransaction(rawTx, args.GetExtraArgs())
		}
		if err == nil {
			break
		}
		logWorkerError("doSwap", "sign tx failed", err, "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin, "signCount", i)
		restInJob(retrySignInterval)
	}
	if err != nil {
		return err
	}

	// update database before sending transaction
	matchTx := &MatchTx{
		SwapTx:    txHash,
		SwapValue: tokens.CalcSwappedValue(pairID, originValue, isSwapin).String(),
		SwapType:  swapType,
		SwapNonce: swapNonce,
	}
	err = updateSwapResult(txid, pairID, bind, matchTx)
	if err != nil {
		logWorkerError("doSwap", "update swap result failed", err, "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		return err
	}

	err = mongodb.UpdateSwapStatus(isSwapin, txid, pairID, bind, mongodb.TxProcessed, now(), "")
	if err != nil {
		logWorkerError("doSwap", "update swap status failed", err, "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		return err
	}

	txHash, err = sendSignedTransaction(resBridge, signedTx, txid, pairID, bind, isSwapin)
	if err == nil {
		logWorker("doSwap", "send tx success", "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin, "swapNonce", swapNonce, "txHash", txHash)
		if nonceSetter, ok := resBridge.(tokens.NonceSetter); ok {
			nonceSetter.SetNonce(pairID, swapNonce+1) // increase for next usage
		}
	}
	return err
}

type swapInfo struct {
	isSwapin bool
	txid     string
	bind     string
	matchTx  string
}

func addSwapHistory(isSwapin bool, txid, bind, matchTx string) {
	// Create the new item as its own ring
	item := ring.New(1)
	item.Value = &swapInfo{
		isSwapin: isSwapin,
		txid:     txid,
		bind:     bind,
		matchTx:  matchTx,
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

func getSwapHistory(isSwapin bool, txid, bind string) *swapInfo {
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
