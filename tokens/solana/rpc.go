package solana

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	bin "github.com/dfuse-io/binary"
	"github.com/dfuse-io/solana-go"
	solanarpc "github.com/dfuse-io/solana-go/rpc"
	"github.com/ybbus/jsonrpc"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

func (b *Bridge) getClients() (clis []*solanarpc.Client) {
	endpoints := b.GatewayConfig.APIAddress
	clis = make([]*solanarpc.Client, 0)
	for _, endpoint := range endpoints {
		cli := solanarpc.NewClient(endpoint)
		if cli != nil {
			clis = append(clis, cli)
		}
	}
	return
}

func (b *Bridge) getURLs() (rpcURL []string) {
	return b.GatewayConfig.APIAddress
}

type RPCError struct {
	errs   []error
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
	rpcError := &RPCError{[]error{}, "GetLatestBlockNumber"}
	for _, cli := range b.getClients() {
		res, err := cli.GetSlot(ctx, "")
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
	ctx := context.Background()
	cli := solanarpc.NewClient(apiAddress)
	rpcError := &RPCError{[]error{}, "GetLatestBlockNumberOf"}
	res, err := cli.GetSlot(ctx, "")
	if err != nil {
		rpcError.log(err)
		return 0, rpcError.Error()
	}
	return uint64(res), nil
}

// GetBalance gets SOL token balance
func (b *Bridge) GetBalance(account string) (balance *big.Int, err error) {
	ctx := context.Background()
	rpcError := &RPCError{[]error{}, "GetBalance"}
	for _, cli := range b.getClients() {
		res, err := cli.GetBalance(ctx, account, "finalized")
		if err == nil {
			return new(big.Int).SetUint64(uint64(res.Value)), nil
		} else {
			rpcError.log(err)
		}
	}
	return big.NewInt(0), rpcError.Error()
}

// GetRecentBlockhash gets recent block hash
func (b *Bridge) GetRecentBlockhash() (string, error) {
	ctx := context.Background()
	rpcError := &RPCError{[]error{}, "GetBalance"}
	for _, cli := range b.getClients() {
		res, err := cli.GetRecentBlockhash(ctx, "finalized")
		if err == nil {
			return res.Value.Blockhash.String(), nil
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
	Transaction *solana.Transaction        `json:"transaction"`
	Meta        *solanarpc.TransactionMeta `json:"meta,omitempty"`
	Slot        bin.Uint64                 `json:"slot,omitempty"`
	BlockTime   bin.Uint64                 `json:"blockTime,omitempty"`
}

// GetTransaction gets tx by hash, returns sdk.Tx
func (b *Bridge) GetTransaction(txHash string) (tx interface{}, err error) {
	rpcError := &RPCError{[]error{}, "GetConfirmedTransaction"}
	params := []interface{}{txHash, "json"}
	for _, rpcURL := range b.getURLs() {
		rpcClient := jsonrpc.NewClient(rpcURL)
		tx := &GetConfirmedTransactonResult{}
		err := rpcClient.CallFor(tx, "getConfirmedTransaction", params...)
		if err == nil {
			return tx, nil
		} else {
			rpcError.log(err)
		}
	}
	return nil, rpcError.Error()
}

// GetTransactionStatus returns tx status
func (b *Bridge) GetTransactionStatus(txHash string) (status *tokens.TxStatus) {
	status = new(tokens.TxStatus)
	params := []interface{}{txHash, "json"}
	rpcError := &RPCError{[]error{}, "GetConfirmedTransaction"}
	for _, rpcURL := range b.getURLs() {
		rpcClient := jsonrpc.NewClient(rpcURL)
		tx := &GetConfirmedTransactonResult{}
		err := rpcClient.CallFor(tx, "getConfirmedTransaction", params...)
		if err == nil && tx.Meta.Err == nil {
			status.Receipt = tx
			status.Confirmations = 1
			status.PrioriFinalized = true
			status.BlockHeight = uint64(tx.Slot)
			status.BlockHash = ""
			status.BlockTime = uint64(tx.BlockTime)
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
	rpcError := &RPCError{[]error{}, "GetBalance"}
	for _, cli := range b.getClients() {
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
func (b *Bridge) GetBlockByNumber(num *big.Int) (block *solanarpc.GetConfirmedBlockResult, err error) {
	ctx := context.Background()
	rpcError := &RPCError{[]error{}, "GetBalance"}
	for _, cli := range b.getClients() {
		block, err := cli.GetConfirmedBlock(ctx, num.Uint64(), "")
		if err == nil {
			return block, nil
		} else {
			rpcError.log(err)
		}
	}
	return nil, rpcError.Error()
}

func (b *Bridge) searchTxs(address string, before, until string, limit uint64) (txs []string, err error) {
	rpcError := &RPCError{[]error{}, "SearchTxs"}
	acct, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		rpcError.log(err)
		return nil, rpcError.Error()
	}

	opts := &solanarpc.GetConfirmedSignaturesForAddress2Opts{
		Limit: limit,
	}
	if until != "" {
		opts.Until = until
	}
	if before != "" {
		opts.Before = before
	}

	ctx := context.Background()
	for _, cli := range b.getClients() {
		res, err := cli.GetConfirmedSignaturesForAddress2(ctx, acct, opts)
		if err != nil {
			rpcError.log(err)
			continue
		}
		txs = make([]string, 0)
		for _, tx := range res {
			txs = append(txs, tx.Signature)
		}
		return txs, nil
	}
	return nil, rpcError.Error()
}

// SearchTxs searches txs for address
func (b *Bridge) SearchTxs(address string, start, end string) (txs []string, err error) {
	before := end
	util := start
	limit := uint64(10)
	txs = make([]string, 0)
	for {
		txs1, err := b.searchTxs(address, before, util, limit)
		if err != nil {
			return nil, err
		}
		txs = append(txs, txs1...)
		if len(txs1) == 0 || strings.EqualFold(txs1[len(txs1)-1], util) {
			break
		}
		before = txs[len(txs)-1]
	}
	if end != "" {
		txs = append([]string{end}, txs...)
	}
	if start != "" {
		txs = append(txs, start)
	}
	return txs, nil
}
