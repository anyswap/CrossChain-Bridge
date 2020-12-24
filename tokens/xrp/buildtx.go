package xrp

import (
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// BuildRawTransaction build raw tx
func (b *Bridge) BuildRawTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	return nil, nil
}

// BuildTransaction build tx
func (b *Bridge) BuildTransaction(from string, receivers []string, amounts []int64, memo string, relayFeePerKb int64) (rawTx interface{}, err error) {
	return nil, nil
}
