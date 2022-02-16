package types

import (
	"github.com/anyswap/CrossChain-Bridge/common"
)

// Hash returns the transaction hash
func (tx *Transaction) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	var h common.Hash
	switch tx.Type() {
	case LegacyTxType:
		h = rlpHash(tx.toLegacyTx())
	case AccessListTxType:
		h = prefixedRlpHash(tx.Type(), tx.toAccessListTx())
	case DynamicFeeTxType:
		h = prefixedRlpHash(tx.Type(), tx.toDynamicFeeTx())
	}
	if h != common.EmptyHash {
		tx.hash.Store(h)
	}
	return h
}
