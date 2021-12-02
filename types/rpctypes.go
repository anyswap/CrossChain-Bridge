// Package types defines the eth-like core types (Transaction, etc) and RPC result types.
package types

import (
	"fmt"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/common/hexutil"
)

// RPCBaseBlock struct
type RPCBaseBlock struct {
	Hash       *common.Hash    `json:"hash"`
	ParentHash *common.Hash    `json:"parentHash"`
	Coinbase   *common.Address `json:"miner"`
	Difficulty *hexutil.Big    `json:"difficulty"`
	Number     *hexutil.Big    `json:"number"`
	GasLimit   *hexutil.Uint64 `json:"gasLimit"`
	GasUsed    *hexutil.Uint64 `json:"gasUsed"`
	Time       *hexutil.Big    `json:"timestamp"`
	BaseFee    *hexutil.Big    `json:"baseFeePerGas"`
}

// RPCBlock struct
type RPCBlock struct {
	Hash         *common.Hash    `json:"hash"`
	ParentHash   *common.Hash    `json:"parentHash"`
	Coinbase     *common.Address `json:"miner"`
	Difficulty   *hexutil.Big    `json:"difficulty"`
	Number       *hexutil.Big    `json:"number"`
	GasLimit     *hexutil.Uint64 `json:"gasLimit"`
	GasUsed      *hexutil.Uint64 `json:"gasUsed"`
	Time         *hexutil.Big    `json:"timestamp"`
	BaseFee      *hexutil.Big    `json:"baseFeePerGas"`
	Transactions []*common.Hash  `json:"transactions"`
}

// RPCTransaction struct
type RPCTransaction struct {
	Type         hexutil.Uint64  `json:"type"`
	Hash         *common.Hash    `json:"hash"`
	TxIndex      *hexutil.Uint   `json:"transactionIndex"`
	BlockNumber  *hexutil.Big    `json:"blockNumber"`
	BlockHash    *common.Hash    `json:"blockHash"`
	From         *common.Address `json:"from"`
	AccountNonce string          `json:"nonce"` // unexpect RSK has leading zero (eg. 0x01)
	Price        *hexutil.Big    `json:"gasPrice"`
	GasTipCap    *hexutil.Big    `json:"maxPriorityFeePerGas,omitempty"`
	GasFeeCap    *hexutil.Big    `json:"maxFeePerGas,omitempty"`
	GasLimit     *hexutil.Uint64 `json:"gas"`
	Recipient    *common.Address `json:"to"`
	Amount       *hexutil.Big    `json:"value"`
	Payload      *hexutil.Bytes  `json:"input"`
	V            *hexutil.Big    `json:"v"`
	R            *hexutil.Big    `json:"r"`
	S            *hexutil.Big    `json:"s"`
	ChainID      *hexutil.Big    `json:"chainId,omitempty"`
}

// FeeHistoryResult fee history result
type FeeHistoryResult struct {
	OldestBlock  interface{}      `json:"oldestBlock"`
	Reward       [][]*hexutil.Big `json:"reward,omitempty"`
	BaseFee      []*hexutil.Big   `json:"baseFeePerGas,omitempty"`
	GasUsedRatio []float64        `json:"gasUsedRatio"`
}

// GetAccountNonce convert
func (tx *RPCTransaction) GetAccountNonce() uint64 {
	if tx == nil || tx.AccountNonce == "" {
		return 0
	}
	if result, err := common.GetUint64FromStr(tx.AccountNonce); err == nil {
		return result
	}
	return 0
}

// RPCLog struct
type RPCLog struct {
	Address *common.Address `json:"address"`
	Topics  []common.Hash   `json:"topics"`
	Data    *hexutil.Bytes  `json:"data"`
	Removed *bool           `json:"removed"`
}

// RPCTxReceipt struct
type RPCTxReceipt struct {
	Type        hexutil.Uint64  `json:"type"`
	TxHash      *common.Hash    `json:"transactionHash"`
	TxIndex     *hexutil.Uint   `json:"transactionIndex"`
	BlockNumber *hexutil.Big    `json:"blockNumber"`
	BlockHash   *common.Hash    `json:"blockHash"`
	Status      *hexutil.Uint64 `json:"status"`
	From        *common.Address `json:"from"`
	Recipient   *common.Address `json:"to"`
	GasUsed     *hexutil.Uint64 `json:"gasUsed"`
	Logs        []*RPCLog       `json:"logs"`
}

// IsStatusOk is status ok
func (r *RPCTxReceipt) IsStatusOk() bool {
	return r != nil && r.Status != nil && *r.Status == 1
}

// FilterQuery struct
type FilterQuery struct {
	BlockHash *common.Hash
	FromBlock *big.Int
	ToBlock   *big.Int
	Addresses []common.Address
	Topics    [][]common.Hash
}

// ToFilterArg query to filter arg
func ToFilterArg(q *FilterQuery) (interface{}, error) {
	arg := map[string]interface{}{
		"address": q.Addresses,
		"topics":  q.Topics,
	}
	if q.BlockHash != nil {
		arg["blockHash"] = *q.BlockHash
		if q.FromBlock != nil || q.ToBlock != nil {
			return nil, fmt.Errorf("cannot specify both BlockHash and FromBlock/ToBlock")
		}
	} else {
		if q.FromBlock == nil {
			arg["fromBlock"] = "0x0"
		} else {
			arg["fromBlock"] = ToBlockNumArg(q.FromBlock)
		}
		arg["toBlock"] = ToBlockNumArg(q.ToBlock)
	}
	return arg, nil
}

// ToBlockNumArg to block number arg
func ToBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	return hexutil.EncodeBig(number)
}
