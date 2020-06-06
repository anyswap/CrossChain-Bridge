package rpcapi

import (
	"net/http"

	"github.com/fsn-dev/crossChain-Bridge/internal/swapapi"
	"github.com/fsn-dev/crossChain-Bridge/params"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

type RpcAPI struct{}

type RpcNullArgs struct{}

func (s *RpcAPI) GetVersionInfo(r *http.Request, args *RpcNullArgs, result *string) error {
	ver := params.VersionWithMeta
	ver += "-rev5"
	*result = ver
	return nil
}

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

func (s *RpcAPI) GetRawSwapin(r *http.Request, txid *string, result *swapapi.Swap) error {
	res, err := swapapi.GetRawSwapin(txid)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

func (s *RpcAPI) GetRawSwapinResult(r *http.Request, txid *string, result *swapapi.SwapResult) error {
	res, err := swapapi.GetRawSwapinResult(txid)
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

func (s *RpcAPI) GetRawSwapout(r *http.Request, txid *string, result *swapapi.Swap) error {
	res, err := swapapi.GetRawSwapout(txid)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

func (s *RpcAPI) GetRawSwapoutResult(r *http.Request, txid *string, result *swapapi.SwapResult) error {
	res, err := swapapi.GetRawSwapoutResult(txid)
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
	Address string `json:"address"`
	Offset  int    `json:"offset"`
	Limit   int    `json:"limit"`
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

type RpcP2shSwapinArgs struct {
	TxID string `json:"txid"`
	Bind string `json:"bind"`
}

func (s *RpcAPI) P2shSwapin(r *http.Request, args *RpcP2shSwapinArgs, result *swapapi.PostResult) error {
	res, err := swapapi.P2shSwapin(&args.TxID, &args.Bind)
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

func (s *RpcAPI) IsValidSwapinBindAddress(r *http.Request, address *string, result *bool) error {
	*result = swapapi.IsValidSwapinBindAddress(address)
	return nil
}

func (s *RpcAPI) IsValidSwapoutBindAddress(r *http.Request, address *string, result *bool) error {
	*result = swapapi.IsValidSwapoutBindAddress(address)
	return nil
}

func (s *RpcAPI) RegisterP2shAddress(r *http.Request, bindAddress *string, result *tokens.P2shAddressInfo) error {
	res, err := swapapi.RegisterP2shAddress(*bindAddress)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

func (s *RpcAPI) GetP2shAddressInfo(r *http.Request, p2shAddress *string, result *tokens.P2shAddressInfo) error {
	res, err := swapapi.GetP2shAddressInfo(*p2shAddress)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}
