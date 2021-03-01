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

type Version int

var (
	Rest3 Version = 3
	Rest4 Version = 4
)

var RestVersion Version = 4

func (b *Bridge) GetBalance(account string) (balance *big.Int, err error) {
	switch RestVersion {
	case Rest3:
		return b.getBalance3(account)
	case Rest4:
		return b.getBalance4(account)
	default:
		return b.getBalance3(account)
	}
	return nil, nil
}

func (b *Bridge) getBalance3(account string) (balance *big.Int, err error) {
	endpoints := b.GatewayConfig.APIAddress
	for _, endpoint := range endpoints {
		endpointURL, err := url.Parse(endpoint)
		if err != nil {
			continue
		}
		endpoint = endpointURL.String()
		client := resty.New()
		resp, err := client.R().Get(fmt.Sprintf("%v/bank/balances/%v", endpoint, account))
		if err != nil || resp.StatusCode() != 200 {
			log.Warn("cosmos rest request error", "request error", err, "func", "GetBalance")
			continue
		}
		var balances sdk.Coins
		err = CDC.UnmarshalJSON(resp.Body(), &balances)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetBalance")
			continue
		}
		for _, bal := range balances {
			if bal.Denom != TheCoin.Denom {
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

func (b *Bridge) GetTokenBalance(tokenType, tokenName, accountAddress string) (balance *big.Int, err error) {
	switch RestVersion {
	case Rest3:
		return b.getTokenBalance3(tokenType, tokenName, accountAddress)
	case Rest4:
		return b.getTokenBalance4(tokenType, tokenName, accountAddress)
	default:
		return b.getTokenBalance3(tokenType, tokenName, accountAddress)
	}
	return nil, nil
}

func (b *Bridge) getTokenBalance3(tokenType, tokenName, accountAddress string) (balance *big.Int, err error) {
	coin, ok := SupportedCoins[tokenName]
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
		resp, err := client.R().Get(fmt.Sprintf("%v/bank/balances/%v", endpoint, accountAddress))
		if err != nil || resp.StatusCode() != 200 {
			log.Warn("cosmos rest request error", "error", err, "func", "GetTokenBalance")
			continue
		}
		var balances sdk.Coins
		err = CDC.UnmarshalJSON(resp.Body(), &balances)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetTokenBalance")
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

func (b *Bridge) GetTokenSupply(tokenType, tokenAddress string) (*big.Int, error) {
	return nil, fmt.Errorf("Cosmos bridges does not support this method")
}

func (b *Bridge) GetTransaction(txHash string) (tx interface{}, err error) {
	switch RestVersion {
	case Rest3:
		return b.getTransaction3(txHash)
	case Rest4:
		return b.getTransaction4(txHash)
	default:
		return b.getTransaction3(txHash)
	}
	return nil, nil
}

func (b *Bridge) getTransaction3(txHash string) (tx interface{}, err error) {
	endpoints := b.GatewayConfig.APIAddress
	for _, endpoint := range endpoints {
		endpointURL, err := url.Parse(endpoint)
		if err != nil {
			continue
		}
		endpoint = endpointURL.String()
		client := resty.New()

		resp, err := client.R().Get(fmt.Sprintf("%v/txs/%v", endpoint, txHash))
		if err != nil || resp.StatusCode() != 200 {
			log.Warn("cosmos rest request error", "request error", err, "func", "GetTransaction")
			continue
		}
		var txResult sdk.TxResponse
		err = CDC.UnmarshalJSON(resp.Body(), &txResult)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetTransaction")
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

const TimeFormat = time.RFC3339Nano

func (b *Bridge) GetTransactionStatus(txHash string) (status *tokens.TxStatus) {
	switch RestVersion {
	case Rest3:
		return b.getTransactionStatus3(txHash)
	case Rest4:
		return b.getTransactionStatus4(txHash)
	default:
		return b.getTransactionStatus3(txHash)
	}
	return nil
}

func (b *Bridge) getTransactionStatus3(txHash string) (status *tokens.TxStatus) {
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

		resp, err := client.R().Get(fmt.Sprintf("%v/txs/%v", endpoint, txHash))
		if err != nil || resp.StatusCode() != 200 {
			log.Warn("cosmos rest request error", "request error", err, "func", "GetTransactionStatus")
			continue
		}

		var txResult sdk.TxResponse
		err = CDC.UnmarshalJSON(resp.Body(), &txResult)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetTransactionStatus")
			return
		}
		tx := txResult.Tx
		err = tx.ValidateBasic()
		if err != nil {
			return
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

func (b *Bridge) GetLatestBlockNumber() (height uint64, err error) {
	switch RestVersion {
	case Rest3:
		return b.getLatestBlockNumber3()
	case Rest4:
		return b.getLatestBlockNumber4()
	default:
		return b.getLatestBlockNumber3()
	}
	return 0, nil
}

func (b *Bridge) getLatestBlockNumber3() (height uint64, err error) {
	endpoints := b.GatewayConfig.APIAddress
	for _, endpoint := range endpoints {
		endpointURL, err := url.Parse(endpoint)
		if err != nil {
			continue
		}
		endpoint = endpointURL.String()
		client := resty.New()

		resp, err := client.R().Get(fmt.Sprintf("%v/blocks/latest", endpoint))
		if err != nil || resp.StatusCode() != 200 {
			log.Warn("cosmos rest request error", "request error", err, "func", "GetLatestBlockNumber")
			continue
		}
		var blockRes ctypes.ResultBlock
		err = CDC.UnmarshalJSON(resp.Body(), &blockRes)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetLatestBlockNumber")
			fmt.Printf("\n\n%+v\n\n", string(resp.Body()))
			continue
		}
		height = uint64(blockRes.Block.Header.Height)
		return height, nil
	}
	return
}

func (b *Bridge) GetLatestBlockNumberOf(apiAddress string) (uint64, error) {
	switch RestVersion {
	case Rest3:
		return b.getLatestBlockNumberOf3(apiAddress)
	case Rest4:
		return b.getLatestBlockNumberOf4(apiAddress)
	default:
		return b.getLatestBlockNumberOf3(apiAddress)
	}
	return 0, nil
}

func (b *Bridge) getLatestBlockNumberOf3(apiAddress string) (uint64, error) {
	endpointURL, err := url.Parse(apiAddress)
	if err != nil {
		return 0, err
	}
	endpoint := endpointURL.String()
	client := resty.New()

	resp, err := client.R().Get(fmt.Sprintf("%v/blocks/latest", endpoint))
	if err != nil || resp.StatusCode() != 200 {
		log.Warn("cosmos rest request error", "request error", err, "func", "GetLatestBlockNumberOf")
		return 0, err
	}
	var blockRes ctypes.ResultBlock
	err = CDC.UnmarshalJSON(resp.Body(), &blockRes)
	if err != nil {
		log.Warn("cosmos rest request error", "unmarshal error", err, "func", "GetLatestBlockNumberOf")
		return 0, err
	}
	height := uint64(blockRes.Block.Header.Height)
	return height, nil
}

func (b *Bridge) GetAccountNumber(address string) (uint64, error) {
	switch RestVersion {
	case Rest3:
		return b.getAccountNumber3(address)
	case Rest4:
		return b.getAccountNumber4(address)
	default:
		return b.getAccountNumber3(address)
	}
	return 0, nil
}

func (b *Bridge) getAccountNumber3(address string) (uint64, error) {
	endpoints := b.GatewayConfig.APIAddress
	for _, endpoint := range endpoints {
		endpointURL, err := url.Parse(endpoint)
		if err != nil {
			continue
		}
		endpoint = endpointURL.String()
		client := resty.New()

		resp, err := client.R().Get(fmt.Sprintf("%v/auth/accounts/%v", endpoint, address))
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

func (b *Bridge) GetPoolNonce(address, height string) (uint64, error) {
	switch RestVersion {
	case Rest3:
		return b.getPoolNonce3(address, height)
	case Rest4:
		return b.getPoolNonce4(address, height)
	default:
		return b.getPoolNonce3(address, height)
	}
	return 0, nil
}

func (b *Bridge) getPoolNonce3(address, height string) (uint64, error) {
	endpoints := b.GatewayConfig.APIAddress
	for _, endpoint := range endpoints {
		endpointURL, err := url.Parse(endpoint)
		if err != nil {
			continue
		}
		endpoint = endpointURL.String()
		client := resty.New()

		resp, err := client.R().Get(fmt.Sprintf("%v/auth/accounts/%v", endpoint, address))
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
			resp, err := client.R().Get(fmt.Sprintf("%v/txs/%v", endpoint, params))
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
			resp, err := client.R().Get(fmt.Sprintf("%v/txs/%v", endpoint, params))
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
func (b *Bridge) SearchTxs(start, end *big.Int) ([]sdk.TxResponse, error) {
	switch RestVersion {
	case Rest3:
		return b.searchTxs3(start, end)
	case Rest4:
		return b.searchTxs4(start, end)
	default:
		return b.searchTxs3(start, end)
	}
	return nil, nil
}

func (b *Bridge) searchTxs3(start, end *big.Int) ([]sdk.TxResponse, error) {
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

func (b *Bridge) BroadcastTx(tx authtypes.StdTx) error {
	switch RestVersion {
	case Rest3:
		return b.broadcastTx3(tx)
	case Rest4:
		return b.broadcastTx4(tx)
	default:
		return b.broadcastTx3(tx)
	}
	return nil
}

func (b *Bridge) broadcastTx3(tx authtypes.StdTx) error {
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
