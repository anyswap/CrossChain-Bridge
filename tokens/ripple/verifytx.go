package ripple

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/data"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/websockets"
)

// VerifyMsgHash verify msg hash
func (b *Bridge) VerifyMsgHash(rawTx interface{}, msgHash []string) (err error) {
	tx, ok := rawTx.(data.Transaction)
	if !ok {
		return fmt.Errorf("Ripple tx type error")
	}
	rebuildMsgHash, _, err := data.SigningHash(tx)
	if err != nil {
		return fmt.Errorf("Rebuild ripple tx msg error, %v", err)
	}

	if len(msgHash) < 1 {
		return fmt.Errorf("Must provide msg hash")
	}
	if strings.EqualFold(rebuildMsgHash.String(), msgHash[0]) {
		return nil
	}
	return fmt.Errorf("Msg hash not match, recover: %v, claiming: %v", rebuildMsgHash.String(), msgHash[0])
}

// VerifyTransaction impl
func (b *Bridge) VerifyTransaction(pairID, txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	if !b.IsSrc {
		return nil, tokens.ErrBridgeDestinationNotSupported
	}
	return b.verifySwapinTxWithPairID(pairID, txHash, allowUnstable)
}

func (b *Bridge) verifySwapinTxWithPairID(pairID, txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.PairID = pairID // PairID
	swapInfo.Hash = txHash   // Hash

	token := b.GetTokenConfig(pairID)
	if token == nil {
		return swapInfo, tokens.ErrUnknownPairID
	}

	tx, err := b.GetTransaction(txHash)
	if err != nil {
		log.Debug("[verifySwapin] "+b.ChainConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return swapInfo, tokens.ErrTxNotFound
	}

	txres, ok := tx.(*websockets.TxResult)
	if !ok {
		// unexpected
		return swapInfo, fmt.Errorf("unexpected: tx type is not data.TxResult")
	}

	if !txres.Validated {
		return swapInfo, fmt.Errorf("ripple tx is not validated")
	}

	h, err := b.GetLatestBlockNumber()
	if err != nil {
		return swapInfo, err
	}

	if h-uint64(txres.TransactionWithMetaData.LedgerSequence) < *b.GetChainConfig().Confirmations {
		return swapInfo, fmt.Errorf("ripple ledger height not stable")
	}

	// Check tx status
	if txres.TransactionWithMetaData.MetaData.TransactionResult != 0 {
		return swapInfo, fmt.Errorf("ripple tx status is not success")
	}

	payment, ok := txres.TransactionWithMetaData.Transaction.(*data.Payment)
	if !ok || payment.GetTransactionType() != data.PAYMENT {
		log.Printf("Not a payment transaction")
		return swapInfo, fmt.Errorf("not a payment transaction")
	}

	txRecipient := payment.Destination.String()
	if !common.IsEqualIgnoreCase(txRecipient, token.DepositAddress) {
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}

	err = b.checkToken(pairID, &txres.TransactionWithMetaData)
	if err != nil {
		return swapInfo, err
	}

	bind, ok := GetBindAddressFromMemos(payment)
	if !ok {
		log.Debug("wrong memos", "memos", payment.Memos)
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}

	if !txres.TransactionWithMetaData.MetaData.DeliveredAmount.IsPositive() {
		return swapInfo, fmt.Errorf("payment amount error")
	}
	amt := big.NewInt(int64(txres.TransactionWithMetaData.MetaData.DeliveredAmount.Float() * 1000000))

	swapInfo.To = token.DepositAddress                        // To
	swapInfo.From = strings.ToLower(payment.Account.String()) // From
	swapInfo.Bind = bind                                      // Bind
	swapInfo.Value = amt

	log.Debug("verify swapin pass", "pairID", swapInfo.PairID, "from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind, "value", swapInfo.Value, "txid", swapInfo.Hash, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	return swapInfo, nil
}

func (b *Bridge) checkToken(pairID string, txmeta *data.TransactionWithMetaData) error {
	token := b.GetTokenConfig(pairID)
	if !strings.EqualFold(token.RippleExtra.Currency, txmeta.MetaData.DeliveredAmount.Currency.Machine()) {
		return fmt.Errorf("ripple currency not match")
	}
	if !txmeta.MetaData.DeliveredAmount.Currency.IsNative() {
		if !strings.EqualFold(token.RippleExtra.Issuer, txmeta.MetaData.DeliveredAmount.Issuer.String()) {
			return fmt.Errorf("ripple currency issuer not match")
		}
	}
	return nil
}

// GetBindAddressFromMemos get bind address
func GetBindAddressFromMemos(tx data.Transaction) (bind string, ok bool) {
	for _, memo := range tx.GetBase().Memos {
		bindStr := string(memo.Memo.MemoData) // hex string
		if tokens.DstBridge.IsValidAddress(bindStr) {
			bind = bindStr
			ok = true
			return
		}
		bindBytes := fmt.Sprintf("%X", memo.Memo.MemoType) // bytes
		if tokens.DstBridge.IsValidAddress(bindBytes) {
			bind = bindBytes
			ok = true
			return
		}
		log.Warn("Bind address is not a valid destination address", "bind ascii", bindStr, "bind hex", bindBytes)
	}
	return "", false
}

func addSwapInfoConsiderError(swapInfo *tokens.TxSwapInfo, err error, swapInfos *[]*tokens.TxSwapInfo, errs *[]error) {
	if !tokens.ShouldRegisterSwapForError(err) {
		return
	}
	*swapInfos = append(*swapInfos, swapInfo)
	*errs = append(*errs, err)
}
