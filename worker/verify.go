package worker

import (
	"sync"

	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc"
)

var (
	swapinVerifyStarter  sync.Once
	swapoutVerifyStarter sync.Once
)

// StartVerifyJob verify job
func StartVerifyJob() {
	go startSwapinVerifyJob()
	go startSwapoutVerifyJob()
}

func startSwapinVerifyJob() {
	swapinVerifyStarter.Do(func() {
		logWorker("verify", "start swapin verify job")
		for {
			res, err := findSwapinsToVerify()
			if err != nil {
				logWorkerError("verify", "find swapins error", err)
			}
			if len(res) > 0 {
				logWorker("verify", "find swapins to verify", "count", len(res))
			}
			for _, swap := range res {
				err = processSwapinVerify(swap)
				switch err {
				case nil, tokens.ErrTxNotStable, tokens.ErrTxNotFound:
				default:
					logWorkerError("verify", "process swapin verify error", err, "txid", swap.TxID)
				}
			}
			restInJob(restIntervalInVerifyJob)
		}
	})
}

func startSwapoutVerifyJob() {
	swapoutVerifyStarter.Do(func() {
		logWorker("verify", "start swapout verify job")
		for {
			res, err := findSwapoutsToVerify()
			if err != nil {
				logWorkerError("verify", "find swapouts error", err)
			}
			if len(res) > 0 {
				logWorker("verify", "find swapouts to verify", "count", len(res))
			}
			for _, swap := range res {
				err = processSwapoutVerify(swap)
				switch err {
				case nil, tokens.ErrTxNotStable, tokens.ErrTxNotFound:
				default:
					logWorkerError("verify", "process swapout verify error", err, "txid", swap.TxID)
				}
			}
			restInJob(restIntervalInVerifyJob)
		}
	})
}

func findSwapinsToVerify() ([]*mongodb.MgoSwap, error) {
	status := mongodb.TxNotStable
	septime := getSepTimeInFind(maxVerifyLifetime)
	return mongodb.FindSwapinsWithStatus(status, septime)
}

func findSwapoutsToVerify() ([]*mongodb.MgoSwap, error) {
	status := mongodb.TxNotStable
	septime := getSepTimeInFind(maxVerifyLifetime)
	return mongodb.FindSwapoutsWithStatus(status, septime)
}

func isInBlacklist(swapInfo *tokens.TxSwapInfo) (isBlacked bool, err error) {
	isBlacked, err = mongodb.QueryBlacklist(swapInfo.From)
	if err != nil {
		return isBlacked, err
	}
	if !isBlacked && swapInfo.Bind != swapInfo.From {
		isBlacked, err = mongodb.QueryBlacklist(swapInfo.Bind)
		if err != nil {
			return isBlacked, err
		}
	}
	return isBlacked, nil
}

func processSwapinVerify(swap *mongodb.MgoSwap) (err error) {
	return processSwapVerify(swap, true)
}

func processSwapoutVerify(swap *mongodb.MgoSwap) error {
	return processSwapVerify(swap, false)
}

func processSwapVerify(swap *mongodb.MgoSwap, isSwapin bool) (err error) {
	txid := swap.TxID
	var bridge tokens.CrossChainBridge
	if isSwapin {
		bridge = tokens.SrcBridge
	} else {
		bridge = tokens.DstBridge
	}

	var swapInfo *tokens.TxSwapInfo
	switch tokens.SwapTxType(swap.TxType) {
	case tokens.SwapinTx, tokens.SwapoutTx:
		swapInfo, err = bridge.VerifyTransaction(txid, false)
	case tokens.P2shSwapinTx:
		if btc.BridgeInstance == nil {
			return tokens.ErrNoBtcBridge
		}
		swapInfo, err = btc.BridgeInstance.VerifyP2shTransaction(txid, swap.Bind, false)
	default:
		return tokens.ErrWrongSwapinTxType
	}
	if swapInfo.Height != 0 &&
		swapInfo.Height < tokens.GetTokenConfig(isSwapin).InitialHeight {
		err = tokens.ErrTxBeforeInitialHeight
		return mongodb.UpdateSwapinStatus(txid, mongodb.TxVerifyFailed, now(), err.Error())
	}
	isBlacked, errf := isInBlacklist(swapInfo)
	if errf != nil {
		return errf
	}
	if isBlacked {
		err = tokens.ErrAddressIsInBlacklist
		return mongodb.UpdateSwapinStatus(txid, mongodb.SwapInBlacklist, now(), err.Error())
	}
	return updateSwapStatus(txid, swapInfo, isSwapin, err)
}

func updateSwapStatus(txid string, swapInfo *tokens.TxSwapInfo, isSwapin bool, err error) error {
	resultStatus := mongodb.MatchTxEmpty

	switch err {
	case tokens.ErrTxNotStable, tokens.ErrTxNotFound:
		return err
	case nil:
		status := mongodb.TxNotSwapped
		if swapInfo.Value.Cmp(tokens.GetBigValueThreshold(isSwapin)) > 0 {
			status = mongodb.TxWithBigValue
			resultStatus = mongodb.TxWithBigValue
		}
		err = mongodb.UpdateSwapStatus(isSwapin, txid, status, now(), "")
	case tokens.ErrTxWithWrongMemo:
		resultStatus = mongodb.TxWithWrongMemo
		err = mongodb.UpdateSwapStatus(isSwapin, txid, mongodb.TxWithWrongMemo, now(), err.Error())
	case tokens.ErrBindAddrIsContract:
		resultStatus = mongodb.BindAddrIsContract
		err = mongodb.UpdateSwapStatus(isSwapin, txid, mongodb.BindAddrIsContract, now(), err.Error())
	case tokens.ErrTxWithWrongValue:
		resultStatus = mongodb.TxWithWrongValue
		err = mongodb.UpdateSwapStatus(isSwapin, txid, mongodb.TxWithWrongValue, now(), err.Error())
	case tokens.ErrTxSenderNotRegistered:
		return mongodb.UpdateSwapStatus(isSwapin, txid, mongodb.TxSenderNotRegistered, now(), err.Error())
	case tokens.ErrTxWithWrongSender:
		return mongodb.UpdateSwapStatus(isSwapin, txid, mongodb.TxWithWrongSender, now(), err.Error())
	case tokens.ErrTxIncompatible:
		return mongodb.UpdateSwapStatus(isSwapin, txid, mongodb.TxIncompatible, now(), err.Error())
	case tokens.ErrTxWithWrongReceipt:
		return mongodb.UpdateSwapStatus(isSwapin, txid, mongodb.TxVerifyFailed, now(), err.Error())
	case tokens.ErrRPCQueryError:
		return mongodb.UpdateSwapStatus(isSwapin, txid, mongodb.RPCQueryError, now(), err.Error())
	default:
		logWorkerWarn("verify", "maybe not considered tx verify error", "err", err)
		return mongodb.UpdateSwapStatus(isSwapin, txid, mongodb.TxVerifyFailed, now(), err.Error())
	}

	if err != nil {
		logWorkerError("verify", "update swap status", err, "txid", txid, "isSwapin", isSwapin)
		return err
	}
	return addInitialSwapResult(swapInfo, resultStatus, isSwapin)
}
