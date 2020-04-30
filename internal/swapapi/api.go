package swapapi

import (
	"time"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/params"
)

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

func GetSwapin(txid *common.Hash) (*SwapInfo, error) {
	txidstr := txid.String()
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

func GetSwapout(txid *common.Hash) (*SwapInfo, error) {
	txidstr := txid.String()
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

func Swapin(txid *common.Hash) (*PostResult, error) {
	txidstr := txid.String()
	swap := &mongodb.MgoSwap{
		Key:       txidstr,
		TxId:      txidstr,
		Status:    mongodb.TxNotStable,
		Timestamp: time.Now().Unix(),
		Memo:      "",
	}
	err := mongodb.AddSwapin(swap)
	if err != nil {
		return nil, err
	}
	return &SuccessPostResult, nil
}

func Swapout(txid *common.Hash) (*PostResult, error) {
	txidstr := txid.String()
	swap := &mongodb.MgoSwap{
		Key:       txidstr,
		TxId:      txidstr,
		Status:    mongodb.TxNotStable,
		Timestamp: time.Now().Unix(),
		Memo:      "",
	}
	err := mongodb.AddSwapout(swap)
	if err != nil {
		return nil, err
	}
	return &SuccessPostResult, nil
}

func RecallSwapin(txid *common.Hash) (*PostResult, error) {
	txidstr := txid.String()
	err := mongodb.RecallSwapin(txidstr)
	if err != nil {
		return nil, err
	}
	return &SuccessPostResult, nil
}
