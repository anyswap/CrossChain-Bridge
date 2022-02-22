package ripple

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/data"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/websockets"
)

// SendTransaction send signed tx
func (b *Bridge) SendTransaction(signedTx interface{}) (txHash string, err error) {
	tx, ok := signedTx.(*data.Payment)
	if !ok {
		return "", fmt.Errorf("Send transaction type assertion error")
	}
	/*_, raw, err := data.Raw(tx)
	if err != nil {
		return "", err
	}
	txBlob := fmt.Sprintf("%X", raw)
	log.Info("Try send ripple tx", "txblob", txBlob)
	data := "{\"method\":\"submit\",\"params\":[{\"tx_blob\":\"" + txBlob + "\"}]}"
	fmt.Println(data)
	res := DoPostRequest("https://s.altnet.rippletest.net:51234", "", data)
	fmt.Printf("\nres:\n%v\n", res)
	return*/
	for i := 0; i < rpcRetryTimes; i++ {
		for url := range b.Remotes {
			r, err0 := websockets.NewRemote(url)
			if err0 != nil {
				log.Warn("Cannot connect to remote", "error", err)
				continue
			}
			resp, err1 := r.Submit(tx)
			if err1 != nil || resp == nil {
				log.Warn("Try sending transaction failed", "error", err)
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

// DoPostRequest only for test
func DoPostRequest(url, api, reqData string) string {
	req := bytes.NewBuffer([]byte(reqData))
	resp, err := http.Post(url+"/"+api, "application/json;charset=utf-8", req)
	if err != nil {
		return ""
	}
	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return string(body)
}
