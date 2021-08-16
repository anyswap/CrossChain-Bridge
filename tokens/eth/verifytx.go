package eth

import (
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/types"
)

// GetTransactionStatus impl
func (b *Bridge) GetTransactionStatus(txHash string) (*tokens.TxStatus, error) {
	txStatus := &tokens.TxStatus{}
	txr, url, err := b.GetTransactionReceipt(txHash)
	if err != nil {
		log.Trace("GetTransactionReceipt fail", "hash", txHash, "err", err)
		return txStatus, err
	}
	txStatus.BlockHeight = txr.BlockNumber.ToInt().Uint64()
	txStatus.BlockHash = txr.BlockHash.String()
	if txStatus.BlockHeight != 0 {
		for i := 0; i < 3; i++ {
			latest, errt := b.GetLatestBlockNumberOf(url)
			if errt == nil {
				if latest > txStatus.BlockHeight {
					txStatus.Confirmations = latest - txStatus.BlockHeight
				}
				break
			}
			time.Sleep(1 * time.Second)
		}
	}
	txStatus.Receipt = txr
	return txStatus, nil
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

func getTxByHash(b *Bridge, txHash string, withExt bool) (*types.RPCTransaction, error) {
	gateway := b.GatewayConfig
	tx, err := getTransactionByHash(txHash, gateway.APIAddress)
	if err != nil && withExt && len(gateway.APIAddressExt) > 0 {
		tx, err = getTransactionByHash(txHash, gateway.APIAddressExt)
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

	return b.verifyNativeSwapinTx(swapInfo, allowUnstable, token)
}

func (b *Bridge) verifyNativeSwapinTx(swapInfo *tokens.TxSwapInfo, allowUnstable bool, token *tokens.TokenConfig) (*tokens.TxSwapInfo, error) {
	tx, err := getTxByHash(b, swapInfo.Hash, !allowUnstable)
	if err != nil {
		log.Debug("[verifySwapin] "+b.ChainConfig.BlockChain+" Bridge::GetTransaction fail", "tx", swapInfo.Hash, "err", err)
		return swapInfo, tokens.ErrTxNotFound
	}
	if tx.Recipient == nil { // ignore contract creation tx
		return swapInfo, tokens.ErrTxWithWrongReceiver
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
		log.Info("verify native swapin stable pass",
			"identifier", params.GetIdentifier(), "pairID", swapInfo.PairID,
			"from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind,
			"value", swapInfo.Value, "txid", swapInfo.Hash,
			"height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	}
	return swapInfo, nil
}

func (b *Bridge) getReceipt(swapInfo *tokens.TxSwapInfo, allowUnstable bool) (*types.RPCTxReceipt, error) {
	if !allowUnstable {
		return b.getStableReceipt(swapInfo)
	}
	receipt, _, err := b.GetTransactionReceipt(swapInfo.Hash)
	if err != nil {
		return nil, err
	}
	swapInfo.Height = receipt.BlockNumber.ToInt().Uint64() // Height
	if *receipt.Status != 1 {
		return nil, tokens.ErrTxWithWrongReceipt
	}
	return receipt, nil
}

func (b *Bridge) getStableReceipt(swapInfo *tokens.TxSwapInfo) (*types.RPCTxReceipt, error) {
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
		return nil, tokens.ErrTxBeforeInitialHeight
	}
	if txStatus.Confirmations < *b.GetChainConfig().Confirmations {
		return nil, tokens.ErrTxNotStable
	}
	receipt, ok := txStatus.Receipt.(*types.RPCTxReceipt)
	if !ok || receipt == nil || *receipt.Status != 1 {
		return nil, tokens.ErrTxWithWrongReceipt
	}
	return receipt, nil
}
