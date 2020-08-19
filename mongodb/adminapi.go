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

// AddToBlacklist add to blacklist
func AddToBlacklist(address string) error {
	mb := &MgoBlackAccount{
		Key:       strings.ToLower(address),
		Timestamp: time.Now().Unix(),
	}
	err := collBlacklist.Insert(mb)
	if err == nil {
		log.Info("mongodb add to black list success", "address", address)
	} else {
		log.Info("mongodb add to black list failed", "address", address, "err", err)
	}
	return mgoError(err)
}

// RemoveFromBlacklist remove from blacklist
func RemoveFromBlacklist(address string) error {
	err := collBlacklist.RemoveId(strings.ToLower(address))
	if err == nil {
		log.Info("mongodb remove from black list success", "address", address)
	} else {
		log.Info("mongodb remove from black list failed", "address", address, "err", err)
	}
	return mgoError(err)
}

// QueryBlacklist query if is blacked
func QueryBlacklist(address string) (isBlacked bool, err error) {
	var result MgoBlackAccount
	err = collBlacklist.FindId(strings.ToLower(address)).One(&result)
	if err == nil {
		return true, nil
	}
	if err == mgo.ErrNotFound {
		return false, nil
	}
	return false, err
}

// PassSwapinBigValue pass swapin big value
func PassSwapinBigValue(txid string) error {
	return passBigValue(txid, true)
}

// PassSwapoutBigValue pass swapout big value
func PassSwapoutBigValue(txid string) error {
	return passBigValue(txid, false)
}

func passBigValue(txid string, isSwapin bool) error {
	swap, err := FindSwap(isSwapin, txid)
	if err != nil {
		return err
	}
	if swap.Status != TxWithBigValue {
		return fmt.Errorf("swap status is %v, not big value status %v", swap.Status.String(), TxWithBigValue.String())
	}
	return UpdateSwapStatus(isSwapin, txid, TxNotSwapped, time.Now().Unix(), "")
}

// ReverifySwapin reverify swapin
func ReverifySwapin(txid string) error {
	return reverifySwap(txid, true)
}

// ReverifySwapout reverify swapout
func ReverifySwapout(txid string) error {
	return reverifySwap(txid, false)
}

func reverifySwap(txid string, isSwapin bool) error {
	swap, err := FindSwap(isSwapin, txid)
	if err != nil {
		return err
	}
	if !swap.Status.CanReverify() {
		return fmt.Errorf("swap status is %v, no need to reverify", swap.Status.String())
	}
	return UpdateSwapStatus(isSwapin, txid, TxNotStable, time.Now().Unix(), "")
}

// Reswapin reswapin
func Reswapin(txid string) error {
	return reswap(txid, true)
}

// Reswapout reswapout
func Reswapout(txid string) error {
	return reswap(txid, false)
}

func reswap(txid string, isSwapin bool) error {
	swap, err := FindSwap(isSwapin, txid)
	if err != nil {
		return err
	}
	if !swap.Status.CanReswap() {
		return fmt.Errorf("swap status is %v, can not reswap", swap.Status.String())
	}
	swapResult, err := FindSwapResult(isSwapin, txid)
	if err != nil {
		return err
	}
	err = checkCanReswap(swapResult, isSwapin)
	if err != nil {
		return err
	}

	log.Info("[reswap] update status to TxNotSwapped to retry", "txid", txid, "swaptx", swapResult.SwapTx)
	err = UpdateSwapResultStatus(isSwapin, txid, MatchTxEmpty, time.Now().Unix(), "")
	if err != nil {
		return err
	}

	return UpdateSwapStatus(isSwapin, txid, TxNotSwapped, time.Now().Unix(), "")
}

func checkCanReswap(res *MgoSwapResult, isSwapin bool) error {
	swapType := tokens.SwapType(res.SwapType)
	switch swapType {
	case tokens.SwapinType:
	case tokens.SwapoutType:
	default:
		return fmt.Errorf("swap type is %v, can not reswap", swapType.String())
	}
	switch res.Status {
	case MatchTxNotStable:
	case MatchTxFailed:
	default:
		return fmt.Errorf("swap result status is %v, can not reswap", res.Status.String())
	}
	if res.SwapTx == "" {
		return errors.New("swap without swaptx")
	}
	var bridge tokens.CrossChainBridge
	if isSwapin {
		bridge = tokens.DstBridge
	} else {
		bridge = tokens.SrcBridge
	}
	_, err := bridge.GetTransaction(res.SwapTx)
	if err == nil {
		return errors.New("swaptx exist in chain or pool")
	}
	nonceGetter, ok := bridge.(tokens.NonceGetter)
	if !ok {
		return nil
	}
	tokenCfg := tokens.GetTokenConfig(bridge.IsSrcEndpoint())
	// eth enhanced, if we fail at nonce a, we should retry after nonce a
	// to ensure tx with nonce a is on blockchain to prevent double swapping
	var nonce uint64
	retryGetNonceCount := 3
	for i := 0; i < retryGetNonceCount; i++ {
		nonce, err = nonceGetter.GetPoolNonce(tokenCfg.DcrmAddress, "latest")
		if err == nil {
			break
		}
		log.Warn("get account nonce failed", "address", tokenCfg.DcrmAddress)
		time.Sleep(time.Second)
	}
	if nonce < res.SwapNonce {
		return errors.New("can not retry swap with lower nonce")
	}
	return nil
}

// ManualManageSwap manual manage swap
func ManualManageSwap(txid, memo string, isSwapin, isPass bool) error {
	swap, err := FindSwap(isSwapin, txid)
	if err != nil {
		return err
	}
	if isPass {
		if swap.Status.CanManualMakePass() {
			return UpdateSwapStatus(isSwapin, txid, TxNotSwapped, time.Now().Unix(), memo)
		}
		if swap.Status.CanReverify() {
			return UpdateSwapStatus(isSwapin, txid, TxNotStable, time.Now().Unix(), memo)
		}
	} else if swap.Status.CanManualMakeFail() {
		return UpdateSwapStatus(isSwapin, txid, ManualMakeFail, time.Now().Unix(), memo)
	}
	return fmt.Errorf("swap status is %v, can not operate. txid=%v isSwapin=%v isPass=%v", swap.Status.String(), txid, isSwapin, isPass)
}
