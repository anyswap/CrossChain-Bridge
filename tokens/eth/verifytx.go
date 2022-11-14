package eth

import (
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
	"github.com/anyswap/CrossChain-Bridge/types"
)

// GetTransactionStatus impl
func (b *Bridge) GetTransactionStatus(txHash string) *tokens.TxStatus {
	var txStatus tokens.TxStatus
	txr, _, err := b.GetTransactionReceipt(txHash)
	if err != nil {
		log.Trace("GetTransactionReceipt fail", "hash", txHash, "err", err)
		return &txStatus
	}
	txStatus.BlockHeight = txr.BlockNumber.ToInt().Uint64()
	txStatus.BlockHash = txr.BlockHash.String()
	if txStatus.BlockHeight != 0 {
		for i := 0; i < 3; i++ {
			confirmations, errt := b.Inherit.GetBlockConfirmations(txr)
			if errt == nil {
				txStatus.Confirmations = confirmations
				break
			}
			time.Sleep(1 * time.Second)
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
		logFunc := log.GetPrintFuncOr(params.IsDebugMode, log.Info, log.Trace)
		logFunc("message hash mismatch", "want", msgHash, "have", sigHash.String(), "tx", tx.RawStr())
		return tokens.ErrMsgHashMismatch
	}
	return nil
}

func getTxByHash(b *Bridge, txHash string, withExt bool) (*types.RPCTransaction, error) {
	gateway := b.GatewayConfig
	tx, err := b.getTransactionByHash(txHash, gateway.APIAddress)
	if err != nil && withExt && len(gateway.APIAddressExt) > 0 {
		tx, err = b.getTransactionByHash(txHash, gateway.APIAddressExt)
	}
	return tx, err
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

	if token.IsErc20() {
		return b.verifyErc20SwapinTx(pairID, txHash, allowUnstable, token)
	}

	_, err := b.getReceipt(swapInfo, allowUnstable)
	if err != nil {
		return swapInfo, err
	}

	tx, err := getTxByHash(b, txHash, !allowUnstable)
	if err != nil {
		log.Debug("[verifySwapin] "+b.ChainConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return swapInfo, tokens.ErrTxNotFound
	}
	if tx.Recipient == nil { // ignore contract creation tx
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}

	txRecipient := strings.ToLower(tx.Recipient.String())
	if !common.IsEqualIgnoreCase(txRecipient, token.DepositAddress) {
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}
	if *tx.From == (common.Address{}) {
		return nil, tokens.ErrTxWithWrongSender
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
		log.Info("verify swapin stable pass", "pairID", swapInfo.PairID, "from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind, "value", swapInfo.Value, "txid", txHash, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
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
			swapInfo, errf := b.verifyErc20SwapinTx(pairID, txHash, allowUnstable, token)
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

func (b *Bridge) getReceipt(swapInfo *tokens.TxSwapInfo, allowUnstable bool) (*types.RPCTxReceipt, error) {
	if !allowUnstable {
		return b.getStableReceipt(swapInfo)
	}
	receipt, _, err := b.GetTransactionReceipt(swapInfo.Hash)
	if err != nil {
		log.Error("get tx receipt failed", "hash", swapInfo.Hash, "err", err)
		return nil, err
	}
	swapInfo.Height = receipt.BlockNumber.ToInt().Uint64() // Height
	if !receipt.IsStatusOk() {
		return nil, tokens.ErrTxWithWrongReceipt
	}
	if *receipt.From == (common.Address{}) {
		return nil, tokens.ErrTxWithWrongSender
	}
	return receipt, nil
}

func (b *Bridge) getStableReceipt(swapInfo *tokens.TxSwapInfo) (*types.RPCTxReceipt, error) {
	txStatus := b.GetTransactionStatus(swapInfo.Hash)
	swapInfo.Height = txStatus.BlockHeight  // Height
	swapInfo.Timestamp = txStatus.BlockTime // Timestamp
	if txStatus.BlockHeight == 0 {
		return nil, tokens.ErrTxNotFound
	}
	if txStatus.Confirmations < *b.GetChainConfig().Confirmations {
		return nil, tokens.ErrTxNotStable
	}
	if txStatus.BlockHeight < *b.ChainConfig.InitialHeight {
		return nil, tokens.ErrTxBeforeInitialHeight
	}
	receipt, ok := txStatus.Receipt.(*types.RPCTxReceipt)
	if !ok || !receipt.IsStatusOk() {
		return nil, tokens.ErrTxWithWrongReceipt
	}
	return receipt, nil
}

func (b *Bridge) checkSwapinInfo(swapInfo *tokens.TxSwapInfo) error {
	if swapInfo.Bind == swapInfo.To {
		return tokens.ErrTxWithWrongSender
	}
	if !tokens.CheckSwapValue(swapInfo, b.IsSrc) {
		return tokens.ErrTxWithWrongValue
	}
	token := b.GetTokenConfig(swapInfo.PairID)
	if token == nil {
		return tokens.ErrUnknownPairID
	}
	bindAddr := swapInfo.Bind
	if !tokens.DstBridge.IsValidAddress(bindAddr) {
		log.Warn("wrong bind address in swapin", "bind", bindAddr)
		return tokens.ErrTxWithWrongMemo
	}
	if params.MustRegisterAccount() && !tools.IsAddressRegistered(bindAddr) {
		return tokens.ErrTxSenderNotRegistered
	}
	if params.IsSwapServer &&
		token.ContractAddress != "" &&
		params.CheckBindAddrIsContract() &&
		common.IsEqualIgnoreCase(swapInfo.TxTo, token.ContractAddress) {
		isContract, err := b.IsContractAddress(bindAddr)
		if err != nil {
			log.Warn("query is contract address failed", "bindAddr", bindAddr, "err", err)
			return err
		}
		if isContract {
			return tokens.ErrBindAddrIsContract
		}
	}
	return nil
}
