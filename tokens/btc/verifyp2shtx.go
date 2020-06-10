package btc

import (
	"fmt"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

// VerifyP2shTransaction verify p2sh tx
func (b *Bridge) VerifyP2shTransaction(txHash string, bindAddress string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
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
		if txStatus.BlockHeight == 0 ||
			txStatus.Confirmations < *token.Confirmations {
			return swapInfo, tokens.ErrTxNotStable
		}
	}
	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug(b.TokenConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return swapInfo, tokens.ErrTxNotFound
	}
	txStatus := tx.Status
	if txStatus.BlockHeight != nil {
		swapInfo.Height = *txStatus.BlockHeight // Height
	}
	if txStatus.BlockTime != nil {
		swapInfo.Timestamp = *txStatus.BlockTime // Timestamp
	}
	var (
		rightReceiver bool
		value         uint64
		from          string
	)
	for _, output := range tx.Vout {
		switch *output.ScriptpubkeyType {
		case "p2sh":
			if *output.ScriptpubkeyAddress != p2shAddress {
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
			input.Prevout.ScriptpubkeyAddress != nil {
			from = *input.Prevout.ScriptpubkeyAddress
			break
		}
	}
	swapInfo.From = from // From

	// check sender
	if swapInfo.From == swapInfo.To {
		return swapInfo, tokens.ErrTxWithWrongSender
	}

	if !tokens.CheckSwapValue(swapInfo.Value, b.IsSrc) {
		return swapInfo, tokens.ErrTxWithWrongValue
	}

	if !allowUnstable {
		log.Debug("verify p2sh swapin pass", "from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind, "value", swapInfo.Value, "txid", swapInfo.Hash, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	}
	return swapInfo, nil
}
