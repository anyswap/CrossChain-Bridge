package main

import (
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/tokens/terra"
)

func main() {
	b := terra.NewCrossChainBridge(true)
	terra.InitSDK()
	address, err := b.PublicKeyToAddress("045c8648793e4867af465691685000ae841dccab0b011283139d2eae454b569d5789f01632e13a75a5aad8480140e895dd671cae3639f935750bea7ae4b5a2512e")
	if err != nil {
		panic(err)
	}
	fmt.Println(address)
}
