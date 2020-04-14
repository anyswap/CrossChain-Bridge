package swapapi

import (
	"github.com/fsn-dev/crossChain-Bridge/common"
)

func GetServerInfo() (*ServerInfo, error) {
	info := &ServerInfo{}
	return info, nil
}

func GetSwapStatistics() (*SwapStatistics, error) {
	stat := &SwapStatistics{}
	return stat, nil
}

func GetSwapin(txid *common.Hash) (*SwapInfo, error) {
	swap := &SwapInfo{}
	return swap, nil
}

func GetSwapout(txid *common.Hash) (*SwapInfo, error) {
	swap := &SwapInfo{}
	return swap, nil
}

func GetSwapinHistory(address *common.Address, offset, limit int) ([]*SwapInfo, error) {
	return nil, nil
}

func GetSwapoutHistory(address *common.Address, offset, limit int) ([]*SwapInfo, error) {
	return nil, nil
}

func Swapin(txid *common.Hash) (*PostResult, error) {
	res := &PostResult{}
	return res, nil
}

func Swapout(txid *common.Hash) (*PostResult, error) {
	res := &PostResult{}
	return res, nil
}

func RecallSwapin(txid *common.Hash) (*PostResult, error) {
	res := &PostResult{}
	return res, nil
}
