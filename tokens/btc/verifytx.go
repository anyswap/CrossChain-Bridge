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

// GetTransaction impl
func (b *Bridge) GetTransaction(txHash string) (interface{}, error) {
	return b.GetTransactionByHash(txHash)
}

// GetTransactionStatus impl
func (b *Bridge) GetTransactionStatus(txHash string) *tokens.TxStatus {
	txStatus := &tokens.TxStatus{}
	elcstStatus, err := b.GetElectTransactionStatus(txHash)
	if err != nil {
		log.Debug("Bridge::GetElectTransactionStatus fail", "tx", txHash, "err", err)
		return txStatus
	}
	if elcstStatus.BlockHash != nil {
		txStatus.BlockHash = *elcstStatus.BlockHash
	}
	if elcstStatus.BlockTime != nil {
		txStatus.BlockTime = *elcstStatus.BlockTime
	}
	if elcstStatus.BlockHeight != nil {
		txStatus.BlockHeight = *elcstStatus.BlockHeight
		latest, err := b.GetLatestBlockNumber()
		if err != nil {
			log.Debug("Bridge::GetLatestBlockNumber fail", "err", err)
			return txStatus
		}
		if latest > txStatus.BlockHeight {
			txStatus.Confirmations = latest - txStatus.BlockHeight
		}
	}
	return txStatus
}

// VerifyMsgHash verify msg hash
func (b *Bridge) VerifyMsgHash(rawTx interface{}, msgHash string, extra interface{}) error {
	authoredTx, ok := rawTx.(*txauthor.AuthoredTx)
	if !ok {
		return tokens.ErrWrongRawTx
	}
	extras, ok := extra.(*tokens.AllExtras)
	if !ok || extras.BtcExtra == nil {
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

// VerifyTransaction impl
func (b *Bridge) VerifyTransaction(txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	if b.IsSrc {
		return b.verifySwapinTx(txHash, allowUnstable)
	}
	return nil, tokens.ErrBridgeDestinationNotSupported
}

func (b *Bridge) verifySwapinTx(txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.Hash = txHash // Hash
	token := b.TokenConfig
	if !allowUnstable {
		txStatus := b.GetTransactionStatus(txHash)
		if txStatus.BlockHeight == 0 ||
			txStatus.Confirmations < *token.Confirmations {
			return swapInfo, tokens.ErrTxNotStable
		}
	}
	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug("Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return swapInfo, tokens.ErrTxNotFound
	}
	txStatus := tx.Status
	dcrmAddress := token.DcrmAddress
	if txStatus.BlockHeight != nil {
		swapInfo.Height = *txStatus.BlockHeight // Height
	}
	if txStatus.BlockTime != nil {
		swapInfo.Timestamp = *txStatus.BlockTime // Timestamp
	}
	var (
		rightReceiver bool
		value         uint64
		memoScript    string
		from          string
	)
	for _, output := range tx.Vout {
		switch *output.ScriptpubkeyType {
		case "op_return":
			memoScript = *output.ScriptpubkeyAsm
			continue
		case "p2pkh":
			if *output.ScriptpubkeyAddress != dcrmAddress {
				continue
			}
			rightReceiver = true
			value += *output.Value
		}
	}
	if !rightReceiver {
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}
	swapInfo.To = dcrmAddress                    // To
	swapInfo.Value = common.BigFromUint64(value) // Value

	bindAddress, bindOk := getBindAddressFromMemoScipt(memoScript)
	swapInfo.Bind = bindAddress // Bind

	for _, input := range tx.Vin {
		if input != nil &&
			input.Prevout != nil &&
			input.Prevout.ScriptpubkeyAddress != nil {
			from = *input.Prevout.ScriptpubkeyAddress
			break
		}
	}
	swapInfo.From = from // From

	// check sender
	if from == dcrmAddress {
		return swapInfo, tokens.ErrTxWithWrongSender
	}

	if !tokens.CheckSwapValue(common.BigFromUint64(value), b.IsSrc) {
		return swapInfo, tokens.ErrTxWithWrongValue
	}

	// NOTE: must verify memo at last step (as it can be recall)
	if !bindOk {
		log.Debug("wrong memo", "memo", memoScript)
		return swapInfo, tokens.ErrTxWithWrongMemo
	} else if !tokens.DstBridge.IsValidAddress(bindAddress) {
		log.Debug("wrong bind address in memo", "bind", bindAddress)
		return swapInfo, tokens.ErrTxWithWrongMemo
	}

	if !allowUnstable {
		log.Debug("verify swapin pass", "from", from, "to", dcrmAddress, "bind", bindAddress, "value", value, "txid", *tx.Txid, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	}
	return swapInfo, nil
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
