package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"

	"github.com/dfuse-io/solana-go"
)

/*
Usage:
	solanaAddress -pubkey="04d38309dfdfd9adf129287b68cf2e1f1124e0cbc40cc98f94e5f2d23c26712fa3b33d63280dd1448319a6a4f4111722d6b3a730ebe07652ed2b3770947b3de2e2"
	KqhC7vpe7D9Sa1UMv9VLKj6xMovgL8QHd1mjW3Aws3t
*/

var pubkeyhex string

func init() {
	flag.StringVar(&pubkeyhex, "pubkey", "", "pubkey hex")
}

func main() {
	flag.Parse()
	address, err := PubkeyHexToAddress(pubkeyhex)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(address)
}

func PubkeyHexToAddress(pubkeyHex string) (string, error) {
	bz, err := hex.DecodeString(pubkeyHex)
	if err != nil {
		return "", errors.New("Decode pubkey hex error")
	}
	pub := PublicKeyFromBytes(bz)
	return fmt.Sprintf("%s", pub), nil
}

func PublicKeyFromBytes(in []byte) (out solana.PublicKey) {
	byteCount := len(in)
	if byteCount == 0 {
		return
	}

	max := 32
	if byteCount < max {
		max = byteCount
	}

	copy(out[:], in[0:max])
	return
}
