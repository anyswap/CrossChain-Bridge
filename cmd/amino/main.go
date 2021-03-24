package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	amino "github.com/tendermint/go-amino"
)

var TokenCDC = amino.NewCodec()

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	TokenCDC.RegisterInterface((*SwapinPromise)(nil), nil)
	TokenCDC.RegisterConcrete(&SolanaSwapinPromise{}, SolanaSwapinPromiseType, nil)

	var p1 SwapinPromise
	p1 = &SolanaSwapinPromise{"asdfasdf", "0xaaaaaaaa"}

	bz, err := TokenCDC.MarshalJSON(p1)
	checkError(err)
	fmt.Printf("promise1: %s\n", bz)

	var p2 SwapinPromise
	err = TokenCDC.UnmarshalJSON(bz, &p2)
	checkError(err)

	VerifySolanaSwapinPromise(p2.(*SolanaSwapinPromise))

	input := []byte(`{"type":"SolanaSwapinPromise","value":{"SolanaDepositAddress":"asdfasdf","ETHBindAddress":"0xaaaaaaaa"}}`)
	args := make(map[string]interface{})
	err = json.Unmarshal(input, &args)
	checkError(err)
	fmt.Printf("\nargs: %+v\n", args)

	p3, err := PromiseFromArgs(args)
	checkError(err)
	VerifySolanaSwapinPromise(p3.(*SolanaSwapinPromise))
}

// SwapinPromise interface
type SwapinPromise interface {
	// Key returns something identifies a swapin group
	// e.g. for solana-eth, Key should be solana address
	Key() string
	// Type returns promise type, e.g. "solana-eth-bindaddress"
	Type() string
	Value() interface{}
	// For solana-eth, Value should return eth binding address
}

type SolanaSwapinPromise struct {
	SolanaDepositAddress string
	ETHBindAddress       string
}

func (p *SolanaSwapinPromise) Key() string {
	depositAddress := strings.ToLower(p.SolanaDepositAddress)
	return fmt.Sprintf("solana-deposit-address-%s", depositAddress)
}

const SolanaSwapinPromiseType = "SolanaSwapinPromise"

func (p *SolanaSwapinPromise) Type() string {
	return SolanaSwapinPromiseType
}

func (p *SolanaSwapinPromise) Value() interface{} {
	return strings.ToLower(p.ETHBindAddress)
}

func VerifySolanaSwapinPromise(p *SolanaSwapinPromise) {
	fmt.Printf("Verify Solana swapin promise: %v\n", p)
	return
}

func PromiseFromArgs(args map[string](interface{})) (SwapinPromise, error) {
	bz, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	var p SwapinPromise
	err = TokenCDC.UnmarshalJSON(bz, &p)
	if err != nil {
		return nil, err
	}
	return p, nil
}
