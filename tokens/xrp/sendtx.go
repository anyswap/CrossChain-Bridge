package xrp

import (
	"fmt"
	"time"

	"github.com/rubblelabs/ripple/data"
)

// SendTransaction send signed tx
func (b *Bridge) SendTransaction(signedTx interface{}) (txHash string, err error) {
	tx, ok := signedTx.(data.Transaction)
	if !ok {
		return "", fmt.Errorf("Send transaction type assertion error")
	}
	for i := 0; i < rpcRetryTimes; i++ {
		for _, r := range b.Remotes {
			resp, err1 := r.Submit(tx)
			if err1 != nil || resp == nil {
				err = err1
				continue
			}
			if resp.EngineResult == 0 {
				return tx.GetBase().Hash.String(), nil
			}
		}
		time.Sleep(rpcRetryInterval)
	}
	return
}
