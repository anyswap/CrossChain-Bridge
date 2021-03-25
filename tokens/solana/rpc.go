package solana

import (
	"context"
	"fmt"
	"math/big"

	"github.com/dfuse-io/solana-go"
	solanarpc "github.com/dfuse-io/solana-go/rpc"
	"github.com/ybbus/jsonrpc"
)

func (b *Bridge) getClients() (clis []*solanarpc.Client) {
	endpoints := b.GatewayConfig.APIAddress
	clis := make([]*solanarpc.Client)
	for _, endpoint := range endpoints {
		cli := solanarpc.NewClient(endpoint)
		if cli != nil {
			clis = append(clis, cli)
		}
	}
	return
}

// Transversal endpoints, request with lower level methods so as to bypass solana-go/rpc,
// because solana-go/rpc not well
func (b *Bridge) getURLs() (rpcURL []string) {
	return b.GatewayConfig.APIAddress
}

type RPCError struct {
	errs []error
	method string
}

func (e *RPCError) log(msg error) {
	log.Warn("[Solana RPC error]", "method", e.method, "msg", msg)
	if len(e.errs) < 1 {
		e.errs = make([]error, 1)
	}
	e.errs = append(e.errs, msg)
}

func (e *RPCError) Error() error {
	return fmt.Errorf("[Solana RPC error] method: %v errors:%+v", e.method, e.errs)
}

// GetLatestBlockNumber returns current finalized block height
func (b *Bridge) GetLatestBlockNumber() (height uint64, err error) {
	ctx := context.Background()
	rpcError := &RPCError{[]error, "GetLatestBlockNumber"}
	for _, cli := range getClients {
		res, err := cli.GetSlot(ctx, "finalized")
		if err == nil {
			return uint64(res), nil
		} else {
			rpcError.log(err)
		}
	}
	return 0, rpcError.Error()
}

// GetLatestBlockNumberOf returns current finalized block height from given node
func (b *Bridge) GetLatestBlockNumberOf(apiAddress string) (uint64, error) {
	cli := solanarpc.NewClient(apiAddress)
	res, err := cli.GetSlot(ctx, "finalized")
	if err != nil {
		return 0, RPCError{[]error{}, "GetLatestBlockNumberOf"}.log(err).Error()
	}
	return uint64(res), nil
}

// GetBalance gets SOL token balance
func (b *Bridge) GetBalance(account string) (balance *big.Int, err error) {
	ctx := context.Background()
	rpcError := &RPCError{[]error, "GetBalance"}
	for _, cli := range getClients {
		res, err := cli.GetBalance(ctx, account, "finalized")
		if err == nil {
			return big.NewInt(uint64(res.Value)), nil
		} else {
			rpcError.log(err)
		}
	}
	return big.NewInt(0), rpcError.Error()
}

// GetRecentBlockhash gets recent block hash
func (b *Bridge) GetRecentBlockhash() (string, error) {
	ctx := context.Background()
	rpcError := &RPCError{[]error, "GetBalance"}
	for _, cli := range getClients {
		res, err := cli.GetRecentBlockhash(ctx, "finalized")
		if err == nil {
			return resRbt.Value.Blockhash.String(), nil
		} else {
			rpcError.log(err)
		}
	}
	return "", rpcError.Error()
}

// GetTokenBalance gets balance for given token
func (b *Bridge) GetTokenBalance(tokenType, tokenName, accountAddress string) (balance *big.Int, err error) {
	return nil, fmt.Errorf("Solana bridges does not support this method")
}

// GetTokenSupply not supported
func (b *Bridge) GetTokenSupply(tokenType, tokenAddress string) (*big.Int, error) {
	return nil, fmt.Errorf("Solana bridges does not support this method")
}

type GetConfirmedTransactonResult struct {
		Transaction *solana.Transaction `json:"transaction"`
		Meta        *TransactionMeta    `json:"meta,omitempty"`
		Slot  bin.Uint64 `json:"slot,omitempty"`
		BlockTime  bin.Uint64 `json:"blockTime,omitempty"`
	}

// GetTransaction gets tx by hash, returns sdk.Tx
func (b *Bridge) GetTransaction(txHash string) (tx interface{}, err error) {
	rpcError := &RPCError{[]error, "GetConfirmedTransaction"}
	for _, rpcURL := range getURLs() {
		rpcClient := jsonrpc.NewClient(rpcURL)
		tx := &GetConfirmedTransactonResult{}
		err := rpcClient.CallFor(tx.(*GetConfirmedTransactonResult), "getConfirmedTransaction", params...)
		if err == nil {
			return
		} else {
			rpcError.log(err)
		}
	}
	return nil, rpcError.Error()
}

// GetTransactionStatus returns tx status
func (b *Bridge) GetTransactionStatus(txHash string) (status *tokens.TxStatus) {
	status = new(token.TxStatus)
	rpcError := &RPCError{[]error, "GetConfirmedTransaction"}
	for _, rpcURL := range getURLs() {
		rpcClient := jsonrpc.NewClient(rpcURL)
		res, err := cli.GetConfirmedTransaction(ctx, txHash)
		if err == nil && res.Meta.Err == nil {
			status.Receipt = res
			status.Confirmations = 1
			status.PrioriFinalized = true
			status.BlockHeight = uint64(res.Slot)
			status.BlockHash = ""
			status.BlockTime = uint64(res.BlockTime)
			return
		} else {
			status.Confirmations = 0
			rpcError.log(err)
			return
		}
	}
	return
}

// BroadcastTx broadcast tx
func (b *Bridge) BroadcastTx(tx *solana.Transaction) (hash string, err error) {
	ctx := context.Background()
	rpcError := &RPCError{[]error, "GetBalance"}
	for _, cli := range getClients {
		hash, err := cli.SendTransaction(ctx, tx)
		if err == nil {
			return hash, nil
		} else {
			rpcError.log(err)
		}
	}
	return "", rpcError.Error()
}

// GetBlockByNumber get block by number
func (b *Bridge) GetBlockByNumber(num *big.Int) (block *solanarpc.GetConfirmedBlockResult{}, err error) {
	ctx := context.Background()
	rpcError := &RPCError{[]error, "GetBalance"}
	for _, cli := range getClients {
		block, err := cli.GetConfirmedBlockResult(ctx, num, "")
		if err == nil {
			return block, nil
		} else {
			rpcError.log(err)
		}
	}
	return nil, rpcError.Error()
}
