package tron

import (
	"context"
	"errors"
	"fmt"
	"time"
	"math/big"
	"strings"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	tronaddress "github.com/fbsobreira/gotron-sdk/pkg/address"
	"google.golang.org/grpc"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var GRPC_TIMEOUT = time.Second * 15

func (b *Bridge) getClients() []*client.GrpcClient {
	endpoints := b.GatewayConfig.APIAddress
	clis := make([]*client.GrpcClient, 0)
	for _, endpoint := range endpoints {
		cli := client.NewGrpcClientWithTimeout(endpoint, GRPC_TIMEOUT)
		if cli != nil {
			clis = append(clis, cli)
		}
	}
	return clis
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
	rpcError := &RPCError{[]error{}, "GetLatestBlockNumber"}
	for _, cli := range b.getClients() {
		err = cli.Start(grpc.WithInsecure())
		if err != nil {
			rpcError.log(err)
			continue
		}
		res, err := cli.GetNowBlock()
		if err == nil {
			if res.BlockHeader.RawData.Number > 0 {
				height = uint64(res.BlockHeader.RawData.Number)
				cli.Stop()
				break
			}
		} else {
			rpcError.log(err)
		}
		cli.Stop()
	}
	if height > 0 {
		return height, nil
	}
	return 0, rpcError.Error()
}

// GetLatestBlockNumberOf returns current finalized block height from given node
func (b *Bridge) GetLatestBlockNumberOf(apiAddress string) (uint64, error) {
	rpcError := &RPCError{[]error{}, "GetLatestBlockNumberOf"}
	cli := client.NewGrpcClientWithTimeout(apiAddress, GRPC_TIMEOUT)
	if cli == nil {
		rpcError.log(errors.New("New client failed"))
		return 0, rpcError.Error()
	}
	err := cli.Start(grpc.WithInsecure())
	if err != nil {
		rpcError.log(err)
		return 0, rpcError.Error()
	}
	res, err := cli.GetNowBlock()
	if err != nil {
		rpcError.log(err)
		return 0, rpcError.Error()
	}
	return uint64(res.BlockHeader.RawData.Number), nil
}

// GetBalance gets SOL token balance
func (b *Bridge) GetBalance(account string) (balance *big.Int, err error) {
	rpcError := &RPCError{[]error{}, "GetBalance"}
	for _, cli := range b.getClients() {
		err = cli.Start(grpc.WithInsecure())
		if err != nil {
			rpcError.log(err)
			continue
		}
		res, err := cli.GetAccount(account)
		if err == nil {
			if res.Balance > 0 {
				balance = big.NewInt(int64(res.Balance))
				cli.Stop()
				break
			}
		} else {
			rpcError.log(err)
		}
		cli.Stop()
	}
	if balance.Cmp(big.NewInt(0)) > 0 {
		return balance, nil
	}
	return big.NewInt(0), rpcError.Error()
}

func (b *Bridge) GetTokenBalance(tokenType, tokenAddress, accountAddress string) (balance *big.Int, err error) {
	switch strings.ToUpper(tokenType) {
	case TRC20TokenType:
		return b.GetTrc20Balance(tokenAddress, accountAddress)
	case TRC10TokenType:
		return nil, fmt.Errorf("[%v] can not get token balance of token with type '%v'", b.ChainConfig.BlockChain, tokenType)
	default:
		return nil, fmt.Errorf("[%v] can not get token balance of token with type '%v'", b.ChainConfig.BlockChain, tokenType)
	}
}

// GetTrc20Balance gets balance for given ERC20 token
func (b *Bridge) GetTrc20Balance(tokenAddress, accountAddress string) (balance *big.Int, err error) {
	rpcError := &RPCError{[]error{}, "GetTrc20Balance"}
	for _, cli := range b.getClients() {
		err = cli.Start(grpc.WithInsecure())
		if err != nil {
			rpcError.log(err)
			continue
		}
		res, err := cli.TRC20ContractBalance(accountAddress, tokenAddress)
		if err == nil {
			balance = res
			cli.Stop()
			break
		} else {
			rpcError.log(err)
		}
		cli.Stop()
	}
	if balance.Cmp(big.NewInt(0)) > 0 {
		return balance, nil
	}
	return big.NewInt(0), rpcError.Error()
}

// GetTokenSupply impl
func (b *Bridge) GetTokenSupply(tokenType, tokenAddress string) (*big.Int, error) {
	switch strings.ToUpper(tokenType) {
	case TRC20TokenType:
		return b.GetErc20TotalSupply(tokenAddress)
	case TRC10TokenType:
		return nil, fmt.Errorf("[%v] can not get token supply of token with type '%v'", b.ChainConfig.BlockChain, tokenType)
	default:
		return nil, fmt.Errorf("[%v] can not get token supply of token with type '%v'", b.ChainConfig.BlockChain, tokenType)
	}
}

// GetTokenSupply not supported
func (b *Bridge) GetErc20TotalSupply(tokenAddress string) (totalSupply *big.Int, err error) {
	totalSupplyMethodSignature := "0x18160ddd"
	rpcError := &RPCError{[]error{}, "GetErc20TotalSupply"}
	for _, cli := range b.getClients() {
		err = cli.Start(grpc.WithInsecure())
		if err != nil {
			rpcError.log(err)
			continue
		}
		result, err := cli.TRC20Call("", tokenAddress, totalSupplyMethodSignature, true, 0)
		if err == nil {
			totalSupply = new(big.Int).SetBytes(result.GetConstantResult()[0])
			cli.Stop()
			break
		} else {
			rpcError.log(err)
		}
		cli.Stop()
	}
	if totalSupply.Cmp(big.NewInt(0)) > 0 {
		return totalSupply, nil
	}
	return big.NewInt(0), rpcError.Error()
}

// GetTransaction gets tx by hash, returns sdk.Tx
func (b *Bridge) GetTransaction(txHash string) (tx interface{}, err error) {
	rpcError := &RPCError{[]error{}, "GetTransaction"}
	for _, cli := range b.getClients() {
		err = cli.Start(grpc.WithInsecure())
		if err != nil {
			rpcError.log(err)
			continue
		}
		tx, err = cli.GetTransactionByID(txHash)
		if err == nil {
			cli.Stop()
			break
		}
		cli.Stop()
	}
	if err != nil {
		return nil, rpcError.Error()
	}
	return
}

// GetTransactionStatus returns tx status
func (b *Bridge) GetTransactionStatus(txHash string) (status *tokens.TxStatus) {
	status = &tokens.TxStatus{}
	var tx *core.TransactionInfo
	rpcError := &RPCError{[]error{}, "GetTransactionStatus"}
	for _, cli := range b.getClients() {
		err := cli.Start(grpc.WithInsecure())
		if err != nil {
			rpcError.log(err)
			continue
		}
		tx, err = cli.GetTransactionInfoByID(txHash)
		if err == nil {
			cli.Stop()
			break
		}
		cli.Stop()
	}
	status.Receipt = tx.Receipt
	status.PrioriFinalized = false
	status.BlockHeight = uint64(tx.BlockNumber)
	status.BlockTime = uint64(tx.BlockTimeStamp / 1000)

	if latest, err := b.GetLatestBlockNumber(); err == nil {
		status.Confirmations = latest - status.BlockHeight
	}
	return
}

// BuildTransfer returns an unsigned tron transfer tx
func (b *Bridge) BuildTransfer(from, to string, amount *big.Int, input []byte) (tx *core.Transaction, err error) {
	to, err = ethToTron(to)
	if err != nil {
		return nil, err
	}
	n, _ := new(big.Int).SetString("18446740000000000000", 0)
	if amount.Cmp(n) > 0 {
		return nil, errors.New("Amount exceed max uint64")
	}
	contract := &core.TransferContract{}
	contract.OwnerAddress, err = common.DecodeCheck(from)
	if err != nil {
		return nil, err
	}
	contract.ToAddress, err = common.DecodeCheck(to)
	if err != nil {
		return nil, err
	}
	contract.Amount = amount.Int64()
	rpcError := &RPCError{[]error{}, "BuildTransfer"}
	ctx, cancel := context.WithTimeout(context.Background(), GRPC_TIMEOUT)
	defer cancel()
	for _, cli := range b.getClients() {
		err = cli.Start(grpc.WithInsecure())
		if err != nil {
			rpcError.log(err)
			continue
		}
		txext, err1 := cli.Client.CreateTransaction2(ctx, contract)
		err = err1
		if err == nil {
			cli.Stop()
			cancel()
			tx = txext.Transaction
			if tx == nil {
				err = fmt.Errorf("%v", txext)
				rpcError.log(err)
			}
			break
		}
		rpcError.log(err)
		cli.Stop()
		cancel()
	}
	if err != nil {
		return nil, rpcError.Error()
	}
	return tx, nil
}

// BuildTRC20Transfer returns an unsigned trc20 transfer tx
func (b *Bridge) BuildTRC20Transfer(from, to, tokenAddress string, amount *big.Int) (tx *core.Transaction, err error) {
	to, err = ethToTron(to)
	if err != nil {
		return nil, err
	}
	n, _ := new(big.Int).SetString("18446740000000000000", 0)
	if amount.Cmp(n) > 0 {
		return nil, errors.New("Amount exceed max uint64")
	}
	contract := &core.TransferContract{}
	contract.OwnerAddress, err = common.DecodeCheck(from)
	if err != nil {
		return nil, err
	}
	contract.ToAddress, err = common.DecodeCheck(to)
	if err != nil {
		return nil, err
	}
	rpcError := &RPCError{[]error{}, "BuildTRC20Transfer"}
	for _, cli := range b.getClients() {
		err = cli.Start(grpc.WithInsecure())
		if err != nil {
			rpcError.log(err)
			continue
		}
		txext, err1 := cli.TRC20Send(from, to, tokenAddress, amount, 0)
		err = err1
		if err == nil {
			tx = txext.Transaction
			cli.Stop()
			break
		}
		rpcError.log(err)
		cli.Stop()
	}
	if err != nil {
		return nil, rpcError.Error()
	}
	return tx, nil
}

// BuildSwapinTx returns an unsigned mapping asset minting tx
func (b *Bridge) BuildSwapinTx(from, to, tokenAddress string, amount *big.Int, txhash string) (tx *core.Transaction, err error) {
	n, _ := new(big.Int).SetString("18446740000000000000", 0)
	if amount.Cmp(n) > 0 {
		return nil, errors.New("Amount exceed max uint64")
	}
	method := "Swapin"
	param := fmt.Sprintf(`[{"string":"%s"},{"address":"%s"},{"uint256":"%v"}]`, txhash, to, amount.Uint64())
	rpcError := &RPCError{[]error{}, "BuildSwapinTx"}
	for _, cli := range b.getClients() {
		err = cli.Start(grpc.WithInsecure())
		if err != nil {
			rpcError.log(err)
			continue
		}
		txext, err1 := cli.TriggerConstantContract(from, tokenAddress, method, param)
		err = err1
		if err == nil {
			tx = txext.Transaction
			cli.Stop()
			break
		}
		rpcError.log(err)
		cli.Stop()
	}
	if err != nil {
		return nil, rpcError.Error()
	}
	return tx, nil
}

// GetCode returns contract bytecode
func (b *Bridge) GetCode(contractAddress string) (data []byte, err error) {
	contractDesc, err := tronaddress.Base58ToAddress(contractAddress)
	if err != nil {
		return nil, err
	}
	message := new(api.BytesMessage)
	message.Value = contractDesc
	rpcError := &RPCError{[]error{}, "GetCode"}
	for _, cli := range b.getClients() {
		err = cli.Start(grpc.WithInsecure())
		ctx, cancel := context.WithTimeout(context.Background(), GRPC_TIMEOUT)
		if err != nil {
			rpcError.log(err)
			continue
		}
		sm, err1 := cli.Client.GetContract(ctx, message)
		err = err1
		if err == nil {
			data = sm.Bytecode
			cli.Stop()
			cancel()
			break
		}
		cli.Stop()
		cancel()
	}
	if err != nil {
		return nil, rpcError.Error()
	}
	return data, nil
}

// GetBlockByLimitNext gets block by limit next
func (b *Bridge) GetBlockByLimitNext(start, end int64) (res *api.BlockListExtention, err error) {
	rpcError := &RPCError{[]error{}, "GetBlockByLimitNext"}
	for _, cli := range b.getClients() {
		err = cli.Start(grpc.WithInsecure())
		if err != nil {
			rpcError.log(err)
			continue
		}
		res, err = cli.GetBlockByLimitNext(start, end)
		if err == nil {
			cli.Stop()
			break
		}
		rpcError.log(err)
	}
	if err != nil {
		return nil, rpcError.Error()
	}
	return res, nil
}

// BroadcastTx broadcast tx to network
func (b *Bridge) BroadcastTx(tx *core.Transaction) (err error) {
	rpcError := &RPCError{[]error{}, "BroadcastTx"}
	for _, cli := range b.getClients() {
		err = cli.Start(grpc.WithInsecure())
		if err != nil {
			rpcError.log(err)
			continue
		}
		res, err := cli.Broadcast(tx)
		if err == nil {
			cli.Stop()
			if res.Code != 0 {
				rpcError.log(fmt.Errorf("bad transaction: %v", string(res.GetMessage())))
			}
			return nil
		}
		rpcError.log(err)
	}
	return rpcError.Error()
}