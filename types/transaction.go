package types

import (
	"math/big"
	"sync"
	"sync/atomic"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/tools/crypto"
	"github.com/anyswap/CrossChain-Bridge/tools/rlp"
	"golang.org/x/crypto/sha3"
)

// Transaction types.
const (
	LegacyTxType = iota
	AccessListTxType
	DynamicFeeTxType
)

// StorageSize type
type StorageSize = common.StorageSize

// hasherPool holds LegacyKeccak256 hashers for rlpHash.
var hasherPool = sync.Pool{
	New: func() interface{} { return sha3.NewLegacyKeccak256() },
}

// Transaction struct
type Transaction struct {
	data txdata
	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

// NewTransaction new tx
func NewTransaction(nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	return newTransaction(nonce, &to, amount, gasLimit, gasPrice, data)
}

// NewContractCreation new contract creation
func NewContractCreation(nonce uint64, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	return newTransaction(nonce, nil, amount, gasLimit, gasPrice, data)
}

func newTransaction(nonce uint64, to *common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}
	d := txdata{
		AccountNonce: nonce,
		Recipient:    to,
		Payload:      data,
		Amount:       new(big.Int),
		GasLimit:     gasLimit,
		Price:        new(big.Int),
		V:            new(big.Int),
		R:            new(big.Int),
		S:            new(big.Int),
	}
	if amount != nil {
		d.Amount.Set(amount)
	}
	if gasPrice != nil {
		d.Price.Set(gasPrice)
	}

	return &Transaction{data: d}
}

// NewDynamicFeeTx new dynamic fee tx for EIP-1559
func NewDynamicFeeTx(chainID *big.Int, nonce uint64, to *common.Address, amount *big.Int,
	gasLimit uint64, gasTipCap, gasFeeCap *big.Int, data []byte, accessList AccessList) *Transaction {
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}
	tx := &DynamicFeeTx{
		ChainID:    new(big.Int),
		Nonce:      nonce,
		GasTipCap:  new(big.Int),
		GasFeeCap:  new(big.Int),
		Gas:        gasLimit,
		To:         to,
		Value:      new(big.Int),
		Data:       data,
		AccessList: make(AccessList, len(accessList)),
		V:          new(big.Int),
		R:          new(big.Int),
		S:          new(big.Int),
	}
	if chainID != nil {
		tx.ChainID.Set(chainID)
	}
	if gasTipCap != nil {
		tx.GasTipCap.Set(gasTipCap)
	}
	if gasFeeCap != nil {
		tx.GasFeeCap.Set(gasFeeCap)
	}
	if amount != nil {
		tx.Value.Set(amount)
	}
	if len(accessList) > 0 {
		copy(tx.AccessList, accessList)
	}

	return &Transaction{data: *tx.getTxData()}
}

// Type returns tx type
func (tx *Transaction) Type() uint8 {
	return tx.data.Type
}

// ChainID returns which chain id this transaction was signed for (if at all)
func (tx *Transaction) ChainID() *big.Int {
	if tx.Type() == LegacyTxType {
		return deriveChainID(tx.data.V)
	}
	return new(big.Int).Set(tx.data.ChainID)
}

// Protected returns whether the transaction is protected from replay protection.
func (tx *Transaction) Protected() bool {
	return isProtectedV(tx.data.V)
}

func isProtectedV(rsvV *big.Int) bool {
	if rsvV.BitLen() <= 8 {
		v := rsvV.Uint64()
		return v != 27 && v != 28
	}
	// anything not 27 or 28 is considered protected
	return true
}

// MarshalJSON encodes the web3 RPC transaction format.
func (tx *Transaction) MarshalJSON() ([]byte, error) {
	hash := tx.Hash()
	data := tx.data
	data.Hash = &hash
	return data.MarshalJSON()
}

// UnmarshalJSON decodes the web3 RPC transaction format.
func (tx *Transaction) UnmarshalJSON(input []byte) error {
	var dec txdata
	if err := dec.UnmarshalJSON(input); err != nil {
		return err
	}
	*tx = Transaction{data: dec}
	return nil
}

// Data tx data
func (tx *Transaction) Data() []byte { return common.CopyBytes(tx.data.Payload) }

// Gas tx gas
func (tx *Transaction) Gas() uint64 { return tx.data.GasLimit }

// GasPrice tx gas price
func (tx *Transaction) GasPrice() *big.Int { return new(big.Int).Set(tx.data.Price) }

// SetGasPrice tx gas price
func (tx *Transaction) SetGasPrice(gasPrice *big.Int) { tx.data.Price.Set(gasPrice) }

// Value tx value
func (tx *Transaction) Value() *big.Int { return new(big.Int).Set(tx.data.Amount) }

// Nonce tx nonce
func (tx *Transaction) Nonce() uint64 { return tx.data.AccountNonce }

// CheckNonce check nonce
func (tx *Transaction) CheckNonce() bool { return true }

// To returns the recipient address of the transaction.
// It returns nil if the transaction is a contract creation.
func (tx *Transaction) To() *common.Address {
	if tx.data.Recipient == nil {
		return nil
	}
	to := *tx.data.Recipient
	return &to
}

// GasTipCap gas tip cap
func (tx *Transaction) GasTipCap() *big.Int {
	if tx.data.MaxPriorityFeePerGas != nil {
		return new(big.Int).Set(tx.data.MaxPriorityFeePerGas)
	}
	return nil
}

// GasFeeCap gas fee cap
func (tx *Transaction) GasFeeCap() *big.Int {
	if tx.data.MaxFeePerGas != nil {
		return new(big.Int).Set(tx.data.MaxFeePerGas)
	}
	return nil
}

// AccessList returns the access list of the transaction.
func (tx *Transaction) AccessList() AccessList {
	length := len(tx.data.AccessList)
	accessList := make(AccessList, length)
	if length > 0 {
		copy(accessList, tx.data.AccessList)
	}
	return accessList
}

// rlpHash encodes x and hashes the encoded bytes.
func rlpHash(x interface{}) (h common.Hash) {
	sha := hasherPool.Get().(crypto.KeccakState)
	defer hasherPool.Put(sha)
	sha.Reset()
	_ = rlp.Encode(sha, x)
	_, _ = sha.Read(h[:])
	return h
}

// prefixedRlpHash writes the prefix into the hasher before rlp-encoding x.
// It's used for typed transactions.
func prefixedRlpHash(prefix byte, x interface{}) (h common.Hash) {
	sha := hasherPool.Get().(crypto.KeccakState)
	defer hasherPool.Put(sha)
	sha.Reset()
	_, _ = sha.Write([]byte{prefix})
	_ = rlp.Encode(sha, x)
	_, _ = sha.Read(h[:])
	return h
}

type writeCounter StorageSize

func (c *writeCounter) Write(b []byte) (int, error) {
	*c += writeCounter(len(b))
	return len(b), nil
}

// Size returns the true RLP encoded storage size of the transaction, either by
// encoding and returning it, or returning a previsouly cached value.
func (tx *Transaction) Size() StorageSize {
	if size := tx.size.Load(); size != nil {
		return size.(StorageSize)
	}
	c := writeCounter(0)
	_ = rlp.Encode(&c, &tx.data)
	tx.size.Store(StorageSize(c))
	return StorageSize(c)
}

// WithSignature returns a new transaction with the given signature.
// This signature needs to be in the [R || S || V] format where V is 0 or 1.
func (tx *Transaction) WithSignature(signer Signer, sig []byte) (*Transaction, error) {
	r, s, v, err := signer.SignatureValues(tx, sig)
	if err != nil {
		return nil, err
	}
	cpy := &Transaction{data: tx.data}
	cpy.data.R, cpy.data.S, cpy.data.V = r, s, v
	return cpy, nil
}

// Cost returns amount + gasprice * gaslimit.
func (tx *Transaction) Cost() *big.Int {
	total := new(big.Int).Mul(tx.data.Price, new(big.Int).SetUint64(tx.data.GasLimit))
	total.Add(total, tx.data.Amount)
	return total
}

// RawSignatureValues returns the V, R, S signature values of the transaction.
// The return values should not be modified by the caller.
func (tx *Transaction) RawSignatureValues() (v, r, s *big.Int) {
	return tx.data.V, tx.data.R, tx.data.S
}
