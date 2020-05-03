package dcrm

import (
	"fmt"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/rpc/client"
)

func newWrongStatusError(errInfo string) error {
	return fmt.Errorf("Wrong status, %v", errInfo)
}

func httpPost(result interface{}, method string, params ...interface{}) error {
	return client.RpcPost(&result, dcrmRpcAddress, method, params...)
}

func GetEnode() (string, error) {
	var result GetEnodeResp
	err := httpPost(&result, "dcrm_getEnode")
	if err != nil {
		return "", err
	}
	if result.Status != "Success" {
		return "", newWrongStatusError(result.Error)
	}
	return result.Data.Enode, nil
}

func GetSignNonce() (uint64, error) {
	var result DataResultResp
	err := httpPost(&result, "dcrm_getSignNonce", keyWrapper.Address.String())
	if err != nil {
		return 0, err
	}
	if result.Status != "Success" {
		return 0, newWrongStatusError(result.Error)
	}
	bi, err := common.GetBigIntFromStr(result.Data.Result)
	if err != nil {
		return 0, newWrongStatusError(err.Error())
	}
	return bi.Uint64(), nil
}

func GetSignStatus(key string) (*SignStatus, error) {
	var result SignStatusResp
	err := httpPost(&result, "dcrm_getSignStatus", key)
	if err != nil {
		return nil, err
	}
	if result.Status != "Success" {
		return nil, newWrongStatusError(result.Error)
	}
	return &SignStatus{
		Rsv:       result.Rsv,
		AllReply:  result.AllReply,
		TimeStamp: result.TimeStamp,
	}, nil
}

func GetCurNodeSignInfo() ([]*SignInfoData, error) {
	var result SignInfoResp
	err := httpPost(&result, "dcrm_getCurNodeSignInfo", keyWrapper.Address.String())
	if err != nil {
		return nil, err
	}
	if result.Status != "Success" {
		return nil, newWrongStatusError(result.Error)
	}
	return result.Data, nil
}

func Sign(raw string) (string, error) {
	var result DataResultResp
	err := httpPost(&result, "dcrm_sign", raw)
	if err != nil {
		return "", err
	}
	if result.Status != "Success" {
		return "", newWrongStatusError(result.Error)
	}
	return result.Data.Result, nil
}

func AcceptSign(raw string) (string, error) {
	var result DataResultResp
	err := httpPost(&result, "dcrm_acceptSign", raw)
	if err != nil {
		return "", err
	}
	if result.Status != "Success" {
		return "", newWrongStatusError(result.Error)
	}
	return result.Data.Result, nil
}
