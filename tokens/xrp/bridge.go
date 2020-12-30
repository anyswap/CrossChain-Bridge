package xrp

import (
	"math/big"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/rubblelabs/ripple/data"
	"github.com/rubblelabs/ripple/websockets"
)

// Bridge block bridge inherit from btc bridge
type Bridge struct {
	*tokens.CrossChainBridgeBase
	Remotes []*websockets.Remote
}

var pairID = "xrp"

// NewCrossChainBridge new bridge
func NewCrossChainBridge(isSrc bool) *Bridge {
	tokens.IsSwapoutToStringAddress = true
	return &Bridge{
		CrossChainBridgeBase: tokens.NewCrossChainBridgeBase(isSrc),
	}
}

// SetChainAndGateway set chain and gateway config
func (b *Bridge) SetChainAndGateway(chainCfg *tokens.ChainConfig, gatewayCfg *tokens.GatewayConfig) {
	b.CrossChainBridgeBase.SetChainAndGateway(chainCfg, gatewayCfg)
	b.VerifyChainConfig()
	b.InitLatestBlockNumber()
	b.InitRemotes()
}

// InitRemotes set ripple remotes
func (b *Bridge) InitRemotes() {
	for _, r := range b.Remotes {
		r.Close()
	}
	remotes := make([]*websockets.Remote, 0)
	for _, apiAddress := range b.GetGatewayConfig().APIAddress {
		remote, err := websockets.NewRemote(apiAddress)
		if err != nil {
			log.Warn("Cannot connect to ripple", "error", err)
		}
		remotes = append(remotes, remote)
	}
	b.Remotes = remotes
}

// VerifyChainConfig verify chain config
func (b *Bridge) VerifyChainConfig() {
	chainCfg := b.ChainConfig
	networkID := strings.ToLower(chainCfg.NetID)
	switch networkID {
	case "mainnet":
		return
	case "testnet":
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
// For ripple, GetLatestBlockNumber returns current ledger version
func (b *Bridge) GetLatestBlockNumber() (num uint64, err error) {
	for i := 0; i < rpcRetryTimes; i++ {
		for _, r := range b.Remotes {
			resp, err1 := r.Ledger("validated", false)
			if err1 != nil || resp == nil {
				err = err1
				continue
			}
			num = uint64(resp.Ledger.LedgerSequence)
			return
		}
		time.Sleep(rpcRetryInterval)
	}
	return
}

//GetLatestBlockNumberOf gets latest block number from single api
// For ripple, GetLatestBlockNumberOf returns current ledger version
func (b *Bridge) GetLatestBlockNumberOf(apiAddress string) (uint64, error) {
	r, err := websockets.NewRemote(apiAddress)
	if err != nil {
		return 0, err
	}
	defer r.Close()
	resp, err := r.Ledger("validated", false)
	if err != nil || resp == nil {
		return 0, err
	}
	return uint64(resp.Ledger.LedgerSequence), nil
}

// GetTransaction impl
func (b *Bridge) GetTransaction(txHash string) (tx interface{}, err error) {
	txhash256, err := data.NewHash256(txHash)
	if err != nil {
		return
	}
	for i := 0; i < rpcRetryTimes; i++ {
		for _, r := range b.Remotes {
			resp, err1 := r.Tx(*txhash256)
			if err1 != nil || resp == nil {
				err = err1
				continue
			}
			tx = resp
			return
		}
		time.Sleep(rpcRetryInterval)
	}
	return
}

// GetTransactionStatus impl
func (b *Bridge) GetTransactionStatus(txHash string) (status *tokens.TxStatus) {
	tx, err := b.GetTransaction(txHash)
	if err != nil {
		return nil
	}
	rippleTx, ok := tx.(*websockets.TxResult)
	if !ok {
		// unexpected
		log.Warn("ripple tx type assertion error")
		return
	}
	status.Receipt = rippleTx
	inledger := rippleTx.LedgerSequence
	status.BlockHeight = uint64(inledger)
	if latest := rippleTx.Transaction.GetBase().LastLedgerSequence; latest != nil {
		status.Confirmations = uint64(*latest - inledger)
	}
	return
}

// GetBlockHash gets block hash
func (b *Bridge) GetBlockHash(num uint64) (hash string, err error) {
	for i := 0; i < rpcRetryTimes; i++ {
		for _, r := range b.Remotes {
			resp, err1 := r.Ledger(num, false)
			if err1 != nil || resp == nil {
				err = err1
				continue
			}
			hash = resp.Ledger.Hash.String()
			return
		}
		time.Sleep(rpcRetryInterval)
	}
	return
}

// GetBlockTxids gets glock txids
func (b *Bridge) GetBlockTxids(num uint64) (txs []string, err error) {
	txs = make([]string, 0)
	for i := 0; i < rpcRetryTimes; i++ {
		for _, r := range b.Remotes {
			resp, err1 := r.Ledger(num, true)
			if err1 != nil || resp == nil {
				err = err1
				continue
			}
			for _, tx := range resp.Ledger.Transactions {
				txs = append(txs, tx.GetBase().Hash.String())
			}
			return
		}
		time.Sleep(rpcRetryInterval)
	}
	return
}

// GetPoolTxidList not supported
func (b *Bridge) GetPoolTxidList() ([]string, error) {
	return nil, nil
}

// GetBalance gets balance
func (b *Bridge) GetBalance(accountAddress string) (*big.Int, error) {
	acct, err := b.GetAccount(accountAddress)
	if err != nil {
		log.Warn("Get balance failed")
		return nil, err
	}
	bal := big.NewInt(int64(acct.AccountData.Balance.Float() * 1000000))
	return bal, nil
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
func (b *Bridge) GetAccount(address string) (acct *websockets.AccountInfoResult, err error) {
	account, err := data.NewAccountFromAddress(address)
	if err != nil {
		return
	}
	for i := 0; i < rpcRetryTimes; i++ {
		for _, r := range b.Remotes {
			acct, err = r.AccountInfo(*account)
			if err != nil || acct == nil {
				log.Warn("Try getting account failed", "error", err)
				continue
			}
			return
		}
		time.Sleep(rpcRetryInterval)
	}
	return
}
