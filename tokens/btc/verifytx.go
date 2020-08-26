package btc

import (
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcwallet/wallet/txauthor"
)

// GetTransaction impl
func (b *Bridge) GetTransaction(txHash string) (interface{}, error) {
	return b.GetTransactionByHash(txHash)
}

// GetTransactionStatus impl
func (b *Bridge) GetTransactionStatus(txHash string) *tokens.TxStatus {
	txStatus := &tokens.TxStatus{}
	electStatus, err := b.GetElectTransactionStatus(txHash)
	if err != nil {
		log.Debug(b.ChainConfig.BlockChain+" Bridge::GetElectTransactionStatus fail", "tx", txHash, "err", err)
		return txStatus
	}
	if !*electStatus.Confirmed {
		return txStatus
	}
	if electStatus.BlockHash != nil {
		txStatus.BlockHash = *electStatus.BlockHash
	}
	if electStatus.BlockTime != nil {
		txStatus.BlockTime = *electStatus.BlockTime
	}
	if electStatus.BlockHeight != nil {
		txStatus.BlockHeight = *electStatus.BlockHeight
		latest, err := b.GetLatestBlockNumber()
		if err != nil {
			log.Debug(b.ChainConfig.BlockChain+" Bridge::GetLatestBlockNumber fail", "err", err)
			return txStatus
		}
		if latest > txStatus.BlockHeight {
			txStatus.Confirmations = latest - txStatus.BlockHeight
		}
	}
	return txStatus
}

// VerifyMsgHash verify msg hash
func (b *Bridge) VerifyMsgHash(pairID string, rawTx interface{}, msgHash []string, extra interface{}) (err error) {
	authoredTx, ok := rawTx.(*txauthor.AuthoredTx)
	if !ok {
		return tokens.ErrWrongRawTx
	}
	for i, preScript := range authoredTx.PrevScripts {
		sigScript := preScript
		if txscript.IsPayToScriptHash(sigScript) {
			sigScript, err = b.getRedeemScriptByOutputScrpit(preScript)
			if err != nil {
				return err
			}
		}
		sigHash, err := txscript.CalcSignatureHash(sigScript, hashType, authoredTx.Tx, i)
		if err != nil {
			return err
		}
		if hex.EncodeToString(sigHash) != msgHash[i] {
			return tokens.ErrMsgHashMismatch
		}
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

func hasLockTimeOrSequence(tx *electrs.ElectTx) bool {
	if *tx.Locktime != 0 {
		return true
	}
	for _, input := range tx.Vin {
		if *input.Sequence != wire.MaxTxInSequenceNum {
			return true
		}
	}
	return false
}

func (b *Bridge) verifySwapinTx(txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	tokenCfg := b.GetTokenConfig(PairID)
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.Hash = txHash // Hash
	if !allowUnstable && !b.checkStable(txHash) {
		return swapInfo, tokens.ErrTxNotStable
	}
	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug(b.ChainConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return swapInfo, tokens.ErrTxNotFound
	}
	txStatus := tx.Status
	if txStatus.BlockHeight != nil {
		swapInfo.Height = *txStatus.BlockHeight // Height
	}
	if txStatus.BlockTime != nil {
		swapInfo.Timestamp = *txStatus.BlockTime // Timestamp
	}
	depositAddress := tokenCfg.DepositAddress
	value, memoScript, rightReceiver := b.getReceivedValue(tx.Vout, depositAddress, anyType)
	if !rightReceiver {
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}
	swapInfo.To = depositAddress                 // To
	swapInfo.Value = common.BigFromUint64(value) // Value

	memoStr, bindAddress, bindOk := getBindAddressFromMemoScipt(memoScript)
	if memoStr == aggregateMemo {
		return swapInfo, tokens.ErrTxIsAggregateTx
	}

	swapInfo.Bind = bindAddress // Bind

	swapInfo.From = getTxFrom(tx.Vin, depositAddress) // From

	// check sender
	if swapInfo.From == swapInfo.To {
		return swapInfo, tokens.ErrTxWithWrongSender
	}

	if !tokens.CheckSwapValue(PairID, swapInfo.Value, b.IsSrc) {
		return swapInfo, tokens.ErrTxWithWrongValue
	}

	if !bindOk {
		log.Debug("wrong memo", "memo", memoScript)
		return swapInfo, tokens.ErrTxWithWrongMemo
	} else if !tokens.DstBridge.IsValidAddress(swapInfo.Bind) {
		log.Debug("wrong bind address in memo", "bind", swapInfo.Bind)
		return swapInfo, tokens.ErrTxWithWrongMemo
	}

	if hasLockTimeOrSequence(tx) {
		return swapInfo, tokens.ErrTxWithLockTimeOrSequence
	}

	if !allowUnstable {
		log.Debug("verify swapin pass", "from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind, "value", swapInfo.Value, "txid", swapInfo.Hash, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	}
	return swapInfo, nil
}

func (b *Bridge) checkStable(txHash string) bool {
	txStatus := b.GetTransactionStatus(txHash)
	confirmations := *b.GetChainConfig().Confirmations
	return txStatus.BlockHeight > 0 && txStatus.Confirmations >= confirmations
}

func (b *Bridge) getReceivedValue(vout []*electrs.ElectTxOut, receiver, pubkeyType string) (value uint64, memoScript string, rightReceiver bool) {
	for _, output := range vout {
		switch *output.ScriptpubkeyType {
		case opReturnType:
			memoScript = *output.ScriptpubkeyAsm
			continue
		case pubkeyType, anyType:
			if *output.ScriptpubkeyAddress != receiver {
				continue
			}
			rightReceiver = true
			value += *output.Value
		}
	}
	return value, memoScript, rightReceiver
}

// return priorityAddress if has it in Vin
// return the first address in Vin if has no priorityAddress
func getTxFrom(vin []*electrs.ElectTxin, priorityAddress string) string {
	from := ""
	for _, input := range vin {
		if input != nil &&
			input.Prevout != nil &&
			input.Prevout.ScriptpubkeyAddress != nil {
			if *input.Prevout.ScriptpubkeyAddress == priorityAddress {
				return priorityAddress
			}
			if from == "" {
				from = *input.Prevout.ScriptpubkeyAddress
			}
		}
	}
	return from
}

func getBindAddressFromMemoScipt(memoScript string) (memoStr, bind string, ok bool) {
	re := regexp.MustCompile("^OP_RETURN OP_PUSHBYTES_[0-9]* ")
	parts := re.Split(memoScript, -1)
	if len(parts) != 2 {
		return "", "", false
	}
	memoHex := strings.TrimSpace(parts[1])
	memo := common.FromHex(memoHex)
	memoStr = string(memo)
	if len(memo) <= len(tokens.LockMemoPrefix) {
		return memoStr, "", false
	}
	if !strings.HasPrefix(string(memo), tokens.LockMemoPrefix) {
		return memoStr, "", false
	}
	bind = string(memo[len(tokens.LockMemoPrefix):])
	return memoStr, bind, true
}
