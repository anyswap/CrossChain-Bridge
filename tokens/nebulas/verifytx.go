package nebulas

import (
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// GetTransactionStatus impl
func (b *Bridge) GetTransactionStatus(txHash string) (*tokens.TxStatus, error) {
	txr, url, err := b.GetTransactionReceipt(txHash)
	if err != nil {
		log.Trace("GetTransactionReceipt fail", "hash", txHash, "err", err)
		return nil, err
	}

	txStatus := &tokens.TxStatus{}
	txStatus.Receipt = txr
	txStatus.BlockHeight = txr.BlockHeight
	block, err := b.GetBlockByNumber(big.NewInt(int64(txr.BlockHeight)))
	if err != nil {
		return nil, err
	}
	txStatus.BlockHash = block.Hash

	if txStatus.BlockHeight != 0 {
		for i := 0; i < 3; i++ {
			latest, errt := b.Inherit.GetLatestBlockNumberOf(url)
			if errt == nil {
				if latest > txStatus.BlockHeight {
					txStatus.Confirmations = latest - txStatus.BlockHeight
				}
				break
			}
			time.Sleep(1 * time.Second)
		}
	}
	return txStatus, nil
}

// VerifyMsgHash verify msg hash
func (b *Bridge) VerifyMsgHash(rawTx interface{}, msgHashes []string) error {
	tx, ok := rawTx.(*Transaction)
	if !ok {
		return tokens.ErrWrongRawTx
	}
	if len(msgHashes) != 1 {
		return tokens.ErrWrongCountOfMsgHashes
	}
	msgHash := msgHashes[0]
	sigHash, err := tx.HashTransaction()
	if err != nil {
		return err
	}
	if sigHash.String() != msgHash {
		logFunc := log.GetPrintFuncOr(params.IsDebugMode, log.Info, log.Trace)
		logFunc("message hash mismatch", "want", msgHash, "have", sigHash.String(), "tx", tx.String())
		return tokens.ErrMsgHashMismatch
	}
	return nil
}

func getTxByHash(b *Bridge, txHash string, withExt bool) (*TransactionResponse, error) {
	gateway := b.GatewayConfig
	tx, err := b.getTransactionByHash(txHash, gateway.APIAddress)
	if err != nil && withExt && len(gateway.APIAddressExt) > 0 {
		tx, err = b.getTransactionByHash(txHash, gateway.APIAddressExt)
	}
	return tx, err
}

// VerifyTransaction impl
func (b *Bridge) VerifyTransaction(pairID, txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.PairID = pairID                // PairID
	swapInfo.Hash = strings.ToLower(txHash) // Hash

	token := b.GetTokenConfig(pairID)

	if token == nil {
		return swapInfo, tokens.ErrUnknownPairID
	}

	if token.DisableSwap {
		return swapInfo, tokens.ErrSwapIsClosed
	}

	receipt, err := b.getReceipt(swapInfo, allowUnstable)
	if err != nil {
		return swapInfo, err
	}

	if !b.IsSrc {
		return b.verifySwapoutTx(swapInfo, allowUnstable, token, receipt)
	}

	if token.IsErc20() {
		return b.verifyErc20SwapinTx(swapInfo, allowUnstable, token, receipt)
	}

	tx, err := getTxByHash(b, swapInfo.Hash, !allowUnstable)
	if err != nil {
		log.Debug("[verifyNativeSwapin] "+b.ChainConfig.BlockChain+" Bridge::GetTransaction fail", "tx", swapInfo.Hash, "err", err)
		return swapInfo, tokens.ErrTxNotFound
	}
	return b.verifyNativeSwapinTx(swapInfo, allowUnstable, token, tx)
}

func (b *Bridge) verifyNativeSwapinTx(swapInfo *tokens.TxSwapInfo, allowUnstable bool, token *tokens.TokenConfig, tx *TransactionResponse) (*tokens.TxSwapInfo, error) {
	if len(tx.To) == 0 { // ignore contract creation tx
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}

	txRecipient := strings.ToLower(tx.To)
	if !common.IsEqualIgnoreCase(txRecipient, token.DepositAddress) {
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}

	swapInfo.TxTo = txRecipient              // TxTo
	swapInfo.To = txRecipient                // To
	swapInfo.From = strings.ToLower(tx.From) // From
	if tx.Type == TxPayloadBinaryType {
		swapInfo.Bind = string(tx.Data)
	} else {
		payload, err := LoadCallPayload(tx.Data)
		if err != nil {
			return nil, err
		}
		args, err := payload.Arguments()
		if err != nil {
			return nil, err
		}
		if len(args) < 3 {
			return nil, errors.New("faile to parse paylad bind address")
		}
		swapInfo.Bind = args[2].(string)
	}
	value, _ := new(big.Int).SetString(tx.Value, 10)
	swapInfo.Value = value

	err := b.checkSwapinInfo(swapInfo)
	if err != nil {
		return swapInfo, err
	}

	if !allowUnstable {
		log.Info("verify native swapin stable pass",
			"identifier", params.GetIdentifier(), "pairID", swapInfo.PairID,
			"from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind,
			"value", swapInfo.Value, "txid", swapInfo.Hash,
			"height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	}
	return swapInfo, nil
}

func (b *Bridge) getReceipt(swapInfo *tokens.TxSwapInfo, allowUnstable bool) (*TransactionResponse, error) {
	if !allowUnstable {
		return b.getStableReceipt(swapInfo)
	}
	receipt, _, err := b.GetTransactionReceipt(swapInfo.Hash)
	if err != nil {
		log.Error("get tx receipt failed", "hash", swapInfo.Hash, "err", err)
		return nil, err
	}
	swapInfo.Height = receipt.BlockHeight
	if receipt.Status != 1 {
		return nil, tokens.ErrTxWithWrongReceipt
	}
	return receipt, nil
}

func (b *Bridge) getStableReceipt(swapInfo *tokens.TxSwapInfo) (*TransactionResponse, error) {
	txStatus, err := b.GetTransactionStatus(swapInfo.Hash)
	if err != nil {
		return nil, err
	}
	if txStatus.BlockHeight == 0 {
		return nil, tokens.ErrTxNotFound
	}
	swapInfo.Height = txStatus.BlockHeight  // Height
	swapInfo.Timestamp = txStatus.BlockTime // Timestamp
	if txStatus.BlockHeight < *b.ChainConfig.InitialHeight {
		log.Warn("transaction before initial block height",
			"initialHeight", *b.ChainConfig.InitialHeight,
			"blockHeight", txStatus.BlockHeight)
		return nil, tokens.ErrTxBeforeInitialHeight
	}
	if txStatus.Confirmations < *b.GetChainConfig().Confirmations {
		return nil, tokens.ErrTxNotStable
	}
	receipt, ok := txStatus.Receipt.(*TransactionResponse)
	if !ok || receipt.Status != 1 {
		return nil, tokens.ErrTxWithWrongReceipt
	}
	return receipt, nil
}
