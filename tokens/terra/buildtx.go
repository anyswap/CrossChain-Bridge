package terra

import (
	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// BuildRawTransaction build raw tx
func (b *Bridge) BuildRawTransaction(args *tokens.BuildTxArgs) (rawTx interface{}, err error) {
	return nil, tokens.ErrTodo
}

// GetPoolNonce impl NonceSetter interface
func (b *Bridge) GetPoolNonce(address, _height string) (uint64, error) {
	return b.GetAccountSequence(address)
}

// GetAccountSequence get account sequence
func (b *Bridge) GetAccountSequence(address string) (uint64, error) {
	urls := append(b.GatewayConfig.APIAddress, b.GatewayConfig.APIAddressExt...)
	var acc *BaseAccount
	var err error
	for _, url := range urls {
		acc, err = GetBaseAccount(url, address)
		if err == nil && acc != nil {
			return common.GetUint64FromStr(acc.Sequence)
		}
	}
	return 0, wrapRPCQueryError(err, "GetAccountSequence")
}

// GetAccountNumber get account number
func (b *Bridge) GetAccountNumber(address string) (uint64, error) {
	urls := append(b.GatewayConfig.APIAddress, b.GatewayConfig.APIAddressExt...)
	var acc *BaseAccount
	var err error
	for _, url := range urls {
		acc, err = GetBaseAccount(url, address)
		if err == nil && acc != nil {
			return common.GetUint64FromStr(acc.AccountNumber)
		}
	}
	return 0, wrapRPCQueryError(err, "GetAccountNumber")
}
