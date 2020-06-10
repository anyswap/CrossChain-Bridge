package dcrm

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/rpc/client"
)

// get dcrm sign status error
var (
	ErrGetSignStatusTimeout = errors.New("GetSignStatus timeout")
	ErrGetSignStatusFailed  = errors.New("GetSignStatus failure")
)

func newWrongStatusError(subject, status, errInfo string) error {
	return fmt.Errorf("[%v] Wrong status \"%v\", err=\"%v\"", subject, status, errInfo)
}

func wrapPostError(method string, err error) error {
	return fmt.Errorf("[post] %v error, %v", method, err)
}

func httpPost(result interface{}, method string, params ...interface{}) error {
	return client.RPCPost(&result, dcrmRPCAddress, method, params...)
}

// GetEnode call dcrm_getEnode
func GetEnode() (string, error) {
	var result GetEnodeResp
	err := httpPost(&result, "dcrm_getEnode")
	if err != nil {
		return "", wrapPostError("dcrm_getEnode", err)
	}
	if result.Status != "Success" {
		return "", newWrongStatusError("GetEnode", result.Status, result.Error)
	}
	return result.Data.Enode, nil
}

// GetSignNonce call dcrm_getSignNonce
func GetSignNonce() (uint64, error) {
	var result DataResultResp
	err := httpPost(&result, "dcrm_getSignNonce", keyWrapper.Address.String())
	if err != nil {
		return 0, wrapPostError("dcrm_getSignNonce", err)
	}
	if result.Status != "Success" {
		return 0, newWrongStatusError("GetSignNonce", result.Status, result.Error)
	}
	bi, err := common.GetBigIntFromStr(result.Data.Result)
	if err != nil {
		return 0, fmt.Errorf("GetSignNonce can't parse result as big int, %v", err)
	}
	return bi.Uint64(), nil
}

// GetSignStatus call dcrm_getSignStatus
func GetSignStatus(key string) (*SignStatus, error) {
	var result DataResultResp
	err := httpPost(&result, "dcrm_getSignStatus", key)
	if err != nil {
		return nil, wrapPostError("dcrm_getSignStatus", err)
	}
	if result.Status != "Success" {
		return nil, newWrongStatusError("GetSignStatus", result.Status, "responce error "+result.Error)
	}
	data := result.Data.Result
	var signStatus SignStatus
	err = json.Unmarshal([]byte(data), &signStatus)
	if err != nil {
		return nil, wrapPostError("dcrm_getSignStatus", err)
	}
	switch signStatus.Status {
	case "Failure":
		log.Info("GetSignStatus Failure", "keyID", key, "status", data)
		return nil, ErrGetSignStatusFailed
	case "Timeout":
		log.Info("GetSignStatus Timeout", "keyID", key, "status", data)
		return nil, ErrGetSignStatusTimeout
	case "Success":
		return &signStatus, nil
	default:
		return nil, newWrongStatusError("GetSignStatus", signStatus.Status, "sign status error "+signStatus.Error)
	}
}

// GetCurNodeSignInfo call dcrm_getCurNodeSignInfo
func GetCurNodeSignInfo() ([]*SignInfoData, error) {
	var result SignInfoResp
	err := httpPost(&result, "dcrm_getCurNodeSignInfo", keyWrapper.Address.String())
	if err != nil {
		return nil, wrapPostError("dcrm_getCurNodeSignInfo", err)
	}
	if result.Status != "Success" {
		return nil, newWrongStatusError("GetCurNodeSignInfo", result.Status, result.Error)
	}
	return result.Data, nil
}

// Sign call dcrm_sign
func Sign(raw string) (string, error) {
	var result DataResultResp
	err := httpPost(&result, "dcrm_sign", raw)
	if err != nil {
		return "", wrapPostError("dcrm_sign", err)
	}
	if result.Status != "Success" {
		return "", newWrongStatusError("Sign", result.Status, result.Error)
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
	if result.Status != "Success" {
		return "", newWrongStatusError("AcceptSign", result.Status, result.Error)
	}
	return result.Data.Result, nil
}

// GetGroupByID call dcrm_getGroupByID
func GetGroupByID(groupID string) (*GroupInfo, error) {
	var result GetGroupByIDResp
	err := httpPost(&result, "dcrm_getGroupByID", groupID)
	if err != nil {
		return nil, wrapPostError("dcrm_getGroupByID", err)
	}
	if result.Status != "Success" {
		return nil, newWrongStatusError("GetGroupByID", result.Status, result.Error)
	}
	return result.Data, nil
}
