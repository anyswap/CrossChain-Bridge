package fsn

import (
	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/common/hexutil"
)

type RPCBlock struct {
	Hash            *common.Hash    `json:"hash"`
	ParentHash      *common.Hash    `json:"parentHash"`
	UncleHash       *common.Hash    `json:"sha3Uncles"`
	Coinbase        *common.Address `json:"miner"`
	Root            *common.Hash    `json:"stateRoot"`
	TxHash          *common.Hash    `json:"transactionsRoot"`
	ReceiptHash     *common.Hash    `json:"receiptsRoot"`
	Bloom           *hexutil.Bytes  `json:"logsBloom"`
	Difficulty      *hexutil.Big    `json:"difficulty"`
	Number          *hexutil.Big    `json:"number"`
	GasLimit        *hexutil.Uint64 `json:"gasLimit"`
	GasUsed         *hexutil.Uint64 `json:"gasUsed"`
	Time            *hexutil.Big    `json:"timestamp"`
	Extra           *hexutil.Bytes  `json:"extraData"`
	MixDigest       *common.Hash    `json:"mixHash"`
	Nonce           *hexutil.Bytes  `json:"nonce"`
	Size            *string         `json:"size"`
	TotalDifficulty *hexutil.Big    `json:"totalDifficulty"`
	Transactions    []*common.Hash  `json:"transactions"`
	Uncles          []*common.Hash  `json:"uncles"`
}

type RPCTransaction struct {
	Hash             *common.Hash    `json:"hash"`
	TransactionIndex *hexutil.Uint   `json:"transactionIndex"`
	BlockNumber      *hexutil.Big    `json:"blockNumber,omitempty"`
	BlockHash        *common.Hash    `json:"blockHash,omitempty"`
	From             *common.Address `json:"from,omitempty"`
	AccountNonce     *hexutil.Uint64 `json:"nonce"`
	Price            *hexutil.Big    `json:"gasPrice"`
	GasLimit         *hexutil.Uint64 `json:"gas"`
	Recipient        *common.Address `json:"to"`
	Amount           *hexutil.Big    `json:"value"`
	Payload          *hexutil.Bytes  `json:"input"`
	V                *hexutil.Big    `json:"v"`
	R                *hexutil.Big    `json:"r"`
	S                *hexutil.Big    `json:"s"`
}

type RPCLog struct {
	Address     *common.Address `json:"address"`
	Topics      []common.Hash   `json:"topics"`
	Data        *hexutil.Bytes  `json:"data"`
	BlockNumber *hexutil.Uint64 `json:"blockNumber"`
	TxHash      *common.Hash    `json:"transactionHash"`
	TxIndex     *hexutil.Uint   `json:"transactionIndex"`
	BlockHash   *common.Hash    `json:"blockHash"`
	Index       *hexutil.Uint   `json:"logIndex"`
	Removed     *bool           `json:"removed"`
}

type RPCTxReceipt struct {
	TxHash            *common.Hash    `json:"transactionHash"`
	TxIndex           *hexutil.Uint   `json:"transactionIndex"`
	BlockNumber       *hexutil.Big    `json:"blockNumber"`
	BlockHash         *common.Hash    `json:"blockHash"`
	PostState         *hexutil.Bytes  `json:"root"`
	Status            *hexutil.Uint64 `json:"status"`
	From              *common.Address `json:"from"`
	Recipient         *common.Address `json:"to"`
	GasUsed           *hexutil.Uint64 `json:"gasUsed"`
	CumulativeGasUsed *hexutil.Uint64 `json:"cumulativeGasUsed"`
	ContractAddress   *common.Address `json:"contractAddress,omitempty"`
	Bloom             *hexutil.Bytes  `json:"logsBloom"`
	FsnLogTopic       *string         `json:"fsnLogTopic,omitempty"`
	FsnLogData        interface{}     `json:"fsnLogData,omitempty"`
	Logs              []*RPCLog       `json:"logs"`
}

type RPCTxAndReceipt struct {
	FsnTxInput   interface{}     `json:"fsnTxInput,omitempty"`
	Tx           *RPCTransaction `json:"tx"`
	Receipt      *RPCTxReceipt   `json:"receipt"`
	ReceiptFound *bool           `json:"receiptFound"`
}
