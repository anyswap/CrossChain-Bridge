package dcrm

import (
	"crypto/rand"
	"encoding/json"
	"errors"
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

const (
	pingCount                  = 3
	retrySignCount             = 3
	retryGetSignStatusCount    = 70
	retryGetSignStatusInterval = 10 * time.Second
)

func pingDcrmNode(nodeInfo *NodeInfo) (err error) {
	rpcAddr := nodeInfo.dcrmRPCAddress
	for j := 0; j < pingCount; j++ {
		_, err = GetEnode(rpcAddr)
		if err == nil {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	log.Error("pingDcrmNode failed", "rpcAddr", rpcAddr, "pingCount", pingCount, "err", err)
	return err
}

// DoSignOne dcrm sign single msgHash with context msgContext
func DoSignOne(signPubkey, msgHash, msgContext string) (keyID string, rsvs []string, err error) {
	return DoSign(signPubkey, []string{msgHash}, []string{msgContext})
}

// DoSign dcrm sign msgHash with context msgContext
func DoSign(signPubkey string, msgHash, msgContext []string) (keyID string, rsvs []string, err error) {
	if !params.IsDcrmEnabled() {
		return "", nil, errors.New("dcrm sign is disabled")
	}
	log.Debug("dcrm DoSign", "msgHash", msgHash, "msgContext", msgContext)
	if signPubkey == "" {
		return "", nil, errors.New("dcrm sign with empty public key")
	}
	var pingOk bool
	for retry := 0; retry < retrySignCount; retry++ {
		for _, dcrmNode := range allInitiatorNodes {
			if err = pingDcrmNode(dcrmNode); err != nil {
				continue
			}
			pingOk = true
			signGroupsCount := int64(len(dcrmNode.signGroups))
			// randomly pick first subgroup to sign
			randIndex, _ := rand.Int(rand.Reader, big.NewInt(signGroupsCount))
			startIndex := randIndex.Int64()
			i := startIndex
			for {
				keyID, rsvs, err = doSignImpl(dcrmNode, i, signPubkey, msgHash, msgContext)
				if err == nil {
					return keyID, rsvs, nil
				}
				i = (i + 1) % signGroupsCount
				if i == startIndex {
					break
				}
			}
		}
	}
	if !pingOk {
		err = errors.New("dcrm sign ping dcrm node failed")
	}
	return "", nil, err
}

func doSignImpl(dcrmNode *NodeInfo, signGroupIndex int64, signPubkey string, msgHash, msgContext []string) (keyID string, rsvs []string, err error) {
	nonce, err := GetSignNonce(dcrmNode.dcrmUser.String(), dcrmNode.dcrmRPCAddress)
	if err != nil {
		return "", nil, err
	}
	txdata := SignData{
		TxType:     "SIGN",
		PubKey:     signPubkey,
		MsgHash:    msgHash,
		MsgContext: msgContext,
		Keytype:    "ECDSA",
		GroupID:    dcrmNode.signGroups[signGroupIndex],
		ThresHold:  dcrmThreshold,
		Mode:       dcrmMode,
		TimeStamp:  common.NowMilliStr(),
	}
	payload, _ := json.Marshal(txdata)
	rawTX, err := BuildDcrmRawTx(nonce, payload, dcrmNode.keyWrapper)
	if err != nil {
		return "", nil, err
	}

	rpcAddr := dcrmNode.dcrmRPCAddress
	keyID, err = Sign(rawTX, rpcAddr)
	if err != nil {
		return "", nil, err
	}

	time.Sleep(retryGetSignStatusInterval)
	var signStatus *SignStatus
	i := 0
	for ; i < retryGetSignStatusCount; i++ {
		signStatus, err = GetSignStatus(keyID, rpcAddr)
		if err == nil {
			rsvs = signStatus.Rsv
			break
		}
		switch err {
		case ErrGetSignStatusFailed, ErrGetSignStatusTimeout:
			return "", nil, err
		}
		log.Warn("retry get sign status as error", "keyID", keyID, "err", err)
		time.Sleep(retryGetSignStatusInterval)
	}
	if i == retryGetSignStatusCount || len(rsvs) == 0 {
		return "", nil, errors.New("get sign status failed")
	}

	return keyID, rsvs, err
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
