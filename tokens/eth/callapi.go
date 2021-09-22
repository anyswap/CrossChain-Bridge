package eth

import (
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/common/hexutil"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/types"
)

var (
	errNotFound               = errors.New("not found")
	errEmptyURLs              = errors.New("empty URLs")
	errTxInOrphanBlock        = errors.New("tx is in orphan block")
	errTxHashMismatch         = errors.New("tx hash mismatch with rpc result")
	errTxBlockHashMismatch    = errors.New("tx block hash mismatch with rpc result")
	errTxReceiptMissBlockInfo = errors.New("tx receipt missing block info")
)

func wrapRPCQueryError(err error, method string, params ...interface{}) error {
	if err == nil {
		err = errNotFound
	}
	return fmt.Errorf("%w: call '%s %v' failed, err='%v'", tokens.ErrRPCQueryError, method, params, err)
}

// RPCCall common RPC calling
func RPCCall(result interface{}, url, method string, params ...interface{}) error {
	if err := client.RPCPost(&result, url, method, params...); err != nil {
		return wrapRPCQueryError(err, method, params)
	}
	return nil
}

// GetLatestBlockNumberOf call eth_blockNumber
func (b *Bridge) GetLatestBlockNumberOf(url string) (latest uint64, err error) {
	var result string
	err = client.RPCPost(&result, url, "eth_blockNumber")
	if err == nil {
		return common.GetUint64FromStr(result)
	}
	return 0, wrapRPCQueryError(err, "eth_blockNumber")
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
	return 0, wrapRPCQueryError(err, "eth_blockNumber")
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
	return nil, wrapRPCQueryError(err, "eth_getBlockByHash", blockHash)
}

// GetBlockByNumber call eth_getBlockByNumber
func (b *Bridge) GetBlockByNumber(number *big.Int) (*types.RPCBlock, error) {
	gateway := b.GatewayConfig
	var result *types.RPCBlock
	var err error
	blockNumber := types.ToBlockNumArg(number)
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress
		err = client.RPCPost(&result, url, "eth_getBlockByNumber", blockNumber, false)
		if err == nil && result != nil {
			return result, nil
		}
	}
	return nil, wrapRPCQueryError(err, "eth_getBlockByNumber", number)
}

// GetBlockHash impl
func (b *Bridge) GetBlockHash(height uint64) (hash string, err error) {
	gateway := b.GatewayConfig
	return b.GetBlockHashOf(gateway.APIAddress, height)
}

// GetBlockHashOf impl
func (b *Bridge) GetBlockHashOf(urls []string, height uint64) (hash string, err error) {
	if len(urls) == 0 {
		return "", errEmptyURLs
	}
	blockNumber := types.ToBlockNumArg(new(big.Int).SetUint64(height))
	var block *types.RPCBaseBlock
	for _, url := range urls {
		err = client.RPCPost(&block, url, "eth_getBlockByNumber", blockNumber, false)
		if err == nil && block != nil {
			return block.Hash.Hex(), nil
		}
	}
	return "", wrapRPCQueryError(err, "eth_getBlockByNumber")
}

// GetTransaction impl
func (b *Bridge) GetTransaction(txHash string) (tx interface{}, err error) {
	gateway := b.GatewayConfig
	tx, err = b.getTransactionByHash(txHash, gateway.APIAddress)
	if err != nil && errors.Is(err, tokens.ErrRPCQueryError) && len(gateway.APIAddressExt) > 0 {
		tx, err = b.getTransactionByHash(txHash, gateway.APIAddressExt)
	}
	return tx, err
}

// GetTransactionByHash call eth_getTransactionByHash
func (b *Bridge) GetTransactionByHash(txHash string) (*types.RPCTransaction, error) {
	gateway := b.GatewayConfig
	return b.getTransactionByHash(txHash, gateway.APIAddress)
}

func (b *Bridge) getTransactionByHash(txHash string, urls []string) (result *types.RPCTransaction, err error) {
	if len(urls) == 0 {
		return nil, errEmptyURLs
	}
	for _, url := range urls {
		err = client.RPCPost(&result, url, "eth_getTransactionByHash", txHash)
		if err == nil && result != nil {
			if !common.IsEqualIgnoreCase(result.Hash.Hex(), txHash) {
				return nil, errTxHashMismatch
			}
			return result, nil
		}
	}
	return nil, wrapRPCQueryError(err, "eth_getTransactionByHash", txHash)
}

// GetTransactionByBlockNumberAndIndex get tx by block number and tx index
func (b *Bridge) GetTransactionByBlockNumberAndIndex(blockNumber *big.Int, txIndex uint) (result *types.RPCTransaction, err error) {
	gateway := b.GatewayConfig
	for _, url := range gateway.APIAddress {
		result, err = getTransactionByBlockNumberAndIndex(blockNumber, txIndex, url)
		if err == nil && result != nil {
			return result, nil
		}
	}
	return nil, wrapRPCQueryError(err, "eth_getTransactionByBlockNumberAndIndex", blockNumber, txIndex)
}

func getTransactionByBlockNumberAndIndex(blockNumber *big.Int, txIndex uint, url string) (result *types.RPCTransaction, err error) {
	err = client.RPCPost(&result, url, "eth_getTransactionByBlockNumberAndIndex", types.ToBlockNumArg(blockNumber), hexutil.Uint64(txIndex))
	if err == nil && result != nil {
		return result, nil
	}
	return nil, wrapRPCQueryError(err, "eth_getTransactionByBlockNumberAndIndex", blockNumber, txIndex)
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
	return nil, wrapRPCQueryError(err, "eth_pendingTransactions")
}

// GetTxBlockInfo impl
func (b *Bridge) GetTxBlockInfo(txHash string) (blockHeight, blockTime uint64) {
	var useExt bool
	gateway := b.GatewayConfig
	receipt, _, err := b.getTransactionReceipt(txHash, gateway.APIAddress)
	if (err != nil || receipt == nil) && len(gateway.APIAddressExt) > 0 {
		useExt = true
		receipt, _, err = b.getTransactionReceipt(txHash, gateway.APIAddressExt)
	}
	if err != nil || receipt == nil {
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
	receipt, url, err = b.getTransactionReceipt(txHash, gateway.APIAddress)
	if err != nil && errors.Is(err, tokens.ErrRPCQueryError) && len(gateway.APIAddressExt) > 0 {
		return b.getTransactionReceipt(txHash, gateway.APIAddressExt)
	}
	return receipt, url, err
}

func (b *Bridge) getTransactionReceipt(txHash string, urls []string) (result *types.RPCTxReceipt, rpcURL string, err error) {
	if len(urls) == 0 {
		return nil, "", errEmptyURLs
	}
	for _, url := range urls {
		err = client.RPCPost(&result, url, "eth_getTransactionReceipt", txHash)
		if err == nil && result != nil {
			if result.BlockNumber == nil || result.BlockHash == nil || result.TxIndex == nil {
				return nil, "", errTxReceiptMissBlockInfo
			}
			if !common.IsEqualIgnoreCase(result.TxHash.Hex(), txHash) {
				return nil, "", errTxHashMismatch
			}
			if b.ChainConfig.EnableCheckTxBlockIndex {
				tx, errt := getTransactionByBlockNumberAndIndex(result.BlockNumber.ToInt(), uint(*result.TxIndex), url)
				if errt != nil {
					return nil, "", errt
				}
				if !common.IsEqualIgnoreCase(tx.Hash.Hex(), txHash) {
					return nil, "", errTxInOrphanBlock
				}
			}
			if b.ChainConfig.EnableCheckTxBlockHash {
				if err = b.checkTxBlockHash(result.BlockNumber.ToInt(), *result.BlockHash); err != nil {
					return nil, "", err
				}
			}
			return result, url, nil
		}
	}
	return nil, "", wrapRPCQueryError(err, "eth_getTransactionReceipt", txHash)
}

func (b *Bridge) checkTxBlockHash(blockNumber *big.Int, blockHash common.Hash) error {
	block, err := b.GetBlockByNumber(blockNumber)
	if err != nil {
		log.Warn("get block by number failed", "number", blockNumber.String(), "err", err)
		return err
	}
	if *block.Hash != blockHash {
		log.Warn("tx block hash mismatch", "number", blockNumber.String(), "have", blockHash.String(), "want", block.Hash.String())
		return errTxBlockHashMismatch
	}
	return nil
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
	return nil, wrapRPCQueryError(err, "eth_getLogs")
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
	return 0, wrapRPCQueryError(err, "eth_getTransactionCount", account, height)
}

// SuggestPrice call eth_gasPrice
func (b *Bridge) SuggestPrice() (*big.Int, error) {
	gateway := b.GatewayConfig
	return getMedianGasPrice(gateway.APIAddress, gateway.APIAddressExt)
}

// get median gas price as the rpc result fluctuates too widely
func getMedianGasPrice(urlsSlice ...[]string) (*big.Int, error) {
	logFunc := log.GetPrintFuncOr(params.IsDebugMode, log.Info, log.Trace)

	allGasPrices := make([]*big.Int, 0, 10)
	urlCount := 0

	var result hexutil.Big
	var err error
	for _, urls := range urlsSlice {
		urlCount += len(urls)
		for _, url := range urls {
			if err = client.RPCPost(&result, url, "eth_gasPrice"); err != nil {
				logFunc("call eth_gasPrice failed", "url", url, "err", err)
				continue
			}
			gasPrice := result.ToInt()
			allGasPrices = append(allGasPrices, gasPrice)
		}
	}
	if len(allGasPrices) == 0 {
		log.Warn("getMedianGasPrice failed", "err", err)
		return nil, wrapRPCQueryError(err, "eth_gasPrice")
	}
	sort.Slice(allGasPrices, func(i, j int) bool {
		return allGasPrices[i].Cmp(allGasPrices[j]) < 0
	})
	var mdGasPrice *big.Int
	count := len(allGasPrices)
	mdInd := (count - 1) / 2
	if count%2 != 0 {
		mdGasPrice = allGasPrices[mdInd]
	} else {
		mdGasPrice = new(big.Int).Add(allGasPrices[mdInd], allGasPrices[mdInd+1])
		mdGasPrice.Div(mdGasPrice, big.NewInt(2))
	}
	logFunc("getMedianGasPrice success", "urls", urlCount, "count", count, "median", mdGasPrice)
	return mdGasPrice, nil
}

// SendSignedTransaction call eth_sendRawTransaction
func (b *Bridge) SendSignedTransaction(tx *types.Transaction) (txHash string, err error) {
	data, err := tx.MarshalBinary()
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
	logFunc := log.GetPrintFuncOr(params.IsDebugMode, log.Info, log.Trace)
	var result string
	for _, url := range urls {
		err = client.RPCPost(&result, url, "eth_sendRawTransaction", hexData)
		if err != nil {
			logFunc("call eth_sendRawTransaction failed", "txHash", result, "url", url, "err", err)
			continue
		}
		logFunc("call eth_sendRawTransaction success", "txHash", result, "url", url)
		if txHash == "" {
			txHash = result
		}
	}
	if txHash != "" {
		return txHash, nil
	}
	return "", wrapRPCQueryError(err, "eth_sendRawTransaction")
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
	return nil, wrapRPCQueryError(err, "eth_chainId")
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
	return nil, wrapRPCQueryError(err, "net_version")
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
	return nil, wrapRPCQueryError(err, "eth_getCode", contract)
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
	if err != nil {
		logFunc := log.GetPrintFuncOr(params.IsDebugMode, log.Info, log.Trace)
		logFunc("call CallContract failed", "contract", contract, "data", data, "err", err)
	}
	return "", wrapRPCQueryError(err, "eth_call", contract)
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
	return nil, wrapRPCQueryError(err, "eth_getBalance", account)
}

// SuggestGasTipCap call eth_maxPriorityFeePerGas
func (b *Bridge) SuggestGasTipCap() (maxGasTipCap *big.Int, err error) {
	gateway := b.GatewayConfig
	if len(gateway.APIAddressExt) > 0 {
		maxGasTipCap, err = getMaxGasTipCap(gateway.APIAddressExt)
	}
	maxGasTipCap2, err2 := getMaxGasTipCap(gateway.APIAddress)
	if err2 == nil {
		if maxGasTipCap == nil || maxGasTipCap2.Cmp(maxGasTipCap) > 0 {
			maxGasTipCap = maxGasTipCap2
		}
	} else {
		err = err2
	}
	if maxGasTipCap != nil {
		return maxGasTipCap, nil
	}
	return nil, err
}

func getMaxGasTipCap(urls []string) (maxGasTipCap *big.Int, err error) {
	if len(urls) == 0 {
		return nil, errEmptyURLs
	}
	var success bool
	var result hexutil.Big
	for _, url := range urls {
		err = client.RPCPost(&result, url, "eth_maxPriorityFeePerGas")
		if err == nil {
			success = true
			if maxGasTipCap == nil || result.ToInt().Cmp(maxGasTipCap) > 0 {
				maxGasTipCap = result.ToInt()
			}
		}
	}
	if success {
		return maxGasTipCap, nil
	}
	return nil, wrapRPCQueryError(err, "eth_maxPriorityFeePerGas")
}

// FeeHistory call eth_feeHistory
func (b *Bridge) FeeHistory(blockCount int, rewardPercentiles []float64) (*types.FeeHistoryResult, error) {
	gateway := b.GatewayConfig
	result, err := getFeeHistory(gateway.APIAddress, blockCount, rewardPercentiles)
	if err != nil && len(gateway.APIAddressExt) > 0 {
		result, err = getFeeHistory(gateway.APIAddressExt, blockCount, rewardPercentiles)
	}
	return result, err
}

func getFeeHistory(urls []string, blockCount int, rewardPercentiles []float64) (*types.FeeHistoryResult, error) {
	if len(urls) == 0 {
		return nil, errEmptyURLs
	}
	var result types.FeeHistoryResult
	var err error
	for _, url := range urls {
		err = client.RPCPost(&result, url, "eth_feeHistory", blockCount, "latest", rewardPercentiles)
		if err == nil {
			return &result, nil
		}
	}
	log.Warn("get fee history failed", "blockCount", blockCount, "err", err)
	return nil, wrapRPCQueryError(err, "eth_feeHistory", blockCount)
}

// GetBaseFee get base fee
func (b *Bridge) GetBaseFee(blockCount int) (*big.Int, error) {
	if blockCount == 0 { // from lastest block header
		block, err := b.GetBlockByNumber(nil)
		if err != nil {
			return nil, err
		}
		return block.BaseFee.ToInt(), nil
	}
	// from fee history
	feeHistory, err := b.FeeHistory(blockCount, nil)
	if err != nil {
		return nil, err
	}
	length := len(feeHistory.BaseFee)
	if length > 0 {
		return feeHistory.BaseFee[length-1].ToInt(), nil
	}
	return nil, wrapRPCQueryError(err, "eth_feeHistory", blockCount)
}

// EstimateGas call eth_estimateGas
func (b *Bridge) EstimateGas(from, to string, value *big.Int, data []byte) (uint64, error) {
	reqArgs := map[string]interface{}{
		"from":  from,
		"to":    to,
		"value": (*hexutil.Big)(value),
		"data":  hexutil.Bytes(data),
	}
	gateway := b.GatewayConfig
	var result hexutil.Uint64
	var err error
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress
		err = client.RPCPost(&result, url, "eth_estimateGas", reqArgs)
		if err == nil {
			return uint64(result), nil
		}
	}
	log.Warn("[rpc] estimate gas failed", "from", from, "to", to, "value", value, "data", hexutil.Bytes(data), "err", err)
	return 0, wrapRPCQueryError(err, "eth_estimateGas")
}
