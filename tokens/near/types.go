package near

import (
	"math/big"
)

type TransactionResult struct {
	Status             Status             `json:"status"`
	Transaction        Transaction        `json:"transaction"`
	TransactionOutcome TransactionOutcome `json:"transaction_outcome"`
	ReceiptsOutcome    []ReceiptsOutcome  `json:"receipts_outcome"`
}

type BlockDetail struct {
	header BlockHeader `json:"header"`
}

type BlockHeader struct {
	hash   string `json:"hash"`
	height string `json:"string"`
}

type NetworkStatus struct {
	chainId  string   `json:"chain_id"`
	syncInfo SyncInfo `json:"sync_info"`
}

type SyncInfo struct {
	latestBlockHash   string `json:"latest_block_hash"`
	latestBlockHeight string `json:"latest_block_height"`
}

type Status struct {
	SuccessValue     string `json:"SuccessValue,omitempty"`
	SuccessReceiptId string `json:"SuccessReceiptId,omitempty"`
	Failure          string `json:"Failure,omitempty"`
	Unknown          string `json:"Unknown,omitempty"`
}

type Transaction struct {
	Actions    []Action `json:"actions"`
	Hash       string   `json:"hash"`
	Nonce      int      `json:"nonce"`
	PublicKey  string   `json:"public_key"`
	ReceiverID string   `json:"receiver_id"`
	Signature  string   `json:"signature"`
	SignerID   string   `json:"signer_id"`
}

type TransactionOutcome struct {
	BlockHash string  `json:"block_hash"`
	ID        string  `json:"id"`
	Outcome   Outcome `json:"outcome"`
	Proof     []Proof `json:"proof"`
}

type ReceiptsOutcome struct {
	BlockHash string  `json:"block_hash"`
	ID        string  `json:"id"`
	Outcome   Outcome `json:"outcome"`
	Proof     []Proof `json:"proof"`
}

type Outcome struct {
	ExecutorID  string        `json:"executor_id"`
	GasBurnt    int64         `json:"gas_burnt"`
	Logs        []interface{} `json:"logs"`
	ReceiptIds  []string      `json:"receipt_ids"`
	Status      Status        `json:"status"`
	TokensBurnt string        `json:"tokens_burnt"`
}

type Proof struct {
	Direction string `json:"direction"`
	Hash      string `json:"hash"`
}

type Action struct {
	FunctionCall FunctionCall
	Transfer     Transfer
}

type Transfer struct {
	Deposit big.Int
}

type FunctionCall struct {
	MethodName string
	Args       []byte
	Gas        uint64
	Deposit    big.Int
}
