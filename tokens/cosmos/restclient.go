package cosmos

import (
	"math/big"
	"net/url"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/go-resty/resty/v2"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
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
		if err != nil || resp.StatusCode != 200 {
			getBalanceError = fmt.Errorf("Cannot connect to resp endpoint")
			continue
		}
		var balances sdk.Coins
		err = json.Unmarshal(resp.Body, &balances)
		if err != nil {
			getBalanceError = fmt.Errorf("Unmarshal balance responce error")
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

func (b *Bridge) GetTokenBalance(tokenType, tokenName, accountAddress string) (*big.Int, error) {
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
		resp, err := client.R().Get(fmt.Sprintf("%vbank/balances/%v", endpoint, account))
		if err != nil || resp.StatusCode != 200 {
			getBalanceError = fmt.Errorf("Cannot connect to resp endpoint")
			continue
		}
		var balances sdk.Coins
		err = json.Unmarshal(resp.Body, &balances)
		if err != nil {
			getBalanceError = fmt.Errorf("Unmarshal balance responce error")
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
		if err != nil || resp.StatusCode != 200 {
			getTxError = fmt.Errorf("Cannot connect to resp endpoint")
			continue
		}
		var txRes sdk.TxResponse
		err = UnmarshalJSON(ret, &txRes)
		if err != nil {
			getTxError = fmt.Errorf("Unmarshal tx response error")
			continue
		}
		tx = txRes.GetTx()
		return tx, nil
	}
	return
}

func (b *Bridge) GetTransactionStatus(txHash string) (status *tokens.TxStatus) {
	endpoints := b.GatewayConfig.APIAddress
	for _, endpoint := range endpoints {
		endpointURL, err := url.Parse(endpoint)
		if err != nil {
			continue
		}
		endpoint = endpointURL.String()
		client := resty.New()

		resp, err := client.R().Get(fmt.Sprintf("%vtxs/%v", endpoint, txHash))
		if err != nil || resp.StatusCode != 200 {
			continue
		}
		var txRes sdk.TxResponse
		err = UnmarshalJSON(ret, &txRes)
		if err != nil {
			continue
		}
		tx = txRes.GetTx()
		status = &tokens.TxStatus{
			Receipt: txRes.Logs,
			//Confirmations
			BlockHeight: txRes.Height,
			//BlockHash
			BlockTime: Timestamp,
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
		if err != nil || resp.StatusCode != 200 {
			getLatestError = fmt.Errorf("Cannot connect to resp endpoint")
			continue
		}
		var blockRes ctypes.ResultBlock
		err = UnmarshalJSON(ret, &blockRes)
		if err != nil {
			getLatestError = fmt.Errorf("Unmarshal block response error")
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
		continue
	}
	endpoint = endpointURL.String()
	client := resty.New()

	resp, err := client.R().Get(fmt.Sprintf("%vblocks/latest", endpoint))
	if err != nil || resp.StatusCode != 200 {
		getLatestError = fmt.Errorf("Cannot connect to resp endpoint")
		continue
	}
	var blockRes ctypes.ResultBlock
	err = UnmarshalJSON(ret, &blockRes)
	if err != nil {
		getLatestError = fmt.Errorf("Unmarshal block response error")
		continue
	}
	height = uint64(blockRes.Block.Header.Height)
	return
}

func (b *Bridge) GetAccountNumber(address string) (uint64, error) {
	// TODO
	return 0, nil
}

func (b *Bridge) GetPoolNonce(address, height string) (uint64, error) {
	// TODO
	return 0, nil
}

// GetBlockByNumber
func (b *Bridge) GetBlockByNumber(number *big.Int) (ctypes.ResultBlock, error) {
	// TODO
	return ctypes.ResultBlock{}, nil
}
