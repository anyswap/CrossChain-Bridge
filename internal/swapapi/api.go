package swapapi

import (
	"time"

	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/params"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	rpcjson "github.com/gorilla/rpc/v2/json2"
)

func newRpcError(ec rpcjson.ErrorCode, message string) error {
	return &rpcjson.Error{
		Code:    ec,
		Message: message,
	}
}

func GetServerInfo() (*ServerInfo, error) {
	config := params.GetConfig()
	if config == nil {
		return nil, nil
	}
	return &ServerInfo{
		SrcToken:  config.SrcToken,
		DestToken: config.DestToken,
	}, nil
}

func GetSwapStatistics() (*SwapStatistics, error) {
	stat := &SwapStatistics{}
	return stat, nil
}

func GetSwapin(txid *string) (*SwapInfo, error) {
	txidstr := *txid
	result, err := mongodb.FindSwapinResult(txidstr)
	if err == nil {
		return ConvertMgoSwapResultToSwapInfo(result), nil
	}
	register, err := mongodb.FindSwapin(txidstr)
	if err == nil {
		return ConvertMgoSwapToSwapInfo(register), nil
	}
	return nil, mongodb.ErrSwapNotFound
}

func GetSwapout(txid *string) (*SwapInfo, error) {
	txidstr := *txid
	result, err := mongodb.FindSwapoutResult(txidstr)
	if err == nil {
		return ConvertMgoSwapResultToSwapInfo(result), nil
	}
	register, err := mongodb.FindSwapout(txidstr)
	if err == nil {
		return ConvertMgoSwapToSwapInfo(register), nil
	}
	return nil, mongodb.ErrSwapNotFound
}

func processHistoryLimit(limit int) int {
	if limit == 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	return limit
}

func GetSwapinHistory(address string, offset, limit int) ([]*SwapInfo, error) {
	limit = processHistoryLimit(limit)
	result, err := mongodb.FindSwapinResults(address, offset, limit)
	if err != nil {
		return nil, err
	}
	return ConvertMgoSwapResultsToSwapInfos(result), nil
}

func GetSwapoutHistory(address string, offset, limit int) ([]*SwapInfo, error) {
	limit = processHistoryLimit(limit)
	result, err := mongodb.FindSwapoutResults(address, offset, limit)
	if err != nil {
		return nil, err
	}
	return ConvertMgoSwapResultsToSwapInfos(result), nil
}

func Swapin(txid *string) (*PostResult, error) {
	txidstr := *txid
	info, err := tokens.SrcBridge.VerifyTransaction(txidstr)
	if err != nil {
		return nil, newRpcError(-32099, "verify swapin failed! "+err.Error())
	}
	swap := &mongodb.MgoSwap{
		Key:       txidstr,
		TxId:      txidstr,
		Status:    mongodb.TxNotStable,
		Timestamp: time.Now().Unix(),
		Memo:      info.Bind,
	}
	err = mongodb.AddSwapin(swap)
	if err != nil {
		return nil, err
	}
	log.Info("[api] add swapin", "swap", swap)
	return &SuccessPostResult, nil
}

func Swapout(txid *string) (*PostResult, error) {
	txidstr := *txid
	info, err := tokens.DstBridge.VerifyTransaction(txidstr)
	if err != nil {
		return nil, newRpcError(-32098, "verify swapout failed! "+err.Error())
	}
	swap := &mongodb.MgoSwap{
		Key:       txidstr,
		TxId:      txidstr,
		Status:    mongodb.TxNotStable,
		Timestamp: time.Now().Unix(),
		Memo:      info.Bind,
	}
	err = mongodb.AddSwapout(swap)
	if err != nil {
		return nil, err
	}
	log.Info("[api] add swapout", "swap", swap)
	return &SuccessPostResult, nil
}

func RecallSwapin(txid *string) (*PostResult, error) {
	txidstr := *txid
	err := mongodb.RecallSwapin(txidstr)
	if err != nil {
		return nil, err
	}
	log.Info("[api] add recall swap", "txid", txidstr)
	return &SuccessPostResult, nil
}
