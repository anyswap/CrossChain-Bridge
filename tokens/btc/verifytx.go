package btc

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/tokens/btc/electrs"
)

func (b *BtcBridge) GetTransactionStatus(txHash string) *tokens.TxStatus {
	txStatus := &tokens.TxStatus{}
	elcstStatus, err := b.GetElectTransactionStatus(txHash)
	if err != nil {
		log.Debug("BtcBridge::GetElectTransactionStatus fail", "tx", txHash, "err", err)
		return txStatus
	}
	if elcstStatus.Block_hash != nil {
		txStatus.Block_hash = *elcstStatus.Block_hash
	}
	if elcstStatus.Block_time != nil {
		txStatus.Block_time = *elcstStatus.Block_time
	}
	if elcstStatus.Block_height != nil {
		txStatus.Block_height = *elcstStatus.Block_height
		latest, err := b.GetLatestBlockNumber()
		if err != nil {
			log.Debug("BtcBridge::GetLatestBlockNumber fail", "err", err)
			return txStatus
		}
		if latest > txStatus.Block_height {
			txStatus.Confirmations = latest - txStatus.Block_height
		}
	}
	return txStatus
}

func (b *BtcBridge) getTransactionStatus(txHash string) (txStatus *electrs.ElectTxStatus, isStable bool) {
	var err error
	txStatus, err = b.GetElectTransactionStatus(txHash)
	if err != nil {
		log.Debug("BtcBridge::GetElectTransactionStatus fail", "tx", txHash, "err", err)
		return nil, false
	}
	if txStatus.Confirmed != nil && !*txStatus.Confirmed {
		return nil, false
	}
	latest, err := b.GetLatestBlockNumber()
	if err != nil {
		log.Debug("BtcBridge::GetLatestBlockNumber fail", "err", err)
		return nil, false
	}
	token := b.TokenConfig
	confirmations := *token.Confirmations
	if *txStatus.Block_height+confirmations > latest {
		return nil, false
	}
	return txStatus, true
}

func (b *BtcBridge) VerifyTransaction(txHash string) (*tokens.TxSwapInfo, error) {
	if b.IsSrc {
		return b.verifySwapinTx(txHash)
	}
	return nil, tokens.ErrBridgeDestinationNotSupported
}

func (b *BtcBridge) verifySwapinTx(txHash string) (*tokens.TxSwapInfo, error) {
	txStatus, isStable := b.getTransactionStatus(txHash)
	if !isStable {
		return nil, tokens.ErrTxNotStable
	}
	tx, err := b.GetTransaction(txHash)
	if err != nil {
		log.Debug("BtcBridge::GetTransaction fail", "tx", txHash, "err", err)
		return nil, tokens.ErrTxNotStable
	}
	token := b.TokenConfig
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
		return nil, tokens.ErrTxWithWrongReceiver
	}
	if !tokens.CheckSwapValue(float64(value), b.IsSrc) {
		return nil, tokens.ErrTxWithWrongValue
	}
	// NOTE: must verify memo at last step (as it can be recall)
	bindAddress, ok := getBindAddressFromMemoScipt(memoScript)
	if !ok {
		log.Debug("wrong memo", "memo", memoScript)
		return nil, tokens.ErrTxWithWrongMemo
	}
	if !tokens.DstBridge.IsValidAddress(bindAddress) {
		log.Debug("wrong bind address", "bind", bindAddress)
		return nil, tokens.ErrTxWithWrongMemo
	}
	for _, input := range tx.Vin {
		if input != nil &&
			input.Prevout != nil &&
			input.Prevout.Scriptpubkey_address != nil {
			from = *input.Prevout.Scriptpubkey_address
			break
		}
	}
	log.Debug("verify swapin pass", "from", from, "to", dcrmAddress, "bind", bindAddress, "value", value, "txid", *tx.Txid, "height", *txStatus.Block_height, "timestamp", *txStatus.Block_time)
	return &tokens.TxSwapInfo{
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
