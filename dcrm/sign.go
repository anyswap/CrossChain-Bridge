package dcrm

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tools/crypto"
	"github.com/anyswap/CrossChain-Bridge/tools/keystore"
	"github.com/anyswap/CrossChain-Bridge/tools/rlp"
	"github.com/anyswap/CrossChain-Bridge/types"
)

const (
	pingCount     = 3
	retrySignLoop = 3
)

var (
	errSignIsDisabled       = errors.New("sign is disabled")
	errSignTimerTimeout     = errors.New("sign timer timeout")
	errDoSignFailed         = errors.New("do sign failed")
	errSignWithoutPublickey = errors.New("sign without public key")
	errGetSignResultFailed  = errors.New("get sign result failed")
	errRValueIsUsed         = errors.New("r value is already used")
	errWrongSignatureLength = errors.New("wrong signature length")
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
		return "", nil, errSignIsDisabled
	}
	log.Debug("dcrm DoSign", "msgHash", msgHash, "msgContext", msgContext)
	if signPubkey == "" {
		return "", nil, errSignWithoutPublickey
	}
	for i := 0; i < retrySignLoop; i++ {
		for _, dcrmNode := range allInitiatorNodes {
			if err = pingDcrmNode(dcrmNode); err != nil {
				continue
			}
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
		time.Sleep(2 * time.Second)
	}
	log.Warn("dcrm DoSign failed", "msgHash", msgHash, "msgContext", msgContext, "err", err)
	return "", nil, errDoSignFailed
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
		Keytype:    dcrmSignType,
		GroupID:    dcrmNode.signGroups[signGroupIndex],
		ThresHold:  dcrmThreshold,
		Mode:       dcrmMode,
		TimeStamp:  common.NowMilliStr(),
	}
	payload, err := json.Marshal(txdata)
	if err != nil {
		return "", nil, err
	}
	if verifySignatureInAccept {
		// append payload signature into the end of message context
		sighash := common.Keccak256Hash(payload)
		signature, errf := crypto.Sign(sighash[:], dcrmNode.keyWrapper.PrivateKey)
		if errf != nil {
			return "", nil, errf
		}
		txdata.MsgContext = append(txdata.MsgContext, common.ToHex(signature))
		payload, _ = json.Marshal(txdata)
	}

	rawTX, err := BuildDcrmRawTx(nonce, payload, dcrmNode.keyWrapper)
	if err != nil {
		return "", nil, err
	}

	rpcAddr := dcrmNode.dcrmRPCAddress
	keyID, err = Sign(rawTX, rpcAddr)
	if err != nil {
		return "", nil, err
	}

	rsvs, err = getSignResult(keyID, rpcAddr)
	if err != nil {
		return "", nil, err
	}

	if isECDSA() && mongodb.HasClient() { // prevent multiple use of same r value
		for _, rsv := range rsvs {
			signature := common.FromHex(rsv)
			if len(signature) != crypto.SignatureLength {
				return "", nil, errWrongSignatureLength
			}
			r := common.ToHex(signature[:32])
			err = mongodb.AddUsedRValue(signPubkey, r)
			if err != nil {
				return "", nil, errRValueIsUsed
			}
		}
	}

	return keyID, rsvs, nil
}

// GetSignStatusByKeyID get sign status by keyID
func GetSignStatusByKeyID(keyID string) (rsvs []string, err error) {
	return getSignResult(keyID, defaultDcrmNode.dcrmRPCAddress)
}

func getSignResult(keyID, rpcAddr string) (rsvs []string, err error) {
	log.Info("start get sign status", "keyID", keyID)
	var signStatus *SignStatus
	i := 0
	signTimer := time.NewTimer(dcrmSignTimeout)
	defer signTimer.Stop()
LOOP_GET_SIGN_STATUS:
	for {
		i++
		select {
		case <-signTimer.C:
			if err == nil {
				err = errSignTimerTimeout
			}
			break LOOP_GET_SIGN_STATUS
		default:
			signStatus, err = GetSignStatus(keyID, rpcAddr)
			if err == nil {
				rsvs = signStatus.Rsv
				break LOOP_GET_SIGN_STATUS
			}
			switch {
			case errors.Is(err, ErrGetSignStatusHasDisagree),
				errors.Is(err, ErrGetSignStatusFailed),
				errors.Is(err, ErrGetSignStatusTimeout):
				break LOOP_GET_SIGN_STATUS
			}
		}
		log.Trace("get sign status failed", "keyID", keyID, "count", i, "err", err)
		time.Sleep(3 * time.Second)
	}
	if len(rsvs) == 0 || err != nil {
		log.Info("get sign status failed", "keyID", keyID, "retryCount", i, "err", err)
		return nil, errGetSignResultFailed
	}
	log.Info("get sign status success", "keyID", keyID, "retryCount", i)
	return rsvs, nil
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

// HasValidSignature has valid signature
func (s *SignInfoData) HasValidSignature() bool {
	msgContextLen := len(s.MsgContext)
	if !verifySignatureInAccept {
		return msgContextLen == 1
	}

	if msgContextLen != 2 {
		return false
	}
	msgContext := s.MsgContext[:msgContextLen-1]
	msgSig := common.FromHex(s.MsgContext[msgContextLen-1])

	txdata := SignData{
		TxType:     "SIGN",
		PubKey:     s.PubKey,
		MsgHash:    s.MsgHash,
		MsgContext: msgContext,
		Keytype:    dcrmSignType,
		GroupID:    s.GroupID,
		ThresHold:  s.ThresHold,
		Mode:       s.Mode,
		TimeStamp:  s.TimeStamp,
	}
	payload, _ := json.Marshal(txdata)
	sighash := common.Keccak256Hash(payload)

	// recover the public key from the signature
	pub, err := crypto.Ecrecover(sighash[:], msgSig)
	if err != nil {
		return false
	}
	if len(pub) == 0 || pub[0] != 4 {
		return false
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(pub[1:])[12:])
	return addr == common.HexToAddress(s.Account)
}
