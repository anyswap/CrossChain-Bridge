package worker

import (
	"errors"
	"fmt"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/types"
	mapset "github.com/deckarep/golang-set"
)

var (
	cachedSwapHistoty        = mapset.NewSet()
	maxCachedSwapHistorySize = 1000

	cachedSwapTasks    = mapset.NewSet()
	maxCachedSwapTasks = 1000

	swapChanSize       = 10
	swapinTaskChanMap  = make(map[string]chan *tokens.BuildTxArgs)
	swapoutTaskChanMap = make(map[string]chan *tokens.BuildTxArgs)

	errAlreadySwapped     = errors.New("already swapped")
	errDBError            = errors.New("database error")
	errSendTxWithDiffHash = errors.New("send tx with different hash")
)

// StartSwapJob swap job
func StartSwapJob() {
	swapinNonces, swapoutNonces := mongodb.LoadAllSwapNonces()
	if tokens.DstNonceSetter != nil {
		tokens.DstNonceSetter.InitNonces(swapinNonces)
	}
	if tokens.SrcNonceSetter != nil {
		tokens.SrcNonceSetter.InitNonces(swapoutNonces)
	}
	for _, pairCfg := range tokens.GetTokenPairsConfig() {
		AddSwapJob(pairCfg)
	}

	mongodb.MgoWaitGroup.Add(2)
	go startSwapinSwapJob()
	go startSwapoutSwapJob()
}

// AddSwapJob add swap job
func AddSwapJob(pairCfg *tokens.TokenPairConfig) {
	swapinDcrmAddr := strings.ToLower(pairCfg.DestToken.DcrmAddress)
	if _, exist := swapinTaskChanMap[swapinDcrmAddr]; !exist {
		swapinTaskChanMap[swapinDcrmAddr] = make(chan *tokens.BuildTxArgs, swapChanSize)
		utils.TopWaitGroup.Add(1)
		go processSwapTask(swapinTaskChanMap[swapinDcrmAddr], swapinDcrmAddr, true)
	}
	swapoutDcrmAddr := strings.ToLower(pairCfg.SrcToken.DcrmAddress)
	if _, exist := swapoutTaskChanMap[swapoutDcrmAddr]; !exist {
		swapoutTaskChanMap[swapoutDcrmAddr] = make(chan *tokens.BuildTxArgs, swapChanSize)
		utils.TopWaitGroup.Add(1)
		go processSwapTask(swapoutTaskChanMap[swapoutDcrmAddr], swapoutDcrmAddr, false)
	}
}

func startSwapinSwapJob() {
	logWorker("swap", "start swapin swap job")
	defer mongodb.MgoWaitGroup.Done()
	for {
		if utils.IsCleanuping() {
			logWorker("swap", "stop swapin swap job")
			return
		}
		processSwapins(mongodb.TxNotSwapped)
		restInJob(restIntervalInDoSwapJob)
	}
}

func startSwapoutSwapJob() {
	logWorker("swap", "start swapout swap job")
	defer mongodb.MgoWaitGroup.Done()
	for {
		if utils.IsCleanuping() {
			logWorker("swap", "stop swapout swap job")
			return
		}
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
		if utils.IsCleanuping() {
			return
		}
		err := processSwapinSwap(swap)
		switch {
		case err == nil,
			errors.Is(err, errAlreadySwapped),
			errors.Is(err, errDBError),
			errors.Is(err, tokens.ErrUnknownPairID),
			errors.Is(err, tokens.ErrAddressIsInBlacklist),
			errors.Is(err, tokens.ErrSwapIsClosed):
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
		if utils.IsCleanuping() {
			return
		}
		err := processSwapoutSwap(swap)
		switch {
		case err == nil,
			errors.Is(err, errAlreadySwapped),
			errors.Is(err, errDBError),
			errors.Is(err, tokens.ErrUnknownPairID),
			errors.Is(err, tokens.ErrAddressIsInBlacklist),
			errors.Is(err, tokens.ErrSwapIsClosed):
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

	cacheKey := getSwapCacheKey(isSwapin, txid, bind)
	if cachedSwapTasks.Contains(cacheKey) {
		return errAlreadySwapped
	}

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
			Reswapping: res.Status == mongodb.Reswapping,
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
	default:
	}
	if res.Status != mongodb.Reswapping {
		if isSwapHistoryExist(isSwapin, res.TxID, res.Bind) {
			logWorkerError("[doSwap]", "forbid reswap by cache", errAlreadySwapped, "isSwapin", isSwapin, "txid", res.TxID, "bind", res.Bind)
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
			txStatus, err := resBridge.GetTransactionStatus(swaphist.SwapTx)
			if err != nil {
				continue
			}
			if txStatus.Receipt != nil {
				receipt, ok := txStatus.Receipt.(*types.RPCTxReceipt)
				if ok && receipt.IsStatusOk() {
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

func processSwapTask(swapChan <-chan *tokens.BuildTxArgs, dcrmAddress string, isSwapin bool) {
	defer utils.TopWaitGroup.Done()
	for {
		select {
		case <-utils.CleanupChan:
			logWorker("doSwap", "stop process swap task", "isSwapin", isSwapin, "dcrmAddress", dcrmAddress)
			return
		case args := <-swapChan:
			if !strings.EqualFold(args.From, dcrmAddress) || args.SwapType != getSwapType(isSwapin) {
				logWorkerWarn("doSwap", "ignore swap task as mismatch reason", "isSwapin", isSwapin, "dcrmAddress", dcrmAddress, "args", args)
				continue
			}
			err := doSwap(args)
			switch {
			case err == nil,
				errors.Is(err, errAlreadySwapped):
			default:
				logWorkerError("doSwap", "process failed", err, "pairID", args.PairID, "txid", args.SwapID, "swapType", args.SwapType.String(), "value", args.OriginValue)
			}
		}
	}
}

func getSwapCacheKey(isSwapin bool, txid, bind string) string {
	return strings.ToLower(fmt.Sprintf("%s:%s:%t", txid, bind, isSwapin))
}

func checkAndUpdateProcessSwapTaskCache(key string) error {
	if cachedSwapTasks.Contains(key) {
		return errAlreadySwapped
	}
	if cachedSwapTasks.Cardinality() >= maxCachedSwapTasks {
		cachedSwapTasks.Pop()
	}
	cachedSwapTasks.Add(key)
	return nil
}

func doSwap(args *tokens.BuildTxArgs) (err error) {
	pairID := args.PairID
	txid := args.SwapID
	bind := args.Bind
	swapType := args.SwapType

	isSwapin := swapType == tokens.SwapinType
	resBridge := tokens.GetCrossChainBridge(!isSwapin)

	cacheKey := getSwapCacheKey(isSwapin, txid, bind)
	err = checkAndUpdateProcessSwapTaskCache(cacheKey)
	if err != nil {
		return err
	}
	logWorker("doSwap", "add swap cache", "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin, "value", args.OriginValue)
	isCachedSwapProcessed := false
	defer func() {
		if !isCachedSwapProcessed {
			logWorkerError("doSwap", "delete swap cache", err, "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin, "value", args.OriginValue)
			cachedSwapTasks.Remove(cacheKey)
		}
	}()

	logWorker("doSwap", "start to process", "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin, "value", args.OriginValue)

	rawTx, err := resBridge.BuildRawTransaction(args)
	if err != nil {
		logWorkerError("doSwap", "build tx failed", err, "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		return err
	}

	swapNonce := args.GetTxNonce()

	var signedTx interface{}
	var signTxHash string
	tokenCfg := resBridge.GetTokenConfig(pairID)
	for i := 1; i <= 3; i++ { // with retry
		if tokenCfg.GetDcrmAddressPrivateKey() != nil {
			signedTx, signTxHash, err = resBridge.SignTransaction(rawTx, pairID)
		} else {
			signedTx, signTxHash, err = resBridge.DcrmSignTransaction(rawTx, args.GetExtraArgs())
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

	// recheck reswap before update db
	res, err := mongodb.FindSwapResult(isSwapin, txid, pairID, bind)
	if err != nil {
		return err
	}
	err = preventReswap(res, isSwapin)
	if err != nil {
		return err
	}

	// update database before sending transaction
	matchTx := &MatchTx{
		SwapTx:    signTxHash,
		SwapType:  swapType,
		SwapNonce: swapNonce,
	}
	if args.SwapValue != nil {
		matchTx.SwapValue = args.SwapValue.String()
	} else {
		matchTx.SwapValue = tokens.CalcSwappedValue(pairID, args.OriginValue, isSwapin).String()
	}
	err = updateSwapResult(txid, pairID, bind, matchTx)
	if err != nil {
		logWorkerError("doSwap", "update swap result failed", err, "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		return err
	}
	isCachedSwapProcessed = true

	err = mongodb.UpdateSwapStatus(isSwapin, txid, pairID, bind, mongodb.TxProcessed, now(), "")
	if err != nil {
		logWorkerError("doSwap", "update swap status failed", err, "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		return err
	}

	txHash, err := sendSignedTransaction(resBridge, signedTx, args)
	if err == nil {
		logWorker("doSwap", "send tx success", "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin, "swapNonce", swapNonce, "txHash", txHash)
		if txHash != signTxHash {
			logWorkerError("doSwap", "send tx success but with different hash", errSendTxWithDiffHash, "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin, "swapNonce", swapNonce, "txHash", txHash, "signTxHash", signTxHash)
			_ = replaceSwapResult(txid, pairID, bind, txHash, matchTx.SwapValue, isSwapin)
		}
	}
	return err
}

// DeleteCachedSwap delete cached swap
func DeleteCachedSwap(isSwapin bool, txid, bind string) {
	cacheKey := getSwapCacheKey(isSwapin, txid, bind)
	cachedSwapTasks.Remove(cacheKey)
}

func addSwapHistory(isSwapin bool, txid, bind string) {
	if cachedSwapHistoty.Cardinality() >= maxCachedSwapHistorySize {
		cachedSwapHistoty.Pop()
	}
	key := getSwapCacheKey(isSwapin, txid, bind)
	cachedSwapHistoty.Add(key)
}

func isSwapHistoryExist(isSwapin bool, txid, bind string) bool {
	key := getSwapCacheKey(isSwapin, txid, bind)
	return cachedSwapHistoty.Contains(key)
}
