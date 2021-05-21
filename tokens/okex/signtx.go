package okex

import (
	"crypto/sha256"

	okextypes "github.com/okex/exchain/x/evm/types"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/tools/rlp"
	"github.com/anyswap/CrossChain-Bridge/types"
)

// CalcTransactionHashImpl calc tx hash
func (b *Bridge) CalcTransactionHashImpl(tx *types.Transaction) (txHash string, err error) {
	txData, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return "", err
	}

	oktx := new(okextypes.MsgEthereumTx)
	err = rlp.DecodeBytes(txData, &oktx.Data)
	if err != nil {
		return "", err
	}

	txBytes, err := okextypes.ModuleCdc.MarshalBinaryLengthPrefixed(oktx)
	if err != nil {
		return "", err
	}

	txHash = common.Hash(sha256.Sum256(txBytes)).Hex()
	return txHash, nil
}
