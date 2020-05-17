package btc

import (
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcwallet/wallet/txauthor"
	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
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

func (b *BtcBridge) VerifyMsgHash(rawTx interface{}, msgHash string, extra interface{}) error {
	authoredTx, ok := rawTx.(*txauthor.AuthoredTx)
	if !ok {
		return tokens.ErrWrongRawTx
	}
	extras, ok := extra.(*tokens.AllExtras)
	if !ok {
		return tokens.ErrWrongExtraArgs
	}
	btcExtra := extras.BtcExtra
	if btcExtra.SignIndex == nil {
		return tokens.ErrWrongSignIndex
	}
	idx := *btcExtra.SignIndex
	if idx >= len(authoredTx.PrevScripts) {
		return tokens.ErrWrongSignIndex
	}
	tx := authoredTx.Tx
	pkscript := authoredTx.PrevScripts[idx]
	sigHash, err := txscript.CalcSignatureHash(pkscript, hashType, tx, idx)
	if err != nil {
		return err
	}
	if hex.EncodeToString(sigHash) != msgHash {
		return tokens.ErrMsgHashMismatch
	}
	return nil
}

func (b *BtcBridge) VerifyTransaction(txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	if b.IsSrc {
		return b.verifySwapinTx(txHash, allowUnstable)
	}
	return nil, tokens.ErrBridgeDestinationNotSupported
}

func (b *BtcBridge) verifySwapinTx(txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	token := b.TokenConfig
	if !allowUnstable {
		txStatus := b.GetTransactionStatus(txHash)
		if txStatus.Block_height == 0 ||
			txStatus.Confirmations < *token.Confirmations {
			return nil, tokens.ErrTxNotStable
		}
	}
	tx, err := b.GetTransaction(txHash)
	if err != nil {
		log.Debug("BtcBridge::GetTransaction fail", "tx", txHash, "err", err)
		return nil, tokens.ErrTxNotFound
	}
	txStatus := tx.Status
	dcrmAddress := token.DcrmAddress
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
	if !tokens.CheckSwapValue(common.BigFromUint64(value), b.IsSrc) {
		return nil, tokens.ErrTxWithWrongValue
	}
	for _, input := range tx.Vin {
		if input != nil &&
			input.Prevout != nil &&
			input.Prevout.Scriptpubkey_address != nil {
			from = *input.Prevout.Scriptpubkey_address
			break
		}
	}
	// check sender
	if from == dcrmAddress {
		return nil, tokens.ErrTxWithWrongSender
	}

	// NOTE: must verify memo at last step (as it can be recall)
	bindAddress, ok := getBindAddressFromMemoScipt(memoScript)
	if !ok {
		log.Debug("wrong memo", "memo", memoScript)
		err = tokens.ErrTxWithWrongMemo
	} else if !tokens.DstBridge.IsValidAddress(bindAddress) {
		log.Debug("wrong bind address in memo", "bind", bindAddress)
		err = tokens.ErrTxWithWrongMemo
	}
	var blockHeight, blockTimestamp uint64
	if txStatus.Block_height != nil {
		blockHeight = *txStatus.Block_height
	}
	if txStatus.Block_time != nil {
		blockTimestamp = *txStatus.Block_time
	}
	log.Debug("verify swapin pass", "from", from, "to", dcrmAddress, "bind", bindAddress, "value", value, "txid", *tx.Txid, "height", blockHeight, "timestamp", blockTimestamp)
	return &tokens.TxSwapInfo{
		Hash:      *tx.Txid,
		Height:    blockHeight,
		Timestamp: blockTimestamp,
		From:      from,
		To:        dcrmAddress,
		Bind:      bindAddress,
		Value:     common.BigFromUint64(value),
	}, err
}

func getBindAddressFromMemoScipt(memoScript string) (bind string, ok bool) {
	re := regexp.MustCompile("^OP_RETURN OP_PUSHBYTES_[0-9]* ")
	parts := re.Split(memoScript, -1)
	if len(parts) != 2 {
		return "", false
	}
	memoHex := strings.TrimSpace(parts[1])
	memo := common.FromHex(memoHex)
	if len(memo) <= len(tokens.LockMemoPrefix) {
		return "", false
	}
	if !strings.HasPrefix(string(memo), tokens.LockMemoPrefix) {
		return "", false
	}
	bind = string(memo[len(tokens.LockMemoPrefix):])
	return bind, true
}
