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
	// TxNotStable status will be reverify at work/verify, add store in result table
	switch err {
	case nil,
		tokens.ErrTxWithWrongMemo,
		tokens.ErrTxWithWrongValue,
		tokens.ErrBindAddrIsContract:
		return TxNotStable
	case tokens.ErrTxSenderNotRegistered:
		return TxSenderNotRegistered
	case tokens.ErrTxWithWrongSender:
		return TxWithWrongSender
	default:
		log.Warn("[mongodb] maybe not considered tx verify error", "err", err)
		return TxNotStable
	}
}
