package cosmos

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	amino "github.com/tendermint/go-amino"

	//ctypes "github.com/tendermint/tendermint/rpc/core/types"
	ttypes "github.com/tendermint/tendermint/types"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

/*
Rest api doc
https://cosmos.network/rpc/v0.39.2
https://lcd.terra.dev/swagger-ui/#/
*/

var CDC = amino.NewCodec()

// TimeFormat is cosmos time format
const TimeFormat = time.RFC3339Nano

// GetBalance gets main token balance
// call  rest api"/bank/balances/"
func (b *Bridge) GetBalance(account string) (balance *big.Int, err error) {
	endpoints := b.GatewayConfig.APIAddress
	for _, endpoint := range endpoints {
		endpointURL, err := url.Parse(endpoint)
		if err != nil {
			continue
		}
		endpoint = endpointURL.String()
		endpoint = strings.TrimSuffix(endpoint, "/")
		endpoint = endpoint + "/"
		client := resty.New()
		resp, err := client.R().Get(fmt.Sprintf("%vbank/balances/%v", endpoint, account))
		if err != nil || resp.StatusCode() != 200 {
			log.Warn("cosmos rest request error", "resp", string(resp.Body()), "request error", err, "func", "GetBalance")
			continue
		}
		/*var balances sdk.Coins
		err = CDC.UnmarshalJSON(resp.Body(), &balances)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetBalance", "resp", string(resp.Body()))
			continue
		}*/
		balanceResp := make(map[string]interface{})
		unmarshalerr := json.Unmarshal(resp.Body(), &balanceResp)
		if err != nil {
			return big.NewInt(0), unmarshalerr
		}
		balances, ok := balanceResp["result"].([]interface{})
		if !ok {
			return big.NewInt(0), errors.New("get balances error")
		}
		for _, bali := range balances {
			bal := bali.(map[string]interface{})
			if bal["denom"].(string) != b.MainCoin.Denom {
				continue
			}
			amountstr, ok := bal["amount"].(string)
			if ok {
				balance, _ = new(big.Int).SetString(amountstr, 0)
			}
		}
		if balance == nil {
			balance = big.NewInt(0)
		}
		return balance, nil
	}
	return
}

// GetTokenBalance gets balance for given token
// call  rest api"/bank/balances/"
func (b *Bridge) GetTokenBalance(tokenType, tokenName, accountAddress string) (balance *big.Int, err error) {
	coin, ok := b.GetCoin(tokenName)
	if !ok {
		return nil, fmt.Errorf("Unsupported coin: %v", tokenName)
	}
	endpoints := b.GatewayConfig.APIAddress
	for _, endpoint := range endpoints {
		endpointURL, err := url.Parse(endpoint)
		if err != nil {
			continue
		}
		endpoint = endpointURL.String()
		endpoint = strings.TrimSuffix(endpoint, "/")
		endpoint = endpoint + "/"
		client := resty.New()
		resp, err := client.R().Get(fmt.Sprintf("%vbank/balances/%v", endpoint, accountAddress))
		if err != nil || resp.StatusCode() != 200 {
			log.Warn("cosmos rest request error", "resp", string(resp.Body()), "error", err, "func", "GetTokenBalance")
			continue
		}
		/*var balances sdk.Coins
		err = CDC.UnmarshalJSON(resp.Body(), &balances)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetTokenBalance", "resp", string(resp.Body()))
			continue
		}*/
		balanceResp := make(map[string]interface{})
		unmarshalerr := json.Unmarshal(resp.Body(), &balanceResp)
		if err != nil {
			return big.NewInt(0), unmarshalerr
		}
		balances, ok := balanceResp["result"].([]interface{})
		if !ok {
			return big.NewInt(0), errors.New("get balances error")
		}
		for _, bali := range balances {
			bal := bali.(map[string]interface{})
			if bal["denom"].(string) != coin.Denom {
				continue
			}
			amountstr, ok := bal["amount"].(string)
			if ok {
				balance, _ = new(big.Int).SetString(amountstr, 0)
			}
		}
		if balance == nil {
			balance = big.NewInt(0)
		}
		return balance, nil
	}
	return
}

// GetTokenSupply not supported
func (b *Bridge) GetTokenSupply(tokenType, tokenAddress string) (*big.Int, error) {
	return nil, fmt.Errorf("Cosmos bridges does not support this method")
}

// GetTransaction gets tx by hash, returns sdk.Tx
// call rest api "/txs/{txhash}"
func (b *Bridge) GetTransaction(txHash string) (tx interface{}, err error) {
	endpoints := b.GatewayConfig.APIAddress
	for _, endpoint := range endpoints {
		endpointURL, err := url.Parse(endpoint)
		if err != nil {
			continue
		}
		endpoint = endpointURL.String()
		endpoint = strings.TrimSuffix(endpoint, "/")
		endpoint = endpoint + "/"
		client := resty.New()

		resp, err := client.R().Get(fmt.Sprintf("%vtxs/%v", endpoint, txHash))
		if err != nil || resp.StatusCode() != 200 {
			log.Warn("cosmos rest request error", "resp", string(resp.Body()), "request error", err, "func", "GetTransaction")
			continue
		}
		var txResult sdk.TxResponse
		err = CDC.UnmarshalJSON(resp.Body(), &txResult)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetTransaction", "resp", string(resp.Body()))
			return nil, err
		}
		if txResult.Code != 0 {
			log.Warn("Check tx code failed", "resp", string(resp.Body()))
			return nil, fmt.Errorf("tx status error: %+v", txResult)
		}
		tx = txResult.Tx
		/*err = tx.(sdk.Tx).ValidateBasic()
		if err != nil {
			log.Warn("Transaction validate basic error", "error", err, "resp", string(resp.Body()))
			return nil, err
		} else {
			log.Debug("Get transaction success", "tx", tx, "resp", string(resp.Body()))
			return tx, err
		}*/
		return tx, err
	}
	return
}

// GetTransactionStatus returns tx status
// call rest api "/txs/{txhash}"
func (b *Bridge) GetTransactionStatus(txHash string) (status *tokens.TxStatus, err1 error) {
	status = &tokens.TxStatus{
		// Receipt
		//Confirmations
		//BlockHeight: uint64(txRes.Height),
		//BlockHash
		//BlockTime
	}
	endpoints := b.GatewayConfig.APIAddress
	for _, endpoint := range endpoints {
		endpointURL, err := url.Parse(endpoint)
		if err != nil {
			continue
		}
		endpoint = endpointURL.String()
		endpoint = strings.TrimSuffix(endpoint, "/")
		endpoint = endpoint + "/"
		client := resty.New()

		resp, err := client.R().Get(fmt.Sprintf("%vtxs/%v", endpoint, txHash))
		if err != nil || resp.StatusCode() != 200 {
			log.Warn("cosmos rest request error", "resp", string(resp.Body()), "request error", err, "func", "GetTransactionStatus")
			continue
		}

		var txResult sdk.TxResponse
		err = CDC.UnmarshalJSON(resp.Body(), &txResult)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetTransactionStatus", "resp", string(resp.Body()))
			//err1 = err
			return
		}
		/*tx := txResult.Tx
		err = tx.ValidateBasic()
		if err != nil {
			err1 = err
			return
		}*/
		status.Receipt = false
		if txResult.Code == 0 {
			status.Receipt = true
		}
		status.BlockHeight = uint64(txResult.Height)
		t, err := time.Parse(TimeFormat, txResult.Timestamp)
		if err == nil {
			status.BlockTime = uint64(t.Unix())
		}

		if status.BlockHeight > 0 {
			status.Confirmations = 10
			status.Finalized = true // asserts that tx has finalized, no need to check everything again
		}
		return
	}
	return
}

type ResultBlock struct {
	//BlockMeta *ttypes.BlockMeta `json:"block_meta"`
	Block *ttypes.Block `json:"block"`
}

// GetLatestBlockNumber returns current block height
// call rest api "/blocks/latest"
func (b *Bridge) GetLatestBlockNumber() (height uint64, err error) {
	endpoints := b.GatewayConfig.APIAddress
	for _, endpoint := range endpoints {
		endpointURL, err := url.Parse(endpoint)
		if err != nil {
			continue
		}
		endpoint = endpointURL.String()
		endpoint = strings.TrimSuffix(endpoint, "/")
		endpoint = endpoint + "/"
		client := resty.New()

		resp, err := client.R().Get(fmt.Sprintf("%vblocks/latest", endpoint))
		if err != nil || resp.StatusCode() != 200 {
			log.Warn("cosmos rest request error", "resp", string(resp.Body()), "request error", err, "func", "GetLatestBlockNumber")
			continue
		}
		//var blockRes ResultBlock
		//err = CDC.UnmarshalJSON(resp.Body(), &blockRes)
		res := make(map[string]interface{})
		err = json.Unmarshal(resp.Body(), &res)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetLatestBlockNumber", "resp", string(resp.Body()))
			continue
		}
		block, ok1 := res["block"].(map[string]interface{})
		if !ok1 {
			return 0, fmt.Errorf("parse block error: %v", resp.Body())
		}
		header, ok2 := block["header"].(map[string]interface{})
		if !ok2 {
			return 0, fmt.Errorf("parse height error: %v", block)
		}
		heightstr, ok3 := header["height"].(string)
		if !ok3 {
			return 0, fmt.Errorf("parse height error: %v", header)
		}
		height, strconverr := strconv.ParseUint(heightstr, 10, 64)
		if strconverr != nil {
			return 0, fmt.Errorf("convert height error: %v", heightstr)
		}
		//height = uint64(blockRes.Block.Header.Height)
		return height, nil
	}
	return
}

// GetLatestBlockNumberOf returns current block height of given node
// call rest api "/blocks/latest"
func (b *Bridge) GetLatestBlockNumberOf(apiAddress string) (uint64, error) {
	endpointURL, err := url.Parse(apiAddress)
	if err != nil {
		return 0, err
	}
	endpoint := endpointURL.String()
	endpoint = strings.TrimSuffix(endpoint, "/")
	endpoint = endpoint + "/"
	client := resty.New()

	resp, err := client.R().Get(fmt.Sprintf("%vblocks/latest", endpoint))
	if err != nil || resp.StatusCode() != 200 {
		log.Warn("cosmos rest request error", "resp", string(resp.Body()), "request error", err, "func", "GetLatestBlockNumber")
		return 0, err
	}
	//var blockRes ResultBlock
	//err = CDC.UnmarshalJSON(resp.Body(), &blockRes)
	res := make(map[string]interface{})
	err = json.Unmarshal(resp.Body(), &res)
	if err != nil {
		log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetLatestBlockNumber", "resp", string(resp.Body()))
		return 0, err
	}
	block, ok1 := res["block"].(map[string]interface{})
	if !ok1 {
		return 0, fmt.Errorf("parse block error: %v", resp.Body())
	}
	header, ok2 := block["header"].(map[string]interface{})
	if !ok2 {
		return 0, fmt.Errorf("parse height error: %v", block)
	}
	heightstr, ok3 := header["height"].(string)
	if !ok3 {
		return 0, fmt.Errorf("parse height error: %v", header)
	}
	height, strconverr := strconv.ParseUint(heightstr, 10, 64)
	if strconverr != nil {
		return 0, fmt.Errorf("convert height error: %v", heightstr)
	}
	//height = uint64(blockRes.Block.Header.Height)
	return height, nil
}

func GetAccountNumber(endpoint, address string) (uint64, error) {
	client := resty.New()

	resp, err := client.R().Get(fmt.Sprintf("%vauth/accounts/%v", endpoint, address))
	if err != nil || resp.StatusCode() != 200 {
		log.Warn("cosmos rest request error", "resp", string(resp.Body()), "request error", err, "func", "GetAccountNumber")
		return 0, err
	}
	tempStruct := make(map[string]interface{})
	err = json.Unmarshal(resp.Body(), &tempStruct)
	if err != nil {
		log.Warn("Marshal resp error", "err", err)
		return 0, err
	}
	result, ok := tempStruct["result"]
	if !ok {
		log.Warn("Get result error")
		return 0, fmt.Errorf("Get result error")
	}
	bz, err := json.Marshal(result)
	if err != nil {
		log.Warn("Marshal result error", "err", err)
		return 0, err
	}
	var accountRes authtypes.BaseAccount
	err = CDC.UnmarshalJSON(bz, &accountRes)
	if err != nil {
		log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetAccountNumber")
		return 0, err
	}
	accountNumber := accountRes.AccountNumber
	return accountNumber, nil
}

// GetAccountNumber gets account number, a series number of account on a cosmos state
// call rest api "/auth/accounts/"
func (b *Bridge) GetAccountNumber(address string) (uint64, error) {
	endpoints := b.GatewayConfig.APIAddress
	for _, endpoint := range endpoints {
		endpointURL, err := url.Parse(endpoint)
		if err != nil {
			continue
		}
		endpoint = endpointURL.String()
		endpoint = strings.TrimSuffix(endpoint, "/")
		endpoint = endpoint + "/"
		accn, err := GetAccountNumber(endpoint, address)
		if err != nil {
			continue
		}
		return accn, nil
	}
	return 0, nil
}

func GetPoolNonce(endpoint, address, height string) (uint64, error) {
	client := resty.New()

	resp, err := client.R().Get(fmt.Sprintf("%vauth/accounts/%v", endpoint, address))
	if err != nil || resp.StatusCode() != 200 {
		log.Warn("cosmos rest request error", "resp", string(resp.Body()), "request error", err, "func", "GetPoolNonce")
		return 0, err
	}
	tempStruct := make(map[string]interface{})
	err = json.Unmarshal(resp.Body(), &tempStruct)
	if err != nil {
		log.Warn("Marshal resp error", "err", err)
		return 0, err
	}
	result, ok := tempStruct["result"]
	if !ok {
		log.Warn("Get result error")
		return 0, fmt.Errorf("Result error")
	}
	bz, err := json.Marshal(result)
	if err != nil {
		log.Warn("Marshal result error", "err", err)
		return 0, err
	}
	var accountRes authtypes.BaseAccount
	err = CDC.UnmarshalJSON(bz, &accountRes)
	if err != nil {
		log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetPoolNonce")
		return 0, err
	}
	seq := accountRes.Sequence
	return seq, nil
}

// GetPoolNonce gets account sequence
// call rest api "/auth/accounts/"
func (b *Bridge) GetPoolNonce(address, height string) (uint64, error) {
	endpoints := b.GatewayConfig.APIAddress
	for _, endpoint := range endpoints {
		endpointURL, err := url.Parse(endpoint)
		if err != nil {
			continue
		}
		endpoint = endpointURL.String()
		endpoint = strings.TrimSuffix(endpoint, "/")
		endpoint = endpoint + "/"
		seq, err := GetPoolNonce(endpoint, address, height)
		if err != nil {
			continue
		}
		return seq, nil
	}
	return 0, nil
}

// SearchTxsHash searches tx in range of blocks
// call rest api "/txs?..."
func (b *Bridge) SearchTxsHash(start, end *big.Int) ([]string, error) {
	txs := make([]string, 0)
	var limit = 100
	var page = 1
	var pageTotal = 1
	endpoints := b.GatewayConfig.APIAddress

	// search send
	for page <= pageTotal {
		log.Debug("Search send msgs", "start", start, "end", end, "limit", limit, "page", page, "pageTotal", pageTotal)
		for _, endpoint := range endpoints {
			endpointURL, err := url.Parse(endpoint)
			if err != nil {
				continue
			}
			endpoint = endpointURL.String()
			endpoint = strings.TrimSuffix(endpoint, "/")
			endpoint = endpoint + "/"
			client := resty.New()
			params := fmt.Sprintf("?message.action=send&page=%v&limit=%v&tx.minheight=%v&tx.maxheight=%v", page, limit, start, end)
			resp, err := client.R().Get(fmt.Sprintf("%vtxs%v", endpoint, params))
			if err != nil || resp.StatusCode() != 200 {
				log.Warn("cosmos rest request error", "resp", string(resp.Body()), "request error", err, "func", "SearchTxsHash")
				continue
			}
			var res sdk.SearchTxsResult
			err = CDC.UnmarshalJSON(resp.Body(), &res)
			if err != nil {
				log.Warn("Search txs unmarshal error", "start", start, "end", end, "page", page, "func", "SearchTxsHash")
				continue
			}
			pageTotal = res.PageTotal
			log.Debug("Txs containing Send msgs", "length", len(res.Txs))
			for _, tx := range res.Txs {
				if tx.Code != 0 {
					log.Debug("discard failed tx")
					continue
				}
				txs = append(txs, tx.TxHash)
			}
			break
		}
		page = page + 1
	}

	// search multisend
	page = 1
	pageTotal = 1
	for page <= pageTotal {
		log.Debug("Search multisend msgs", "start", start, "end", end, "limit", limit, "page", page, "pageTotal", pageTotal)
		for _, endpoint := range endpoints {
			endpointURL, err := url.Parse(endpoint)
			if err != nil {
				continue
			}
			endpoint = endpointURL.String()
			endpoint = strings.TrimSuffix(endpoint, "/")
			endpoint = endpoint + "/"
			client := resty.New()
			params := fmt.Sprintf("?message.action=multisend&page=%v&limit=%v&tx.minheight=%v&tx.maxheight=%v", page, limit, start, end)
			resp, err := client.R().Get(fmt.Sprintf("%vtxs%v", endpoint, params))
			if err != nil || resp.StatusCode() != 200 {
				log.Warn("cosmos rest request error", "resp", string(resp.Body()), "request error", err, "func", "SearchTxsHash")
				continue
			}
			var res sdk.SearchTxsResult
			err = CDC.UnmarshalJSON(resp.Body(), &res)
			if err != nil {
				log.Warn("Search txs unmarshal error", "start", start, "end", end, "page", page, "func", "SearchTxsHash")
				continue
			}
			pageTotal = res.PageTotal
			log.Debug("Txs containing MultiSend msgs", "length", len(res.Txs))
			for _, tx := range res.Txs {
				if tx.Code != 0 {
					log.Debug("discard failed tx")
					continue
				}
				txs = append(txs, tx.TxHash)
			}
			break
		}
		page = page + 1
	}
	log.Debug("Search txs finish")
	return txs, nil
}

// SearchTxs searches tx in range of blocks
// call rest api "/txs?..."
func (b *Bridge) SearchTxs(start, end *big.Int) ([]sdk.TxResponse, error) {
	txs := make([]sdk.TxResponse, 0)
	var limit = 100

	// search send
	var page = 1
	var pageTotal = 1
	endpoints := b.GatewayConfig.APIAddress
	for page <= pageTotal {
		log.Debug("Search send msgs", "start", start, "end", end, "limit", limit, "page", page, "pageTotal", pageTotal)
		for _, endpoint := range endpoints {
			endpointURL, err := url.Parse(endpoint)
			if err != nil {
				continue
			}
			endpoint = endpointURL.String()
			endpoint = strings.TrimSuffix(endpoint, "/")
			endpoint = endpoint + "/"
			client := resty.New()
			params := fmt.Sprintf("?message.action=send&page=%v&limit=%v&tx.minheight=%v&tx.maxheight=%v", page, limit, start, end)
			resp, err := client.R().Get(fmt.Sprintf("%vtxs%v", endpoint, params))
			if err != nil || resp.StatusCode() != 200 {
				log.Warn("cosmos rest request error", "resp", string(resp.Body()), "request error", err, "func", "SearchTxs")
				continue
			}
			var res sdk.SearchTxsResult
			err = CDC.UnmarshalJSON(resp.Body(), &res)
			if err != nil {
				log.Warn("Search txs unmarshal error", "start", start, "end", end, "page", page, "func", "SearchTxs")
				continue
			}
			pageTotal = res.PageTotal
			log.Debug("Txs containing Send msgs", "length", len(res.Txs))
			for _, txresp := range res.Txs {
				if txresp.Code != 0 {
					log.Debug("discard failed tx")
					continue
				}
				txs = append(txs, txresp)
			}
			break
		}
		page = page + 1
	}

	// search multisend
	page = 1
	pageTotal = 1
	for page <= pageTotal {
		log.Debug("Search multisend msgs", "start", start, "end", end, "limit", limit, "page", page, "pageTotal", pageTotal)
		for _, endpoint := range endpoints {
			endpointURL, err := url.Parse(endpoint)
			if err != nil {
				continue
			}
			endpoint = endpointURL.String()
			endpoint = strings.TrimSuffix(endpoint, "/")
			endpoint = endpoint + "/"
			client := resty.New()
			params := fmt.Sprintf("?message.action=multisend&page=%v&limit=%v&tx.minheight=%v&tx.maxheight=%v", page, limit, start, end)
			resp, err := client.R().Get(fmt.Sprintf("%v/txs%v", endpoint, params))
			if err != nil || resp.StatusCode() != 200 {
				log.Warn("cosmos rest request error", "resp", string(resp.Body()), "request error", err, "func", "SearchTxs")
				continue
			}
			var res sdk.SearchTxsResult
			err = CDC.UnmarshalJSON(resp.Body(), &res)
			if err != nil {
				log.Warn("Search txs unmarshal error", "start", start, "end", end, "page", page, "func", "SearchTxs")
				continue
			}
			pageTotal = res.PageTotal
			log.Debug("Txs containing MultiSend msgs", "length", len(res.Txs))
			for _, txresp := range res.Txs {
				if txresp.Code != 0 {
					log.Debug("discard failed tx")
					continue
				}
				txs = append(txs, txresp)
			}
			break
		}
		page = page + 1
	}
	return txs, nil
}

// BroadcastTx broadcast tx
// post "txs" to rest api
// mode: block
func (b *Bridge) BroadcastTx(tx HashableStdTx) (string, error) {
	txhash := ""
	stdtx := tx.ToStdTx()

	bz, err := CDC.MarshalJSON(stdtx)
	if err != nil {
		return txhash, err
	}
	// Take "value" from the json struct
	tempStr := make(map[string]interface{})
	err = json.Unmarshal(bz, &tempStr)
	if err != nil {
		return txhash, err
	}
	value, ok := tempStr["value"].(map[string]interface{})
	if !ok {
		return txhash, fmt.Errorf("tx value error")
	}
	// repass account number and sequence
	signatures, ok := value["signatures"].([]interface{})
	if !ok || len(signatures) < 1 {
		return txhash, fmt.Errorf("tx value not contain signature")
	}
	signatures[0].(map[string]interface{})["account_number"] = fmt.Sprintf("%v", tx.AccountNumber)
	signatures[0].(map[string]interface{})["sequence"] = fmt.Sprintf("%v", tx.Sequence)
	value["signatures"] = signatures
	bz2, err := json.Marshal(value)
	if err != nil {
		return txhash, fmt.Errorf("Remarshal, std tx error: %v", err)
	}
	data := fmt.Sprintf(`{"tx":%v,"mode":"block"}`, string(bz2))

	log.Info("!!! broadcast", "data", data)

	endpoints := b.GatewayConfig.APIAddress
	for _, endpoint := range endpoints {
		endpointURL, err := url.Parse(endpoint)
		if err != nil {
			continue
		}
		endpoint = endpointURL.String()
		endpoint = strings.TrimSuffix(endpoint, "/")
		api := endpoint + "/txs"
		log.Info("!!! broadcast", "api", api)

		client := &http.Client{}

		req, err := http.NewRequest("POST", api, strings.NewReader(data))
		if err != nil {
			continue
		}
		req.Header.Set("accept", "application/json")
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			log.Warn("Broadcast tx error", "error", err)
			continue
		}
		bodyText, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		log.Info("Broadcast tx", "resp", bodyText)

		var res map[string]interface{}
		err = json.Unmarshal([]byte(bodyText), &res)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err, "func", "BroadcastTx")
			continue
		}
		height, ok1 := res["height"].(string)
		restxhash, ok2 := res["txhash"].(string)
		if !ok1 || !ok2 || height == "0" {
			log.Warn("Broadcast tx failed", "response", bodyText)
			continue
		}
		txhash = restxhash
		log.Debug("Broadcast tx success", "txhash", restxhash, "height", height)
		return txhash, nil
	}
	return txhash, nil
}
