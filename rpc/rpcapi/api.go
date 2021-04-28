package rpcapi

import (
	"errors"
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

// GetTokenPairInfo api
func (s *RPCAPI) GetTokenPairInfo(r *http.Request, pairID *string, result *tokens.TokenPairConfig) error {
	res, err := swapapi.GetTokenPairInfo(*pairID)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// GetSwapStatistics api
func (s *RPCAPI) GetSwapStatistics(r *http.Request, pairID *string, result *swapapi.SwapStatistics) error {
	res, err := swapapi.GetSwapStatistics(*pairID)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// RPCTxAndPairIDArgs txid and pairID
type RPCTxAndPairIDArgs struct {
	TxID   string `json:"txid"`
	PairID string `json:"pairid"`
	Bind   string `json:"bind"`
}

func (args *RPCTxAndPairIDArgs) getTxAndPairID() (txid, pairID, bind *string, err error) {
	txid = &args.TxID
	pairID = &args.PairID
	bind = &args.Bind
	if *txid == "" {
		return nil, nil, nil, errors.New("empty tx id")
	}
	if *pairID == "" {
		return nil, nil, nil, errors.New("empty pair id")
	}
	return txid, pairID, bind, nil
}

// GetRawSwapin api
func (s *RPCAPI) GetRawSwapin(r *http.Request, args *RPCTxAndPairIDArgs, result *swapapi.Swap) error {
	txid, pairID, bind, err := args.getTxAndPairID()
	if err != nil {
		return err
	}
	res, err := swapapi.GetRawSwapin(txid, pairID, bind)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// GetRawSwapinResult api
func (s *RPCAPI) GetRawSwapinResult(r *http.Request, args *RPCTxAndPairIDArgs, result *swapapi.SwapResult) error {
	txid, pairID, bind, err := args.getTxAndPairID()
	if err != nil {
		return err
	}
	res, err := swapapi.GetRawSwapinResult(txid, pairID, bind)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// GetSwapin api
func (s *RPCAPI) GetSwapin(r *http.Request, args *RPCTxAndPairIDArgs, result *swapapi.SwapInfo) error {
	txid, pairID, bind, err := args.getTxAndPairID()
	if err != nil {
		return err
	}
	res, err := swapapi.GetSwapin(txid, pairID, bind)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// GetRawSwapout api
func (s *RPCAPI) GetRawSwapout(r *http.Request, args *RPCTxAndPairIDArgs, result *swapapi.Swap) error {
	txid, pairID, bind, err := args.getTxAndPairID()
	if err != nil {
		return err
	}
	res, err := swapapi.GetRawSwapout(txid, pairID, bind)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// GetRawSwapoutResult api
func (s *RPCAPI) GetRawSwapoutResult(r *http.Request, args *RPCTxAndPairIDArgs, result *swapapi.SwapResult) error {
	txid, pairID, bind, err := args.getTxAndPairID()
	if err != nil {
		return err
	}
	res, err := swapapi.GetRawSwapoutResult(txid, pairID, bind)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// GetSwapout api
func (s *RPCAPI) GetSwapout(r *http.Request, args *RPCTxAndPairIDArgs, result *swapapi.SwapInfo) error {
	txid, pairID, bind, err := args.getTxAndPairID()
	if err != nil {
		return err
	}
	res, err := swapapi.GetSwapout(txid, pairID, bind)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// RPCQueryHistoryArgs args
type RPCQueryHistoryArgs struct {
	Address string `json:"address"`
	PairID  string `json:"pairid"`
	Offset  int    `json:"offset"`
	Limit   int    `json:"limit"`
}

// GetSwapinHistory api
func (s *RPCAPI) GetSwapinHistory(r *http.Request, args *RPCQueryHistoryArgs, result *[]*swapapi.SwapInfo) error {
	res, err := swapapi.GetSwapinHistory(args.Address, args.PairID, args.Offset, args.Limit)
	if err == nil && res != nil {
		*result = res
	}
	return err
}

// GetSwapoutHistory api
func (s *RPCAPI) GetSwapoutHistory(r *http.Request, args *RPCQueryHistoryArgs, result *[]*swapapi.SwapInfo) error {
	res, err := swapapi.GetSwapoutHistory(args.Address, args.PairID, args.Offset, args.Limit)
	if err == nil && res != nil {
		*result = res
	}
	return err
}

// Swapin api
func (s *RPCAPI) Swapin(r *http.Request, args *RPCTxAndPairIDArgs, result *swapapi.PostResult) error {
	txid, pairID, _, err := args.getTxAndPairID()
	if err != nil {
		return err
	}
	res, err := swapapi.Swapin(txid, pairID)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// RetrySwapin api
func (s *RPCAPI) RetrySwapin(r *http.Request, args *RPCTxAndPairIDArgs, result *swapapi.PostResult) error {
	txid, pairID, _, err := args.getTxAndPairID()
	if err != nil {
		return err
	}
	res, err := swapapi.RetrySwapin(txid, pairID)
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
func (s *RPCAPI) Swapout(r *http.Request, args *RPCTxAndPairIDArgs, result *swapapi.PostResult) error {
	txid, pairID, _, err := args.getTxAndPairID()
	if err != nil {
		return err
	}
	res, err := swapapi.Swapout(txid, pairID)
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

type SwapAgreementArgs map[string](interface{})

// AddSwapAgreement api
func (s *RPCAPI) AddSwapAgreement(r *http.Request, args *SwapAgreementArgs, result *swapapi.PostResult) error {
	res, err := swapapi.AddSwapAgreement(*args)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// CancelSwapAgreement api
func (s *RPCAPI) CancelSwapAgreement(r *http.Request, pkey *string, result *swapapi.PostResult) error {
	res, err := swapapi.CancelSwapAgreement(*pkey)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// UpdateSwapAgreement api
func (s *RPCAPI) UpdateSwapAgreement(r *http.Request, args *SwapAgreementArgs, result *swapapi.PostResult) error {
	res, err := swapapi.UpdateSwapAgreement(*args)
	if err == nil && res != nil {
		*result = *res
	}
	return err
}

// GetSwapAgreement api
func (s *RPCAPI) GetSwapAgreement(r *http.Request, pkey *string, result *swapapi.SwapAgreement) error {
	res, err := swapapi.GetSwapAgreement(*pkey)
	if err == nil && res != nil {
		*result = res
	}
	return err
}

// GetLatestScanInfo api
func (s *RPCAPI) GetLatestScannedSolanaTxid(r *http.Request, address *string, result *string) error {
	res := swapapi.GetLatestScannedSolanaTxid(*address)
	*result = res
	return nil
}
