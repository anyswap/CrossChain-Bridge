package xrp

import (
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// VerifyMsgHash verify msg hash
func (b *Bridge) VerifyMsgHash(rawTx interface{}, msgHash []string) (err error) {
	return nil
}

// VerifyTransaction impl
func (b *Bridge) VerifyTransaction(pairID, txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	if !b.IsSrc {
		return nil, tokens.ErrBridgeDestinationNotSupported
	}
	return b.verifySwapinTx(pairID, txHash, allowUnstable)
}

func (b *Bridge) verifySwapinTx(pairID, txHash string, allowUnstable bool) (*tokens.TxSwapInfo, error) {
	return nil, nil
}

func (b *Bridge) checkStable(txHash string) bool {
	return false
}

// GetBindAddressFromMemoScipt get bind address
func GetBindAddressFromMemoScipt(memoScript string) (bind string, ok bool) {
	return "", true
}
