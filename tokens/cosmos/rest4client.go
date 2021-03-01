package cosmos

import (
	"math/big"

	legacysdk "github.com/cosmos/cosmos-sdk/types"
	legacyauthtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/anyswap/CrossChain-Bridge/tokens"
)

func (b *Bridge) getBalance4(account string) (balance *big.Int, err error) {
	return nil, nil
}

func (b *Bridge) getTokenBalance4(tokenType, tokenName, accountAddress string) (balance *big.Int, err error) {
	return nil, nil
}

func (b *Bridge) getTransaction4(txHash string) (tx interface{}, err error) {
	return nil, nil
}

func (b *Bridge) getTransactionStatus4(txHash string) (status *tokens.TxStatus) {
	return nil
}

func (b *Bridge) getLatestBlockNumber4() (height uint64, err error) {
	return 0, nil
}

func (b *Bridge) getLatestBlockNumberOf4(apiAddress string) (uint64, error) {
	return 0, nil
}

func (b *Bridge) getAccountNumber4(address string) (uint64, error) {
	return 0, nil
}

func (b *Bridge) getPoolNonce4(address, height string) (uint64, error) {
	return 0, nil
}

func (b *Bridge) searchTxsHash4(start, end *big.Int) ([]string, error) {
	return nil, nil
}

func (b *Bridge) searchTxs4(start, end *big.Int) ([]legacysdk.TxResponse, error) {
	return nil, nil
}

func (b *Bridge) broadcastTx4(tx legacyauthtypes.StdTx) error {
	return nil
}
