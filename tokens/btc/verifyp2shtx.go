package btc

import (
	"fmt"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

func (b *BtcBridge) VerifyP2shTransaction(txHash string, bindAddress string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	if !b.IsSrc {
		return nil, tokens.ErrBridgeDestinationNotSupported
	}
	p2shAddress, _, err := b.GetP2shAddress(bindAddress)
	if err != nil {
		return nil, fmt.Errorf("verify p2sh tx, wrong bind address %v", bindAddress)
	}
	token := b.TokenConfig
	if !allowUnstable {
		txStatus := b.GetTransactionStatus(txHash)
		if txStatus.Block_height == 0 ||
			txStatus.Confirmations < *token.Confirmations {
			return nil, tokens.ErrTxNotStable
		}
	}
	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug("BtcBridge::GetTransaction fail", "tx", txHash, "err", err)
		return nil, tokens.ErrTxNotFound
	}
	var (
		rightReceiver bool
		value         uint64
		from          string
	)
	for _, output := range tx.Vout {
		switch *output.Scriptpubkey_type {
		case "p2sh":
			if *output.Scriptpubkey_address != p2shAddress {
				continue
			}
			rightReceiver = true
			value += *output.Value
		}
	}
	if !rightReceiver {
		return nil, tokens.ErrTxWithWrongReceiver
	}
	if !tokens.CheckSwapValue(common.BigFromUint64(value), b.IsSrc) {
		return nil, tokens.ErrTxWithWrongValue
	}
	for _, input := range tx.Vin {
		if input != nil &&
			input.Prevout != nil &&
			input.Prevout.Scriptpubkey_address != nil {
			from = *input.Prevout.Scriptpubkey_address
			break
		}
	}
	// check sender
	if from == b.TokenConfig.DcrmAddress {
		return nil, tokens.ErrTxWithWrongSender
	}

	var blockHeight, blockTimestamp uint64
	txStatus := tx.Status
	if txStatus.Block_height != nil {
		blockHeight = *txStatus.Block_height
	}
	if txStatus.Block_time != nil {
		blockTimestamp = *txStatus.Block_time
	}
	if !allowUnstable {
		log.Debug("verify p2sh swapin pass", "from", from, "to", p2shAddress, "bind", bindAddress, "value", value, "txid", *tx.Txid, "height", blockHeight, "timestamp", blockTimestamp)
	}
	return &tokens.TxSwapInfo{
		Hash:      *tx.Txid,
		Height:    blockHeight,
		Timestamp: blockTimestamp,
		From:      from,
		To:        p2shAddress,
		Bind:      bindAddress,
		Value:     common.BigFromUint64(value),
	}, err
}
