package arbitrum

import (
	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/tokens/eth/abicoder"
	"github.com/anyswap/CrossChain-Bridge/types"
)

// ------------------------ arbitrum override apis -----------------------------

const arbQueryConfirmationsContract = "0x00000000000000000000000000000000000000C8"

// function getL1Confirmations(bytes32 blockHash) external view returns (uint64 confirmations)
var getL1ConfirmationsFuncHash = common.FromHex("0xe5ca238c")

// GetBlockConfirmations override this method
// call getL1Confirmations to 0x00000000000000000000000000000000000000C8
func (b *Bridge) GetBlockConfirmations(receipt *types.RPCTxReceipt) (uint64, error) {
	res, err := b.CallContract(
		arbQueryConfirmationsContract,
		abicoder.PackDataWithFuncHash(getL1ConfirmationsFuncHash, *receipt.BlockHash),
		"latest",
	)
	if err != nil {
		return 0, err
	}
	return common.GetBigInt(common.FromHex(res), 0, 32).Uint64(), nil
}
