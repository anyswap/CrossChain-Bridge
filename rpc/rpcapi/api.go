package rpcapi

import (
	"net/http"

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

func (s *RpcAPI) GetSwapin(r *http.Request, txid *string, result *swapapi.SwapInfo) error {
	res, err := swapapi.GetSwapin(txid)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

func (s *RpcAPI) GetSwapout(r *http.Request, txid *string, result *swapapi.SwapInfo) error {
	res, err := swapapi.GetSwapout(txid)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

type RpcQueryHistoryArgs struct {
	Address string
	Offset  int
	Limit   int
}

func (args *RpcQueryHistoryArgs) getQueryArgs() (address string, offset int, limit int, err error) {
	address = args.Address
	offset = args.Offset
	limit = args.Limit
	return address, offset, limit, nil
}

func (s *RpcAPI) GetSwapinHistory(r *http.Request, args *RpcQueryHistoryArgs, result *[]*swapapi.SwapInfo) error {
	address, offset, limit, err := args.getQueryArgs()
	if err != nil {
		return err
	}
	res, err := swapapi.GetSwapinHistory(address, offset, limit)
	if err == nil && res != nil {
		*result = res
	}
	return err
}

func (s *RpcAPI) GetSwapoutHistory(r *http.Request, args *RpcQueryHistoryArgs, result *[]*swapapi.SwapInfo) error {
	address, offset, limit, err := args.getQueryArgs()
	if err != nil {
		return err
	}
	res, err := swapapi.GetSwapoutHistory(address, offset, limit)
	if err == nil && res != nil {
		*result = res
	}
	return err
}

func (s *RpcAPI) Swapin(r *http.Request, txid *string, result *swapapi.PostResult) error {
	res, err := swapapi.Swapin(txid)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

func (s *RpcAPI) Swapout(r *http.Request, txid *string, result *swapapi.PostResult) error {
	res, err := swapapi.Swapout(txid)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

func (s *RpcAPI) RecallSwapin(r *http.Request, txid *string, result *swapapi.PostResult) error {
	res, err := swapapi.RecallSwapin(txid)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}
