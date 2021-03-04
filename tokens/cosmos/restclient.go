package cosmos

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/go-resty/resty/v2"
	amino "github.com/tendermint/go-amino"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

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
		client := resty.New()
		resp, err := client.R().Get(fmt.Sprintf("%vbank/balances/%v", endpoint, account))
		if err != nil || resp.StatusCode() != 200 {
			log.Warn("cosmos rest request error", "request error", err, "func", "GetBalance")
			continue
		}
		var balances sdk.Coins
		err = CDC.UnmarshalJSON(resp.Body(), &balances)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetBalance", "resp", string(resp.Body()))
			continue
		}
		for _, bal := range balances {
			if bal.Denom != b.MainCoin.Denom {
				continue
			}
			balance = bal.Amount.BigInt()
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
		client := resty.New()
		resp, err := client.R().Get(fmt.Sprintf("%vbank/balances/%v", endpoint, accountAddress))
		if err != nil || resp.StatusCode() != 200 {
			log.Warn("cosmos rest request error", "error", err, "func", "GetTokenBalance")
			continue
		}
		var balances sdk.Coins
		err = CDC.UnmarshalJSON(resp.Body(), &balances)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetTokenBalance", "resp", string(resp.Body()))
			continue
		}
		for _, bal := range balances {
			if bal.Denom != coin.Denom {
				continue
			}
			balance = bal.Amount.BigInt()
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
		client := resty.New()

		resp, err := client.R().Get(fmt.Sprintf("%vtxs/%v", endpoint, txHash))
		if err != nil || resp.StatusCode() != 200 {
			log.Warn("cosmos rest request error", "request error", err, "func", "GetTransaction")
			continue
		}
		var txResult sdk.TxResponse
		err = CDC.UnmarshalJSON(resp.Body(), &txResult)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetTransaction", "resp", string(resp.Body()))
			return nil, err
		}
		if txResult.Code != 0 {
			return nil, fmt.Errorf("tx status error: %+v", txResult)
		}
		tx = txResult.Tx
		err = tx.(sdk.Tx).ValidateBasic()
		if err != nil {
			return nil, err
		} else {
			return tx, err
		}
	}
	return
}

// GetTransactionStatus returns tx status
// call rest api "/txs/{txhash}"
func (b *Bridge) GetTransactionStatus(txHash string) (status *tokens.TxStatus) {
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
		client := resty.New()

		resp, err := client.R().Get(fmt.Sprintf("%vtxs/%v", endpoint, txHash))
		if err != nil || resp.StatusCode() != 200 {
			log.Warn("cosmos rest request error", "request error", err, "func", "GetTransactionStatus")
			continue
		}

		var txResult sdk.TxResponse
		err = CDC.UnmarshalJSON(resp.Body(), &txResult)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetTransactionStatus", "resp", string(resp.Body()))
			return
		}
		tx := txResult.Tx
		err = tx.ValidateBasic()
		if err != nil {
			return
		}
		if txResult.Code != 0 {
			status.Confirmations = 0
		} else {
			status.Confirmations = 1
		}
		status.BlockHeight = uint64(txResult.Height)
		t, err := time.Parse(TimeFormat, txResult.Timestamp)
		if err == nil {
			status.BlockTime = uint64(t.Unix())
		}
		return
	}
	return
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
		client := resty.New()

		resp, err := client.R().Get(fmt.Sprintf("%vblocks/latest", endpoint))
		if err != nil || resp.StatusCode() != 200 {
			log.Warn("cosmos rest request error", "request error", err, "func", "GetLatestBlockNumber")
			continue
		}
		var blockRes ctypes.ResultBlock
		err = CDC.UnmarshalJSON(resp.Body(), &blockRes)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetLatestBlockNumber", "resp", string(resp.Body()))
			continue
		}
		height = uint64(blockRes.Block.Header.Height)
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
	client := resty.New()

	resp, err := client.R().Get(fmt.Sprintf("%vblocks/latest", endpoint))
	if err != nil || resp.StatusCode() != 200 {
		log.Warn("cosmos rest request error", "request error", err, "func", "GetLatestBlockNumberOf")
		return 0, err
	}
	var blockRes ctypes.ResultBlock
	err = CDC.UnmarshalJSON(resp.Body(), &blockRes)
	if err != nil {
		log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetLatestBlockNumberOf", "resp", string(resp.Body()))
		return 0, err
	}
	height := uint64(blockRes.Block.Header.Height)
	return height, nil
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
		client := resty.New()

		resp, err := client.R().Get(fmt.Sprintf("%vauth/accounts/%v", endpoint, address))
		if err != nil || resp.StatusCode() != 200 {
			log.Warn("cosmos rest request error", "request error", err, "func", "GetAccountNumber")
			continue
		}
		var accountRes authtypes.BaseAccount
		err = CDC.UnmarshalJSON(resp.Body(), &accountRes)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetAccountNumber")
			continue
		}
		accountNumber := accountRes.AccountNumber
		return accountNumber, nil
	}
	return 0, nil
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
		client := resty.New()

		resp, err := client.R().Get(fmt.Sprintf("%vauth/accounts/%v", endpoint, address))
		if err != nil || resp.StatusCode() != 200 {
			log.Warn("cosmos rest request error", "request error", err, "func", "GetPoolNonce")
			continue
		}
		var accountRes authtypes.BaseAccount
		err = CDC.UnmarshalJSON(resp.Body(), &accountRes)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetPoolNonce")
			continue
		}
		seq := accountRes.Sequence
		return seq, nil
	}
	return 0, nil
}

// SearchTxsHash searches tx in range of blocks
// call rest api "/txs?..."
func (b *Bridge) SearchTxsHash(start, end *big.Int) ([]string, error) {
	txs := make([]string, 0)
	var limit = 100
	var page = 0
	var pageTotal = 1
	endpoints := b.GatewayConfig.APIAddress

	// search send
	for page < pageTotal {
		for _, endpoint := range endpoints {
			endpointURL, err := url.Parse(endpoint)
			if err != nil {
				continue
			}
			endpoint = endpointURL.String()
			client := resty.New()
			params := fmt.Sprintf("?message.action=send&page=%v&limit=%v&tx.minheight=%v&tx.maxheight=%v", page, limit, start, end)
			resp, err := client.R().Get(fmt.Sprintf("%vtxs%v", endpoint, params))
			if err != nil || resp.StatusCode() != 200 {
				log.Warn("cosmos rest request error", "request error", err, "func", "SearchTxsHash")
				continue
			}
			var res sdk.SearchTxsResult
			err = CDC.UnmarshalJSON(resp.Body(), &res)
			if err != nil {
				log.Warn("Search txs unmarshal error", "start", start, "end", end, "page", page, "func", "SearchTxsHash")
				continue
			}
			pageTotal = res.PageTotal
			for _, tx := range res.Txs {
				if tx.Code != 0 {
					continue
				}
				txs = append(txs, tx.TxHash)
			}
			break
		}
		page = page + 1
	}

	// search multisend
	page = 0
	pageTotal = 1
	for page < pageTotal {
		for _, endpoint := range endpoints {
			endpointURL, err := url.Parse(endpoint)
			if err != nil {
				continue
			}
			endpoint = endpointURL.String()
			client := resty.New()
			params := fmt.Sprintf("?message.action=send&page=%v&limit=%v&tx.minheight=%v&tx.maxheight=%v", page, limit, start, end)
			resp, err := client.R().Get(fmt.Sprintf("%vtxs/%v", endpoint, params))
			if err != nil || resp.StatusCode() != 200 {
				log.Warn("cosmos rest request error", "request error", err, "func", "SearchTxsHash")
				continue
			}
			var res sdk.SearchTxsResult
			err = CDC.UnmarshalJSON(resp.Body(), &res)
			if err != nil {
				log.Warn("Search txs unmarshal error", "start", start, "end", end, "page", page, "func", "SearchTxsHash")
				continue
			}
			pageTotal = res.PageTotal
			for _, tx := range res.Txs {
				if tx.Code != 0 {
					continue
				}
				txs = append(txs, tx.TxHash)
			}
			break
		}
		page = page + 1
	}
	return txs, nil
}

// SearchTxs searches tx in range of blocks
// call rest api "/txs?..."
func (b *Bridge) SearchTxs(start, end *big.Int) ([]sdk.TxResponse, error) {
	txs := make([]sdk.TxResponse, 0)
	var limit = 100

	// search send
	var page = 0
	var pageTotal = 1
	endpoints := b.GatewayConfig.APIAddress
	for page < pageTotal {
		for _, endpoint := range endpoints {
			endpointURL, err := url.Parse(endpoint)
			if err != nil {
				continue
			}
			endpoint = endpointURL.String()
			client := resty.New()
			params := fmt.Sprintf("?message.action=send&page=%v&limit=%v&tx.minheight=%v&tx.maxheight=%v", page, limit, start, end)
			resp, err := client.R().Get(fmt.Sprintf("%vtxs/%v", endpoint, params))
			if err != nil || resp.StatusCode() != 200 {
				log.Warn("cosmos rest request error", "request error", err, "func", "SearchTxs")
				continue
			}
			var res sdk.SearchTxsResult
			err = CDC.UnmarshalJSON(resp.Body(), &res)
			if err != nil {
				log.Warn("Search txs unmarshal error", "start", start, "end", end, "page", page, "func", "SearchTxs")
				continue
			}
			pageTotal = res.PageTotal
			for _, txresp := range res.Txs {
				if txresp.Code != 0 {
					continue
				}
				txs = append(txs, txresp)
			}
			break
		}
		page = page + 1
	}

	// search multisend
	page = 0
	pageTotal = 1
	for page < pageTotal {
		for _, endpoint := range endpoints {
			endpointURL, err := url.Parse(endpoint)
			if err != nil {
				continue
			}
			endpoint = endpointURL.String()
			client := resty.New()
			params := fmt.Sprintf("?message.action=multisend&page=%v&limit=%v&tx.minheight=%v&tx.maxheight=%v", page, limit, start, end)
			resp, err := client.R().Get(fmt.Sprintf("%v/txs/%v", endpoint, params))
			if err != nil || resp.StatusCode() != 200 {
				log.Warn("cosmos rest request error", "request error", err, "func", "SearchTxs")
				continue
			}
			var res sdk.SearchTxsResult
			err = CDC.UnmarshalJSON(resp.Body(), &res)
			if err != nil {
				log.Warn("Search txs unmarshal error", "start", start, "end", end, "page", page, "func", "SearchTxs")
				continue
			}
			pageTotal = res.PageTotal
			for _, txresp := range res.Txs {
				if txresp.Code != 0 {
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
func (b *Bridge) BroadcastTx(tx authtypes.StdTx) error {
	bz, err := json.Marshal(tx)
	if err != nil {
		return err
	}
	data := fmt.Sprintf(`{"tx":%v,"mode":"block"}`, string(bz))

	endpoints := b.GatewayConfig.APIAddress
	for _, endpoint := range endpoints {
		endpointURL, err := url.Parse(endpoint)
		if err != nil {
			continue
		}
		endpoint = endpointURL.String()
		client := resty.New()
		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(data).
			Post(endpoint)
		if err != nil || resp.StatusCode() != 200 {
			log.Warn("cosmos rest request error", "request error", err, "func", "BroadcastTx")
			continue
		}
		var res ctypes.ResultBroadcastTxCommit
		err = CDC.UnmarshalJSON(resp.Body(), res)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err, "func", "BroadcastTx")
			continue
		}
		log.Debug("Send tx success", "res", res)
	}
	return nil
}
