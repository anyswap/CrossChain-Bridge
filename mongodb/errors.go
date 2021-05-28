package mongodb

import (
	"errors"

	rpcjson "github.com/gorilla/rpc/v2/json2"
	"gopkg.in/mgo.v2"
)

func newError(ec rpcjson.ErrorCode, message string) error {
	return &rpcjson.Error{
		Code:    ec,
		Message: message,
	}
}

func mgoError(err error) error {
	if err != nil {
		if errors.Is(err, mgo.ErrNotFound) {
			return ErrItemNotFound
		}
		if mgo.IsDup(err) {
			return ErrItemIsDup
		}
		return newError(-32001, "mgoError: "+err.Error())
	}
	return nil
}

// mongodb special errors
var (
	ErrItemNotFound       = newError(-32002, "mgoError: Item not found")
	ErrItemIsDup          = newError(-32003, "mgoError: Item is duplicate")
	ErrSwapNotFound       = newError(-32011, "mgoError: Swap is not found")
	ErrWrongKey           = newError(-32012, "mgoError: Wrong key")
	ErrForbidUpdateNonce  = newError(-32013, "mgoError: Forbid update swap nonce")
	ErrForbidUpdateSwapTx = newError(-32014, "mgoError: Forbid update swap tx")
)
