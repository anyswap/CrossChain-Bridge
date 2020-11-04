package dcrm

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
)

// get dcrm sign status error
var (
	ErrGetSignStatusTimeout = errors.New("getSignStatus timeout")
	ErrGetSignStatusFailed  = errors.New("getSignStatus failure")
)

const (
	successStatus = "Success"
)

func newWrongStatusError(subject, status, errInfo string) error {
	return fmt.Errorf("[%v] Wrong status \"%v\", err=\"%v\"", subject, status, errInfo)
}

func wrapPostError(method string, err error) error {
	return fmt.Errorf("[post] %v error, %v", method, err)
}

func httpPost(result interface{}, method string, params ...interface{}) error {
	return client.RPCPost(&result, defaultDcrmNode.dcrmRPCAddress, method, params...)
}

func httpPostTo(result interface{}, rpcAddress, method string, params ...interface{}) error {
	return client.RPCPost(&result, rpcAddress, method, params...)
}

// GetEnode call dcrm_getEnode
func GetEnode(rpcAddr string) (string, error) {
	var result GetEnodeResp
	err := httpPostTo(&result, rpcAddr, "dcrm_getEnode")
	if err != nil {
		return "", wrapPostError("dcrm_getEnode", err)
	}
	if result.Status != successStatus {
		return "", newWrongStatusError("getEnode", result.Status, result.Error)
	}
	return result.Data.Enode, nil
}

// GetSignNonce call dcrm_getSignNonce
func GetSignNonce(dcrmUser, rpcAddr string) (uint64, error) {
	var result DataResultResp
	err := httpPostTo(&result, rpcAddr, "dcrm_getSignNonce", dcrmUser)
	if err != nil {
		return 0, wrapPostError("dcrm_getSignNonce", err)
	}
	if result.Status != successStatus {
		return 0, newWrongStatusError("getSignNonce", result.Status, result.Error)
	}
	bi, err := common.GetBigIntFromStr(result.Data.Result)
	if err != nil {
		return 0, fmt.Errorf("getSignNonce can't parse result as big int, %v", err)
	}
	return bi.Uint64(), nil
}

// GetSignStatus call dcrm_getSignStatus
func GetSignStatus(key, rpcAddr string) (*SignStatus, error) {
	var result DataResultResp
	err := httpPostTo(&result, rpcAddr, "dcrm_getSignStatus", key)
	if err != nil {
		return nil, wrapPostError("dcrm_getSignStatus", err)
	}
	if result.Status != successStatus {
		return nil, newWrongStatusError("getSignStatus", result.Status, "response error "+result.Error)
	}
	data := result.Data.Result
	var signStatus SignStatus
	err = json.Unmarshal([]byte(data), &signStatus)
	if err != nil {
		return nil, wrapPostError("dcrm_getSignStatus", err)
	}
	switch signStatus.Status {
	case "Failure":
		log.Info("getSignStatus Failure", "keyID", key, "status", data)
		return nil, ErrGetSignStatusFailed
	case "Timeout":
		log.Info("getSignStatus Timeout", "keyID", key, "status", data)
		return nil, ErrGetSignStatusTimeout
	case successStatus:
		return &signStatus, nil
	default:
		return nil, newWrongStatusError("getSignStatus", signStatus.Status, "sign status error "+signStatus.Error)
	}
}

// GetCurNodeSignInfo call dcrm_getCurNodeSignInfo
func GetCurNodeSignInfo() ([]*SignInfoData, error) {
	var result SignInfoResp
	err := httpPost(&result, "dcrm_getCurNodeSignInfo", defaultDcrmNode.keyWrapper.Address.String())
	if err != nil {
		return nil, wrapPostError("dcrm_getCurNodeSignInfo", err)
	}
	if result.Status != successStatus {
		return nil, newWrongStatusError("getCurNodeSignInfo", result.Status, result.Error)
	}
	return result.Data, nil
}

// Sign call dcrm_sign
func Sign(raw, rpcAddr string) (string, error) {
	var result DataResultResp
	err := httpPostTo(&result, rpcAddr, "dcrm_sign", raw)
	if err != nil {
		return "", wrapPostError("dcrm_sign", err)
	}
	if result.Status != successStatus {
		return "", newWrongStatusError("sign", result.Status, result.Error)
	}
	return result.Data.Result, nil
}

// AcceptSign call dcrm_acceptSign
func AcceptSign(raw string) (string, error) {
	var result DataResultResp
	err := httpPost(&result, "dcrm_acceptSign", raw)
	if err != nil {
		return "", wrapPostError("dcrm_acceptSign", err)
	}
	if result.Status != successStatus {
		return "", newWrongStatusError("acceptSign", result.Status, result.Error)
	}
	return result.Data.Result, nil
}

// GetGroupByID call dcrm_getGroupByID
func GetGroupByID(groupID, rpcAddr string) (*GroupInfo, error) {
	var result GetGroupByIDResp
	err := httpPostTo(&result, rpcAddr, "dcrm_getGroupByID", groupID)
	if err != nil {
		return nil, wrapPostError("dcrm_getGroupByID", err)
	}
	if result.Status != successStatus {
		return nil, newWrongStatusError("getGroupByID", result.Status, result.Error)
	}
	return result.Data, nil
}

// GetBip32ChildKey get bip32 child key
func GetBip32ChildKey(rootPubkey, inputCode string) (string, error) {
	var result DataResultResp
	err := httpPost(&result, "dcrm_getBip32ChildKey", rootPubkey, inputCode)
	if err != nil {
		return "", wrapPostError("dcrm_getBip32ChildKey", err)
	}
	if result.Status != successStatus {
		return "", newWrongStatusError("getBip32ChildKey", result.Status, result.Error)
	}
	return result.Data.Result, nil
}
