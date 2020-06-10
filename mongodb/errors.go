package mongodb

import (
	rpcjson "github.com/gorilla/rpc/v2/json2"
)

func newError(ec rpcjson.ErrorCode, message string) error {
	return &rpcjson.Error{
		Code:    ec,
		Message: message,
	}
}

func mgoError(err error) error {
	if err != nil {
		return newError(-32001, "mgoError: "+err.Error())
	}
	return nil
}

// mongodb special errors
var (
	ErrSwapNotFound              = newError(-32002, "mgoError: Swap is not found")
	ErrSwapinTxNotStable         = newError(-32003, "mgoError: Swap in tx is not stable")
	ErrSwapinRecallExist         = newError(-32004, "mgoError: Swap in recall is exist")
	ErrSwapinRecalledOrForbidden = newError(-32005, "mgoError: Swap in is already recalled or can not recall")
)
