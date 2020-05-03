package dcrm

import (
	"encoding/json"
	"math/big"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/tools/crypto"
	"github.com/fsn-dev/crossChain-Bridge/tools/rlp"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

func DoSign(msgHash string) (string, error) {
	nonce, err := GetSignNonce()
	if err != nil {
		return "", err
	}
	txdata := SignData{
		TxType:    "SIGN",
		PubKey:    pubkey,
		MsgHash:   msgHash,
		Keytype:   "ECDSA",
		GroupID:   groupID,
		ThresHold: threshold,
		Mode:      mode,
		TimeStamp: common.NowMilliStr(),
	}
	payload, _ := json.Marshal(txdata)
	rawTX, err := BuildDcrmRawTx(nonce, payload)
	if err != nil {
		return "", err
	}
	return Sign(rawTX)
}

func DoAcceptSign(keyID string, agreeResult string) (string, error) {
	nonce := uint64(0)
	data := AcceptData{
		TxType:    "ACCEPTSIGN",
		Key:       keyID,
		Accept:    agreeResult,
		TimeStamp: common.NowMilliStr(),
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	rawTX, err := BuildDcrmRawTx(nonce, payload)
	if err != nil {
		return "", err
	}
	return AcceptSign(rawTX)
}

func BuildDcrmRawTx(nonce uint64, payload []byte) (string, error) {
	tx := types.NewTransaction(
		nonce,             // nonce
		DcrmToAddr,        // to address
		big.NewInt(0),     // value
		100000,            // gasLimit
		big.NewInt(80000), // gasPrice
		payload,           // data
	)
	signature, err := crypto.Sign(Signer.Hash(tx).Bytes(), keyWrapper.PrivateKey)
	if err != nil {
		return "", err
	}
	sigTx, err := tx.WithSignature(Signer, signature)
	if err != nil {
		return "", err
	}
	txdata, err := rlp.EncodeToBytes(sigTx)
	if err != nil {
		return "", err
	}
	rawTX := common.ToHex(txdata)
	return rawTX, nil
}
