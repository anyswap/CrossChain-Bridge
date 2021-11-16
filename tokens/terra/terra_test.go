package terra

import (
	"fmt"
	"testing"

	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens/cosmos"

	"github.com/stretchr/testify/assert"
)

func newTestBridge() *Bridge {
	b := &Bridge{Bridge: cosmos.NewCrossChainBridge(true)}
	b.BeforeConfig()
	params.SetConfig(&params.BridgeConfig{
		Identifier: "TEQUILA2RINKEBY",
	})
	cosmos.GetFeeAmount = b.FeeGetter()
	b.SupportedCoins["LUNA"] = cosmos.CosmosCoin{Denom: "uluna", Decimal: 6}
	b.SupportedCoins["USD"] = cosmos.CosmosCoin{Denom: "uusd", Decimal: 6}
	b.MainCoin = b.SupportedCoins["LUNA"]
	return b
}

func TestAddress(t *testing.T) {
	b := newTestBridge()
	addr1, err1 := b.PublicKeyToAddress("0458e8769080f3a91cc65312a67c3edcf133467810ee35e715a347bc0906506cae7df559f771f306fbb25d09be30ce9fe8b36ab4c226d49c39d39260ff68919716")
	addr2, err2 := b.PublicKeyToAddress("04d38309dfdfd9adf129287b68cf2e1f1124e0cbc40cc98f94e5f2d23c26712fa3b33d63280dd1448319a6a4f4111722d6b3a730ebe07652ed2b3770947b3de2e2")
	addr3, err3 := b.PublicKeyToAddress("04a244fd6711d00d96ec2ab8189867c0ec57844ed9baa4c941e86542974c5a43c9c1dd537a4fad80d6319ac785639d7bb6a26f875d90aea9b38c9b73ab1620b065")
	assert.Nil(t, err1)
	assert.Nil(t, err2)
	assert.Nil(t, err3)
	assert.Equal(t, addr1, "terra1tgfzuquds5y3au839k3j7uxtxmf238mrspja4w")
	assert.Equal(t, addr2, "terra10rf55rx37vrtc4ws7l8v950whvwq9znmk7d9ka")
	assert.Equal(t, addr3, "terra1fg9wpzn3289sklzz67928lfgmu7c7ckp7l7fph")
	fmt.Printf("addr1: %v\n", addr1)
	fmt.Printf("addr2: %v\n", addr2)
	fmt.Printf("addr2: %v\n", addr3)
}
