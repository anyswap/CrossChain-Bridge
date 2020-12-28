package xrp

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/rubblelabs/ripple/data"
)

// VerifyMsgHash verify msg hash
func (b *Bridge) VerifyMsgHash(rawTx interface{}, msgHash []string) (err error) {
	tx, ok := rawTx.(data.Transaction)
	if !ok {
		return fmt.Errorf("Type assertion error, tx is not xrp transaction")
	}
	if len(msgHash) < 1 {
		return fmt.Errorf("Must provide msg hash")
	}
	if strings.EqualFold(tx.GetHash().String(), msgHash[0]) {
		return nil
	}
	return fmt.Errorf("Msg hash not match, recover: %v, claiming: %v", tx.GetHash().String(), msgHash[0])
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

	if !allowUnstable && !b.checkStable(txHash) {
		return swapInfo, tokens.ErrTxNotStable
	}

	tx, err := b.GetTransaction(txHash)
	if err != nil {
		log.Debug("[verifySwapin] "+b.ChainConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return swapInfo, tokens.ErrTxNotFound
	}

	xrpTx, ok := tx.(data.Transaction)
	if !ok || xrpTx.GetTransactionType() != data.PAYMENT {
		return swapInfo, fmt.Errorf("Not a payment transaction")
	}

	payment := tx.(data.Payment)

	txRecipient := payment.Destination.String()
	if !common.IsEqualIgnoreCase(txRecipient, token.DepositAddress) {
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}

	bind, ok := GetBindAddressFromMemos(xrpTx)
	if !ok {
		log.Debug("wrong memoa", "memos", payment.Memos)
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}

	if payment.Amount.Currency.Machine() != "XRP" {
		log.Warn("Ripple payment currency is not XRP", "currency", payment.Amount.Currency.Machine())
		return nil, fmt.Errorf("Ripple payment currency is not XRP, currency: %v", payment.Amount.Currency)
	}

	// TODO check issuer

	amt := big.NewInt(int64(payment.Amount.Float() * 1000000))

	swapInfo.To = token.DepositAddress                        // To
	swapInfo.From = strings.ToLower(payment.Account.String()) // From
	swapInfo.Bind = bind                                      // Bind
	swapInfo.Value = amt

	if !allowUnstable {
		log.Debug("verify swapin pass", "pairID", swapInfo.PairID, "from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind, "value", swapInfo.Value, "txid", swapInfo.Hash, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	}
	return nil, nil
}

func (b *Bridge) checkStable(txHash string) bool {
	// always true
	return true
}

// GetBindAddressFromMemos get bind address
func GetBindAddressFromMemos(tx data.Transaction) (bind string, ok bool) {
	for _, memo := range tx.GetBase().Memos {
		if strings.EqualFold(memo.Memo.MemoType.String(), "BIND") {
			bind = memo.Memo.MemoData.String()
			if tokens.DstBridge.IsValidAddress(bind) {
				ok = true
				return
			}
		}
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
