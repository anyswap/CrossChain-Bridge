package ripple

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/data"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/websockets"
)

// Bridge block bridge inherit from btc bridge
type Bridge struct {
	*tokens.CrossChainBridgeBase
	*NonceSetterBase
	Remotes map[string]*websockets.Remote
}

// NewCrossChainBridge new bridge
func NewCrossChainBridge(isSrc bool) *Bridge {
	tokens.IsSwapoutToStringAddress = true
	return &Bridge{
		CrossChainBridgeBase: tokens.NewCrossChainBridgeBase(isSrc),
		Remotes:              make(map[string]*websockets.Remote),
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
	log.Info("XRP init remotes")
	for _, r := range b.Remotes {
		if r != nil {
			r.Close()
		}
	}
	b.Remotes = make(map[string]*websockets.Remote)
	for _, apiAddress := range b.GetGatewayConfig().APIAddress {
		remote, err := websockets.NewRemote(apiAddress)
		if err != nil || remote == nil {
			log.Warn("Cannot connect to ripple", "address", apiAddress, "error", err)
			continue
		}
		log.Info("Connected to remote api", "", apiAddress)
		b.Remotes[apiAddress] = remote
		remote.Close()
	}
	if len(b.Remotes) < 1 {
		log.Error("No available remote api")
	}
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
	case "devnet":
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
		for url := range b.Remotes {
			r, err0 := websockets.NewRemote(url)
			if err0 != nil {
				log.Warn("Cannot connect to remote", "error", err)
				continue
			}
			resp, err1 := r.Ledger(nil, false)
			if err1 != nil || resp == nil {
				err = err1
				log.Warn("Try get latest block number failed", "error", err1)
				r.Close()
				continue
			}
			num = uint64(resp.Ledger.LedgerSequence)
			r.Close()
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
		log.Warn("Cannot connect to remote", "error", err)
		return 0, err
	}
	defer r.Close()
	resp, err := r.Ledger(nil, false)
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
		for url := range b.Remotes {
			r, err0 := websockets.NewRemote(url)
			if err0 != nil {
				log.Warn("Cannot connect to remote", "error", err)
				continue
			}
			resp, err1 := r.Tx(*txhash256)
			/*randNum := rand.New(rand.NewSource(time.Now().Unix())).Int()
			if randNum%3 == 0 {
				err1 = fmt.Errorf("Simulate error: Connection closed")
			}*/
			if err1 != nil || resp == nil {
				log.Warn("Try get transaction failed", "error", err1)
				err = err1
				r.Close()
				continue
			}
			tx = resp
			r.Close()
			return
		}
		time.Sleep(rpcRetryInterval)
	}
	return
}

var (
	ErrTxResultType = errors.New("tx type is not data.TxResult")
	ErrTxNotSuccess = func(txres interface{}) error { return fmt.Errorf("ripple tx status is not success: %v", txres) }
)

// GetTransactionStatus impl
func (b *Bridge) GetTransactionStatus(txHash string) (status *tokens.TxStatus, err error) {
	status = new(tokens.TxStatus)
	tx, err := b.GetTransaction(txHash)
	if err != nil {
		return nil, err
	}

	txres, ok := tx.(*websockets.TxResult)
	if !ok {
		// unexpected
		log.Warn("Ripple GetTransactionStatus", "error", ErrTxResultType)
		return nil, ErrTxResultType
	}

	// Check tx status
	if txres.TransactionWithMetaData.MetaData.TransactionResult != 0 {
		log.Warn("Ripple tx status is not success", "result", txres.TransactionWithMetaData.MetaData.TransactionResult)
		return nil, ErrTxNotSuccess(txres.TransactionWithMetaData.MetaData.TransactionResult)
	}

	status.Receipt = nil
	inledger := txres.LedgerSequence
	status.BlockHeight = uint64(inledger)

	if latest, err := b.GetLatestBlockNumber(); err == nil {
		status.Confirmations = latest - uint64(inledger)
	}
	return
}

// GetBlockHash gets block hash
func (b *Bridge) GetBlockHash(num uint64) (hash string, err error) {
	for i := 0; i < rpcRetryTimes; i++ {
		for url := range b.Remotes {
			r, err0 := websockets.NewRemote(url)
			if err0 != nil {
				log.Warn("Cannot connect to remote", "error", err)
				continue
			}
			resp, err1 := r.Ledger(num, false)
			if err1 != nil || resp == nil {
				err = err1
				log.Warn("Try get block hash failed", "error", err1)
				r.Close()
				continue
			}
			hash = resp.Ledger.Hash.String()
			r.Close()
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
		for url := range b.Remotes {
			r, err0 := websockets.NewRemote(url)
			if err0 != nil {
				log.Warn("Cannot connect to remote", "error", err)
				continue
			}
			resp, err1 := r.Ledger(num, true)
			if err1 != nil || resp == nil {
				err = err1
				log.Warn("Try get block tx ids failed", "error", err1)
				r.Close()
				continue
			}
			r.Close()
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
		for url := range b.Remotes {
			r, err0 := websockets.NewRemote(url)
			if err0 != nil {
				log.Warn("Cannot connect to remote", "error", err)
				continue
			}
			acct, err = r.AccountInfo(*account)
			if err != nil || acct == nil {
				r.Close()
				continue
			}
			r.Close()
			return
		}
		time.Sleep(rpcRetryInterval)
	}
	return
}
