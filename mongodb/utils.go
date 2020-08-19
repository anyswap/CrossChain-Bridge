package mongodb

import (
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// GetStatusByTokenVerifyError get status by token verify error
func GetStatusByTokenVerifyError(err error) SwapStatus {
	if !tokens.ShouldRegisterSwapForError(err) {
		return TxVerifyFailed
	}
	switch err {
	case nil, tokens.ErrTxWithWrongMemo:
		return TxNotStable
	case tokens.ErrTxWithWrongReceipt:
		return TxVerifyFailed
	case tokens.ErrTxSenderNotRegistered:
		return TxSenderNotRegistered
	case tokens.ErrTxWithWrongSender:
		return TxWithWrongSender
	case tokens.ErrTxWithWrongValue:
		return TxWithWrongValue
	case tokens.ErrTxIncompatible:
		return TxIncompatible
	case tokens.ErrBindAddrIsContract:
		return BindAddrIsContract
	default:
		log.Warn("[mongodb] maybe not considered tx verify error", "err", err)
		return TxNotStable
	}
}
