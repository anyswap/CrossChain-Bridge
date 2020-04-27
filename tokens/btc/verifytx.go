package btc

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/fsn-dev/crossChain-Bridge/log"
	. "github.com/fsn-dev/crossChain-Bridge/tokens"
	. "github.com/fsn-dev/crossChain-Bridge/tokens/btc/electrs"
)

func (b *BtcBridge) getTransactionStatus(txHash string) (txStatus *TxStatus, isStable bool) {
	var err error
	txStatus, err = b.GetTransactionStatus(txHash)
	if err != nil {
		log.Debug("BtcBridge::GetTransactionStatus fail", "tx", txHash, "err", err)
		return nil, false
	}
	if !*txStatus.Confirmed {
		return nil, false
	}
	latest, err := b.GetLatestBlockNumber()
	if err != nil {
		log.Debug("BtcBridge::GetLatestBlockNumber fail", "err", err)
		return nil, false
	}
	token, _ := b.GetTokenAndGateway()
	confirmations := *token.Confirmations
	if *txStatus.Block_height+confirmations > latest {
		return nil, false
	}
	return txStatus, true
}

func (b *BtcBridge) IsTransactionStable(txHash string) bool {
	_, isStable := b.getTransactionStatus(txHash)
	return isStable
}

func (b *BtcBridge) VerifyTransaction(txHash string) (*TxSwapInfo, error) {
	if b.IsSrc {
		return b.verifySwapinTx(txHash)
	}
	return nil, ErrBridgeDestinationNotSupported
}

func (b *BtcBridge) verifySwapinTx(txHash string) (*TxSwapInfo, error) {
	txStatus, isStable := b.getTransactionStatus(txHash)
	if !isStable {
		return nil, ErrTxNotStable
	}
	tx, err := b.GetTransaction(txHash)
	if err != nil {
		log.Debug("BtcBridge::GetTransaction fail", "tx", txHash, "err", err)
		return nil, ErrTxNotStable
	}
	token, _ := b.GetTokenAndGateway()
	dcrmAddress := *token.DcrmAddress
	var (
		rightReceiver bool
		value         uint64
		memoScript    string
		from          string
	)
	for _, output := range tx.Vout {
		switch *output.Scriptpubkey_type {
		case "op_return":
			memoScript = *output.Scriptpubkey_asm
			continue
		case "p2pkh":
			if *output.Scriptpubkey_address != dcrmAddress {
				continue
			}
			rightReceiver = true
			value += *output.Value
		}
	}
	if !rightReceiver {
		return nil, ErrTxWithWrongReceiver
	}
	if value == 0 {
		return nil, ErrTxWithWrongValue
	}
	// NOTE: must verify memo at last step (as it can be recall)
	bindAddress, ok := getBindAddressFromMemoScipt(memoScript)
	if !ok {
		return nil, ErrTxWithWrongMemo
	}
	if !DstBridge.IsValidAddress(bindAddress) {
		return nil, ErrTxWithWrongMemo
	}
	for _, input := range tx.Vin {
		if input != nil &&
			input.Prevout != nil &&
			input.Prevout.Scriptpubkey_address != nil {
			from = *input.Prevout.Scriptpubkey_address
			break
		}
	}
	return &TxSwapInfo{
		Hash:      *tx.Txid,
		Height:    *txStatus.Block_height,
		Timestamp: *txStatus.Block_time,
		From:      from,
		To:        dcrmAddress,
		Bind:      bindAddress,
		Value:     fmt.Sprintf("%d", value),
	}, nil
}

func getBindAddressFromMemoScipt(memoScript string) (bind string, ok bool) {
	re := regexp.MustCompile("^OP_RETURN .*" + tokens.LockMemoPrefix)
	parts := re.Split(memoScript, -1)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1]), true
	}
	return "", false
}
