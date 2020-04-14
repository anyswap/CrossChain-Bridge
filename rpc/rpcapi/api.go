package rpcapi

import (
	"net/http"

	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/internal/swapapi"
)

type RpcAPI struct{}

type RpcNullArgs struct{}

func (s *RpcAPI) GetServerInfo(r *http.Request, args *RpcNullArgs, result *swapapi.ServerInfo) error {
	res, err := swapapi.GetServerInfo()
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

func (s *RpcAPI) GetSwapStatistics(r *http.Request, args *RpcNullArgs, result *swapapi.SwapStatistics) error {
	res, err := swapapi.GetSwapStatistics()
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

func (s *RpcAPI) GetSwapin(r *http.Request, txid *common.Hash, result *swapapi.SwapInfo) error {
	res, err := swapapi.GetSwapin(txid)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

func (s *RpcAPI) GetSwapout(r *http.Request, txid *common.Hash, result *swapapi.SwapInfo) error {
	res, err := swapapi.GetSwapout(txid)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

type RpcQueryHistoryArgs struct {
	Address *common.Address
	Offset  int
	Limit   int
}

func (s *RpcAPI) GetSwapinHistory(r *http.Request, args *RpcQueryHistoryArgs, result *[]*swapapi.SwapInfo) error {
	res, err := swapapi.GetSwapinHistory(args.Address, args.Offset, args.Limit)
	if err == nil && res != nil {
		*result = res
	}
	return err
}

func (s *RpcAPI) GetSwapoutHistory(r *http.Request, args *RpcQueryHistoryArgs, result *[]*swapapi.SwapInfo) error {
	res, err := swapapi.GetSwapoutHistory(args.Address, args.Offset, args.Limit)
	if err == nil && res != nil {
		*result = res
	}
	return err
}

func (s *RpcAPI) Swapin(r *http.Request, txid *common.Hash, result *swapapi.PostResult) error {
	res, err := swapapi.Swapin(txid)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

func (s *RpcAPI) Swapout(r *http.Request, txid *common.Hash, result *swapapi.PostResult) error {
	res, err := swapapi.Swapout(txid)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

func (s *RpcAPI) RecallSwapin(r *http.Request, txid *common.Hash, result *swapapi.PostResult) error {
	res, err := swapapi.RecallSwapin(txid)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}
