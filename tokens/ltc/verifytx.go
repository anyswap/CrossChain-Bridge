package ltc

import (
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
	"github.com/ltcsuite/ltcwallet/wallet/txauthor"
)

var (
	regexMemo = regexp.MustCompile(`^OP_RETURN OP_PUSHBYTES_\d* `)
)

// GetTransaction impl
func (b *Bridge) GetTransaction(txHash string) (interface{}, error) {
	return b.GetTransactionByHash(txHash)
}

// GetTransactionStatus impl
func (b *Bridge) GetTransactionStatus(txHash string) (*tokens.TxStatus, error) {
	txStatus := &tokens.TxStatus{}
	electStatus, err := b.GetElectTransactionStatus(txHash)
	if err != nil {
		log.Trace(b.ChainConfig.BlockChain+" Bridge::GetElectTransactionStatus fail", "tx", txHash, "err", err)
		return txStatus, err
	}
	if !*electStatus.Confirmed {
		return txStatus, tokens.ErrTxNotStable
	}
	if electStatus.BlockHash != nil {
		txStatus.BlockHash = *electStatus.BlockHash
	}
	if electStatus.BlockTime != nil {
		txStatus.BlockTime = *electStatus.BlockTime
	}
	if electStatus.BlockHeight != nil {
		txStatus.BlockHeight = *electStatus.BlockHeight
		latest, errt := b.GetLatestBlockNumber()
		if errt == nil {
			if latest > txStatus.BlockHeight {
				txStatus.Confirmations = latest - txStatus.BlockHeight
			}
		}
	}
	return txStatus, nil
}

// VerifyMsgHash verify msg hash
func (b *Bridge) VerifyMsgHash(rawTx interface{}, msgHash []string) (err error) {
	authoredTx, ok := rawTx.(*txauthor.AuthoredTx)
	if !ok {
		return tokens.ErrWrongRawTx
	}
	for i, preScript := range authoredTx.PrevScripts {
		sigScript := preScript
		if b.IsPayToScriptHash(sigScript) {
			sigScript, err = b.getRedeemScriptByOutputScrpit(preScript)
			if err != nil {
				return err
			}
		}
		sigHash, err := b.CalcSignatureHash(sigScript, authoredTx.Tx, i)
		if err != nil {
			return err
		}
		if hex.EncodeToString(sigHash) != msgHash[i] {
			log.Trace("message hash mismatch", "index", i, "want", msgHash[i], "have", hex.EncodeToString(sigHash))
			return tokens.ErrMsgHashMismatch
		}
	}
	return nil
}

// VerifyTransaction impl
func (b *Bridge) VerifyTransaction(pairID, txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	if !b.IsSrc {
		return nil, tokens.ErrBridgeDestinationNotSupported
	}
	return b.verifySwapinTx(pairID, txHash, allowUnstable)
}

func (b *Bridge) verifySwapinTx(pairID, txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	tokenCfg := b.GetTokenConfig(pairID)
	if tokenCfg == nil {
		return nil, tokens.ErrUnknownPairID
	}
	if tokenCfg.DisableSwap {
		return nil, tokens.ErrSwapIsClosed
	}
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.PairID = pairID // PairID
	swapInfo.Hash = txHash   // Hash
	if !allowUnstable && !b.checkStable(txHash) {
		return swapInfo, tokens.ErrTxNotStable
	}
	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug("[verifySwapin] "+b.ChainConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return swapInfo, tokens.ErrTxNotFound
	}
	txStatus := tx.Status
	if txStatus.BlockHeight != nil {
		swapInfo.Height = *txStatus.BlockHeight // Height
	} else if *tx.Locktime != 0 {
		// tx with locktime should be on chain, prvent DDOS attack
		return swapInfo, tokens.ErrTxNotStable
	}
	if txStatus.BlockTime != nil {
		swapInfo.Timestamp = *txStatus.BlockTime // Timestamp
	}
	depositAddress := tokenCfg.DepositAddress
	value, memoScript, rightReceiver := b.GetReceivedValue(tx.Vout, depositAddress, p2pkhType)
	if !rightReceiver {
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}
	bindAddress, bindOk := GetBindAddressFromMemoScipt(memoScript)

	swapInfo.To = depositAddress                      // To
	swapInfo.Value = common.BigFromUint64(value)      // Value
	swapInfo.Bind = bindAddress                       // Bind
	swapInfo.From = getTxFrom(tx.Vin, depositAddress) // From

	err = b.checkSwapinInfo(swapInfo)
	if err != nil {
		return swapInfo, err
	}
	if !bindOk {
		log.Debug("wrong memo", "memo", memoScript)
		return swapInfo, tokens.ErrTxWithWrongMemo
	}

	if !allowUnstable {
		log.Debug("verify swapin pass", "pairID", swapInfo.PairID, "from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind, "value", swapInfo.Value, "txid", swapInfo.Hash, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	}
	return swapInfo, nil
}

func (b *Bridge) checkSwapinInfo(swapInfo *tokens.TxSwapInfo) error {
	if swapInfo.From == swapInfo.To {
		return tokens.ErrTxWithWrongSender
	}
	if !tokens.CheckSwapValue(swapInfo.PairID, swapInfo.Value, b.IsSrc) {
		return tokens.ErrTxWithWrongValue
	}
	if !tokens.DstBridge.IsValidAddress(swapInfo.Bind) {
		log.Debug("wrong bind address in swapin", "bind", swapInfo.Bind)
		return tokens.ErrTxWithWrongMemo
	}
	return nil
}

func (b *Bridge) checkStable(txHash string) bool {
	txStatus, err := b.GetTransactionStatus(txHash)
	if err != nil {
		return false
	}
	confirmations := *b.GetChainConfig().Confirmations
	return txStatus.BlockHeight > 0 && txStatus.Confirmations >= confirmations
}

// GetReceivedValue get received value
func (b *Bridge) GetReceivedValue(vout []*electrs.ElectTxOut, receiver, pubkeyType string) (value uint64, memoScript string, rightReceiver bool) {
	for _, output := range vout {
		switch *output.ScriptpubkeyType {
		case opReturnType:
			memoScript = *output.ScriptpubkeyAsm
			continue
		case pubkeyType:
			if output.ScriptpubkeyAddress == nil || *output.ScriptpubkeyAddress != receiver {
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

// GetBindAddressFromMemoScipt get bind address
func GetBindAddressFromMemoScipt(memoScript string) (bind string, ok bool) {
	parts := regexMemo.Split(memoScript, -1)
	if len(parts) != 2 {
		return "", false
	}
	memoHex := strings.TrimSpace(parts[1])
	memo := common.FromHex(memoHex)
	memoStr := string(memo)
	if memoStr == tokens.AggregateMemo {
		return "", false
	}
	if len(memo) <= len(tokens.LockMemoPrefix) {
		return "", false
	}
	if !strings.HasPrefix(memoStr, tokens.LockMemoPrefix) {
		return "", false
	}
	bind = string(memo[len(tokens.LockMemoPrefix):])
	return bind, true
}
