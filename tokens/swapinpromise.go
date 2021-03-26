package tokens

import (
	"encoding/json"

	amino "github.com/tendermint/go-amino"
)

var TokenCDC = amino.NewCodec()

func init() {
	TokenCDC.RegisterInterface((*SwapinPromise)(nil), nil)
}

// SwapinPromise interface
// to be implemented with definition of swapin binding rule
type SwapinPromise interface {
	// Type returns promise type
	// e.g. "solana-eth-bindaddress"
	Type() string
	// Key returns something identifies a swapin binding rule
	// e.g. for solana-eth, Key should be solana address
	Key() string
	// Value returns data required to define a binding rule
	// For solana-eth, Value should return an eth binding address
	Value() interface{}
}

// PromiseFromArgs takes args from api, returns SwapinPromise
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
