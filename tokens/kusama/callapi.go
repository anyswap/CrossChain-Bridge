package kusama

import (
	"errors"
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/common/hexutil"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	errNotFound = errors.New("not found")
)

func wrapRPCQueryError(err error, method string, params ...interface{}) error {
	if err == nil {
		err = errNotFound
	}
	return fmt.Errorf("%w: call '%s %v' failed, err='%v'", tokens.ErrRPCQueryError, method, params, err)
}

// ------------------------ kusama override apis -----------------------------

// GetLatestBlockNumberOf call eth_blockNumber
func (b *Bridge) GetLatestBlockNumberOf(url string) (latest uint64, err error) {
	blockHash, err := b.KsmGetFinalizedHead(url)
	if err != nil {
		return 0, err
	}
	header, err := b.KsmGetHeader(blockHash.String())
	if err != nil {
		return 0, err
	}
	return header.Number.ToInt().Uint64(), nil
}

// ------------------------ kusama specific apis -----------------------------

// KsmGetFinalizedHead call chain_getFinalizedHead
func (b *Bridge) KsmGetFinalizedHead(url string) (result common.Hash, err error) {
	err = client.RPCPost(&result, url, "chain_getFinalizedHead")
	if err == nil {
		return result, nil
	}
	return result, wrapRPCQueryError(err, "chain_getFinalizedHead")
}

// KsmHeader struct
type KsmHeader struct {
	ParentHash *common.Hash `json:"parentHash"`
	Number     *hexutil.Big `json:"number"`
}

// KsmGetHeader call chain_getHeader
func (b *Bridge) KsmGetHeader(blockHash string) (result *KsmHeader, err error) {
	gateway := b.GatewayConfig
	result, err = b.ksmGetHeader(blockHash, gateway.APIAddress)
	if err != nil && len(gateway.APIAddressExt) > 0 {
		result, err = b.ksmGetHeader(blockHash, gateway.APIAddressExt)
	}
	return result, err
}

func (b *Bridge) ksmGetHeader(blockHash string, urls []string) (result *KsmHeader, err error) {
	for _, url := range urls {
		err = client.RPCPost(&result, url, "chain_getHeader", blockHash)
		if err == nil && result != nil {
			return result, nil
		}
	}
	return nil, wrapRPCQueryError(err, "chain_getHeader", blockHash)
}
