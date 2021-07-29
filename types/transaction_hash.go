package types

import (
	"crypto/sha256"
	"math/big"
	"sync"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/cosmos/cosmos-sdk/codec"
)

var (
	okexCdc       *codec.Codec
	okexCdcInitor sync.Once
	okexChainID   = big.NewInt(66)
)

func getOkexCdc() *codec.Codec {
	if okexCdc == nil {
		okexCdcInitor.Do(func() {
			okexCdc = codec.New()
			okexCdc.RegisterConcrete(MsgEthereumTx{}, "ethermint/MsgEthereumTx", nil)
			okexCdc.Seal()
		})
	}
	return okexCdc
}

// Hash returns the transaction hash
func (tx *Transaction) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	var h common.Hash
	chainID := tx.ChainID()
	switch {
	case chainID.Cmp(okexChainID) == 0:
		h, _ = CalcOkexTransactionHash(tx)
	default:
		h = rlpHash(tx)
	}
	if h != common.EmptyHash {
		tx.hash.Store(h)
	}
	return h
}

// MsgEthereumTx encapsulates an Ethereum transaction as an SDK message.
type MsgEthereumTx struct {
	Data txdata
}

// CalcOkexTransactionHash calc okex tx hash
func CalcOkexTransactionHash(tx *Transaction) (hash common.Hash, err error) {
	txBytes, err := getOkexCdc().MarshalBinaryLengthPrefixed(MsgEthereumTx{tx.data})
	if err != nil {
		return hash, err
	}

	hash = common.Hash(sha256.Sum256(txBytes))
	return hash, nil
}

// MarshalAmino defines custom encoding scheme
func (td txdata) MarshalAmino() ([]byte, error) {
	gasPrice, err := common.MarshalBigInt(td.Price)
	if err != nil {
		return nil, err
	}

	amount, err := common.MarshalBigInt(td.Amount)
	if err != nil {
		return nil, err
	}

	v, err := common.MarshalBigInt(td.V)
	if err != nil {
		return nil, err
	}

	r, err := common.MarshalBigInt(td.R)
	if err != nil {
		return nil, err
	}

	s, err := common.MarshalBigInt(td.S)
	if err != nil {
		return nil, err
	}

	e := encodableTxData{
		AccountNonce: td.AccountNonce,
		Price:        gasPrice,
		GasLimit:     td.GasLimit,
		Recipient:    td.Recipient,
		Amount:       amount,
		Payload:      td.Payload,
		V:            v,
		R:            r,
		S:            s,
		Hash:         td.Hash,
	}

	return getOkexCdc().MarshalBinaryBare(e)
}

// encodableTxData implements the Ethereum transaction data structure. It is used
// solely as intended in Ethereum abiding by the protocol.
type encodableTxData struct {
	AccountNonce uint64
	Price        string
	GasLimit     uint64
	Recipient    *common.Address
	Amount       string
	Payload      []byte

	// signature values
	V string
	R string
	S string

	// hash is only used when marshaling to JSON
	Hash *common.Hash
}
