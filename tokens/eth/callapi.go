package eth

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/common/hexutil"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tools/rlp"
	"github.com/anyswap/CrossChain-Bridge/types"
)

var errEmptyURLs = errors.New("empty URLs")

// GetLatestBlockNumberOf call eth_blockNumber
func (b *Bridge) GetLatestBlockNumberOf(url string) (latest uint64, err error) {
	var result string
	err = client.RPCPost(&result, url, "eth_blockNumber")
	if err == nil {
		return common.GetUint64FromStr(result)
	}
	return 0, err
}

// GetLatestBlockNumber call eth_blockNumber
func (b *Bridge) GetLatestBlockNumber() (uint64, error) {
	gateway := b.GatewayConfig
	maxHeight, err := getMaxLatestBlockNumber(gateway.APIAddress)
	if maxHeight > 0 {
		tokens.CmpAndSetLatestBlockHeight(maxHeight, b.IsSrcEndpoint())
		return maxHeight, nil
	}
	return 0, err
}

func getMaxLatestBlockNumber(urls []string) (maxHeight uint64, err error) {
	if len(urls) == 0 {
		return 0, errEmptyURLs
	}
	var result string
	for _, url := range urls {
		err = client.RPCPost(&result, url, "eth_blockNumber")
		if err == nil {
			height, _ := common.GetUint64FromStr(result)
			if height > maxHeight {
				maxHeight = height
			}
		}
	}
	if maxHeight > 0 {
		return maxHeight, nil
	}
	return 0, err
}

// GetBlockByHash call eth_getBlockByHash
func (b *Bridge) GetBlockByHash(blockHash string) (*types.RPCBlock, error) {
	gateway := b.GatewayConfig
	return getBlockByHash(blockHash, gateway.APIAddress)
}

func getBlockByHash(blockHash string, urls []string) (result *types.RPCBlock, err error) {
	if len(urls) == 0 {
		return nil, errEmptyURLs
	}
	for _, url := range urls {
		err = client.RPCPost(&result, url, "eth_getBlockByHash", blockHash, false)
		if err == nil && result != nil {
			return result, nil
		}
	}
	if result == nil {
		return nil, errors.New("block not found")
	}
	return nil, err
}

// GetBlockByNumber call eth_getBlockByNumber
func (b *Bridge) GetBlockByNumber(number *big.Int) (*types.RPCBlock, error) {
	gateway := b.GatewayConfig
	var result *types.RPCBlock
	var err error
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress
		err = client.RPCPost(&result, url, "eth_getBlockByNumber", types.ToBlockNumArg(number), false)
		if err == nil && result != nil {
			return result, nil
		}
	}
	if result == nil {
		return nil, errors.New("block not found")
	}
	return nil, err
}

// GetTransactionByHash call eth_getTransactionByHash
func (b *Bridge) GetTransactionByHash(txHash string) (*types.RPCTransaction, error) {
	gateway := b.GatewayConfig
	return getTransactionByHash(txHash, gateway.APIAddress)
}

func getTransactionByHash(txHash string, urls []string) (result *types.RPCTransaction, err error) {
	if len(urls) == 0 {
		return nil, errEmptyURLs
	}
	for _, url := range urls {
		err = client.RPCPost(&result, url, "eth_getTransactionByHash", txHash)
		if err == nil && result != nil {
			return result, nil
		}
	}
	if result == nil {
		return nil, errors.New("tx not found")
	}
	return nil, err
}

// GetPendingTransactions call eth_pendingTransactions
func (b *Bridge) GetPendingTransactions() (result []*types.RPCTransaction, err error) {
	gateway := b.GatewayConfig
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress
		err = client.RPCPost(&result, url, "eth_pendingTransactions")
		if err == nil {
			return result, nil
		}
	}
	return nil, err
}

// GetTxBlockInfo impl
func (b Bridge) GetTxBlockInfo(txHash string) (blockHeight, blockTime uint64) {
	var useExt bool
	gateway := b.GatewayConfig
	receipt, _, _ := getTransactionReceipt(txHash, gateway.APIAddress)
	if receipt == nil && len(gateway.APIAddressExt) > 0 {
		useExt = true
		receipt, _, _ = getTransactionReceipt(txHash, gateway.APIAddressExt)
	}
	if receipt == nil {
		return 0, 0
	}
	blockHeight = receipt.BlockNumber.ToInt().Uint64()
	if !useExt {
		block, err := b.GetBlockByHash(receipt.BlockHash.Hex())
		if err == nil {
			blockTime = block.Time.ToInt().Uint64()
		}
	}
	return blockHeight, blockTime
}

// GetTransactionReceipt call eth_getTransactionReceipt
func (b *Bridge) GetTransactionReceipt(txHash string) (receipt *types.RPCTxReceipt, url string, err error) {
	gateway := b.GatewayConfig
	receipt, url, err = getTransactionReceipt(txHash, gateway.APIAddress)
	if err != nil && len(gateway.APIAddressExt) > 0 {
		return getTransactionReceipt(txHash, gateway.APIAddressExt)
	}
	return receipt, url, err
}

func getTransactionReceipt(txHash string, urls []string) (result *types.RPCTxReceipt, rpcURL string, err error) {
	if len(urls) == 0 {
		return nil, "", errEmptyURLs
	}
	for _, url := range urls {
		err = client.RPCPost(&result, url, "eth_getTransactionReceipt", txHash)
		if err == nil && result != nil {
			return result, url, nil
		}
	}
	if result == nil {
		return nil, "", errors.New("tx receipt not found")
	}
	return nil, "", err
}

// GetContractLogs get contract logs
func (b *Bridge) GetContractLogs(contractAddresses []common.Address, logTopics [][]common.Hash, blockHeight uint64) ([]*types.RPCLog, error) {
	height := new(big.Int).SetUint64(blockHeight)

	filter := &types.FilterQuery{
		FromBlock: height,
		ToBlock:   height,
		Addresses: contractAddresses,
		Topics:    logTopics,
	}
	return b.GetLogs(filter)
}

// GetLogs call eth_getLogs
func (b *Bridge) GetLogs(filterQuery *types.FilterQuery) (result []*types.RPCLog, err error) {
	args, err := types.ToFilterArg(filterQuery)
	if err != nil {
		return nil, err
	}
	gateway := b.GatewayConfig
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress
		err = client.RPCPost(&result, url, "eth_getLogs", args)
		if err == nil {
			return result, nil
		}
	}
	return nil, err
}

// GetPoolNonce call eth_getTransactionCount
func (b *Bridge) GetPoolNonce(address, height string) (uint64, error) {
	account := common.HexToAddress(address)
	gateway := b.GatewayConfig
	return getMaxPoolNonce(account, height, gateway.APIAddress)
}

func getMaxPoolNonce(account common.Address, height string, urls []string) (maxNonce uint64, err error) {
	if len(urls) == 0 {
		return 0, errEmptyURLs
	}
	var success bool
	var result hexutil.Uint64
	for _, url := range urls {
		err = client.RPCPost(&result, url, "eth_getTransactionCount", account, height)
		if err == nil {
			success = true
			if uint64(result) > maxNonce {
				maxNonce = uint64(result)
			}
		}
	}
	if success {
		return maxNonce, nil
	}
	return 0, err
}

// SuggestPrice call eth_gasPrice
func (b *Bridge) SuggestPrice() (maxGasPrice *big.Int, err error) {
	gateway := b.GatewayConfig
	if len(gateway.APIAddressExt) > 0 {
		maxGasPrice, err = getMaxGasPrice(gateway.APIAddressExt)
	}
	maxGasPrice2, err2 := getMaxGasPrice(gateway.APIAddress)
	if err2 == nil {
		if maxGasPrice == nil || maxGasPrice2.Cmp(maxGasPrice) > 0 {
			maxGasPrice = maxGasPrice2
		}
	} else {
		err = err2
	}
	if maxGasPrice != nil {
		return maxGasPrice, nil
	}
	return nil, err
}

func getMaxGasPrice(urls []string) (maxGasPrice *big.Int, err error) {
	if len(urls) == 0 {
		return nil, errEmptyURLs
	}
	var success bool
	var result hexutil.Big
	for _, url := range urls {
		err = client.RPCPost(&result, url, "eth_gasPrice")
		if err == nil {
			success = true
			if maxGasPrice == nil || result.ToInt().Cmp(maxGasPrice) > 0 {
				maxGasPrice = result.ToInt()
			}
		}
	}
	if success {
		return maxGasPrice, nil
	}
	return nil, err
}

// SendSignedTransaction call eth_sendRawTransaction
func (b *Bridge) SendSignedTransaction(tx *types.Transaction) (txHash string, err error) {
	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return "", err
	}
	hexData := common.ToHex(data)
	gateway := b.GatewayConfig
	txHash, _ = sendRawTransaction(hexData, gateway.APIAddressExt)
	txHash2, err := sendRawTransaction(hexData, gateway.APIAddress)
	if txHash != "" {
		return txHash, nil
	}
	if txHash2 != "" {
		return txHash2, nil
	}
	return "", err
}

func sendRawTransaction(hexData string, urls []string) (txHash string, err error) {
	if len(urls) == 0 {
		return "", errEmptyURLs
	}
	var result string
	for _, url := range urls {
		err = client.RPCPost(&result, url, "eth_sendRawTransaction", hexData)
		if err != nil {
			log.Trace("call eth_sendRawTransaction failed", "txHash", result, "url", url, "err", err)
			continue
		}
		log.Trace("call eth_sendRawTransaction success", "txHash", result, "url", url)
		if txHash == "" {
			txHash = result
		}
	}
	if txHash != "" {
		return txHash, nil
	}
	if err != nil {
		return "", err
	}
	return "", errors.New("call eth_sendRawTransaction failed")
}

// ChainID call eth_chainId
// Notice: eth_chainId return 0x0 for mainnet which is wrong (use net_version instead)
func (b *Bridge) ChainID() (*big.Int, error) {
	gateway := b.GatewayConfig
	var result hexutil.Big
	var err error
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress
		err = client.RPCPost(&result, url, "eth_chainId")
		if err == nil {
			return result.ToInt(), nil
		}
	}
	return nil, err
}

// NetworkID call net_version
func (b *Bridge) NetworkID() (*big.Int, error) {
	gateway := b.GatewayConfig
	var result string
	var err error
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress
		err = client.RPCPost(&result, url, "net_version")
		if err == nil {
			version := new(big.Int)
			if _, ok := version.SetString(result, 10); !ok {
				return nil, fmt.Errorf("invalid net_version result %q", result)
			}
			return version, nil
		}
	}
	return nil, err
}

// GetSignerChainID default way to get signer chain id
// use chain ID first, if missing then use network ID instead.
// normally this way works, but sometimes it failed (eg. ETC),
// then we should overwrite this function
func (b *Bridge) GetSignerChainID() (*big.Int, error) {
	chainID, err := b.ChainID()
	if err != nil {
		return nil, err
	}
	if chainID.Sign() != 0 {
		return chainID, nil
	}
	return b.NetworkID()
}

// GetCode call eth_getCode
func (b *Bridge) GetCode(contract string) (code []byte, err error) {
	gateway := b.GatewayConfig
	code, err = getCode(contract, gateway.APIAddress)
	if err != nil && len(gateway.APIAddressExt) > 0 {
		return getCode(contract, gateway.APIAddressExt)
	}
	return code, err
}

func getCode(contract string, urls []string) ([]byte, error) {
	if len(urls) == 0 {
		return nil, errEmptyURLs
	}
	var result hexutil.Bytes
	var err error
	for _, url := range urls {
		err = client.RPCPost(&result, url, "eth_getCode", contract, "latest")
		if err == nil {
			return []byte(result), nil
		}
	}
	return nil, err
}

// CallContract call eth_call
func (b *Bridge) CallContract(contract string, data hexutil.Bytes, blockNumber string) (string, error) {
	reqArgs := map[string]interface{}{
		"to":   contract,
		"data": data,
	}
	gateway := b.GatewayConfig
	var result string
	var err error
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress
		err = client.RPCPost(&result, url, "eth_call", reqArgs, blockNumber)
		if err == nil {
			return result, nil
		}
	}
	return "", err
}

// GetBalance call eth_getBalance
func (b *Bridge) GetBalance(account string) (*big.Int, error) {
	gateway := b.GatewayConfig
	var result hexutil.Big
	var err error
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress
		err = client.RPCPost(&result, url, "eth_getBalance", account, "latest")
		if err == nil {
			return result.ToInt(), nil
		}
	}
	return nil, err
}
