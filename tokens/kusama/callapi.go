package kusama

import (
	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/common/hexutil"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/types"
)

var (
	wrapRPCQueryError = tokens.WrapRPCQueryError
)

// ------------------------ kusama override apis -----------------------------

// KsmHeader struct
type KsmHeader struct {
	ParentHash *common.Hash `json:"parentHash"`
	Number     *hexutil.Big `json:"number"`
}

// GetBlockConfirmations override this method
func (b *Bridge) GetBlockConfirmations(receipt *types.RPCTxReceipt) (uint64, error) {
	latest, err := b.GetFinalizedBlockNumber()
	if err != nil {
		return 0, err
	}
	blockNumber := receipt.BlockNumber.ToInt().Uint64()
	if latest > blockNumber {
		return latest - blockNumber, nil
	}
	return 0, nil
}

// GetFinalizedBlockNumber call chain_getFinalizedHead and chain_getHeader
func (b *Bridge) GetFinalizedBlockNumber() (uint64, error) {
	blockHash, err := b.KsmGetFinalizedHead()
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
func (b *Bridge) KsmGetFinalizedHead() (result *common.Hash, err error) {
	urls := append(b.GatewayConfig.APIAddress, b.GatewayConfig.APIAddressExt...)
	for _, url := range urls {
		err = client.RPCPost(&result, url, "chain_getFinalizedHead")
		if err == nil && result != nil {
			return result, nil
		}
	}
	return nil, wrapRPCQueryError(err, "chain_getFinalizedHead")
}

// KsmGetHeader call chain_getHeader
func (b *Bridge) KsmGetHeader(blockHash string) (result *KsmHeader, err error) {
	urls := append(b.GatewayConfig.APIAddress, b.GatewayConfig.APIAddressExt...)
	for _, url := range urls {
		err = client.RPCPost(&result, url, "chain_getHeader", blockHash)
		if err == nil && result != nil {
			return result, nil
		}
	}
	return nil, wrapRPCQueryError(err, "chain_getHeader", blockHash)
}
