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
	ctypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	cdc       = authtypes.ModuleCdc
	txDecoder = authtypes.DefaultTxDecoder(cdc)
	txEncoder = authtypes.DefaultTxEncoder(cdc)
)

func (b *Bridge) GetBalance(account string) (balance *big.Int, getBalanceError error) {
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
			log.Warn("cosmos rest request error", "request error", err)
			continue
		}
		var balances sdk.Coins
		err = json.Unmarshal(resp.Body(), &balances)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", getBalanceError)
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

func (b *Bridge) GetTokenBalance(tokenType, tokenName, accountAddress string) (balance *big.Int, getBalanceError error) {
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
		resp, err := client.R().Get(fmt.Sprintf("%vbank/balances/%v", endpoint, accountAddress))
		if err != nil || resp.StatusCode() != 200 {
			log.Warn("cosmos rest request error", "error", err)
			continue
		}
		var balances sdk.Coins
		err = json.Unmarshal(resp.Body(), &balances)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", getBalanceError)
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

func (b *Bridge) GetTransaction(txHash string) (tx interface{}, getTxError error) {
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
			log.Warn("cosmos rest request error", "request error", err)
			continue
		}
		var txResult sdk.TxResponse
		err = json.Unmarshal(resp.Body(), &txResult)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err)
			return nil, err
		}
		tx = txResult.Tx
		err = tx.(sdk.Tx).ValidateBasic()
		if err != nil {
			return nil, err
		}
		return
	}
	return
}

const TimeFormat = time.RFC3339Nano

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
			log.Warn("cosmos rest request error", "request error", err)
			continue
		}

		var txResult sdk.TxResponse
		err = json.Unmarshal(resp.Body(), &txResult)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err)
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

func (b *Bridge) GetLatestBlockNumber() (height uint64, getLatestError error) {
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
			log.Warn("cosmos rest request error", "request error", err)
			continue
		}
		var blockRes ctypes.ResultBlock
		err = json.Unmarshal(resp.Body(), &blockRes)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err)
			continue
		}
		height = uint64(blockRes.Block.Header.Height)
		return
	}
	return
}

func (b *Bridge) GetLatestBlockNumberOf(apiAddress string) (uint64, error) {
	endpointURL, err := url.Parse(apiAddress)
	if err != nil {
		return 0, err
	}
	endpoint := endpointURL.String()
	client := resty.New()

	resp, err := client.R().Get(fmt.Sprintf("%vblocks/latest", endpoint))
	if err != nil || resp.StatusCode() != 200 {
		getLatestError := fmt.Errorf("Cannot connect to resp endpoint")
		log.Warn("cosmos rest request error", "request error", getLatestError)
		return 0, getLatestError
	}
	var blockRes ctypes.ResultBlock
	err = json.Unmarshal(resp.Body(), &blockRes)
	if err != nil {
		log.Warn("cosmos rest request error", "unmarshal error", err)
		return 0, err
	}
	height := uint64(blockRes.Block.Header.Height)
	return height, nil
}

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
			log.Warn("cosmos rest request error", "request error", err)
			continue
		}
		var accountRes authtypes.BaseAccount
		err = json.Unmarshal(resp.Body(), &accountRes)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err)
			continue
		}
		accountNumber := accountRes.AccountNumber
		return accountNumber, nil
	}
	return 0, nil
}

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
			log.Warn("cosmos rest request error", "request error", err)
			continue
		}
		var accountRes authtypes.BaseAccount
		err = json.Unmarshal(resp.Body(), &accountRes)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err)
			continue
		}
		seq := accountRes.Sequence
		return seq, nil
	}
	return 0, nil
}

// SearchTxs searches tx in range of blocks
func (b *Bridge) SearchTxs(start, end *big.Int) ([]string, error) {
	txs := make([]string, 0)
	var limit = 100
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
				log.Warn("cosmos rest request error", "request error", err)
				continue
			}
			var res sdk.SearchTxsResult
			err = json.Unmarshal(resp.Body(), &res)
			if err != nil {
				log.Warn("Search txs unmarshal error", "start", start, "end", end, "page", page)
				continue
			}
			pageTotal = res.PageTotal
			for _, tx := range res.Txs {
				txs = append(txs, tx.TxHash)
			}
			break
		}
		page = page + 1
	}
	return txs, nil
}

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
			log.Warn("cosmos rest request error", "request error", err)
			continue
		}
		var res ctypes.ResultBroadcastTxCommit
		err = json.Unmarshal(resp.Body(), res)
		if err != nil {
			log.Warn("cosmos rest request error", "unmarshal error", err)
			continue
		}
		log.Debug("Send tx success", "res", res)
	}
	return nil
}
