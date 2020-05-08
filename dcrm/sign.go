package dcrm

import (
	"crypto/rand"
	"encoding/json"
	"math/big"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/tools/crypto"
	"github.com/fsn-dev/crossChain-Bridge/tools/rlp"
	"github.com/fsn-dev/crossChain-Bridge/types"
)

func DoSign(msgHash, msgContext string) (string, error) {
	nonce, err := GetSignNonce()
	if err != nil {
		return "", err
	}
	// randomly pick sub-group to sign
	randIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(len(SignGroups))))
	signGroup := SignGroups[randIndex.Int64()]
	txdata := SignData{
		TxType:     "SIGN",
		PubKey:     signPubkey,
		MsgHash:    msgHash,
		MsgContext: msgContext,
		Keytype:    "ECDSA",
		GroupID:    signGroup,
		ThresHold:  threshold,
		Mode:       mode,
		TimeStamp:  common.NowMilliStr(),
	}
	payload, _ := json.Marshal(txdata)
	rawTX, err := BuildDcrmRawTx(nonce, payload)
	if err != nil {
		return "", err
	}
	return Sign(rawTX)
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
