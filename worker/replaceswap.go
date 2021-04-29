package worker

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	errSwapWithoutSwapTx  = errors.New("swap without swaptx")
	errWrongResultStatus  = errors.New("swap result status is not 'MatchTxNotStable'")
	errSwapTxWithHeight   = errors.New("swaptx with block height")
	errSwapTxIsOnChain    = errors.New("swaptx exist in chain")
	errGetNonceFailed     = errors.New("get nonce failed")
	errSwapNoncePassed    = errors.New("can not replace swap with old nonce")
	errBuildTxFailed      = errors.New("build tx failed")
	errSignTxFailed       = errors.New("sign tx failed")
	errUpdateOldTxsFailed = errors.New("update old swaptxs failed")
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
	if res.SwapTx == "" {
		return nil, nil, errSwapWithoutSwapTx
	}
	if res.Status != mongodb.MatchTxNotStable {
		return nil, nil, errWrongResultStatus
	}
	if res.SwapHeight != 0 {
		return nil, nil, errSwapTxWithHeight
	}
	bridge := tokens.GetCrossChainBridge(!isSwapin)
	nonceSetter, ok := bridge.(tokens.NonceSetter)
	if !ok {
		return nil, nil, errors.New("not nonce support bridge")
	}
	if isSwapResultTxOnChain(nonceSetter, res) {
		return nil, nil, errSwapTxIsOnChain
	}

	tokenCfg := bridge.GetTokenConfig(pairID)
	if tokenCfg == nil {
		return nil, nil, fmt.Errorf("no token config for pairID '%v'", pairID)
	}
	nonce, err := nonceSetter.GetPoolNonce(tokenCfg.DcrmAddress, "latest")
	if err != nil {
		return nil, nil, errGetNonceFailed
	}
	if nonce > res.SwapNonce {
		if isSwapResultTxOnChain(nonceSetter, res) {
			return nil, nil, errSwapTxIsOnChain
		}
		_ = markSwapResultFailed(txid, pairID, bind, isSwapin)
		return nil, nil, errSwapNoncePassed
	}

	return swap, res, nil
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

	bridge := tokens.GetCrossChainBridge(!isSwapin)
	tokenCfg := bridge.GetTokenConfig(pairID)
	swapType := getSwapType(isSwapin)

	value, err := common.GetBigIntFromStr(res.Value)
	if err != nil {
		return "", fmt.Errorf("wrong value %v", res.Value)
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
		OriginValue: value,
		Extra: &tokens.AllExtras{
			EthExtra: &tokens.EthExtraArgs{
				GasPrice: gasPrice,
				Nonce:    &nonce,
			},
		},
	}
	rawTx, err := bridge.BuildRawTransaction(args)
	if err != nil {
		logWorkerError("replaceSwap", "build tx failed", err, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		return "", errBuildTxFailed
	}
	var signedTx interface{}
	if tokenCfg.GetDcrmAddressPrivateKey() != nil {
		signedTx, txHash, err = bridge.SignTransaction(rawTx, pairID)
	} else {
		signedTx, txHash, err = bridge.DcrmSignTransaction(rawTx, args.GetExtraArgs())
	}
	if err != nil {
		logWorkerError("replaceSwap", "sign tx failed", err, "txid", txid, "bind", bind, "isSwapin", isSwapin)
		return "", errSignTxFailed
	}

	err = replaceSwapResult(res, txHash, isSwapin)
	if err != nil {
		return "", errUpdateOldTxsFailed
	}
	err = sendSignedTransaction(bridge, signedTx, txid, pairID, bind, isSwapin)
	return txHash, err
}

func replaceSwapResult(swapResult *mongodb.MgoSwapResult, txHash string, isSwapin bool) (err error) {
	txid := swapResult.TxID
	pairID := swapResult.PairID
	bind := swapResult.Bind
	var oldSwapTxs []string
	if len(swapResult.OldSwapTxs) > 0 {
		var existsInOld bool
		for _, oldSwapTx := range swapResult.OldSwapTxs {
			if oldSwapTx == txHash {
				existsInOld = true
				break
			}
		}
		if !existsInOld {
			oldSwapTxs = swapResult.OldSwapTxs
			oldSwapTxs = append(oldSwapTxs, txHash)
		}
	} else if swapResult.SwapTx != "" && txHash != swapResult.SwapTx {
		oldSwapTxs = []string{swapResult.SwapTx, txHash}
	}
	swapType := tokens.SwapType(swapResult.SwapType).String()
	err = updateOldSwapTxs(txid, pairID, bind, oldSwapTxs, isSwapin)
	if err != nil {
		logWorkerError("replace", "replaceSwapResult", err, "txid", txid, "pairID", pairID, "bind", bind, "swaptx", txHash, "swapType", swapType, "nonce", swapResult.SwapNonce)
	} else {
		logWorker("replace", "replaceSwapResult", "txid", txid, "pairID", pairID, "bind", bind, "swaptx", txHash, "swapType", swapType, "nonce", swapResult.SwapNonce)
	}
	return err
}
