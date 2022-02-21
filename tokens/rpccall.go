package tokens

import (
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/rpc/client"
)

// WrapRPCQueryError wrap rpc error
func WrapRPCQueryError(err error, method string, params ...interface{}) error {
	if err == nil {
		return fmt.Errorf("call '%s %v' failed, err='%w'", method, params, ErrNotFound)
	}
	return fmt.Errorf("%w: call '%s %v' failed, err='%v'", ErrRPCQueryError, method, params, err)
}

// RPCCall common RPC calling
func RPCCall(result interface{}, urls []string, method string, params ...interface{}) (err error) {
	for _, url := range urls {
		err = client.RPCPost(&result, url, method, params...)
		if err == nil {
			return nil
		}
	}
	return WrapRPCQueryError(err, method, params...)
}

// RPCCallWithTimeout common RPC calling with specified timeout
func RPCCallWithTimeout(timeout int, result interface{}, urls []string, method string, params ...interface{}) (err error) {
	for _, url := range urls {
		err = client.RPCPostWithTimeout(timeout, &result, url, method, params...)
		if err == nil {
			return nil
		}
	}
	return WrapRPCQueryError(err, method, params...)
}
