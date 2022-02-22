package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/anyswap/CrossChain-Bridge/tokens/ripple"
)

var pubkey string

func init() {
	flag.StringVar(&pubkey, "pubkey", "", "pubkey hex")
}

func main() {
	flag.Parse()
	addr, err := ripple.PublicKeyHexToAddress(pubkey)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("address: %v\n", addr)
}
