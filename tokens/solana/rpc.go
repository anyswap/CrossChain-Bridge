package solana

import (
	"context"
	"fmt"
	"math/big"

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

func (b *Bridge) getWSURLs() (wsURL []string) {
	return b.GatewayConfig.Extras.WSEndpoints
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

type AccountSubscription struct {
	solanaws.Subscription
}

func (s *AccountSubscription) Recv() (*solanaws.AccountResult, error) {
	res, err := s.Subscription.Recv()
	if res != nil, err == nil {
		acctres, ok := res.(*solanaws.AccountResult)
		if !ok {
			return nil, errors.New("Account subscription result type error")
		}
		return acctres, nil
	}
	return nil, err
}

type SlotSubscription struct {
	solanaws.Subscription
}

func (s *SlotSubscription) Recv() (*solanaws.SlotResult, error) {
	res, err := s.Subscription.Recv()
	if res != nil, err == nil {
		acctres, ok := res.(*solanaws.SlotResult)
		if !ok {
			return nil, errors.New("Account subscription result type error")
		}
		return acctres, nil
	}
	return nil, err
}

// SubscribeAccount subscribe account
func (b *Bridge) SubscribeAccount(account string) (*AccountSubscription, error) {
	rpcError := &RPCError{[]error{}, "SubscribeAccount"}
	acct, err := solana.PublicKeyFromBase58(account)
	if err != nil {
		rpcError.log(err)
		return nil, rpcError.Error()
	}
	ctx := context.Background()
	for _, endpoint := range getWSURLs() {
		cli, err := ws.Dial(ctx, endpoint)
		if err != nil {
			rpcError.log(err)
			continue
		}
		sbscrpt, err := cli.AccountSubscribe(acct, "finalized")
		if err != nil {
			rpcError.log(err)
			continue
		}
		return *AccountSubscription(sbscrpt), nil
	}
	return nil, rpcError.Error()
}

// SubscribeSlot subscribe slot
func (b *Bridge) SubscribeSlot(account string) (*SlotSubscription, error) {
	rpcError := &RPCError{[]error{}, "SubscribeAccount"}
	ctx := context.Background()
	for _, endpoint := range getWSURLs() {
		cli, err := ws.Dial(ctx, endpoint)
		if err != nil {
			rpcError.log(err)
			continue
		}
		sbscrpt, err := cli.SlotSubscribe()
		if err != nil {
			rpcError.log(err)
			continue
		}
		return *SlotSubscription(sbscrpt), nil
	}
	return nil, rpcError.Error()
}
