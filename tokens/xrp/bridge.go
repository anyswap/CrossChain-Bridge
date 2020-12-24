package xrp

import (
	"encoding/json"
	"io/ioutil"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

// Bridge block bridge inherit from btc bridge
type Bridge struct {
	*tokens.CrossChainBridgeBase
}

var pairID = "xrp"

// NewCrossChainBridge new fsn bridge
func NewCrossChainBridge(isSrc bool) *Bridge {
	return nil
}

// SetChainAndGateway set chain and gateway config
func (b *Bridge) SetChainAndGateway(chainCfg *tokens.ChainConfig, gatewayCfg *tokens.GatewayConfig) {
	b.CrossChainBridgeBase.SetChainAndGateway(chainCfg, gatewayCfg)
	b.VerifyChainConfig()
	b.InitLatestBlockNumber()
}

// VerifyChainConfig verify chain config
func (b *Bridge) VerifyChainConfig() {
	chainCfg := b.ChainConfig
	networkID := strings.ToLower(chainCfg.NetID)
	switch networkID {
	case "mainnet":
		return
	default:
		log.Fatal("unsupported bitcoin network", "netID", chainCfg.NetID)
	}
}

// VerifyTokenConfig verify token config
func (b *Bridge) VerifyTokenConfig(tokenCfg *tokens.TokenConfig) error {
	return nil
}

// InitLatestBlockNumber init latest block number
func (b *Bridge) InitLatestBlockNumber() {
	chainCfg := b.ChainConfig
	gatewayCfg := b.GatewayConfig
	var latest uint64
	var err error
	for {
		latest, err = b.GetLatestBlockNumber()
		if err == nil {
			tokens.SetLatestBlockHeight(latest, b.IsSrc)
			log.Info("get latst block number succeed.", "number", latest, "BlockChain", chainCfg.BlockChain, "NetID", chainCfg.NetID)
			break
		}
		log.Error("get latst block number failed.", "BlockChain", chainCfg.BlockChain, "NetID", chainCfg.NetID, "err", err)
		log.Println("retry query gateway", gatewayCfg.APIAddress)
		time.Sleep(3 * time.Second)
	}
}

var (
	rpcRetryTimes    = 3
	rpcRetryInterval = 1 * time.Second
)

// GetLatestBlockNumber gets latest block number
func (b *Bridge) GetLatestBlockNumber() (uint64, error) {
	return 0, nil
}

// GetTransaction impl
func (b *Bridge) GetTransaction(txHash string) (interface{}, error) {
	return nil, nil
}

// GetTransactionStatus impl
func (b *Bridge) GetTransactionStatus(txHash string) *tokens.TxStatus {
	return nil
}

//GetLatestBlockNumberOf gets latest block number from single api
func (b *Bridge) GetLatestBlockNumberOf(apiAddress string) (uint64, error) {
	return 0, nil
}

// GetBlockHash gets block hash
func (b *Bridge) GetBlockHash(num uint64) (string, error) {
	return "", nil
}

// GetBlockTxids gets glock txids
func (b *Bridge) GetBlockTxids(blk string) ([]string, error) {
	return nil, nil
}

// GetPoolTxidList gets pool txs
func (b *Bridge) GetPoolTxidList() ([]string, error) {
	return nil, nil
}

// GetBalance gets balance
func (b *Bridge) GetBalance(accountAddress string) (*big.Int, error) {
	return nil, nil
}

// GetTokenBalance not supported
func (b *Bridge) GetTokenBalance(tokenType, tokenAddress, accountAddress string) (*big.Int, error) {
	return nil, nil
}

// GetTokenSupply not supported
func (b *Bridge) GetTokenSupply(tokenType, tokenAddress string) (*big.Int, error) {
	return nil, nil
}

// GetAccount returns account
func (b *Bridge) GetAccount(address string) (acct Account, err error) {
	for i := 0; i < rpcRetryTimes; i++ {
		for _, apiAddress := range b.GetGatewayConfig().APIAddress {
			acct, err = b.getAccount(address, apiAddress)
			if err != nil {
				continue
			}
			return
		}
		time.Sleep(rpcRetryInterval)
	}
	return
}

func (b *Bridge) getAccount(address string, apiAddress string) (Account, error) {
	reader := strings.NewReader("{\"method\":\"account_info\",\"params\":[{\"account\":\"" + address + "\"}]}")
	request, err := http.NewRequest("POST", apiAddress, reader)
	if err != nil {
		return Account{}, err
	}
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return Account{}, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	acctResp := new(AccountResp)
	err = json.Unmarshal(body, acctResp)
	if err != nil {
		return Account{}, err
	}
	return acctResp.Result.Account_data, nil
}
