package nebulas

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"sort"
	"strconv"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens"
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

func getNebState(url string) (*GetNebStateResponse, error) {
	var result NebResponse
	url = fmt.Sprintf("%s/v1/user/nebstate", url)
	err := client.RPCGet(&result, url)
	if err == nil {
		return &result.Result, nil
	}
	return nil, wrapRPCQueryError(err, "v1/user/nebstate")
}

// GetLatestBlockNumberOf call eth_blockNumber
func (b *Bridge) GetLatestBlockNumberOf(url string) (latest uint64, err error) {
	resp, err := getNebState(url)
	if err != nil {
		return 0, err
	}
	return resp.Height, nil
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
	var result *GetNebStateResponse
	for _, url := range urls {
		result, err = getNebState(url)
		if err == nil {
			height := result.Height
			if height > maxHeight {
				maxHeight = height
			}
		}
	}
	if maxHeight > 0 {
		return maxHeight, nil
	}
	return 0, wrapRPCQueryError(err, "v1/user/nebstate")
}

// GetBlockByHash call eth_getBlockByHash
func (b *Bridge) GetBlockByHash(blockHash string) (*BlockResponse, error) {
	gateway := b.GatewayConfig
	return getBlockByHash(blockHash, gateway.APIAddress)
}

func getBlockByHash(blockHash string, urls []string) (result *BlockResponse, err error) {
	if len(urls) == 0 {
		return nil, errEmptyURLs
	}
	var resp *http.Response
	for _, url := range urls {
		url = fmt.Sprintf("%s/v1/user/getBlockByHash", url)
		params := make(map[string]interface{})
		params["hash"] = blockHash
		params["full_fill_transaction"] = false
		resp, err = client.HTTPPost(url, params, nil, nil, 60)
		if err == nil && resp != nil {
			block := new(BlockResponse)
			err = ParseReponse(resp, block)
			if err != nil {
				return nil, err
			}
			return block, nil
		}
	}
	return nil, wrapRPCQueryError(err, "getBlockByHash", blockHash)
}

// GetBlockByNumber call eth_getBlockByNumber
func (b *Bridge) GetBlockByNumber(number *big.Int) (*BlockResponse, error) {
	gateway := b.GatewayConfig
	var resp *http.Response
	var err error
	for _, apiAddress := range gateway.APIAddress {
		url := fmt.Sprintf("%s/v1/user/getBlockByHeight", apiAddress)
		params := make(map[string]interface{})
		params["height"] = number.Int64()
		params["full_fill_transaction"] = false
		resp, err = client.HTTPPost(url, params, nil, nil, 60)
		if err == nil && resp != nil {
			block := new(BlockResponse)
			err = ParseReponse(resp, block)
			if err != nil {
				return nil, err
			}
			return block, nil
		}
	}
	return nil, wrapRPCQueryError(err, "getBlockByHeight", number)
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
	var resp *http.Response
	for _, url := range urls {
		url = fmt.Sprintf("%s/v1/user/getBlockByHeight", url)
		params := make(map[string]interface{})
		params["height"] = height
		params["full_fill_transaction"] = false
		resp, err = client.HTTPPost(url, params, nil, nil, 60)
		if err == nil && resp != nil {
			block := new(BlockResponse)
			err = ParseReponse(resp, block)
			if err != nil {
				return "", err
			}
			return block.Hash, nil
		}
	}
	return "", wrapRPCQueryError(err, "getBlockByHeight")
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
func (b *Bridge) GetTransactionByHash(txHash string) (*TransactionResponse, error) {
	gateway := b.GatewayConfig
	return b.getTransactionByHash(txHash, gateway.APIAddress)
}

func (b *Bridge) getTransactionByHash(txHash string, urls []string) (result *TransactionResponse, err error) {
	if len(urls) == 0 {
		return nil, errEmptyURLs
	}
	var resp *http.Response
	for _, url := range urls {
		url = fmt.Sprintf("%s/v1/user/getTransactionReceipt", url)
		params := make(map[string]interface{})
		params["hash"] = txHash
		resp, err = client.HTTPPost(url, params, nil, nil, 60)
		if err == nil && resp != nil {
			result = new(TransactionResponse)
			err = ParseReponse(resp, result)
			return
		}
	}
	return nil, wrapRPCQueryError(err, "getTransactionReceipt", txHash)
}

// GetTxBlockInfo impl
func (b *Bridge) GetTxBlockInfo(txHash string) (blockHeight, blockTime uint64) {
	receipt, err := b.GetTransactionByHash(txHash)
	if err != nil || receipt == nil {
		return 0, 0
	}
	blockHeight = receipt.BlockHeight
	block, err := b.GetBlockByNumber(big.NewInt(int64(blockHeight)))
	if err != nil || block == nil {
		return 0, 0
	}
	return blockHeight, uint64(block.Timestamp)
}

// GetTransactionReceipt call eth_getTransactionReceipt
func (b *Bridge) GetTransactionReceipt(txHash string) (receipt *TransactionResponse, url string, err error) {
	gateway := b.GatewayConfig
	receipt, url, err = b.getTransactionReceipt(txHash, gateway.APIAddress)
	if err != nil && errors.Is(err, tokens.ErrRPCQueryError) && len(gateway.APIAddressExt) > 0 {
		return b.getTransactionReceipt(txHash, gateway.APIAddressExt)
	}
	return receipt, url, err
}

func (b *Bridge) getTransactionReceipt(txHash string, urls []string) (result *TransactionResponse, rpcURL string, err error) {
	if len(urls) == 0 {
		return nil, "", errEmptyURLs
	}
	for _, url := range urls {

		pathUrl := fmt.Sprintf("%s/v1/user/getTransactionReceipt", url)
		params := make(map[string]interface{})
		params["hash"] = txHash
		var resp *http.Response
		resp, err = client.HTTPPost(pathUrl, params, nil, nil, 60)
		if err == nil && resp != nil {
			tx := new(TransactionResponse)
			err = ParseReponse(resp, tx)
			if err != nil {
				return nil, url, err
			}
			if tx.BlockHeight <= 0 {
				return nil, "", errTxReceiptMissBlockInfo
			}
			return tx, url, nil
		}
	}
	return nil, "", wrapRPCQueryError(err, "getTransactionReceipt", txHash)
}

// GetPoolNonce call eth_getTransactionCount
func (b *Bridge) GetPoolNonce(address, height string) (uint64, error) {
	account, err := AddressParse(address)
	if err != nil {
		return 0, err
	}
	gateway := b.GatewayConfig
	return getMaxPoolNonce(account, height, gateway.APIAddress)
}

func getMaxPoolNonce(account *Address, height string, urls []string) (maxNonce uint64, err error) {
	if len(urls) == 0 {
		return 0, errEmptyURLs
	}
	var success bool
	for _, url := range urls {
		pathUrl := fmt.Sprintf("%s/v1/user/accountstate", url)
		params := make(map[string]interface{})
		params["address"] = account.String()
		var resp *http.Response
		resp, err = client.HTTPPost(pathUrl, params, nil, nil, 60)
		if err == nil {
			success = true
			astate := new(GetAccountStateResponse)
			err = ParseReponse(resp, astate)
			if err != nil {
				return 0, err
			}
			nonce, err := strconv.ParseUint(astate.Nonce, 10, 64)
			if err != nil {
				return 0, err
			}
			if nonce > maxNonce {
				maxNonce = nonce
			}
		}
	}
	if success {
		return maxNonce, nil
	}
	return 0, wrapRPCQueryError(err, "accountstate", account, height)
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

	var err error
	for _, urls := range urlsSlice {
		urlCount += len(urls)
		for _, url := range urls {
			var result PriceResponse
			url = fmt.Sprintf("%s/v1/user/getGasPrice", url)
			err = client.RPCGet(&result, url)
			if err != nil {
				logFunc("call getGasPrice failed", "url", url, "err", err)
				continue
			}
			n := new(big.Int)
			gasPrice, ok := n.SetString(result.Result.GasPrice, 10)
			if !ok {
				logFunc("call getGasPrice parse failed")
				continue
			}
			allGasPrices = append(allGasPrices, gasPrice)
		}
	}
	if len(allGasPrices) == 0 {
		log.Warn("getMedianGasPrice failed", "err", err)
		return nil, wrapRPCQueryError(err, "getGasPrice")
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

// SendSignedTransaction call sendRawTransaction
func (b *Bridge) SendSignedTransaction(tx *Transaction) (txHash string, err error) {
	data, err := tx.Bytes()
	if err != nil {
		return "", err
	}
	gateway := b.GatewayConfig
	txHash, _ = sendRawTransaction(data, gateway.APIAddressExt)
	txHash2, err := sendRawTransaction(data, gateway.APIAddress)
	if txHash != "" {
		return txHash, nil
	}
	if txHash2 != "" {
		return txHash2, nil
	}
	return "", err
}

func sendRawTransaction(data []byte, urls []string) (txHash string, err error) {
	if len(urls) == 0 {
		return "", errEmptyURLs
	}
	logFunc := log.GetPrintFuncOr(params.IsDebugMode, log.Info, log.Trace)

	for _, url := range urls {
		pathUrl := fmt.Sprintf("%s/v1/user/rawtransaction", url)
		params := make(map[string]interface{})
		params["data"] = data
		var resp *http.Response
		resp, err = client.HTTPPost(pathUrl, params, nil, nil, 60)
		if err == nil {
			sResp := new(SendTransactionResponse)
			err = ParseReponse(resp, sResp)
			if err != nil {
				logFunc("call rawtransaction failed", "txHash", sResp, "url", url, "err", err)
				continue
			}
			if txHash == "" {
				txHash = sResp.Txhash
			}
		}
	}
	if txHash != "" {
		return txHash, nil
	}
	return "", wrapRPCQueryError(err, "rawtransaction")
}

// ChainID call eth_chainId
// Notice: eth_chainId return 0x0 for mainnet which is wrong (use net_version instead)
func (b *Bridge) ChainID() (*big.Int, error) {
	gateway := b.GatewayConfig
	var err error
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress
		var result *GetNebStateResponse
		result, err = getNebState(url)
		if err == nil {
			return big.NewInt(int64(result.ChainId)), nil
		}
	}
	return nil, wrapRPCQueryError(err, "chainID")
}

// NetworkID call net_version
func (b *Bridge) NetworkID() (*big.Int, error) {
	return b.ChainID()
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

func ParseReponse(resp *http.Response, v interface{}) error {
	defer func() {
		_ = resp.Body.Close()
	}()
	const maxReadContentLength int64 = 1024 * 1024 * 10 // 10M
	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, maxReadContentLength))
	if err != nil {
		return fmt.Errorf("read body error: %w", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("wrong response status %v. message: %v", resp.StatusCode, string(body))
	}
	if len(body) == 0 {
		return fmt.Errorf("empty response body")
	}

	var mapResult map[string]interface{}
	err = json.Unmarshal(body, &mapResult)
	if err != nil {
		return fmt.Errorf("unmarshal body error, body is \"%v\" err=\"%w\"", string(body), err)
	}

	rbytes, err := json.Marshal(mapResult["result"])
	if err != nil {
		return nil
	}
	return json.Unmarshal(rbytes, v)

}

// CallContract call eth_call
func (b *Bridge) CallContract(contract string, value string, fun string, args string) (string, error) {
	reqArgs := buildTxArgs("", contract, value, fun, args)
	log.Debug("CallContract", "reqArgs", reqArgs)
	gateway := b.GatewayConfig
	var err error
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress

		pathUrl := fmt.Sprintf("%s/v1/user/call", url)
		var resp *http.Response
		resp, err = client.HTTPPost(pathUrl, reqArgs, nil, nil, 60)
		if err == nil {
			respObj := new(CallResponse)
			err = ParseReponse(resp, respObj)
			if err == nil {
				return respObj.Result, nil
			}
		}

	}
	if err != nil {
		logFunc := log.GetPrintFuncOr(params.IsDebugMode, log.Info, log.Trace)
		logFunc("call CallContract failed", "contract", contract, "data", fun, "err", err)
	}
	return "", wrapRPCQueryError(err, "call", contract)
}

func buildTxArgs(from string, contract string, value string, fun string, args string) map[string]interface{} {
	if len(from) == 0 {
		from = "n1gczhpkT54RaT4PB55CNoYbqmEQcfo4hqq"
	}
	contractArgs := map[string]string{
		"function": fun,
		"args":     args,
	}
	reqArgs := map[string]interface{}{
		"from":      from,
		"to":        contract,
		"value":     value,
		"nonce":     1,
		"gas_price": "20000000000",
		"gas_limit": "10000000000",
		"contract":  contractArgs,
	}
	return reqArgs
}

// GetBalance call eth_getBalance
func (b *Bridge) GetBalance(account string) (*big.Int, error) {
	gateway := b.GatewayConfig
	var err error
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress
		pathUrl := fmt.Sprintf("%s/v1/user/accountstate", url)
		params := make(map[string]interface{})
		params["address"] = account
		var resp *http.Response
		resp, err = client.HTTPPost(pathUrl, params, nil, nil, 60)
		if err == nil {
			respObj := new(GetAccountStateResponse)
			err = ParseReponse(resp, respObj)
			if err != nil {
				return nil, err
			}
			balance, _ := new(big.Int).SetString(respObj.Balance, 10)
			return balance, nil
		}
	}
	return nil, wrapRPCQueryError(err, "getBalance", account)
}

// EstimateGas call eth_estimateGas
func (b *Bridge) EstimateGas(from, to string, value *big.Int, input []byte) (uint64, error) {
	payload, er := LoadCallPayload(input)
	if er != nil {
		return 0, er
	}
	reqArgs := buildTxArgs(from, to, value.String(), payload.Function, payload.Args)
	gateway := b.GatewayConfig
	var err error
	for _, apiAddress := range gateway.APIAddress {
		url := apiAddress
		pathUrl := fmt.Sprintf("%s/v1/user/estimateGas", url)
		var resp *http.Response
		resp, err = client.HTTPPost(pathUrl, reqArgs, nil, nil, 60)
		if err == nil {
			respObj := new(GasResponse)
			err = ParseReponse(resp, respObj)
			if err == nil {
				return strconv.ParseUint(respObj.Gas, 10, 64)
			}
		}
	}
	log.Warn("[rpc] estimate gas failed", "from", from, "to", to, "value", value, "func", payload.Function, "err", err)
	return 0, wrapRPCQueryError(err, "estimateGas")
}
