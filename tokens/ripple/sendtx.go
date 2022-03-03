package ripple

import (
	"fmt"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/data"
)

// SendTransaction send signed tx
func (b *Bridge) SendTransaction(signedTx interface{}) (txHash string, err error) {
	tx, ok := signedTx.(*data.Payment)
	if !ok {
		return "", fmt.Errorf("Send transaction type assertion error")
	}
	for i := 0; i < rpcRetryTimes; i++ {
		for _, r := range b.Remotes {
			resp, err1 := r.Submit(tx)
			if err1 != nil || resp == nil {
				log.Warn("Try sending transaction failed", "error", err1)
				err = err1
				continue
			}
			if !resp.EngineResult.Success() {
				log.Warn("send tx with error result", "result", resp.EngineResult, "message", resp.EngineResultMessage)
			}
			return tx.GetBase().Hash.String(), nil
		}
		time.Sleep(rpcRetryInterval)
	}
	return
}

// DoPostRequest only for test
func DoPostRequest(url, api, reqData string) string {
	apiAddress := url + "/" + api
	res, err := client.RPCRawPost(apiAddress, reqData)
	if err != nil {
		log.Warn("do post request failed", "url", apiAddress, "data", reqData, "err", err)
	}
	return res
}
