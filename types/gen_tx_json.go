package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/common/hexutil"
)

type txJSON struct {
	Type hexutil.Uint64 `json:"type"`

	// Common transaction fields:
	AccountNonce         *hexutil.Uint64 `json:"nonce"`
	Price                *hexutil.Big    `json:"gasPrice"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas,omitempty"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas,omitempty"`
	GasLimit             *hexutil.Uint64 `json:"gas"`
	Recipient            *common.Address `json:"to"`
	Amount               *hexutil.Big    `json:"value"`
	Payload              *hexutil.Bytes  `json:"input"`
	V                    *hexutil.Big    `json:"v"`
	R                    *hexutil.Big    `json:"r"`
	S                    *hexutil.Big    `json:"s"`

	// Access list transaction fields:
	ChainID    *hexutil.Big `json:"chainId,omitempty"`
	AccessList *AccessList  `json:"accessList,omitempty"`

	// Only used for encoding:
	Hash *common.Hash `json:"hash"`
}

// MarshalJSON marshals as JSON.
func (t *txdata) MarshalJSON() ([]byte, error) {
	var enc txJSON
	enc.Type = hexutil.Uint64(t.Type)
	enc.AccountNonce = (*hexutil.Uint64)(&t.AccountNonce)
	enc.GasLimit = (*hexutil.Uint64)(&t.GasLimit)
	enc.Recipient = t.Recipient
	enc.Amount = (*hexutil.Big)(t.Amount)
	enc.Payload = (*hexutil.Bytes)(&t.Payload)
	enc.V = (*hexutil.Big)(t.V)
	enc.R = (*hexutil.Big)(t.R)
	enc.S = (*hexutil.Big)(t.S)
	enc.Hash = t.Hash

	if t.Type != LegacyTxType {
		enc.ChainID = (*hexutil.Big)(t.ChainID)
		enc.AccessList = &t.AccessList

	}
	if t.Type == DynamicFeeTxType {
		enc.MaxFeePerGas = (*hexutil.Big)(t.MaxFeePerGas)
		enc.MaxPriorityFeePerGas = (*hexutil.Big)(t.MaxPriorityFeePerGas)
	} else {
		enc.Price = (*hexutil.Big)(t.Price)
	}
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (t *txdata) UnmarshalJSON(input []byte) error {
	var dec txJSON
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	t.Type = uint8(uint64(dec.Type))
	if dec.AccountNonce == nil {
		return errors.New("missing required field 'nonce' for txdata")
	}
	t.AccountNonce = uint64(*dec.AccountNonce)
	if t.Type != LegacyTxType {
		if dec.AccessList != nil {
			t.AccessList = *dec.AccessList
		}
		if dec.ChainID == nil {
			return errors.New("missing required field 'chainId' in transaction")
		}
		t.ChainID = (*big.Int)(dec.ChainID)
	}
	if t.Type == DynamicFeeTxType {
		if dec.MaxPriorityFeePerGas == nil {
			return errors.New("missing required field 'maxPriorityFeePerGas' for txdata")
		}
		t.MaxPriorityFeePerGas = (*big.Int)(dec.MaxPriorityFeePerGas)
		if dec.MaxFeePerGas == nil {
			return errors.New("missing required field 'maxFeePerGas' for txdata")
		}
		t.MaxFeePerGas = (*big.Int)(dec.MaxFeePerGas)
	} else {
		if dec.Price == nil {
			return errors.New("missing required field 'gasPrice' for txdata")
		}
		t.Price = (*big.Int)(dec.Price)
	}
	if dec.GasLimit == nil {
		return errors.New("missing required field 'gas' for txdata")
	}
	t.GasLimit = uint64(*dec.GasLimit)
	if dec.Recipient != nil {
		t.Recipient = dec.Recipient
	}
	if dec.Amount == nil {
		return errors.New("missing required field 'value' for txdata")
	}
	t.Amount = (*big.Int)(dec.Amount)
	if dec.Payload == nil {
		return errors.New("missing required field 'input' for txdata")
	}
	t.Payload = *dec.Payload
	if dec.V == nil {
		return errors.New("missing required field 'v' for txdata")
	}
	t.V = (*big.Int)(dec.V)
	if dec.R == nil {
		return errors.New("missing required field 'r' for txdata")
	}
	t.R = (*big.Int)(dec.R)
	if dec.S == nil {
		return errors.New("missing required field 's' for txdata")
	}
	t.S = (*big.Int)(dec.S)
	return nil
}

// PrintPretty print pretty (json)
func (tx *Transaction) PrintPretty() {
	bs, _ := json.MarshalIndent(tx, "", "  ")
	fmt.Println(string(bs))
}

// PrintRaw print raw encoded (hex string)
func (tx *Transaction) PrintRaw() {
	bs, _ := tx.MarshalBinary()
	fmt.Println(hexutil.Bytes(bs))
}

// RawStr return raw encoded (hex string)
func (tx *Transaction) RawStr() string {
	bs, _ := tx.MarshalBinary()
	return common.ToHex(bs)
}
