package dcrm

import (
	"encoding/json"

	"github.com/anyswap/CrossChain-Bridge/common"
)

// DoAcceptSign accept sign
func DoAcceptSign(keyID, agreeResult string, msgHash, msgContext []string) (string, error) {
	nonce := uint64(0)
	data := AcceptData{
		TxType:  "ACCEPTSIGN",
		Key:     keyID,
		Accept:  agreeResult,
		MsgHash: msgHash,
		//MsgContext: msgContext, // context is verified on top level
		TimeStamp: common.NowMilliStr(),
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	rawTX, err := BuildDcrmRawTx(nonce, payload, defaultDcrmNode.keyWrapper)
	if err != nil {
		return "", err
	}
	return AcceptSign(rawTX)
}
