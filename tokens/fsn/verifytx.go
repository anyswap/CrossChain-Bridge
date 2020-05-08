package fsn

import (
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

func (b *FsnBridge) GetTransactionStatus(txHash string) *tokens.TxStatus {
	var txStatus tokens.TxStatus
	txr, err := b.GetTransactionReceipt(txHash)
	if err != nil {
		log.Debug("GetTransactionReceipt fail", "hash", txHash, "err", err)
		return &txStatus
	}
	txStatus.Block_height = txr.BlockNumber.ToInt().Uint64()
	txStatus.Block_hash = txr.BlockHash.String()
	block, err := b.GetBlockByHash(txStatus.Block_hash)
	if err == nil {
		txStatus.Block_time = block.Time.ToInt().Uint64()
	} else {
		log.Debug("GetBlockByHash fail", "hash", txStatus.Block_hash, "err", err)
	}
	if *txr.Status == 1 {
		latest, err := b.GetLatestBlockNumber()
		if err == nil {
			txStatus.Confirmations = latest - txStatus.Block_height
		} else {
			log.Debug("GetLatestBlockNumber fail", "err", err)
		}
	}
	return &txStatus
}

func (b *FsnBridge) VerifyMsgHash(rawTx interface{}, msgHash string) error {
	tx, ok := rawTx.(*types.Transaction)
	if !ok {
		return tokens.ErrWrongRawTx
	}
	signer := b.Signer
	sigHash := signer.Hash(tx)
	if sigHash.String() != msgHash {
		return tokens.ErrMsgHashMismatch
	}
	return nil
}

func (b *FsnBridge) VerifyTransaction(txHash string) (*tokens.TxSwapInfo, error) {
	return nil, tokens.ErrTodo
}
