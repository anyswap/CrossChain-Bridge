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
		return newError(-32001, err.Error())
	}
	return nil
}

var (
	ErrSwapNotFound              = newError(-32002, "Swap is not found")
	ErrSwapinTxNotStable         = newError(-32003, "Swap in tx is not stable")
	ErrSwapinRecallExist         = newError(-32004, "Swap in recall is exist")
	ErrSwapinRecalledOrForbidden = newError(-32005, "Swap in is already recalled or can not recall")
)
