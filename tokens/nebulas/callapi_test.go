package nebulas

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCallBridge(t *testing.T) {
	bridge := NewCrossChainBridge(true)

	baseurl := "https://testnet.nebulas.io"

	hash := "c7bf89548169d52c23827e839cf5d241fd6dd0a4a78f537ac2fc1b2cffcfe02a"
	resp, err := bridge.getTransactionByHash(hash, []string{baseurl})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
}
