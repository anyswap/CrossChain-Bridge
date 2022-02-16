package nebulas

import (
	"testing"

	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/stretchr/testify/assert"
)

func TestCallBridgeReceipt(t *testing.T) {
	bridge := NewCrossChainBridge(true)

	baseurl := "https://testnet.nebulas.io"

	hash := "c7bf89548169d52c23827e839cf5d241fd6dd0a4a78f537ac2fc1b2cffcfe02a"
	resp, err := bridge.getTransactionByHash(hash, []string{baseurl})
	assert.Nil(t, err)
	assert.NotNil(t, resp)

	price, err := getMedianGasPrice([]string{baseurl})
	assert.Nil(t, err)
	assert.NotNil(t, price)
}

func TestCallBridgeBalance(t *testing.T) {
	baseurl := "https://testnet.nebulas.io"
	bridge := NewCrossChainBridge(true)
	bridge.GatewayConfig = &tokens.GatewayConfig{
		APIAddress: []string{baseurl},
	}

	account := "n1SQm23yLbKn1uEDspWJEdhY2whsh1sz5jJ"
	resp, err := bridge.GetBalance(account)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
}

func TestCallBridgePrice(t *testing.T) {
	baseurl := "https://testnet.nebulas.io"

	price, err := getMedianGasPrice([]string{baseurl})
	assert.Nil(t, err)
	assert.NotNil(t, price)
}
