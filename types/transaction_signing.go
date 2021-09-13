package types

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/tools/crypto"
)

// sign tx errors
var (
	ErrInvalidChainID     = errors.New("invalid chain id for signer")
	ErrInvalidSig         = errors.New("invalid transaction v, r, s values")
	ErrTxTypeNotSupported = errors.New("transaction type not supported")
)

// sigCache is used to cache the derived sender and contains
// the signer used to derive it.
type sigCache struct {
	signer Signer
	from   common.Address
}

// MakeSigner make signer
func MakeSigner(signType string, chainID *big.Int) Signer {
	var signer Signer
	switch signType {
	case "London":
		signer = NewLondonSigner(chainID)
	default:
		signer = NewEIP155Signer(chainID)
	}
	return signer
}

// SignTx signs the transaction using the given signer and private key
func SignTx(tx *Transaction, s Signer, prv *ecdsa.PrivateKey) (*Transaction, error) {
	h := s.Hash(tx)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(s, sig)
}

// Sender returns the address derived from the signature (V, R, S) using secp256k1
// elliptic curve and an error if it failed deriving or upon an incorrect
// signature.
//
// Sender may cache the address, allowing it to be used regardless of
// signing method. The cache is invalidated if the cached signer does
// not match the signer used in the current call.
func Sender(signer Signer, tx *Transaction) (common.Address, error) {
	if sc := tx.from.Load(); sc != nil {
		cache := sc.(sigCache)
		// If the signer used to derive from in a previous
		// call is not the same as used current, invalidate
		// the cache.
		if cache.signer.Equal(signer) {
			return cache.from, nil
		}
	}

	addr, err := signer.Sender(tx)
	if err != nil {
		return common.Address{}, err
	}
	tx.from.Store(sigCache{signer: signer, from: addr})
	return addr, nil
}

// Signer encapsulates transaction signature handling. Note that this interface is not a
// stable API and may change at any time to accommodate new protocol rules.
type Signer interface {
	// Sender returns the sender address of the transaction.
	Sender(tx *Transaction) (common.Address, error)
	// SignatureValues returns the raw R, S, V values corresponding to the
	// given signature.
	SignatureValues(tx *Transaction, sig []byte) (r, s, v *big.Int, err error)
	// Hash returns the hash to be signed.
	Hash(tx *Transaction) common.Hash
	// Equal returns true if the given signer is the same as the receiver.
	Equal(Signer) bool
}

type londonSigner struct{ eip2930Signer }

// NewLondonSigner returns a signer that accepts
// - EIP-1559 dynamic fee transactions
// - EIP-2930 access list transactions,
// - EIP-155 replay protected transactions, and
// - legacy Homestead transactions.
func NewLondonSigner(chainID *big.Int) Signer {
	return londonSigner{eip2930Signer{NewEIP155Signer(chainID)}}
}

func (s londonSigner) Sender(tx *Transaction) (common.Address, error) {
	if tx.Type() != DynamicFeeTxType {
		return s.eip2930Signer.Sender(tx)
	}
	V, R, S := tx.RawSignatureValues()
	// DynamicFee txs are defined to use 0 and 1 as their recovery
	// id, add 27 to become equivalent to unprotected Homestead signatures.
	V = new(big.Int).Add(V, big.NewInt(27))
	if tx.ChainID().Cmp(s.chainID) != 0 {
		return common.Address{}, ErrInvalidChainID
	}
	return recoverPlain(s.Hash(tx), R, S, V, true)
}

func (s londonSigner) Equal(s2 Signer) bool {
	x, ok := s2.(londonSigner)
	return ok && x.chainID.Cmp(s.chainID) == 0
}

func (s londonSigner) SignatureValues(tx *Transaction, sig []byte) (R, S, V *big.Int, err error) {
	if tx.Type() != DynamicFeeTxType {
		return s.eip2930Signer.SignatureValues(tx, sig)
	}
	// Check that chain ID of tx matches the signer. We also accept ID zero here,
	// because it indicates that the chain ID was not specified in the tx.
	if tx.data.ChainID != nil && tx.data.ChainID.Sign() != 0 && tx.data.ChainID.Cmp(s.chainID) != 0 {
		return nil, nil, nil, ErrInvalidChainID
	}
	R, S, _ = decodeSignature(sig)
	V = big.NewInt(int64(sig[64]))
	return R, S, V, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (s londonSigner) Hash(tx *Transaction) common.Hash {
	if tx.Type() != DynamicFeeTxType {
		return s.eip2930Signer.Hash(tx)
	}
	return prefixedRlpHash(
		tx.Type(),
		[]interface{}{
			s.chainID,
			tx.data.AccountNonce,
			tx.data.MaxPriorityFeePerGas,
			tx.data.MaxFeePerGas,
			tx.data.GasLimit,
			tx.data.Recipient,
			tx.data.Amount,
			tx.data.Payload,
			tx.data.AccessList,
		})
}

type eip2930Signer struct{ EIP155Signer }

// NewEIP2930Signer returns a signer that accepts EIP-2930 access list transactions,
// EIP-155 replay protected transactions, and legacy Homestead transactions.
func NewEIP2930Signer(chainID *big.Int) Signer {
	return eip2930Signer{NewEIP155Signer(chainID)}
}

func (s eip2930Signer) ChainID() *big.Int {
	return s.chainID
}

func (s eip2930Signer) Equal(s2 Signer) bool {
	x, ok := s2.(eip2930Signer)
	return ok && x.chainID.Cmp(s.chainID) == 0
}

func (s eip2930Signer) Sender(tx *Transaction) (common.Address, error) {
	V, R, S := tx.RawSignatureValues()
	switch tx.Type() {
	case LegacyTxType:
		if !tx.Protected() {
			return HomesteadSigner{}.Sender(tx)
		}
		V = new(big.Int).Sub(V, s.chainIDMul)
		V.Sub(V, big8)
	case AccessListTxType:
		// AL txs are defined to use 0 and 1 as their recovery
		// id, add 27 to become equivalent to unprotected Homestead signatures.
		V = new(big.Int).Add(V, big.NewInt(27))
	default:
		return common.Address{}, ErrTxTypeNotSupported
	}
	if tx.ChainID().Cmp(s.chainID) != 0 {
		return common.Address{}, ErrInvalidChainID
	}
	return recoverPlain(s.Hash(tx), R, S, V, true)
}

func (s eip2930Signer) SignatureValues(tx *Transaction, sig []byte) (R, S, V *big.Int, err error) {
	switch tx.Type() {
	case LegacyTxType:
		return s.EIP155Signer.SignatureValues(tx, sig)
	case AccessListTxType, DynamicFeeTxType:
		// Check that chain ID of tx matches the signer. We also accept ID zero here,
		// because it indicates that the chain ID was not specified in the tx.
		if tx.data.ChainID != nil && tx.data.ChainID.Sign() != 0 && tx.data.ChainID.Cmp(s.chainID) != 0 {
			return nil, nil, nil, ErrInvalidChainID
		}
		R, S, _ = decodeSignature(sig)
		V = big.NewInt(int64(sig[64]))
	default:
		return nil, nil, nil, ErrTxTypeNotSupported
	}
	return R, S, V, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (s eip2930Signer) Hash(tx *Transaction) common.Hash {
	switch tx.Type() {
	case LegacyTxType:
		return s.EIP155Signer.Hash(tx)
	case AccessListTxType:
		return prefixedRlpHash(
			tx.Type(),
			[]interface{}{
				s.chainID,
				tx.data.AccountNonce,
				tx.data.Price,
				tx.data.GasLimit,
				tx.data.Recipient,
				tx.data.Amount,
				tx.data.Payload,
				tx.data.AccessList,
			})
	default:
		// This _should_ not happen, but in case someone sends in a bad
		// json struct via RPC, it's probably more prudent to return an
		// empty hash instead of killing the node with a panic
		//panic("Unsupported transaction type: %d", tx.typ)
		return common.Hash{}
	}
}

// EIP155Signer implements Signer using the EIP155 rules.
type EIP155Signer struct {
	chainID, chainIDMul *big.Int
}

// NewEIP155Signer new EIP155Signer
func NewEIP155Signer(chainID *big.Int) EIP155Signer {
	if chainID == nil {
		chainID = new(big.Int)
	}
	return EIP155Signer{
		chainID:    chainID,
		chainIDMul: new(big.Int).Mul(chainID, big.NewInt(2)),
	}
}

// Equal compare signer
func (s EIP155Signer) Equal(s2 Signer) bool {
	eip155, ok := s2.(EIP155Signer)
	return ok && eip155.chainID.Cmp(s.chainID) == 0
}

var big8 = big.NewInt(8)

// Sender get sender
func (s EIP155Signer) Sender(tx *Transaction) (common.Address, error) {
	if !tx.Protected() {
		return HomesteadSigner{}.Sender(tx)
	}
	if tx.ChainID().Cmp(s.chainID) != 0 {
		return common.Address{}, ErrInvalidChainID
	}
	V := new(big.Int).Sub(tx.data.V, s.chainIDMul)
	V.Sub(V, big8)
	return recoverPlain(s.Hash(tx), tx.data.R, tx.data.S, V, true)
}

// SignatureValues returns signature values. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (s EIP155Signer) SignatureValues(tx *Transaction, sig []byte) (rsvR, rsvS, rsvV *big.Int, err error) {
	rsvR, rsvS, rsvV, err = HomesteadSigner{}.SignatureValues(tx, sig)
	if err != nil {
		return nil, nil, nil, err
	}
	if s.chainID.Sign() != 0 {
		rsvV = big.NewInt(int64(sig[64] + 35))
		rsvV.Add(rsvV, s.chainIDMul)
	}
	return rsvR, rsvS, rsvV, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (s EIP155Signer) Hash(tx *Transaction) common.Hash {
	return rlpHash([]interface{}{
		tx.data.AccountNonce,
		tx.data.Price,
		tx.data.GasLimit,
		tx.data.Recipient,
		tx.data.Amount,
		tx.data.Payload,
		s.chainID, uint(0), uint(0),
	})
}

// HomesteadSigner implements TransactionInterface using the
// homestead rules.
type HomesteadSigner struct{ FrontierSigner }

// Equal compare signer
func (hs HomesteadSigner) Equal(s2 Signer) bool {
	_, ok := s2.(HomesteadSigner)
	return ok
}

// SignatureValues returns signature values. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (hs HomesteadSigner) SignatureValues(tx *Transaction, sig []byte) (r, s, v *big.Int, err error) {
	return hs.FrontierSigner.SignatureValues(tx, sig)
}

// Sender get sender
func (hs HomesteadSigner) Sender(tx *Transaction) (common.Address, error) {
	return recoverPlain(hs.Hash(tx), tx.data.R, tx.data.S, tx.data.V, true)
}

// FrontierSigner frontier signer
type FrontierSigner struct{}

// Equal compare signer
func (fs FrontierSigner) Equal(s2 Signer) bool {
	_, ok := s2.(FrontierSigner)
	return ok
}

// SignatureValues returns signature values. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (fs FrontierSigner) SignatureValues(tx *Transaction, sig []byte) (r, s, v *big.Int, err error) {
	if tx.Type() != LegacyTxType {
		return nil, nil, nil, ErrTxTypeNotSupported
	}
	r, s, v = decodeSignature(sig)
	return r, s, v, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (fs FrontierSigner) Hash(tx *Transaction) common.Hash {
	return rlpHash([]interface{}{
		tx.data.AccountNonce,
		tx.data.Price,
		tx.data.GasLimit,
		tx.data.Recipient,
		tx.data.Amount,
		tx.data.Payload,
	})
}

// Sender get sender
func (fs FrontierSigner) Sender(tx *Transaction) (common.Address, error) {
	return recoverPlain(fs.Hash(tx), tx.data.R, tx.data.S, tx.data.V, false)
}

func decodeSignature(sig []byte) (r, s, v *big.Int) {
	if len(sig) != crypto.SignatureLength {
		panic(fmt.Sprintf("wrong size for signature: got %d, want %d", len(sig), crypto.SignatureLength))
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v
}

func recoverPlain(sighash common.Hash, rsvR, rsvS, rsvV *big.Int, homestead bool) (common.Address, error) {
	if rsvV.BitLen() > 8 {
		return common.Address{}, ErrInvalidSig
	}
	v := byte(rsvV.Uint64() - 27)
	if !crypto.ValidateSignatureValues(v, rsvR, rsvS, homestead) {
		return common.Address{}, ErrInvalidSig
	}
	// encode the signature in uncompressed format
	r, s := rsvR.Bytes(), rsvS.Bytes()
	sig := make([]byte, crypto.SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = v
	// recover the public key from the signature
	pub, err := crypto.Ecrecover(sighash[:], sig)
	if err != nil {
		return common.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return common.Address{}, errors.New("invalid public key")
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(pub[1:])[12:])
	return addr, nil
}

// deriveChainID derives the chain id from the given v parameter
func deriveChainID(v *big.Int) *big.Int {
	if v.BitLen() <= 64 {
		v := v.Uint64()
		if v == 27 || v == 28 {
			return new(big.Int)
		}
		return new(big.Int).SetUint64((v - 35) / 2)
	}
	v = new(big.Int).Sub(v, big.NewInt(35))
	return v.Div(v, big.NewInt(2))
}
