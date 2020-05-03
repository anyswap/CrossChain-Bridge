package eth

import (
	"github.com/fsn-dev/crossChain-Bridge/dcrm"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

func (b *EthBridge) DcrmSignTransaction(rawTx interface{}) (signedTx interface{}, err error) {
	tx, ok := rawTx.(*types.Transaction)
	if !ok {
		return nil, err
	}
	msgHash := tx.Hash().String()
	res, err := dcrm.DoSign(msgHash)
	return res, err
}
