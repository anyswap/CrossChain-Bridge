package eth

import (
	"errors"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/dcrm"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

func (b *EthBridge) DcrmSignTransaction(rawTx interface{}) (interface{}, error) {
	tx, ok := rawTx.(*types.Transaction)
	if !ok {
		return nil, errors.New("wrong raw tx param")
	}
	msgHash := dcrm.Signer.Hash(tx)
	keyID, err := dcrm.DoSign(msgHash.String())
	if err != nil {
		return nil, err
	}

	var rsv string
	retryCount := 10
	retryInterval := 60 * time.Second
	i := 0
	for ; i < retryCount; i++ {
		signStatus, err := dcrm.GetSignStatus(keyID)
		if err != nil {
			time.Sleep(retryInterval)
			continue
		}
		rsv = signStatus.Rsv
		break
	}
	if i == retryCount {
		return nil, errors.New("get sign status failed")
	}

	signature := common.FromHex(rsv)
	signer := dcrm.Signer

	signedTx, err := tx.WithSignature(signer, signature)
	if err != nil {
		return nil, err
	}

	sender, err := types.Sender(signer, signedTx)
	if err != nil {
		return nil, err
	}

	if sender != dcrm.DcrmFromAddress() {
		return nil, err
	}
	return signedTx, err
}
