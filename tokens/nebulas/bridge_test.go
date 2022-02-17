package nebulas_test

import (
	"fmt"
	"testing"

	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens/nebulas"
	"github.com/stretchr/testify/assert"
)

func TestCallApi(t *testing.T) {
	bridge := nebulas.NewCrossChainBridge(true)

	baseurl := "https://testnet.nebulas.io"
	height, err := bridge.GetLatestBlockNumberOf(baseurl)
	assert.Nil(t, err)
	assert.NotEqual(t, 0, height)

	// block
	url := fmt.Sprintf("%s/v1/user/getBlockByHeight", baseurl)
	params := make(map[string]interface{})
	params["height"] = height
	params["full_fill_transaction"] = true
	resp, err := client.HTTPPost(url, params, nil, nil, 60)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	block := new(nebulas.BlockResponse)
	err = nebulas.ParseReponse(resp, block)
	assert.Nil(t, err)
	assert.NotEqual(t, nil, block)

	//transaction
	url = fmt.Sprintf("%s/v1/user/getTransactionReceipt", baseurl)
	params = make(map[string]interface{})
	params["hash"] = "1785f264ae4eb55f279633843b9d04b105fa4c58a6e27dfdd8b2f5254147d84d"
	resp, err = client.HTTPPost(url, params, nil, nil, 60)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	result := new(nebulas.TransactionResponse)
	err = nebulas.ParseReponse(resp, result)
	assert.Nil(t, err)
	assert.NotEqual(t, nil, block)
}
