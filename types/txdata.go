package types

import (
	"bytes"
	"errors"
	"io"
	"math/big"
	"sync"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/tools/rlp"
)

var (
	errEmptyTypedTx = errors.New("empty typed transaction bytes")
)

// deriveBufferPool holds temporary encoder buffers for DeriveSha and TX encoding.
var encodeBufferPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

// AccessList is an EIP-2930 access list.
type AccessList []AccessTuple

// AccessTuple is the element type of an access list.
type AccessTuple struct {
	Address     common.Address `json:"address"        gencodec:"required"`
	StorageKeys []common.Hash  `json:"storageKeys"    gencodec:"required"`
}

type txdata struct {
	Type uint8

	// Common transaction fields:
	AccountNonce         uint64
	Price                *big.Int
	MaxPriorityFeePerGas *big.Int
	MaxFeePerGas         *big.Int
	GasLimit             uint64
	Recipient            *common.Address
	Amount               *big.Int
	Payload              []byte
	V, R, S              *big.Int

	// Access list transaction fields:
	ChainID    *big.Int
	AccessList AccessList

	// Only used for encoding:
	Hash *common.Hash
}

// LegacyTx is the transaction data of regular Ethereum transactions.
type LegacyTx struct {
	Nonce    uint64          // nonce of sender account
	GasPrice *big.Int        // wei per gas
	Gas      uint64          // gas limit
	To       *common.Address `rlp:"nil"` // nil means contract creation
	Value    *big.Int        // wei amount
	Data     []byte          // contract invocation input data
	V, R, S  *big.Int        // signature values
}

// AccessListTx is the data of EIP-2930 access list transactions.
type AccessListTx struct {
	ChainID    *big.Int        // destination chain ID
	Nonce      uint64          // nonce of sender account
	GasPrice   *big.Int        // wei per gas
	Gas        uint64          // gas limit
	To         *common.Address `rlp:"nil"` // nil means contract creation
	Value      *big.Int        // wei amount
	Data       []byte          // contract invocation input data
	AccessList AccessList      // EIP-2930 access list
	V, R, S    *big.Int        // signature values
}

// DynamicFeeTx is the data of EIP-1559 dynamic fee transactions.
type DynamicFeeTx struct {
	ChainID    *big.Int        // destination chain ID
	Nonce      uint64          // nonce of sender account
	GasTipCap  *big.Int        // maxPriorityFeePerGas
	GasFeeCap  *big.Int        // maxFeePerGas
	Gas        uint64          // gas limit
	To         *common.Address `rlp:"nil"` // nil means contract creation
	Value      *big.Int        // wei amount
	Data       []byte          // contract invocation input data
	AccessList AccessList      // EIP-2930 access list
	V, R, S    *big.Int        // signature values
}

func (tx *Transaction) toLegacyTx() *LegacyTx {
	return &LegacyTx{
		Nonce:    tx.data.AccountNonce,
		GasPrice: tx.data.Price,
		Gas:      tx.data.GasLimit,
		To:       tx.data.Recipient,
		Value:    tx.data.Amount,
		Data:     tx.data.Payload,
		V:        tx.data.V,
		R:        tx.data.R,
		S:        tx.data.S,
	}
}

func (tx *Transaction) toAccessListTx() *AccessListTx {
	return &AccessListTx{
		ChainID:    tx.data.ChainID,
		Nonce:      tx.data.AccountNonce,
		GasPrice:   tx.data.Price,
		Gas:        tx.data.GasLimit,
		To:         tx.data.Recipient,
		Value:      tx.data.Amount,
		Data:       tx.data.Payload,
		AccessList: tx.data.AccessList,
		V:          tx.data.V,
		R:          tx.data.R,
		S:          tx.data.S,
	}
}

func (tx *Transaction) toDynamicFeeTx() *DynamicFeeTx {
	return &DynamicFeeTx{
		ChainID:    tx.data.ChainID,
		Nonce:      tx.data.AccountNonce,
		GasTipCap:  tx.data.MaxPriorityFeePerGas,
		GasFeeCap:  tx.data.MaxFeePerGas,
		Gas:        tx.data.GasLimit,
		To:         tx.data.Recipient,
		Value:      tx.data.Amount,
		Data:       tx.data.Payload,
		AccessList: tx.data.AccessList,
		V:          tx.data.V,
		R:          tx.data.R,
		S:          tx.data.S,
	}
}

func (tx *LegacyTx) getTxData() *txdata {
	return &txdata{
		Type:         LegacyTxType,
		AccountNonce: tx.Nonce,
		Price:        tx.GasPrice,
		GasLimit:     tx.Gas,
		Recipient:    tx.To,
		Amount:       tx.Value,
		Payload:      tx.Data,
		V:            tx.V,
		R:            tx.R,
		S:            tx.S,
	}
}

func (tx *AccessListTx) getTxData() *txdata {
	return &txdata{
		Type:         AccessListTxType,
		ChainID:      tx.ChainID,
		AccountNonce: tx.Nonce,
		Price:        tx.GasPrice,
		GasLimit:     tx.Gas,
		Recipient:    tx.To,
		Amount:       tx.Value,
		Payload:      tx.Data,
		AccessList:   tx.AccessList,
		V:            tx.V,
		R:            tx.R,
		S:            tx.S,
	}
}

func (tx *DynamicFeeTx) getTxData() *txdata {
	return &txdata{
		Type:                 DynamicFeeTxType,
		ChainID:              tx.ChainID,
		AccountNonce:         tx.Nonce,
		MaxPriorityFeePerGas: tx.GasTipCap,
		MaxFeePerGas:         tx.GasFeeCap,
		GasLimit:             tx.Gas,
		Recipient:            tx.To,
		Amount:               tx.Value,
		Payload:              tx.Data,
		AccessList:           tx.AccessList,
		V:                    tx.V,
		R:                    tx.R,
		S:                    tx.S,
	}
}

// EncodeRLP implements rlp.Encoder
func (tx *Transaction) EncodeRLP(w io.Writer) error {
	if tx.Type() == LegacyTxType {
		return rlp.Encode(w, tx.toLegacyTx())
	}
	// It's an EIP-2718 typed TX envelope.
	buf := encodeBufferPool.Get().(*bytes.Buffer)
	defer encodeBufferPool.Put(buf)
	buf.Reset()
	if err := tx.encodeTyped(buf); err != nil {
		return err
	}
	return rlp.Encode(w, buf.Bytes())
}

// encodeTyped writes the canonical encoding of a typed transaction to w.
func (tx *Transaction) encodeTyped(w *bytes.Buffer) error {
	w.WriteByte(tx.Type())
	switch tx.Type() {
	case AccessListTxType:
		return rlp.Encode(w, tx.toAccessListTx())
	case DynamicFeeTxType:
		return rlp.Encode(w, tx.toDynamicFeeTx())
	default:
		return ErrTxTypeNotSupported
	}
}

// MarshalBinary returns the canonical encoding of the transaction.
// For legacy transactions, it returns the RLP encoding. For EIP-2718 typed
// transactions, it returns the type and payload.
func (tx *Transaction) MarshalBinary() ([]byte, error) {
	if tx.Type() == LegacyTxType {
		return rlp.EncodeToBytes(tx.toLegacyTx())
	}
	var buf bytes.Buffer
	err := tx.encodeTyped(&buf)
	return buf.Bytes(), err
}

// DecodeRLP implements rlp.Decoder
func (tx *Transaction) DecodeRLP(s *rlp.Stream) error {
	kind, size, err := s.Kind()
	switch {
	case err != nil:
		return err
	case kind == rlp.List:
		// It's a legacy transaction.
		var inner LegacyTx
		err = s.Decode(&inner)
		if err == nil {
			tx.setDecoded(inner.getTxData(), int(size))
		}
		return err
	case kind == rlp.String:
		// It's an EIP-2718 typed TX envelope.
		var b []byte
		if b, err = s.Bytes(); err != nil {
			return err
		}
		txData, err := tx.decodeTyped(b)
		if err == nil {
			tx.setDecoded(txData, len(b))
		}
		return err
	default:
		return rlp.ErrExpectedList
	}
}

// decodeTyped decodes a typed transaction from the canonical format.
func (tx *Transaction) decodeTyped(b []byte) (*txdata, error) {
	if len(b) == 0 {
		return nil, errEmptyTypedTx
	}
	switch b[0] {
	case AccessListTxType:
		var inner AccessListTx
		err := rlp.DecodeBytes(b[1:], &inner)
		if err != nil {
			return nil, err
		}
		return inner.getTxData(), nil
	case DynamicFeeTxType:
		var inner DynamicFeeTx
		err := rlp.DecodeBytes(b[1:], &inner)
		if err != nil {
			return nil, err
		}
		return inner.getTxData(), nil
	default:
		return nil, ErrTxTypeNotSupported
	}
}

// UnmarshalBinary decodes the canonical encoding of transactions.
// It supports legacy RLP transactions and EIP2718 typed transactions.
func (tx *Transaction) UnmarshalBinary(b []byte) error {
	if len(b) > 0 && b[0] > 0x7f {
		// It's a legacy transaction.
		var inner LegacyTx
		err := rlp.DecodeBytes(b, &inner)
		if err != nil {
			return err
		}
		tx.setDecoded(inner.getTxData(), len(b))
		return nil
	}
	// It's an EIP2718 typed transaction envelope.
	txData, err := tx.decodeTyped(b)
	if err != nil {
		return err
	}
	tx.setDecoded(txData, len(b))
	return nil
}

// setDecoded sets the inner transaction and size after decoding.
func (tx *Transaction) setDecoded(inner *txdata, size int) {
	tx.data = *inner
	if size > 0 {
		tx.size.Store(StorageSize(size))
	}
}
