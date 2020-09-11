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
		log.Debug("GetTransactionReceipt fail", "hash", txHash, "err", err)
		return &txStatus
	}
	if *txr.Status != 1 {
		log.Debug("transaction with wrong receipt status", "hash", txHash, "status", txr.Status)
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
func (b *Bridge) VerifyMsgHash(rawTx interface{}, msgHashes []string, extra interface{}) error {
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
		return tokens.ErrMsgHashMismatch
	}
	return nil
}

// VerifyTransaction impl
func (b *Bridge) VerifyTransaction(txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	if !b.IsSrc {
		return b.verifySwapoutTx(txHash, allowUnstable)
	}
	return b.verifySwapinTx(txHash, allowUnstable)
}

func (b *Bridge) verifySwapinTx(txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	if b.TokenConfig.IsErc20() {
		return b.verifyErc20SwapinTx(txHash, allowUnstable)
	}

	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.Hash = txHash // Hash
	token := b.TokenConfig

	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug("[verifySwapin] "+b.TokenConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return swapInfo, tokens.ErrTxNotFound
	}
	if tx.BlockNumber != nil {
		swapInfo.Height = tx.BlockNumber.ToInt().Uint64() // Height
	}
	if tx.Recipient != nil {
		swapInfo.To = strings.ToLower(tx.Recipient.String()) // To
	}
	swapInfo.From = strings.ToLower(tx.From.String()) // From
	swapInfo.Bind = swapInfo.From                     // Bind
	swapInfo.Value = tx.Amount.ToInt()                // Value

	if !allowUnstable {
		txStatus := b.GetTransactionStatus(txHash)
		swapInfo.Height = txStatus.BlockHeight  // Height
		swapInfo.Timestamp = txStatus.BlockTime // Timestamp
		receipt, ok := txStatus.Receipt.(*types.RPCTxReceipt)
		if !ok || receipt == nil {
			return swapInfo, tokens.ErrTxNotStable
		}
		if *receipt.Status != 1 {
			return swapInfo, tokens.ErrTxWithWrongReceipt
		}
		if txStatus.BlockHeight == 0 ||
			txStatus.Confirmations < *token.Confirmations {
			return swapInfo, tokens.ErrTxNotStable
		}
	}

	if !common.IsEqualIgnoreCase(swapInfo.To, token.DepositAddress) {
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}

	// check sender
	if swapInfo.From == swapInfo.To {
		return swapInfo, tokens.ErrTxWithWrongSender
	}

	if !tokens.CheckSwapValue(swapInfo.Value, b.IsSrc) {
		return swapInfo, tokens.ErrTxWithWrongValue
	}

	err = b.checkSwapinBindAddress(swapInfo.Bind)
	if err != nil {
		return swapInfo, err
	}

	log.Debug("verify swapin stable pass", "from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind, "value", swapInfo.Value, "txid", txHash, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	return swapInfo, nil
}

func (b *Bridge) checkSwapinBindAddress(bindAddr string) error {
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
