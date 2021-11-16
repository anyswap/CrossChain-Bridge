package ltc

import (
	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// VerifyP2shTransaction verify p2sh tx
func (b *Bridge) VerifyP2shTransaction(pairID, txHash, bindAddress string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	if !b.IsSrc {
		return nil, tokens.ErrBridgeDestinationNotSupported
	}
	return b.verifyP2shSwapinTx(pairID, txHash, bindAddress, allowUnstable)
}

func (b *Bridge) verifyP2shSwapinTx(pairID, txHash, bindAddress string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	tokenCfg := b.GetTokenConfig(pairID)
	if tokenCfg == nil {
		return nil, tokens.ErrUnknownPairID
	}
	swapInfo := &tokens.TxSwapInfo{}
	swapInfo.PairID = pairID    // PairID
	swapInfo.Hash = txHash      // Hash
	swapInfo.Bind = bindAddress // Bind
	p2shAddress, _, err := b.GetP2shAddress(bindAddress)
	if err != nil {
		return swapInfo, tokens.ErrWrongP2shBindAddress
	}
	if !allowUnstable && !b.checkStable(txHash) {
		return swapInfo, tokens.ErrTxNotStable
	}
	tx, err := b.GetTransactionByHash(txHash)
	if err != nil {
		log.Debug("[verifyP2sh] "+b.ChainConfig.BlockChain+" Bridge::GetTransaction fail", "tx", txHash, "err", err)
		return swapInfo, tokens.ErrTxNotFound
	}
	txStatus := tx.Status
	if txStatus.BlockHeight != nil {
		swapInfo.Height = *txStatus.BlockHeight // Height
	} else if *tx.Locktime != 0 {
		// tx with locktime should be on chain, prvent DDOS attack
		return swapInfo, tokens.ErrTxNotStable
	}
	if txStatus.BlockTime != nil {
		swapInfo.Timestamp = *txStatus.BlockTime // Timestamp
	}
	value, _, rightReceiver := b.GetReceivedValue(tx.Vout, p2shAddress, p2shType)
	if !rightReceiver {
		return swapInfo, tokens.ErrTxWithWrongReceiver
	}
	swapInfo.To = p2shAddress                      // To
	swapInfo.Value = common.BigFromUint64(value)   // Value
	swapInfo.From = getTxFrom(tx.Vin, p2shAddress) // From

	err = b.checkSwapinInfo(swapInfo)
	if err != nil {
		return swapInfo, err
	}

	if !allowUnstable {
		log.Debug("verify p2sh swapin pass", "pairID", swapInfo.PairID, "from", swapInfo.From, "to", swapInfo.To, "bind", swapInfo.Bind, "value", swapInfo.Value, "txid", swapInfo.Hash, "height", swapInfo.Height, "timestamp", swapInfo.Timestamp)
	}
	return swapInfo, nil
}
