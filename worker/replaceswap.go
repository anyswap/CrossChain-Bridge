package worker

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	errSwapWithErrStatus  = errors.New("swap with error status to replace")
	errSwapTxWithHeight   = errors.New("swaptx with block height")
	errSwapTxIsOnChain    = errors.New("swaptx exist in chain")
	errGetNonceFailed     = errors.New("get nonce failed")
	errSwapNoncePassed    = errors.New("can not replace swap with old nonce")
	errSwapNonceTooBig    = errors.New("forbid replace swap with too big nonce than latest")
	errBuildTxFailed      = errors.New("build tx failed")
	errSignTxFailed       = errors.New("sign tx failed")
	errUpdateOldTxsFailed = errors.New("update old swaptxs failed")
	errNotNonceSupport    = errors.New("not nonce support bridge")

	updateOldSwapTxsLock sync.Mutex

	maxDistanceOfSwapNonce = uint64(5)
)

// ReplaceSwapin api
func ReplaceSwapin(txid, pairID, bind, gasPrice string) (string, error) {
	return replaceSwap(txid, pairID, bind, gasPrice, true)
}

// ReplaceSwapout api
func ReplaceSwapout(txid, pairID, bind, gasPrice string) (string, error) {
	return replaceSwap(txid, pairID, bind, gasPrice, false)
}

func verifyReplaceSwap(txid, pairID, bind string, isSwapin bool) (*mongodb.MgoSwap, *mongodb.MgoSwapResult, error) {
	swap, err := mongodb.FindSwap(isSwapin, txid, pairID, bind)
	if err != nil {
		return nil, nil, err
	}
	res, err := mongodb.FindSwapResult(isSwapin, txid, pairID, bind)
	if err != nil {
		return nil, nil, err
	}
	if res.SwapHeight != 0 {
		return nil, nil, errSwapTxWithHeight
	}
	if res.Status != mongodb.MatchTxNotStable {
		return nil, nil, errSwapWithErrStatus
	}

	bridge := tokens.GetCrossChainBridge(!isSwapin)
	err = checkIfSwapNonceHasPassed(bridge, res, true)
	if err != nil {
		return nil, nil, err
	}

	err = preventReplaceswapByHistory(res, isSwapin)
	if err != nil {
		return nil, nil, err
	}

	return swap, res, nil
}

func checkIfSwapNonceHasPassed(bridge tokens.CrossChainBridge, res *mongodb.MgoSwapResult, isReplace bool) error {
	nonceSetter, ok := bridge.(tokens.NonceSetter)
	if !ok {
		return errNotNonceSupport
	}

	pairID := res.PairID
	txid := res.TxID
	bind := res.Bind
	isSwapin := tokens.SwapType(res.SwapType) == tokens.SwapinType

	tokenCfg := bridge.GetTokenConfig(pairID)
	if tokenCfg == nil {
		return fmt.Errorf("no token config for pairID '%v'", pairID)
	}
	nonce, err := nonceSetter.GetPoolNonce(tokenCfg.DcrmAddress, "latest")
	if err != nil {
		return errGetNonceFailed
	}
	if isReplace && res.SwapNonce > nonce+maxDistanceOfSwapNonce {
		return errSwapNonceTooBig
	}

	// only check if nonce has passed when tx is not onchain.
	if isSwapResultTxOnChain(nonceSetter, res) {
		if isReplace {
			return errSwapTxIsOnChain
		}
		return nil
	}

	if nonce > res.SwapNonce && res.SwapNonce > 0 {
		var iden string
		if isReplace {
			iden = "[replace]"
		} else {
			iden = "[stable]"
		}
		if res.Timestamp < getSepTimeInFind(treatAsNoncePassedInterval) {
			if isSwapResultTxOnChain(nonceSetter, res) { // recheck
				if isReplace {
					return errSwapTxIsOnChain
				}
				return nil
			}
			logWorkerWarn(iden, "mark swap result failed with nonce passed", "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin, "swaptime", res.Timestamp, "nowtime", now(), "swapNonce", res.SwapNonce, "latestNonce", nonce)
			_ = markSwapResultFailed(txid, pairID, bind, isSwapin)
		}
		if isReplace {
			return errSwapNoncePassed
		}
	}
	return nil
}

func replaceSwap(txid, pairID, bind, gasPriceStr string, isSwapin bool) (txHash string, err error) {
	var gasPrice *big.Int
	if gasPriceStr != "" {
		var ok bool
		gasPrice, ok = new(big.Int).SetString(gasPriceStr, 0)
		if !ok {
			return "", errors.New("wrong gas price: " + gasPriceStr)
		}
	}

	swap, res, err := verifyReplaceSwap(txid, pairID, bind, isSwapin)
	if err != nil {
		return "", err
	}

	srcBridge := tokens.GetCrossChainBridge(isSwapin)
	swapInfo, err := verifySwapTransaction(srcBridge, pairID, txid, bind, tokens.SwapTxType(swap.TxType))
	if err != nil {
		return "", fmt.Errorf("[replace] reverify swap failed, %w", err)
	}
	if swapInfo.Value.String() != res.Value {
		return "", fmt.Errorf("[replace] reverify swap value mismatch, in db %v != %v", res.Value, swapInfo.Value)
	}
	if !strings.EqualFold(swapInfo.Bind, bind) {
		return "", fmt.Errorf("[replace] reverify swap bind address mismatch, in db %v != %v", bind, swapInfo.Bind)
	}

	bridge := tokens.GetCrossChainBridge(!isSwapin)
	tokenCfg := bridge.GetTokenConfig(pairID)
	swapType := getSwapType(isSwapin)

	replaceNum := uint64(len(res.OldSwapTxs))
	if replaceNum == 0 {
		replaceNum++
	}

	nonce := res.SwapNonce
	args := &tokens.BuildTxArgs{
		SwapInfo: tokens.SwapInfo{
			//#Identifier: params.GetReplaceIdentifier(),
			Identifier: params.GetIdentifier(),
			PairID:     pairID,
			SwapID:     txid,
			SwapType:   swapType,
			TxType:     tokens.SwapTxType(swap.TxType),
			Bind:       bind,
		},
		From:        tokenCfg.DcrmAddress,
		OriginValue: swapInfo.Value,
		Extra: &tokens.AllExtras{
			EthExtra: &tokens.EthExtraArgs{
				GasPrice: gasPrice,
				Nonce:    &nonce,
			},
			ReplaceNum: replaceNum,
		},
	}
	rawTx, err := bridge.BuildRawTransaction(args)
	if err != nil {
		logWorkerError("replaceSwap", "build tx failed", err, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		return "", errBuildTxFailed
	}
	var signedTx interface{}
	var signTxHash string
	if tokenCfg.GetDcrmAddressPrivateKey() != nil {
		signedTx, signTxHash, err = bridge.SignTransaction(rawTx, pairID)
	} else {
		signedTx, signTxHash, err = bridge.DcrmSignTransaction(rawTx, args.GetExtraArgs())
	}
	if err != nil {
		logWorkerError("replaceSwap", "sign tx failed", err, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		return "", errSignTxFailed
	}

	swapValue := ""
	if args.SwapValue != nil {
		swapValue = args.SwapValue.String()
	}
	err = replaceSwapResult(txid, pairID, bind, signTxHash, swapValue, isSwapin)
	if err != nil {
		return "", errUpdateOldTxsFailed
	}
	txHash, err = sendSignedTransaction(bridge, signedTx, args)
	if err == nil && txHash != signTxHash {
		logWorkerError("replaceSwap", "send tx success but with different hash", errSendTxWithDiffHash, "pairID", pairID, "txid", txid, "bind", bind, "isSwapin", isSwapin, "swapNonce", nonce, "txHash", txHash, "signTxHash", signTxHash)
		_ = replaceSwapResult(txid, pairID, bind, txHash, swapValue, isSwapin)
	}
	return txHash, err
}

func replaceSwapResult(txid, pairID, bind, txHash, swapValue string, isSwapin bool) (err error) {
	updateOldSwapTxsLock.Lock()
	defer updateOldSwapTxsLock.Unlock()

	res, err := mongodb.FindSwapResult(isSwapin, txid, pairID, bind)
	if err != nil {
		return err
	}

	oldSwapTxs := res.OldSwapTxs
	oldSwapVals := res.OldSwapVals
	if len(oldSwapTxs) > 0 {
		for _, oldSwapTx := range oldSwapTxs {
			if oldSwapTx == txHash {
				return nil
			}
		}
		oldSwapTxs = append(oldSwapTxs, txHash)
		oldSwapVals = append(oldSwapVals, swapValue)
	} else {
		if txHash == res.SwapTx {
			return nil
		}
		if res.SwapTx == "" {
			oldSwapTxs = []string{txHash}
			oldSwapVals = []string{swapValue}
		} else {
			oldSwapTxs = []string{res.SwapTx, txHash}
			oldSwapVals = []string{res.SwapValue, swapValue}
		}
	}
	swapType := tokens.SwapType(res.SwapType).String()
	err = updateOldSwapTxs(txid, pairID, bind, txHash, oldSwapTxs, oldSwapVals, isSwapin)
	if err != nil {
		logWorkerError("replace", "replaceSwapResult", err, "txid", txid, "pairID", pairID, "bind", bind, "swaptx", txHash, "swapType", swapType, "nonce", res.SwapNonce, "swapValue", swapValue)
	} else {
		logWorker("replace", "replaceSwapResult", "txid", txid, "pairID", pairID, "bind", bind, "swaptx", txHash, "swapType", swapType, "nonce", res.SwapNonce, "swapValue", swapValue)
	}
	return err
}

func preventReplaceswapByHistory(res *mongodb.MgoSwapResult, isSwapin bool) error {
	swapHistories, _ := mongodb.GetSwapHistory(isSwapin, res.TxID, res.Bind)
	if len(swapHistories) == 0 {
		return nil
	}
	resBridge := tokens.GetCrossChainBridge(!isSwapin)
	nonceSetter, ok := resBridge.(tokens.NonceSetter)
	if !ok {
		return errNotNonceSupport
	}
	for _, swaphist := range swapHistories {
		if isTransactionOnChain(nonceSetter, swaphist.SwapTx) {
			logWorkerError("[replace]", "forbid replace by history", errSwapTxIsOnChain,
				"isSwapin", isSwapin, "txid", res.TxID, "bind", res.Bind, "swaptx", swaphist.SwapTx)
			return errSwapTxIsOnChain
		}
	}
	return nil
}
