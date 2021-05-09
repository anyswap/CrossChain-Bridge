package worker

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	swapChanSize       = 10
	swapinTaskChanMap  = make(map[string]chan *tokens.BuildTxArgs)
	swapoutTaskChanMap = make(map[string]chan *tokens.BuildTxArgs)

	maxSwapRoutineCount = uint64(20)
	signSwapTxChan      chan *signContent

	errAlreadySwapped = errors.New("already swapped")
	errDBError        = errors.New("database error")
)

type signContent struct {
	rawTx interface{}
	args  *tokens.BuildTxArgs
}

// StartSwapJob swap job
func StartSwapJob() {
	maxRoutines := params.GetConfig().MaxSwapRoutineCount
	if maxRoutines > 0 {
		maxSwapRoutineCount = maxRoutines
	}
	signSwapTxChan = make(chan *signContent, maxSwapRoutineCount)
	go doSignSwapTxs()

	swapinNonces, swapoutNonces := mongodb.LoadAllSwapNonces()
	if srcNonceSetter != nil {
		srcNonceSetter.InitNonces(swapoutNonces)
	}
	if dstNonceSetter != nil {
		dstNonceSetter.InitNonces(swapinNonces)
	}

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
	processSwapins(pairID, mongodb.TxProcessing)
	for {
		processSwapins(pairID, mongodb.TxNotSwapped)
		restInJob(restIntervalInDoSwapJob)
	}
}

func startSwapoutSwapJob(pairID string) {
	logWorker("swapout", "start swapout swap job")
	processSwapouts(pairID, mongodb.TxProcessing)
	for {
		processSwapouts(pairID, mongodb.TxNotSwapped)
		restInJob(restIntervalInDoSwapJob)
	}
}

func processSwapins(pairID string, status mongodb.SwapStatus) {
	swapins, err := findSwapinsToSwap(pairID, status)
	if err != nil {
		logWorkerError("swapin", "find swapins error", err)
		return
	}
	if len(swapins) == 0 {
		return
	}
	logWorker("swapin", "find swapins to swap", "count", len(swapins))
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

func processSwapouts(pairID string, status mongodb.SwapStatus) {
	swapouts, err := findSwapoutsToSwap(pairID, status)
	if err != nil {
		logWorkerError("swapout", "find swapouts error", err)
	}
	if len(swapouts) == 0 {
		return
	}
	logWorker("swapout", "find swapouts to swap", "count", len(swapouts))
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

func findSwapinsToSwap(pairID string, status mongodb.SwapStatus) ([]*mongodb.MgoSwap, error) {
	septime := getSepTimeInFind(maxDoSwapLifetime)
	return mongodb.FindSwapinsWithPairIDAndStatus(pairID, status, septime)
}

func findSwapoutsToSwap(pairID string, status mongodb.SwapStatus) ([]*mongodb.MgoSwap, error) {
	septime := getSepTimeInFind(maxDoSwapLifetime)
	return mongodb.FindSwapoutsWithPairIDAndStatus(pairID, status, septime)
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

	err = preventDoubleSwap(res, isSwapin)
	if err != nil {
		return err
	}

	logWorker("swap", "start process swap", "pairID", pairID, "txid", txid, "bind", bind, "status", swap.Status, "isSwapin", isSwapin, "value", res.Value)

	value, dcrmAddress, err := checkSwapResult(res, isSwapin)
	if err != nil {
		return err
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
		OriginValue: value,
	}

	matchTx := &MatchTx{
		SwapValue: tokens.CalcSwappedValue(pairID, value, isSwapin).String(),
		SwapType:  swapType,
	}

	// NOTE: only assign 'nonceSetter' when no swap nonce is saved in db
	var nonceSetter tokens.NonceSetter
	if res.SwapNonce > 0 {
		args.SetTxNonce(res.SwapNonce)
	} else {
		resBridge := tokens.GetCrossChainBridge(!isSwapin)
		nonceSetter, _ = resBridge.(tokens.NonceSetter)
		if nonceSetter != nil {
			swapNonce := assignSwapNonce(nonceSetter, pairID, dcrmAddress)
			matchTx.SwapNonce = swapNonce // update swap nonce first
			args.SetTxNonce(swapNonce)
		}
	}

	err = updateSwapResult(txid, pairID, bind, matchTx)
	if err != nil {
		logWorkerError("swap", "update swap result", err, "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		return err
	}

	if nonceSetter != nil {
		// update after previous swap nonce is saved in db
		nonceSetter.SetNonce(pairID, matchTx.SwapNonce+1) // increase for next usage
	}

	err = mongodb.UpdateSwapStatus(isSwapin, txid, pairID, bind, mongodb.TxProcessing, now(), "")
	if err != nil {
		logWorkerError("swap", "update swap status to prcessing failed", err, "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		return err
	}

	return dispatchSwapTask(args)
}

func checkSwapResult(res *mongodb.MgoSwapResult, isSwapin bool) (value *big.Int, dcrmAddress string, err error) {
	pairID := res.PairID
	txid := res.TxID
	bind := res.Bind

	fromTokenCfg, toTokenCfg := tokens.GetTokenConfigsByDirection(pairID, isSwapin)
	if fromTokenCfg == nil || toTokenCfg == nil {
		logWorkerTrace("swap", "swap is not configed", "pairID", pairID, "isSwapin", isSwapin)
		return nil, "", tokens.ErrUnknownPairID
	}
	if fromTokenCfg.DisableSwap {
		logWorkerTrace("swap", "swap is disabled", "pairID", pairID, "isSwapin", isSwapin)
		return nil, "", tokens.ErrSwapIsClosed
	}
	isBlacked, err := isSwapInBlacklist(res)
	if err != nil {
		return nil, "", errDBError
	}
	if isBlacked {
		logWorkerTrace("swap", "address is in blacklist", "txid", txid, "bind", bind, "isSwapin", isSwapin)
		err = tokens.ErrAddressIsInBlacklist
		_ = mongodb.UpdateSwapStatus(isSwapin, txid, pairID, bind, mongodb.SwapInBlacklist, now(), err.Error())
		return nil, "", err
	}

	value, err = common.GetBigIntFromStr(res.Value)
	if err != nil {
		return nil, "", fmt.Errorf("wrong value %v", res.Value)
	}

	return value, toTokenCfg.DcrmAddress, nil
}

func preventDoubleSwap(res *mongodb.MgoSwapResult, isSwapin bool) error {
	if res.SwapTx != "" || res.SwapHeight != 0 || len(res.OldSwapTxs) > 0 {
		if res.Status == mongodb.TxProcessing && res.SwapTx != "" {
			go doReplaceSwap(res)
		}
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
	return nil
}

func getSwapType(isSwapin bool) tokens.SwapType {
	if isSwapin {
		return tokens.SwapinType
	}
	return tokens.SwapoutType
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
	logWorker("doSwap", "dispatch swap task", "pairID", args.PairID, "txid", args.SwapID, "bind", args.Bind, "swapType", args.SwapType.String(), "value", args.OriginValue, "swapNonce", args.GetTxNonce())
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
	err = preventDoubleSwap(res, isSwapin)
	if err != nil {
		return err
	}

	swapNonce := args.GetTxNonce()
	if swapNonce != res.SwapNonce {
		return fmt.Errorf("swap nonce mismatch, in args %v, in db %v", swapNonce, res.SwapNonce)
	}

	logWorker("doSwap", "start to process", "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin, "value", originValue, "swapNonce", swapNonce)

	rawTx, err := resBridge.BuildRawTransaction(args)
	if err != nil {
		logWorkerError("doSwap", "build tx failed", err, "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		return err
	}

	signSwapTxChan <- &signContent{rawTx, args} // producer of sign
	return err
}

func doSignSwapTxs() { // consumer of sign
	for {
		sc := <-signSwapTxChan
		_ = signAndSendResultTx(sc)
	}
}

func signAndSendResultTx(sc *signContent) (err error) {
	args := sc.args
	pairID := args.PairID
	txid := args.SwapID
	bind := args.Bind
	swapType := args.SwapType

	isSwapin := swapType == tokens.SwapinType
	resBridge := tokens.GetCrossChainBridge(!isSwapin)

	var signedTx interface{}
	var txHash string
	tokenCfg := resBridge.GetTokenConfig(pairID)
	for i := 1; ; i++ { // retry sign until success
		if tokenCfg.GetDcrmAddressPrivateKey() != nil {
			signedTx, txHash, err = resBridge.SignTransaction(sc.rawTx, pairID)
		} else {
			signedTx, txHash, err = resBridge.DcrmSignTransaction(sc.rawTx, args.GetExtraArgs())
		}
		if err == nil {
			break
		}
		logWorkerError("doSwap", "sign tx failed", err, "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin, "signCount", i)
		restInJob(retrySignInterval)
	}

	// update database before sending transaction
	err = updateSwapResultTx(txid, pairID, bind, txHash, isSwapin, mongodb.MatchTxNotStable)
	if err != nil {
		logWorkerError("doSwap", "update swap result swaptx failed", err, "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		return err
	}

	err = mongodb.UpdateSwapStatus(isSwapin, txid, pairID, bind, mongodb.TxProcessed, now(), "")
	if err != nil {
		logWorkerError("doSwap", "update swap status to prcessed failed", err, "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		return err
	}

	return sendSignedTransaction(resBridge, signedTx, txid, pairID, bind, isSwapin)
}
