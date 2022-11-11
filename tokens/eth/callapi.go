package eth

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/common/hexutil"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/types"
)

var (
	errEmptyURLs              = errors.New("empty URLs")
	errTxInOrphanBlock        = errors.New("tx is in orphan block")
	errTxHashMismatch         = errors.New("tx hash mismatch with rpc result")
	errTxBlockHashMismatch    = errors.New("tx block hash mismatch with rpc result")
	errTxReceiptMissBlockInfo = errors.New("tx receipt missing block info")

	wrapRPCQueryError = tokens.WrapRPCQueryError
)

// GetBlockConfirmations some chain may override this method
func (b *Bridge) GetBlockConfirmations(receipt *types.RPCTxReceipt) (uint64, error) {
	latest, err := b.GetLatestBlockNumber()
	if err != nil {
		return 0, err
	}
	blockNumber := receipt.BlockNumber.ToInt().Uint64()
	if latest > blockNumber {
		return latest - blockNumber, nil
	}
	return 0, nil
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
	if err != nil && tokens.IsRPCQueryOrNotFoundError(err) && len(gateway.APIAddressExt) > 0 {
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

// GetTxBlockInfo impl
func (b *Bridge) GetTxBlockInfo(txHash string) (blockHeight, blockTime uint64) {
	gateway := b.GatewayConfig
	receipt, err := b.getTransactionReceipt(txHash, gateway.APIAddress)
	if (err != nil || receipt == nil) && len(gateway.APIAddressExt) > 0 {
		receipt, err = b.getTransactionReceipt(txHash, gateway.APIAddressExt)
	}
	if err != nil || receipt == nil {
		return 0, 0
	}
	blockHeight = receipt.BlockNumber.ToInt().Uint64()
	return blockHeight, blockTime
}

// GetTransactionReceipt call eth_getTransactionReceipt
func (b *Bridge) GetTransactionReceipt(txHash string) (receipt *types.RPCTxReceipt, err error) {
	gateway := b.GatewayConfig
	receipt, err = b.getTransactionReceipt(txHash, gateway.APIAddress)
	if err != nil && tokens.IsRPCQueryOrNotFoundError(err) && len(gateway.APIAddressExt) > 0 {
		return b.getTransactionReceipt(txHash, gateway.APIAddressExt)
	}
	return receipt, err
}

func (b *Bridge) getTransactionReceipt(txHash string, urls []string) (result *types.RPCTxReceipt, err error) {
	if len(urls) == 0 {
		return nil, errEmptyURLs
	}
	for _, url := range urls {
		err = client.RPCPost(&result, url, "eth_getTransactionReceipt", txHash)
		if err == nil && result != nil {
			if result.BlockNumber == nil || result.BlockHash == nil || result.TxIndex == nil {
				return nil, errTxReceiptMissBlockInfo
			}
			if !common.IsEqualIgnoreCase(result.TxHash.Hex(), txHash) {
				return nil, errTxHashMismatch
			}
			if b.ChainConfig.EnableCheckTxBlockIndex {
				tx, errt := getTransactionByBlockNumberAndIndex(result.BlockNumber.ToInt(), uint(*result.TxIndex), url)
				if errt != nil {
					return nil, errt
				}
				if !common.IsEqualIgnoreCase(tx.Hash.Hex(), txHash) {
					return nil, errTxInOrphanBlock
				}
			}
			if b.ChainConfig.EnableCheckTxBlockHash {
				if err = b.checkTxBlockHash(result.BlockNumber.ToInt(), *result.BlockHash); err != nil {
					return nil, err
				}
			}
			return result, nil
		}
	}
	return nil, wrapRPCQueryError(err, "eth_getTransactionReceipt", txHash)
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

// GetPoolNonce call eth_getTransactionCount
func (b *Bridge) GetPoolNonce(address, height string) (uint64, error) {
	account := common.HexToAddress(address)
	gateway := b.GatewayConfig
	return getMedianPoolNonce(account, height, gateway.APIAddress)
}

func getMedianPoolNonce(account common.Address, height string, urls []string) (mdPoolNonce uint64, err error) {
	if len(urls) == 0 {
		return 0, errEmptyURLs
	}
	allPoolNonces := make([]uint64, 0, 10)
	for _, url := range urls {
		var result hexutil.Uint64
		err = client.RPCPost(&result, url, "eth_getTransactionCount", account, height)
		if err == nil {
			allPoolNonces = append(allPoolNonces, uint64(result))
			log.Info("call eth_getTransactionCount success", "url", url, "account", account, "nonce", uint64(result))
		}
	}
	if len(allPoolNonces) == 0 {
		log.Warn("GetPoolNonce failed", "account", account, "height", height, "err", err)
		return 0, wrapRPCQueryError(err, "eth_getTransactionCount", account, height)
	}
	sort.Slice(allPoolNonces, func(i, j int) bool {
		return allPoolNonces[i] < allPoolNonces[j]
	})
	count := len(allPoolNonces)
	mdInd := (count - 1) / 2
	if count%2 != 0 {
		mdPoolNonce = allPoolNonces[mdInd]
	} else {
		mdPoolNonce = (allPoolNonces[mdInd] + allPoolNonces[mdInd+1]) / 2
	}
	log.Info("GetPoolNonce success", "account", account, "urls", len(urls), "validCount", count, "median", mdPoolNonce)
	return mdPoolNonce, nil
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
	log.Info("call eth_sendRawTransaction start", "txHash", tx.Hash().String())
	hexData := common.ToHex(data)
	gateway := b.GatewayConfig
	urlCount := len(gateway.APIAddressExt) + len(gateway.APIAddress)
	ch := make(chan *sendTxResult, urlCount)
	wg := new(sync.WaitGroup)
	wg.Add(urlCount)
	go func() {
		wg.Wait()
		close(ch)
		log.Info("call eth_sendRawTransaction finished", "txHash", txHash)
	}()
	for _, url := range gateway.APIAddress {
		go sendRawTransaction(wg, hexData, url, ch)
	}
	for _, url := range gateway.APIAddressExt {
		go sendRawTransaction(wg, hexData, url, ch)
	}
	for i := 0; i < urlCount; i++ {
		res := <-ch
		txHash, err = res.txHash, res.err
		if err == nil && txHash != "" {
			return txHash, nil
		}
	}
	return "", wrapRPCQueryError(err, "eth_sendRawTransaction")
}

type sendTxResult struct {
	txHash string
	err    error
}

func sendRawTransaction(wg *sync.WaitGroup, hexData string, url string, ch chan<- *sendTxResult) {
	defer wg.Done()
	var result string
	err := client.RPCPost(&result, url, "eth_sendRawTransaction", hexData)
	if err != nil {
		log.Trace("call eth_sendRawTransaction failed", "txHash", result, "url", url, "err", err)
	} else {
		log.Trace("call eth_sendRawTransaction success", "txHash", result, "url", url)
	}
	ch <- &sendTxResult{result, err}
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
		err = client.RPCPost(&result, url, "eth_getBalance", account, params.GetBalanceBlockNumberOpt)
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
