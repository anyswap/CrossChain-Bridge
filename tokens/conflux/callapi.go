package conflux

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

// ------------------------ conflux override apis -----------------------------

// CfxBlock struct
type CfxBlock struct {
	Hash        *common.Hash `json:"hash"`
	ParentHash  *common.Hash `json:"parentHash"`
	EpochNumber *hexutil.Big `json:"epochNumber"`
	BlockNumber *hexutil.Big `json:"blockNumber"`
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

// GetFinalizedBlockNumber call cfx_getBlockByEpochNumber
func (b *Bridge) GetFinalizedBlockNumber() (latest uint64, err error) {
	urls := b.GatewayConfig.FinalizeAPIAddress
	var maxHeight uint64
	for _, url := range urls {
		var result *CfxBlock
		err = client.RPCPost(&result, url, "cfx_getBlockByEpochNumber", "latest_finalized", false)
		if err == nil && result != nil {
			h := result.EpochNumber.ToInt().Uint64()
			if h > maxHeight {
				maxHeight = h
			}
		}
	}
	if maxHeight > 0 {
		return maxHeight, nil
	}
	return 0, wrapRPCQueryError(err, "cfx_getBlockByEpochNumber")
}
