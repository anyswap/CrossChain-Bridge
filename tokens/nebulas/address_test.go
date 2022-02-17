package nebulas

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddress(t *testing.T) {
	pubKey := "0458e8769080f3a91cc65312a67c3edcf133467810ee35e715a347bc0906506cae7df559f771f306fbb25d09be30ce9fe8b36ab4c226d49c39d39260ff68919716"
	pubKeyBytes, _ := hex.DecodeString(pubKey)
	address, err := NewAddressFromPublicKey(pubKeyBytes)
	assert.Nil(t, err)
	assert.Equal(t, address.String(), "n1avapCUsTfyZDkNkYYFofjtak3bmroSYmY")
	t.Log(address)
}
