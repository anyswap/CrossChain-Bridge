package rpcapi

import (
	"net/http"

	"github.com/anyswap/CrossChain-Bridge/internal/swapapi"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// RPCAPI rpc api handler
type RPCAPI struct{}

// RPCNullArgs null args
type RPCNullArgs struct{}

// GetVersionInfo api
func (s *RPCAPI) GetVersionInfo(r *http.Request, args *RPCNullArgs, result *string) error {
	version := params.VersionWithMeta
	*result = version
	return nil
}

// GetServerInfo api
func (s *RPCAPI) GetServerInfo(r *http.Request, args *RPCNullArgs, result *swapapi.ServerInfo) error {
	res, err := swapapi.GetServerInfo()
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// GetSwapStatistics api
func (s *RPCAPI) GetSwapStatistics(r *http.Request, args *RPCNullArgs, result *swapapi.SwapStatistics) error {
	res, err := swapapi.GetSwapStatistics()
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// GetRawSwapin api
func (s *RPCAPI) GetRawSwapin(r *http.Request, txid *string, result *swapapi.Swap) error {
	res, err := swapapi.GetRawSwapin(txid)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// GetRawSwapinResult api
func (s *RPCAPI) GetRawSwapinResult(r *http.Request, txid *string, result *swapapi.SwapResult) error {
	res, err := swapapi.GetRawSwapinResult(txid)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// GetSwapin api
func (s *RPCAPI) GetSwapin(r *http.Request, txid *string, result *swapapi.SwapInfo) error {
	res, err := swapapi.GetSwapin(txid)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// GetRawSwapout api
func (s *RPCAPI) GetRawSwapout(r *http.Request, txid *string, result *swapapi.Swap) error {
	res, err := swapapi.GetRawSwapout(txid)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// GetRawSwapoutResult api
func (s *RPCAPI) GetRawSwapoutResult(r *http.Request, txid *string, result *swapapi.SwapResult) error {
	res, err := swapapi.GetRawSwapoutResult(txid)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// GetSwapout api
func (s *RPCAPI) GetSwapout(r *http.Request, txid *string, result *swapapi.SwapInfo) error {
	res, err := swapapi.GetSwapout(txid)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// RPCQueryHistoryArgs args
type RPCQueryHistoryArgs struct {
	Address string `json:"address"`
	Offset  int    `json:"offset"`
	Limit   int    `json:"limit"`
}

// GetSwapinHistory api
func (s *RPCAPI) GetSwapinHistory(r *http.Request, args *RPCQueryHistoryArgs, result *[]*swapapi.SwapInfo) error {
	res, err := swapapi.GetSwapinHistory(args.Address, args.Offset, args.Limit)
	if err == nil && res != nil {
		*result = res
	}
	return err
}

// GetSwapoutHistory api
func (s *RPCAPI) GetSwapoutHistory(r *http.Request, args *RPCQueryHistoryArgs, result *[]*swapapi.SwapInfo) error {
	res, err := swapapi.GetSwapoutHistory(args.Address, args.Offset, args.Limit)
	if err == nil && res != nil {
		*result = res
	}
	return err
}

// Swapin api
func (s *RPCAPI) Swapin(r *http.Request, txid *string, result *swapapi.PostResult) error {
	res, err := swapapi.Swapin(txid)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// RetrySwapin api
func (s *RPCAPI) RetrySwapin(r *http.Request, txid *string, result *swapapi.PostResult) error {
	res, err := swapapi.RetrySwapin(txid)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// RPCP2shSwapinArgs args
type RPCP2shSwapinArgs struct {
	TxID string `json:"txid"`
	Bind string `json:"bind"`
}

// P2shSwapin api
func (s *RPCAPI) P2shSwapin(r *http.Request, args *RPCP2shSwapinArgs, result *swapapi.PostResult) error {
	res, err := swapapi.P2shSwapin(&args.TxID, &args.Bind)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// Swapout api
func (s *RPCAPI) Swapout(r *http.Request, txid *string, result *swapapi.PostResult) error {
	res, err := swapapi.Swapout(txid)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// IsValidSwapinBindAddress api
func (s *RPCAPI) IsValidSwapinBindAddress(r *http.Request, address *string, result *bool) error {
	*result = swapapi.IsValidSwapinBindAddress(address)
	return nil
}

// IsValidSwapoutBindAddress api
func (s *RPCAPI) IsValidSwapoutBindAddress(r *http.Request, address *string, result *bool) error {
	*result = swapapi.IsValidSwapoutBindAddress(address)
	return nil
}

// RegisterP2shAddress api
func (s *RPCAPI) RegisterP2shAddress(r *http.Request, bindAddress *string, result *tokens.P2shAddressInfo) error {
	res, err := swapapi.RegisterP2shAddress(*bindAddress)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// GetP2shAddressInfo api
func (s *RPCAPI) GetP2shAddressInfo(r *http.Request, p2shAddress *string, result *tokens.P2shAddressInfo) error {
	res, err := swapapi.GetP2shAddressInfo(*p2shAddress)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// GetLatestScanInfo api
func (s *RPCAPI) GetLatestScanInfo(r *http.Request, isSrc *bool, result *swapapi.LatestScanInfo) error {
	res, err := swapapi.GetLatestScanInfo(*isSrc)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// RegisterAddress api
func (s *RPCAPI) RegisterAddress(r *http.Request, address *string, result *swapapi.PostResult) error {
	res, err := swapapi.RegisterAddress(*address)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// GetRegisteredAddress api
func (s *RPCAPI) GetRegisteredAddress(r *http.Request, address *string, result *swapapi.RegisteredAddress) error {
	res, err := swapapi.GetRegisteredAddress(*address)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}
