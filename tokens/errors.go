package tokens

import (
	"errors"
)

// common errors
var (
	ErrSwapTypeNotSupported          = errors.New("swap type not supported in this endpoint")
	ErrBridgeSourceNotSupported      = errors.New("bridge source not supported")
	ErrBridgeDestinationNotSupported = errors.New("bridge destination not supported")
	ErrUnknownSwapType               = errors.New("unknown swap type")
	ErrMsgHashMismatch               = errors.New("message hash mismatch")
	ErrWrongCountOfMsgHashes         = errors.New("wrong count of msg hashed")
	ErrWrongRawTx                    = errors.New("wrong raw tx")
	ErrWrongExtraArgs                = errors.New("wrong extra args")
	ErrNoBtcBridge                   = errors.New("no btc bridge exist")
	ErrWrongSwapinTxType             = errors.New("wrong swapin tx type")
	ErrBuildSwapTxInWrongEndpoint    = errors.New("build swap in/out tx in wrong endpoint")
	ErrTxBeforeInitialHeight         = errors.New("transaction before initial block height")
	ErrAddressIsInBlacklist          = errors.New("address is in black list")
	ErrSwapIsClosed                  = errors.New("swap is closed")

	ErrTodo = errors.New("developing: TODO")

	ErrNotFound             = errors.New("not found")
	ErrTxNotFound           = errors.New("tx not found")
	ErrTxNotStable          = errors.New("tx not stable")
	ErrTxWithWrongReceiver  = errors.New("tx with wrong receiver")
	ErrTxWithWrongContract  = errors.New("tx with wrong contract")
	ErrTxWithWrongInput     = errors.New("tx with wrong input data")
	ErrTxWithWrongLogData   = errors.New("tx with wrong log data")
	ErrTxIsAggregateTx      = errors.New("tx is aggregate tx")
	ErrWrongP2shBindAddress = errors.New("wrong p2sh bind address")
	ErrWrongMemoBindAddress = errors.New("wrong memo bind address")
	ErrTxFuncHashMismatch   = errors.New("tx func hash mismatch")
	ErrDepositLogNotFound   = errors.New("deposit log not found or removed")
	ErrSwapoutLogNotFound   = errors.New("swapout log not found or removed")
	ErrUnknownPairID        = errors.New("unknown pair ID")
	ErrBindAddressMismatch  = errors.New("bind address mismatch")
	ErrRPCQueryError        = errors.New("rpc query error")
	ErrWrongSwapValue       = errors.New("wrong swap value")
	ErrTxIncompatible       = errors.New("tx incompatible")
	ErrTxWithWrongReceipt   = errors.New("tx with wrong receipt")
	ErrEstimateGasFailed    = errors.New("estimate gas failed")
	ErrMissTokenPrice       = errors.New("miss token price")
	ErrTxWithWrongSender    = errors.New("tx with wrong sender")
	ErrTxWithWrongStatus    = errors.New("tx with wrong status")
	ErrTxWithNoPayment      = errors.New("tx with no payment")
	ErrTxIsNotValidated     = errors.New("tx is not validated")

	// errors should register
	ErrTxWithWrongMemo       = errors.New("tx with wrong memo")
	ErrTxWithWrongValue      = errors.New("tx with wrong value")
	ErrTxSenderNotRegistered = errors.New("tx sender not registered")
	ErrBindAddrIsContract    = errors.New("bind address is contract")
)

// ShouldRegisterSwapForError return true if this error should record in database
func ShouldRegisterSwapForError(err error) bool {
	switch {
	case err == nil:
	case errors.Is(err, ErrTxWithWrongMemo):
	case errors.Is(err, ErrTxWithWrongValue):
	case errors.Is(err, ErrTxSenderNotRegistered):
	case errors.Is(err, ErrBindAddrIsContract):
	default:
		return false
	}
	return true
}

// IsRPCQueryOrNotFoundError is rpc or not found error
func IsRPCQueryOrNotFoundError(err error) bool {
	return errors.Is(err, ErrRPCQueryError) || errors.Is(err, ErrNotFound)
}
