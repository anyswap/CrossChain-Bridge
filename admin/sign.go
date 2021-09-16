// Package admin provides methods to sign message and to verify signed message
package admin

import (
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/common/hexutil"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tools"
	"github.com/anyswap/CrossChain-Bridge/tools/keystore"
	"github.com/anyswap/CrossChain-Bridge/tools/rlp"
	"github.com/anyswap/CrossChain-Bridge/types"
)

const (
	// swapAdminToAddress used in swap admin sign and verify
	swapAdminToAddress = "0x00000000000000000000000000000000000000cc"
	// swapAdminChainID to make swap admin signer
	swapAdminChainID = 30300
)

var (
	adminSigner = types.MakeSigner("EIP155", big.NewInt(swapAdminChainID))
	adminToAddr = common.HexToAddress(swapAdminToAddress)

	keyWrapper *keystore.Key

	// admin tx lifetime
	maxExpireSeconds int64 = 120
	maxFutureSeconds int64 = 30
)

// CallArgs call args
type CallArgs struct {
	Method    string   `json:"method"`
	Params    []string `json:"params"`
	Timestamp int64    `json:"timestamp"`
}

// Sign sign
func Sign(method string, params []string) (rawTx string, err error) {
	log.Info("admin Sign", "method", method, "params", params)
	payload, err := encodeCallArgs(method, params)
	if err != nil {
		return "", err
	}

	tx := types.NewTransaction(
		0,             // nonce
		adminToAddr,   // to address
		big.NewInt(0), // value
		0,             // gasLimit
		big.NewInt(0), // gasPrice
		payload,       // data
	)

	signedTx, err := types.SignTx(tx, adminSigner, keyWrapper.PrivateKey)
	if err != nil {
		return "", err
	}

	txdata, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		return "", err
	}

	return common.ToHex(txdata), nil
}

// LoadKeyStore load keystore
func LoadKeyStore(keyfile, passfile string) error {
	key, err := tools.LoadKeyStore(keyfile, passfile)
	if err != nil {
		return err
	}
	keyWrapper = key
	log.Info("[admin] load keystore success", "address", keyWrapper.Address.String())
	return nil
}

func encodeCallArgs(method string, params []string) ([]byte, error) {
	args := CallArgs{
		Method:    method,
		Params:    params,
		Timestamp: time.Now().Unix(),
	}
	return json.Marshal(args)
}

func decodeCallArgs(data []byte) (*CallArgs, error) {
	var args CallArgs
	err := json.Unmarshal(data, &args)
	if err != nil {
		return nil, err
	}
	return &args, nil
}

// VerifyTransaction get sender
func VerifyTransaction(tx *types.Transaction) (*common.Address, *CallArgs, error) {
	if tx.To() == nil || *tx.To() != adminToAddr {
		return nil, nil, errors.New("wrong admin tx to address")
	}
	args, err := decodeCallArgs(tx.Data())
	if err != nil {
		return nil, nil, err
	}
	timestamp := args.Timestamp
	now := time.Now().Unix()
	if now-timestamp > maxExpireSeconds {
		return nil, nil, errors.New("expired admin tx timestamp")
	}
	if now+maxFutureSeconds < timestamp {
		return nil, nil, errors.New("future admin tx timestamp")
	}
	sender, err := adminSigner.Sender(tx) // will verify signature
	if err != nil {
		return nil, nil, err
	}
	return &sender, args, nil
}

// DecodeTransaction decode tx from hex string
func DecodeTransaction(rawTx string) (*types.Transaction, error) {
	data, err := hexutil.Decode(rawTx)
	if err != nil {
		return nil, err
	}

	var tx types.Transaction
	err = rlp.DecodeBytes(data, &tx)
	if err != nil {
		return nil, err
	}

	return &tx, nil
}
