package btc

import (
	"fmt"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

func (b *BtcBridge) VerifyP2shTransaction(txHash string, bindAddress string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.Hash = txHash // Hash
	if !b.IsSrc {
		return swapInfo, tokens.ErrBridgeDestinationNotSupported
	}
	p2shAddress, _, err := b.GetP2shAddress(bindAddress)
	if err != nil {
		return swapInfo, fmt.Errorf("verify p2sh tx, wrong bind address %v", bindAddress)
	}
	token := b.TokenConfig
	if !allowUnstable {
		txStatus := b.GetTransactionStatus(txHash)
		if txStatus.Block_height == 0 ||
			txStatus.Confirmations < *token.Confirmations {
			return swapInfo, tokens.ErrTxNotStable
		}
	}
	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug("BtcBridge::GetTransaction fail", "tx", txHash, "err", err)
		return swapInfo, tokens.ErrTxNotFound
	}
	txStatus := tx.Status
	if txStatus.Block_height != nil {
		swapInfo.Height = *txStatus.Block_height // Height
	}
	if txStatus.Block_time != nil {
		swapInfo.Timestamp = *txStatus.Block_time // Timestamp
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
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}
	swapInfo.To = p2shAddress                    // To
	swapInfo.Bind = bindAddress                  // Bind
	swapInfo.Value = common.BigFromUint64(value) // Value

	for _, input := range tx.Vin {
		if input != nil &&
			input.Prevout != nil &&
			input.Prevout.Scriptpubkey_address != nil {
			from = *input.Prevout.Scriptpubkey_address
			break
		}
	}
	swapInfo.From = from // From

	// check sender
	if from == b.TokenConfig.DcrmAddress {
		return swapInfo, tokens.ErrTxWithWrongSender
	}

	if !tokens.CheckSwapValue(common.BigFromUint64(value), b.IsSrc) {
		return swapInfo, tokens.ErrTxWithWrongValue
	}

	if !allowUnstable {
		log.Debug("verify p2sh swapin pass", "from", from, "to", p2shAddress, "bind", bindAddress, "value", value, "txid", *tx.Txid, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	}
	return swapInfo, nil
}
