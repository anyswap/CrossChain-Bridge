package mongodb

import (
	"errors"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// GetStatusByTokenVerifyError get status by token verify error
func GetStatusByTokenVerifyError(err error) SwapStatus {
	if !tokens.ShouldRegisterSwapForError(err) {
		return TxVerifyFailed
	}
	// TxNotStable status will be reverify at work/verify, add store in result table
	switch {
	case err == nil,
		errors.Is(err, tokens.ErrTxWithWrongMemo),
		errors.Is(err, tokens.ErrTxWithWrongValue),
		errors.Is(err, tokens.ErrBindAddrIsContract):
		return TxNotStable
	case errors.Is(err, tokens.ErrTxSenderNotRegistered):
		return TxSenderNotRegistered
	case errors.Is(err, tokens.ErrTxWithWrongSender):
		return TxWithWrongSender
	default:
		log.Warn("[mongodb] maybe not considered tx verify error", "err", err)
		return TxNotStable
	}
}
