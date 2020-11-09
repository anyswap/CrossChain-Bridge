package dcrm

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tools/crypto"
	"github.com/anyswap/CrossChain-Bridge/tools/keystore"
	"github.com/anyswap/CrossChain-Bridge/tools/rlp"
	"github.com/anyswap/CrossChain-Bridge/types"
)

func getDcrmNode() *NodeInfo {
	countOfInitiators := len(allInitiatorNodes)
	if countOfInitiators < 2 {
		return defaultDcrmNode
	}
	i, pingCount := 0, 3
	for {
		nodeInfo := allInitiatorNodes[i]
		rpcAddr := nodeInfo.dcrmRPCAddress
		for j := 0; j < pingCount; j++ {
			_, err := GetEnode(rpcAddr)
			if err == nil {
				return nodeInfo
			}
			log.Error("GetEnode of initiator failed", "rpcAddr", rpcAddr, "times", j+1, "err", err)
			time.Sleep(1 * time.Second)
		}
		i = (i + 1) % countOfInitiators
		if i == 0 {
			log.Error("GetEnode of initiator failed all")
			time.Sleep(60 * time.Second)
		}
	}
}

// DoSignOne dcrm sign single msgHash with context msgContext
func DoSignOne(signPubkey, inputCode, msgHash, msgContext string) (rpcAddr, result string, err error) {
	return DoSign(signPubkey, inputCode, []string{msgHash}, []string{msgContext})
}

// DoSign dcrm sign msgHash with context msgContext
func DoSign(signPubkey, inputCode string, msgHash, msgContext []string) (rpcAddr, result string, err error) {
	if !params.IsDcrmEnabled() {
		return "", "", fmt.Errorf("dcrm sign is disabled")
	}
	log.Debug("dcrm DoSign", "msgHash", msgHash, "msgContext", msgContext)
	if signPubkey == "" {
		return "", "", fmt.Errorf("dcrm sign with empty public key")
	}
	dcrmNode := getDcrmNode()
	if dcrmNode == nil {
		return "", "", fmt.Errorf("dcrm sign with nil node info")
	}
	nonce, err := GetSignNonce(dcrmNode.dcrmUser.String(), dcrmNode.dcrmRPCAddress)
	if err != nil {
		return "", "", err
	}
	// randomly pick sub-group to sign
	signGroups := dcrmNode.signGroups
	randIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(len(signGroups))))
	signGroup := signGroups[randIndex.Int64()]
	txdata := SignData{
		TxType:     "SIGN",
		PubKey:     signPubkey,
		InputCode:  inputCode,
		MsgHash:    msgHash,
		MsgContext: msgContext,
		Keytype:    "ECDSA",
		GroupID:    signGroup,
		ThresHold:  dcrmThreshold,
		Mode:       dcrmMode,
		TimeStamp:  common.NowMilliStr(),
	}
	payload, _ := json.Marshal(txdata)
	rawTX, err := BuildDcrmRawTx(nonce, payload, dcrmNode.keyWrapper)
	if err != nil {
		return "", "", err
	}
	rpcAddr = dcrmNode.dcrmRPCAddress
	result, err = Sign(rawTX, rpcAddr)
	return rpcAddr, result, err
}

// BuildDcrmRawTx build dcrm raw tx
func BuildDcrmRawTx(nonce uint64, payload []byte, keyWrapper *keystore.Key) (string, error) {
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
