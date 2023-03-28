package ripple

import (
	"errors"
	"fmt"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/data"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/websockets"
)

var errTxResultType = errors.New("tx type is not data.TxResult")

// VerifyMsgHash verify msg hash
func (b *Bridge) VerifyMsgHash(rawTx interface{}, msgHashes []string) (err error) {
	if len(msgHashes) < 1 {
		return fmt.Errorf("Must provide msg hash")
	}
	tx, ok := rawTx.(data.Transaction)
	if !ok {
		return fmt.Errorf("Ripple tx type error")
	}
	msgHash, msg, err := data.SigningHash(tx)
	if err != nil {
		return fmt.Errorf("Rebuild ripple tx msg error, %w", err)
	}
	msg = append(tx.SigningPrefix().Bytes(), msg...)

	pubkey := tx.GetPublicKey().Bytes()
	isEd := isEd25519Pubkey(pubkey)
	var signContent string
	if isEd {
		signContent = common.ToHex(msg)
	} else {
		signContent = msgHash.String()
	}

	if !strings.EqualFold(signContent, msgHashes[0]) {
		return fmt.Errorf("msg hash not match, recover: %v, claiming: %v", signContent, msgHashes[0])
	}

	return nil
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
		return swapInfo, errTxResultType
	}

	if !txres.Validated {
		return swapInfo, tokens.ErrTxIsNotValidated
	}

	if !allowUnstable {
		h, errf := b.GetLatestBlockNumber()
		if errf != nil {
			return swapInfo, errf
		}

		if h < uint64(txres.TransactionWithMetaData.LedgerSequence)+*b.GetChainConfig().Confirmations {
			return swapInfo, tokens.ErrTxNotStable
		}
		if h < *b.ChainConfig.InitialHeight {
			return swapInfo, tokens.ErrTxBeforeInitialHeight
		}
	}

	// Check tx status
	if !txres.TransactionWithMetaData.MetaData.TransactionResult.Success() {
		return swapInfo, tokens.ErrTxWithWrongStatus
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
		return swapInfo, tokens.ErrWrongMemoBindAddress
	}

	if !txres.TransactionWithMetaData.MetaData.DeliveredAmount.IsPositive() {
		return swapInfo, tokens.ErrTxWithNoPayment
	}
	amt := tokens.ToBits(txres.TransactionWithMetaData.MetaData.DeliveredAmount.Float(), *token.Decimals)

	swapInfo.To = token.DepositAddress                        // To
	swapInfo.From = strings.ToLower(payment.Account.String()) // From
	swapInfo.Bind = bind                                      // Bind
	swapInfo.Value = amt

	err = b.checkSwapinInfo(swapInfo)
	if err != nil {
		return swapInfo, err
	}

	if !allowUnstable {
		log.Info("verify swapin pass", "pairID", swapInfo.PairID, "from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind, "value", swapInfo.Value, "txid", swapInfo.Hash, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	}
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
	} else if !txmeta.MetaData.DeliveredAmount.Issuer.IsZero() {
		return fmt.Errorf("ripple native issuer is not zero")
	}
	return nil
}

// GetBindAddressFromMemos get bind address
func GetBindAddressFromMemos(tx data.Transaction) (bind string, ok bool) {
	for _, memo := range tx.GetBase().Memos {
		bindStr := memo.Memo.MemoData.String() // hex string
		if tokens.DstBridge.IsValidAddress(bindStr) {
			bind = bindStr
			ok = true
			return
		}
		bindBytes := string(memo.Memo.MemoData.Bytes()) // bytes
		if tokens.DstBridge.IsValidAddress(bindBytes) {
			bind = bindBytes
			ok = true
			return
		}
		log.Warn("Bind address is not a valid destination address", "bindStr", bindStr, "bindBytes", bindBytes)
	}
	return "", false
}

func (b *Bridge) checkSwapinInfo(swapInfo *tokens.TxSwapInfo) error {
	token := b.GetTokenConfig(swapInfo.PairID)
	if token == nil {
		return tokens.ErrUnknownPairID
	}
	if strings.EqualFold(swapInfo.From, token.DepositAddress) ||
		strings.EqualFold(swapInfo.From, token.DcrmAddress) {
		return tokens.ErrTxWithWrongSender
	}
	if !tokens.CheckSwapValue(swapInfo, b.IsSrc) {
		return tokens.ErrTxWithWrongValue
	}
	bindAddr := swapInfo.Bind
	if !tokens.DstBridge.IsValidAddress(bindAddr) {
		log.Warn("wrong bind address in swapin", "bind", bindAddr)
		return tokens.ErrWrongMemoBindAddress
	}
	return nil
}
