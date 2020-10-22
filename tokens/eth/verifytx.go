package eth

import (
	"strings"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
	"github.com/anyswap/CrossChain-Bridge/types"
)

// GetTransaction impl
func (b *Bridge) GetTransaction(txHash string) (interface{}, error) {
	return b.GetTransactionByHash(txHash)
}

// GetTransactionStatus impl
func (b *Bridge) GetTransactionStatus(txHash string) *tokens.TxStatus {
	var txStatus tokens.TxStatus
	txr, err := b.GetTransactionReceipt(txHash)
	if err != nil {
		log.Trace("GetTransactionReceipt fail", "hash", txHash, "err", err)
		return &txStatus
	}
	txStatus.BlockHeight = txr.BlockNumber.ToInt().Uint64()
	txStatus.BlockHash = txr.BlockHash.String()
	block, err := b.GetBlockByHash(txStatus.BlockHash)
	if err == nil {
		txStatus.BlockTime = block.Time.ToInt().Uint64()
	} else {
		log.Debug("GetBlockByHash fail", "hash", txStatus.BlockHash, "err", err)
	}
	if txStatus.BlockHeight != 0 {
		latest, err := b.GetLatestBlockNumber()
		if err == nil {
			if latest > txStatus.BlockHeight {
				txStatus.Confirmations = latest - txStatus.BlockHeight
			}
		} else {
			log.Debug("GetLatestBlockNumber fail", "err", err)
		}
	}
	txStatus.Receipt = txr
	return &txStatus
}

// VerifyMsgHash verify msg hash
func (b *Bridge) VerifyMsgHash(rawTx interface{}, msgHashes []string) error {
	tx, ok := rawTx.(*types.Transaction)
	if !ok {
		return tokens.ErrWrongRawTx
	}
	if len(msgHashes) != 1 {
		return tokens.ErrWrongCountOfMsgHashes
	}
	msgHash := msgHashes[0]
	signer := b.Signer
	sigHash := signer.Hash(tx)
	if sigHash.String() != msgHash {
		log.Trace("message hash mismatch", "want", msgHash, "have", sigHash.String())
		return tokens.ErrMsgHashMismatch
	}
	return nil
}

// VerifyTransaction impl
func (b *Bridge) VerifyTransaction(pairID, txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	if !b.IsSrc {
		return b.verifySwapoutTxWithPairID(pairID, txHash, allowUnstable)
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

	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug("[verifySwapin] "+b.ChainConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return swapInfo, tokens.ErrTxNotFound
	}

	if tx.Recipient == nil { // ignore contract creation tx
		if token.IsErc20() {
			return swapInfo, tokens.ErrTxWithWrongContract
		}
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}

	if token.IsErc20() {
		return b.verifyErc20SwapinTx(tx, pairID, token, allowUnstable)
	}

	if !allowUnstable {
		_, err = b.getStableReceipt(swapInfo)
		if err != nil {
			return swapInfo, err
		}
	}

	txRecipient := strings.ToLower(tx.Recipient.String())
	if !common.IsEqualIgnoreCase(txRecipient, token.DepositAddress) {
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}

	swapInfo.TxTo = txRecipient                       // TxTo
	swapInfo.To = txRecipient                         // To
	swapInfo.From = strings.ToLower(tx.From.String()) // From
	swapInfo.Bind = swapInfo.From                     // Bind
	swapInfo.Value = tx.Amount.ToInt()                // Value

	err = b.checkSwapinInfo(swapInfo)
	if err != nil {
		return swapInfo, err
	}

	if !allowUnstable {
		log.Debug("verify swapin stable pass", "pairID", swapInfo.PairID, "from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind, "value", swapInfo.Value, "txid", txHash, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	}
	return swapInfo, nil
}

// verifySwapinTx verify swapin (in scan job)
func (b *Bridge) verifySwapinTx(txHash string, allowUnstable bool) (swapInfos []*tokens.TxSwapInfo, errs []error) {
	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug(b.ChainConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		addSwapInfoConsiderError(nil, tokens.ErrTxNotFound, &swapInfos, &errs)
		return swapInfos, errs
	}
	if tx.Recipient == nil { // ignore contract creation tx
		addSwapInfoConsiderError(nil, tokens.ErrTxWithWrongReceiver, &swapInfos, &errs)
		return swapInfos, errs
	}
	txRecipient := strings.ToLower(tx.Recipient.String())
	tokenCfgs, pairIDs := tokens.FindTokenConfig(txRecipient, true)
	if len(pairIDs) == 0 {
		addSwapInfoConsiderError(nil, tokens.ErrTxWithWrongReceiver, &swapInfos, &errs)
		return swapInfos, errs
	}

	for i, pairID := range pairIDs {
		token := tokenCfgs[i]

		if token.IsErc20() {
			swapInfo, errf := b.verifyErc20SwapinTx(tx, pairID, token, allowUnstable)
			addSwapInfoConsiderError(swapInfo, errf, &swapInfos, &errs)
			continue
		}

		if !common.IsEqualIgnoreCase(txRecipient, token.DepositAddress) {
			continue
		}

		swapInfo := &tokens.TxSwapInfo{}
		swapInfo.Hash = txHash                            // Hash
		swapInfo.PairID = pairID                          // PairID
		swapInfo.TxTo = txRecipient                       // TxTo
		swapInfo.To = txRecipient                         // To
		swapInfo.From = strings.ToLower(tx.From.String()) // From
		swapInfo.Bind = swapInfo.From                     // Bind
		swapInfo.Value = tx.Amount.ToInt()                // Value

		if !allowUnstable {
			_, err = b.getStableReceipt(swapInfo)
			if err != nil {
				addSwapInfoConsiderError(swapInfo, err, &swapInfos, &errs)
				continue
			}
		}

		err = b.checkSwapinInfo(swapInfo)
		addSwapInfoConsiderError(swapInfo, err, &swapInfos, &errs)

		if !allowUnstable && err == nil {
			log.Debug("verify swapin stable pass", "pairID", swapInfo.PairID, "from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind, "value", swapInfo.Value, "txid", txHash, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
		}
	}

	return swapInfos, errs
}

func addSwapInfoConsiderError(swapInfo *tokens.TxSwapInfo, err error, swapInfos *[]*tokens.TxSwapInfo, errs *[]error) {
	if !tokens.ShouldRegisterSwapForError(err) {
		return
	}
	*swapInfos = append(*swapInfos, swapInfo)
	*errs = append(*errs, err)
}

func (b *Bridge) getStableReceipt(swapInfo *tokens.TxSwapInfo) (*types.RPCTxReceipt, error) {
	txStatus := b.GetTransactionStatus(swapInfo.Hash)
	swapInfo.Height = txStatus.BlockHeight  // Height
	swapInfo.Timestamp = txStatus.BlockTime // Timestamp
	receipt, ok := txStatus.Receipt.(*types.RPCTxReceipt)
	if !ok || receipt == nil {
		return nil, tokens.ErrTxNotStable
	}
	if *receipt.Status != 1 {
		return nil, tokens.ErrTxWithWrongReceipt
	}
	if txStatus.BlockHeight == 0 ||
		txStatus.Confirmations < *b.GetChainConfig().Confirmations {
		return nil, tokens.ErrTxNotStable
	}
	return receipt, nil
}

func (b *Bridge) checkSwapinInfo(swapInfo *tokens.TxSwapInfo) error {
	if swapInfo.Bind == swapInfo.To {
		return tokens.ErrTxWithWrongSender
	}
	if !tokens.CheckSwapValue(swapInfo.PairID, swapInfo.Value, b.IsSrc) {
		return tokens.ErrTxWithWrongValue
	}
	return b.checkSwapinBindAddress(swapInfo.Bind)
}

func (b *Bridge) checkSwapinBindAddress(bindAddr string) error {
	if !tokens.DstBridge.IsValidAddress(bindAddr) {
		log.Warn("wrong bind address in swapin", "bind", bindAddr)
		return tokens.ErrTxWithWrongMemo
	}
	if !tools.IsAddressRegistered(bindAddr) {
		return tokens.ErrTxSenderNotRegistered
	}
	isContract, err := b.IsContractAddress(bindAddr)
	if err != nil {
		log.Warn("query is contract address failed", "bindAddr", bindAddr, "err", err)
		return tokens.ErrRPCQueryError
	}
	if isContract {
		return tokens.ErrBindAddrIsContract
	}
	return nil
}
