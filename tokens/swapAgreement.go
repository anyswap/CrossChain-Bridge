package tokens

import (
	"encoding/json"

	amino "github.com/tendermint/go-amino"
)

var TokenCDC = amino.NewCodec()

func init() {
	TokenCDC.RegisterInterface((*SwapAgreement)(nil), nil)
}

// SwapAgreement interface
// to be implemented with definition of swap binding rule
type SwapAgreement interface {
	// Type returns agreement type
	// e.g. "solana-eth-bindaddress"
	Type() string
	// Key returns something identifies a swap binding rule
	// e.g. for solana-eth, Key should be solana address
	Key() string
	// Value returns data required to define a binding rule
	// For solana-eth, Value should return an eth binding address
	Value() interface{}
}

// AgreementFromArgs takes args from api, returns SwapAgreement
func AgreementFromArgs(args map[string](interface{})) (SwapAgreement, error) {
	bz, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	var p SwapAgreement
	err = TokenCDC.UnmarshalJSON(bz, &p)
	if err != nil {
		return nil, err
	}
	return p, nil
}
