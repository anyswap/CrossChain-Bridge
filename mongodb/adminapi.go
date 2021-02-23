package mongodb

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"gopkg.in/mgo.v2"
)

// --------------- blacklist --------------------------------

func getBlacklistKey(address, pairID string) string {
	return strings.ToLower(address + ":" + pairID)
}

// AddToBlacklist add to blacklist
func AddToBlacklist(address, pairID string) error {
	mb := &MgoBlackAccount{
		Key:       getBlacklistKey(address, pairID),
		Address:   strings.ToLower(address),
		PairID:    strings.ToLower(pairID),
		Timestamp: time.Now().Unix(),
	}
	err := collBlacklist.Insert(mb)
	if err == nil {
		log.Info("mongodb add to black list success", "address", address, "pairID", pairID)
	} else {
		log.Info("mongodb add to black list failed", "address", address, "pairID", pairID, "err", err)
	}
	return mgoError(err)
}

// RemoveFromBlacklist remove from blacklist
func RemoveFromBlacklist(address, pairID string) error {
	err := collBlacklist.RemoveId(getBlacklistKey(address, pairID))
	if err == nil {
		log.Info("mongodb remove from black list success", "address", address, "pairID", pairID)
	} else {
		log.Info("mongodb remove from black list failed", "address", address, "pairID", pairID, "err", err)
	}
	return mgoError(err)
}

// QueryBlacklist query if is blacked
func QueryBlacklist(address, pairID string) (isBlacked bool, err error) {
	var result MgoBlackAccount
	err = collBlacklist.FindId(getBlacklistKey(address, pairID)).One(&result)
	if err == nil {
		return true, nil
	}
	if err == mgo.ErrNotFound {
		return false, nil
	}
	return false, err
}

// PassSwapinBigValue pass swapin big value
func PassSwapinBigValue(txid, pairID, bind string) error {
	return passBigValue(txid, pairID, bind, true)
}

// PassSwapoutBigValue pass swapout big value
func PassSwapoutBigValue(txid, pairID, bind string) error {
	return passBigValue(txid, pairID, bind, false)
}

func passBigValue(txid, pairID, bind string, isSwapin bool) error {
	swap, err := FindSwap(isSwapin, txid, pairID, bind)
	if err != nil {
		return err
	}
	if swap.Status != TxWithBigValue {
		return fmt.Errorf("swap status is %v, not big value status %v", swap.Status.String(), TxWithBigValue.String())
	}
	return UpdateSwapStatus(isSwapin, txid, pairID, bind, TxNotSwapped, time.Now().Unix(), "")
}

// ReverifySwapin reverify swapin
func ReverifySwapin(txid, pairID, bind string) error {
	return reverifySwap(txid, pairID, bind, true)
}

// ReverifySwapout reverify swapout
func ReverifySwapout(txid, pairID, bind string) error {
	return reverifySwap(txid, pairID, bind, false)
}

func reverifySwap(txid, pairID, bind string, isSwapin bool) error {
	swap, err := FindSwap(isSwapin, txid, pairID, bind)
	if err != nil {
		return err
	}
	if !swap.Status.CanReverify() {
		return fmt.Errorf("swap status is %v, no need to reverify", swap.Status.String())
	}
	return UpdateSwapStatus(isSwapin, txid, pairID, bind, TxNotStable, time.Now().Unix(), "")
}

// Reswapin reswapin
func Reswapin(txid, pairID, bind, forceOpt string) error {
	return reswap(txid, pairID, bind, forceOpt, true)
}

// Reswapout reswapout
func Reswapout(txid, pairID, bind, forceOpt string) error {
	return reswap(txid, pairID, bind, forceOpt, false)
}

func reswap(txid, pairID, bind, forceOpt string, isSwapin bool) error {
	swap, err := FindSwap(isSwapin, txid, pairID, bind)
	if err != nil {
		return err
	}
	if !swap.Status.CanReswap() {
		return fmt.Errorf("swap status is %v, can not reswap", swap.Status.String())
	}
	swapResult, err := FindSwapResult(isSwapin, txid, pairID, bind)
	if err != nil {
		return err
	}
	err = checkCanReswap(swapResult, forceOpt, isSwapin)
	if err != nil {
		return err
	}

	log.Info("[reswap] update status to TxNotSwapped to retry", "txid", txid, "pairID", pairID, "bind", bind, "swaptx", swapResult.SwapTx)
	err = UpdateSwapResultStatus(isSwapin, txid, pairID, bind, MatchTxEmpty, time.Now().Unix(), "")
	if err != nil {
		return err
	}

	return UpdateSwapStatus(isSwapin, txid, pairID, bind, TxNotSwapped, time.Now().Unix(), "")
}

func checkCanReswap(res *MgoSwapResult, forceOpt string, isSwapin bool) error {
	swapType := tokens.SwapType(res.SwapType)
	switch swapType {
	case tokens.SwapinType:
	case tokens.SwapoutType:
	default:
		return fmt.Errorf("swap type is %v, can not reswap", swapType.String())
	}
	switch res.Status {
	case TxSwapFailed:
	case MatchTxNotStable:
	case MatchTxFailed:
	default:
		return fmt.Errorf("swap result status is %v, can not reswap", res.Status.String())
	}
	if res.SwapTx == "" {
		return errors.New("swap without swaptx")
	}
	bridge := tokens.GetCrossChainBridge(!isSwapin)
	_, err := bridge.GetTransaction(res.SwapTx)
	if err == nil && res.Status != MatchTxFailed {
		return errors.New("swaptx exist in chain or pool")
	}
	if err != nil && res.Status == MatchTxFailed {
		return errors.New("failed swaptx not exist in chain or pool")
	}
	return checkReswapNonce(bridge, res, forceOpt)
}

func checkReswapNonce(bridge tokens.CrossChainBridge, res *MgoSwapResult, forceOpt string) (err error) {
	const forceFlag = "--force"
	if forceOpt == forceFlag {
		return nil
	}
	nonceSetter, ok := bridge.(tokens.NonceSetter)
	if !ok {
		return nil
	}
	tokenCfg := bridge.GetTokenConfig(res.PairID)
	if tokenCfg == nil {
		return fmt.Errorf("no token config for pairID '%v'", res.PairID)
	}
	// eth enhanced, if we fail at nonce a, we should retry after nonce a
	// to ensure tx with nonce a is on blockchain to prevent double swapping
	var nonce uint64
	retryGetNonceCount := 3
	for i := 0; i < retryGetNonceCount; i++ {
		nonce, err = nonceSetter.GetPoolNonce(tokenCfg.DcrmAddress, "latest")
		if err == nil {
			break
		}
		log.Warn("get account nonce failed", "address", tokenCfg.DcrmAddress)
		time.Sleep(time.Second)
	}
	if nonce <= res.SwapNonce {
		return errors.New("can not retry swap with lower nonce")
	}
	return nil
}

// ManualManageSwap manual manage swap
func ManualManageSwap(txid, pairID, bind, memo string, isSwapin, isPass bool) error {
	swap, err := FindSwap(isSwapin, txid, pairID, bind)
	if err != nil {
		return err
	}
	if isPass {
		if swap.Status.CanManualMakePass() {
			return UpdateSwapStatus(isSwapin, txid, pairID, bind, TxNotSwapped, time.Now().Unix(), memo)
		}
		if swap.Status.CanReverify() {
			return UpdateSwapStatus(isSwapin, txid, pairID, bind, TxNotStable, time.Now().Unix(), memo)
		}
	} else if swap.Status.CanManualMakeFail() {
		return UpdateSwapStatus(isSwapin, txid, pairID, bind, ManualMakeFail, time.Now().Unix(), memo)
	}
	return fmt.Errorf("swap status is %v, can not operate. txid=%v pairID=%v bind=%v isSwapin=%v isPass=%v", swap.Status.String(), txid, pairID, bind, isSwapin, isPass)
}
