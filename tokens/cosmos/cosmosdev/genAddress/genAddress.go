package main

import (
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/tokens/cosmos"
	"github.com/anyswap/CrossChain-Bridge/tokens/terra"
)

func main() {
	pubkey := "045c8648793e4867af465691685000ae841dccab0b011283139d2eae454b569d5789f01632e13a75a5aad8480140e895dd671cae3639f935750bea7ae4b5a2512e"

	cosmosAddr := cosmosAddress(pubkey)
	fmt.Printf("cosmos address: %v\n", cosmosAddr)

	terraAddr := terraAddress(pubkey)
	fmt.Printf("terra address: %v\n", terraAddr)
}

func cosmosAddress(pubkey string) string {
	b := cosmos.NewCrossChainBridge(true)
	address, err := b.PublicKeyToAddress(pubkey)
	if err != nil {
		panic(err)
	}
	return address
}

func terraAddress(pubkey string) string {
	b := terra.NewCrossChainBridge(true)
	terra.InitSDK()
	address, err := b.PublicKeyToAddress(pubkey)
	if err != nil {
		panic(err)
	}
	return address
}
