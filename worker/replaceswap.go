package worker

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// ReplaceSwapin api
func ReplaceSwapin(txid, pairID, bind, gasPrice string) error {
	return replaceSwap(txid, pairID, bind, gasPrice, true)
}

// ReplaceSwapout api
func ReplaceSwapout(txid, pairID, bind, gasPrice string) error {
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
		return nil, nil, errors.New("swap without swaptx")
	}
	if res.Status != mongodb.MatchTxNotStable {
		return nil, nil, errors.New("swap result status is not 'MatchTxNotStable'")
	}
	if res.SwapHeight != 0 {
		return nil, nil, errors.New("swaptx with block height")
	}
	bridge := tokens.GetCrossChainBridge(!isSwapin)
	txStat := getSwapTxStatus(bridge, res)
	if txStat != nil && txStat.BlockHeight > 0 {
		return nil, nil, errors.New("swaptx exist in chain")
	}

	nonceSetter, ok := bridge.(tokens.NonceSetter)
	if !ok {
		return nil, nil, errors.New("not nonce support bridge")
	}

	tokenCfg := bridge.GetTokenConfig(pairID)
	if tokenCfg == nil {
		return nil, nil, fmt.Errorf("no token config for pairID '%v'", pairID)
	}
	nonce, err := nonceSetter.GetPoolNonce(tokenCfg.DcrmAddress, "latest")
	if err != nil {
		return nil, nil, fmt.Errorf("get nonce failed, %v", err)
	}
	if nonce > res.SwapNonce {
		return nil, nil, errors.New("can not replace swap with old nonce")
	}

	return swap, res, nil
}

func replaceSwap(txid, pairID, bind, gasPriceStr string, isSwapin bool) error {
	var gasPrice *big.Int
	if gasPriceStr != "" {
		var ok bool
		gasPrice, ok = new(big.Int).SetString(gasPriceStr, 0)
		if !ok {
			return errors.New("wrong gas price: " + gasPriceStr)
		}
	}

	swap, res, err := verifyReplaceSwap(txid, pairID, bind, isSwapin)
	if err != nil {
		return err
	}

	bridge := tokens.GetCrossChainBridge(!isSwapin)
	tokenCfg := bridge.GetTokenConfig(pairID)
	swapType := getSwapType(isSwapin)

	value, err := common.GetBigIntFromStr(res.Value)
	if err != nil {
		return fmt.Errorf("wrong value %v", res.Value)
	}

	nonce := res.SwapNonce
	args := &tokens.BuildTxArgs{
		SwapInfo: tokens.SwapInfo{
			Identifier: tokens.ReplaceSwapIdentifier,
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
		return err
	}
	var signedTx interface{}
	var txHash string
	if tokenCfg.GetDcrmAddressPrivateKey() != nil {
		signedTx, txHash, err = bridge.SignTransaction(rawTx, pairID)
	} else {
		signedTx, txHash, err = dcrmSignTransaction(bridge, rawTx, args.GetExtraArgs())
	}
	if err != nil {
		return err
	}

	err = replaceSwapResult(res, txHash, isSwapin)
	if err != nil {
		return err
	}
	return sendSignedTransaction(bridge, signedTx, txid, pairID, bind, isSwapin, true)
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
	err = updateOldSwapTxs(txid, pairID, bind, oldSwapTxs, isSwapin)
	if err != nil {
		logWorkerError("replace", "replaceSwapResult", err, "txid", txid, "pairID", pairID, "bind", bind, "swaptx", txHash, "nonce", swapResult.SwapNonce)
	} else {
		logWorker("replace", "replaceSwapResult", "txid", txid, "pairID", pairID, "bind", bind, "swaptx", txHash, "nonce", swapResult.SwapNonce)
	}
	return err
}
