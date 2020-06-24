package dcrm

import (
	"crypto/rand"
	"encoding/json"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tools/crypto"
	"github.com/anyswap/CrossChain-Bridge/tools/rlp"
	"github.com/anyswap/CrossChain-Bridge/types"
)

// DoSignOne dcrm sign single msgHash with context msgContext
func DoSignOne(msgHash, msgContext string) (string, error) {
	return DoSign([]string{msgHash}, []string{msgContext})
}

// DoSign dcrm sign msgHash with context msgContext
func DoSign(msgHash, msgContext []string) (string, error) {
	log.Debug("dcrm DoSign", "msgHash", msgHash, "msgContext", msgContext)
	nonce, err := GetSignNonce()
	if err != nil {
		return "", err
	}
	// randomly pick sub-group to sign
	randIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(len(signGroups))))
	signGroup := signGroups[randIndex.Int64()]
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

// BuildDcrmRawTx build dcrm raw tx
func BuildDcrmRawTx(nonce uint64, payload []byte) (string, error) {
	tx := types.NewTransaction(
		nonce,             // nonce
		dcrmToAddr,        // to address
		big.NewInt(0),     // value
		100000,            // gasLimit
		big.NewInt(80000), // gasPrice
		payload,           // data
	)
	signature, err := crypto.Sign(dcrmSigner.Hash(tx).Bytes(), keyWrapper.PrivateKey)
	if err != nil {
		return "", err
	}
	sigTx, err := tx.WithSignature(dcrmSigner, signature)
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
