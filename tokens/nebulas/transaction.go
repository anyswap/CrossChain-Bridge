// Copyright (C) 2017 go-nebulas authors
//
// This file is part of the go-nebulas library.
//
// the go-nebulas library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-nebulas library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-nebulas library.  If not, see <http://www.gnu.org/licenses/>.
//

package nebulas

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"encoding/json"

	"github.com/anyswap/CrossChain-Bridge/tokens/nebulas/byteutils"
	corepb "github.com/anyswap/CrossChain-Bridge/tokens/nebulas/pb"
	"github.com/anyswap/CrossChain-Bridge/tokens/nebulas/util"
	"github.com/gogo/protobuf/proto"
	"github.com/nebulasio/go-nebulas/crypto/keystore"
	"golang.org/x/crypto/sha3"
)

const (
	// TxHashByteLength invalid tx hash length(len of []byte)
	TxHashByteLength = 32
)

var (
	// TransactionMaxGasPrice max gasPrice:1 * 10 ** 12
	TransactionMaxGasPrice, _ = util.NewUint128FromString("1000000000000")

	// TransactionMaxGas max gas:50 * 10 ** 9
	TransactionMaxGas, _ = util.NewUint128FromString("50000000000")

	// TransactionGasPrice default gasPrice : 2*10**10
	TransactionGasPrice, _ = util.NewUint128FromString("20000000000")

	// GenesisGasPrice default gasPrice : 1*10**6
	GenesisGasPrice, _ = util.NewUint128FromInt(1000000)

	// MinGasCountPerTransaction default gas for normal transaction
	MinGasCountPerTransaction, _ = util.NewUint128FromInt(20000)

	// GasCountPerByte per byte of data attached to a transaction gas cost
	GasCountPerByte, _ = util.NewUint128FromInt(1)

	// MaxDataPayLoadLength Max data length in transaction
	MaxDataPayLoadLength = 128 * 1024
	// MaxDataBinPayloadLength Max data length in binary transaction
	MaxDataBinPayloadLength = 64

	// MaxEventErrLength Max error length in event
	MaxEventErrLength = 256

	// MaxResultLength max execution result length
	MaxResultLength = 256
)

// Transaction type is used to handle all transaction data.
type Transaction struct {
	hash      byteutils.Hash
	from      *Address
	to        *Address
	value     *big.Int
	nonce     uint64
	timestamp int64
	data      *corepb.Data
	chainID   uint32
	gasPrice  *big.Int
	gasLimit  uint64

	// Signature
	alg  keystore.Algorithm
	sign byteutils.Hash // Signature values
}

// SetTimestamp update the timestamp.
func (tx *Transaction) SetTimestamp(timestamp int64) {
	tx.timestamp = timestamp
}

// SetSignature update tx sign
func (tx *Transaction) SetSignature(alg keystore.Algorithm, sign byteutils.Hash) {
	tx.alg = alg
	tx.sign = sign
}

// From return from address
func (tx *Transaction) From() *Address {
	return tx.from
}

// Timestamp return timestamp
func (tx *Transaction) Timestamp() int64 {
	return tx.timestamp
}

// To return to address
func (tx *Transaction) To() *Address {
	return tx.to
}

// ChainID return chainID
func (tx *Transaction) ChainID() uint32 {
	return tx.chainID
}

// Value return tx value
func (tx *Transaction) Value() *big.Int {
	return tx.value
}

// Nonce return tx nonce
func (tx *Transaction) Nonce() uint64 {
	return tx.nonce
}

// SetNonce update th nonce
func (tx *Transaction) SetNonce(newNonce uint64) {
	tx.nonce = newNonce
}

// Type return tx type
func (tx *Transaction) Type() string {
	return tx.data.Type
}

// Data return tx data
func (tx *Transaction) Data() []byte {
	return tx.data.Payload
}

// ToProto converts domain Tx to proto Tx
func (tx *Transaction) ToProto() (proto.Message, error) {
	tvalue, err := util.NewUint128FromBigInt(tx.value)
	if err != nil {
		return nil, err
	}
	value, err := tvalue.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	tgasPrice, err := util.NewUint128FromBigInt(tx.gasPrice)
	if err != nil {
		return nil, err
	}
	gasPrice, err := tgasPrice.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	gasLimit, err := util.NewUint128FromUint(tx.gasLimit).ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	return &corepb.Transaction{
		Hash:      tx.hash,
		From:      tx.from.address,
		To:        tx.to.address,
		Value:     value,
		Nonce:     tx.nonce,
		Timestamp: tx.timestamp,
		Data:      tx.data,
		ChainId:   tx.chainID,
		GasPrice:  gasPrice,
		GasLimit:  gasLimit,
		Alg:       uint32(tx.alg),
		Sign:      tx.sign,
	}, nil
}

// FromProto converts proto Tx into domain Tx
func (tx *Transaction) FromProto(msg proto.Message) error {
	if msg, ok := msg.(*corepb.Transaction); ok {
		if msg != nil {

			tx.hash = msg.Hash
			from, err := AddressParseFromBytes(msg.From)
			if err != nil {
				return err
			}
			tx.from = from

			to, err := AddressParseFromBytes(msg.To)
			if err != nil {
				return err
			}
			tx.to = to

			value, err := util.NewUint128FromFixedSizeByteSlice(msg.Value)
			if err != nil {
				return err
			}
			tx.value = value.Value()

			tx.nonce = msg.Nonce
			tx.timestamp = msg.Timestamp
			tx.chainID = msg.ChainId

			tx.data = msg.Data

			gasPrice, err := util.NewUint128FromFixedSizeByteSlice(msg.GasPrice)
			if err != nil {
				return err
			}
			tx.gasPrice = gasPrice.Value()

			gasLimit, err := util.NewUint128FromFixedSizeByteSlice(msg.GasLimit)
			if err != nil {
				return err
			}
			tx.gasLimit = gasLimit.Uint64()

			tx.alg = keystore.Algorithm(msg.Alg)
			tx.sign = msg.Sign
			return nil
		}
		return errors.New("Invalid transaction")
	}
	return errors.New("Invalid transaction")
}

func (tx *Transaction) String() string {
	return fmt.Sprintf(`{"chainID":%d, "hash":"%s", "from":"%s", "to":"%s", "nonce":%d, "value":"%s", "timestamp":%d, "gasprice": "%s", "gaslimit":"%d", "data": "%s", "type":"%s"}`,
		tx.chainID,
		tx.hash.String(),
		tx.from.String(),
		tx.to.String(),
		tx.nonce,
		tx.value.String(),
		tx.timestamp,
		tx.gasPrice.String(),
		tx.gasLimit,
		tx.Data(),
		tx.Type(),
	)
}

func (tx *Transaction) Bytes() ([]byte, error) {
	pb, _ := tx.ToProto()
	data, err := proto.Marshal(pb)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (tx *Transaction) StringWithoutData() string {
	return fmt.Sprintf(`{"chainID":%d, "hash":"%s", "from":"%s", "to":"%s", "nonce":%d, "value":"%s", "timestamp":%d, "gasprice": "%s", "gaslimit":"%d", "type":"%s"}`,
		tx.chainID,
		tx.hash.String(),
		tx.from.String(),
		tx.to.String(),
		tx.nonce,
		tx.value.String(),
		tx.timestamp,
		tx.gasPrice.String(),
		tx.gasLimit,
		tx.Type(),
	)
}

// JSONString of transaction
func (tx *Transaction) JSONString() string {
	txJSONObj := make(map[string]interface{})
	txJSONObj["chainID"] = tx.chainID
	txJSONObj["hash"] = tx.hash.String()
	txJSONObj["from"] = tx.from.String()
	txJSONObj["to"] = tx.to.String()
	txJSONObj["nonce"] = tx.nonce
	txJSONObj["value"] = tx.value.String()
	txJSONObj["timestamp"] = tx.timestamp
	txJSONObj["gasprice"] = tx.gasPrice.String()
	txJSONObj["gaslimit"] = tx.gasLimit
	txJSONObj["data"] = string(tx.Data())
	txJSONObj["type"] = tx.Type()
	txJSON, _ := json.Marshal(txJSONObj)
	return string(txJSON)
}

// NewTransaction create #Transaction instance.
func NewTransaction(chainID uint32, from, to *Address, value *big.Int, nonce uint64, payloadType string, payload []byte, gasPrice *big.Int, gasLimit uint64) (*Transaction, error) {
	tx := &Transaction{
		from:      from,
		to:        to,
		value:     value,
		nonce:     nonce,
		timestamp: time.Now().Unix(),
		chainID:   chainID,
		data:      &corepb.Data{Type: payloadType, Payload: payload},
		gasPrice:  gasPrice,
		gasLimit:  gasLimit,
	}
	return tx, nil
}

// Hash return the hash of transaction.
func (tx *Transaction) Hash() byteutils.Hash {
	return tx.hash
}

// SetHash set hash to in args
func (tx *Transaction) SetHash(in byteutils.Hash) {
	tx.hash = in
}

// GasPrice returns gasPrice
func (tx *Transaction) GasPrice() *big.Int {
	return tx.gasPrice
}

// GasLimit returns gasLimit
func (tx *Transaction) GasLimit() uint64 {
	return tx.gasLimit
}

// GasCountOfTxBase calculate the actual amount for a tx with data
func (tx *Transaction) GasCountOfTxBase() (*util.Uint128, error) {
	txGas := MinGasCountPerTransaction
	if tx.DataLen() > 0 {
		dataLen, err := util.NewUint128FromInt(int64(tx.DataLen()))
		if err != nil {
			return nil, err
		}
		dataGas, err := dataLen.Mul(GasCountPerByte)
		if err != nil {
			return nil, err
		}
		baseGas, err := txGas.Add(dataGas)
		if err != nil {
			return nil, err
		}
		txGas = baseGas
	}
	return txGas, nil
}

// DataLen return the length of payload
func (tx *Transaction) DataLen() int {
	return len(tx.data.Payload)
}

// Sign sign transaction,sign algorithm is
func (tx *Transaction) Sign(signature keystore.Signature) error {
	if signature == nil {
		return ErrNilArgument
	}
	hash, err := tx.HashTransaction()
	if err != nil {
		return err
	}
	sign, err := signature.Sign(hash)
	if err != nil {
		return err
	}
	tx.hash = hash
	tx.alg = signature.Algorithm()
	tx.sign = sign
	return nil
}

// VerifyIntegrity return transaction verify result, including Hash and Signature.
func (tx *Transaction) VerifyIntegrity(chainID uint32) error {
	// check ChainID.
	if tx.chainID != chainID {
		return ErrInvalidChainID
	}

	// check Hash.
	wantedHash, err := tx.HashTransaction()
	if err != nil {
		return err
	}
	if wantedHash.Equals(tx.hash) == false {
		return ErrInvalidTransactionHash
	}

	// check Signature.
	return tx.verifySign()

}

func (tx *Transaction) verifySign() error {
	signer, err := RecoverSignerFromSignature(tx.alg, tx.hash, tx.sign)
	if err != nil {
		return err
	}
	if !tx.from.Equals(signer) {
		return ErrInvalidTransactionSigner
	}
	return nil
}

// HashTransaction hash the transaction.
func (tx *Transaction) HashTransaction() (byteutils.Hash, error) {
	hasher := sha3.New256()

	tvalue, err := util.NewUint128FromBigInt(tx.value)
	if err != nil {
		return nil, err
	}
	value, err := tvalue.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	tgasPrice, err := util.NewUint128FromBigInt(tx.gasPrice)
	if err != nil {
		return nil, err
	}
	gasPrice, err := tgasPrice.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	gasLimit, err := util.NewUint128FromUint(tx.gasLimit).ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	data, err := proto.Marshal(tx.data)
	if err != nil {
		return nil, err
	}

	hasher.Write(tx.from.address)
	hasher.Write(tx.to.address)
	hasher.Write(value)
	hasher.Write(byteutils.FromUint64(tx.nonce))
	hasher.Write(byteutils.FromInt64(tx.timestamp))
	hasher.Write(data)
	hasher.Write(byteutils.FromUint32(tx.chainID))
	hasher.Write(gasPrice)
	hasher.Write(gasLimit)

	return hasher.Sum(nil), nil
}
