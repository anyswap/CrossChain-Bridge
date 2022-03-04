package ripple

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/base"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/data"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/websockets"
)

var (
	// ensure Bridge impl tokens.CrossChainBridge
	_ tokens.CrossChainBridge = &Bridge{}
	// ensure Bridge impl tokens.NonceSetter
	_ tokens.NonceSetter = &Bridge{}

	currencyMap = make(map[string]data.Currency)
	issuerMap   = make(map[string]*data.Account)
)

// Bridge block bridge inherit from btc bridge
type Bridge struct {
	*base.NonceSetterBase
	Remotes map[string]*websockets.Remote
}

// NewCrossChainBridge new bridge
func NewCrossChainBridge(isSrc bool) *Bridge {
	tokens.IsSwapoutToStringAddress = true
	if !isSrc {
		log.Fatalf("ripple::NewCrossChainBridge error %v", tokens.ErrBridgeDestinationNotSupported)
	}
	return &Bridge{
		NonceSetterBase: base.NewNonceSetterBase(isSrc),
		Remotes:         make(map[string]*websockets.Remote),
	}
}

// SetChainAndGateway set chain and gateway config
func (b *Bridge) SetChainAndGateway(chainCfg *tokens.ChainConfig, gatewayCfg *tokens.GatewayConfig) {
	b.CrossChainBridgeBase.SetChainAndGateway(chainCfg, gatewayCfg)
	b.InitRemotes()
	b.VerifyChainConfig()
	b.InitLatestBlockNumber()
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
	if tokenCfg.RippleExtra == nil {
		return fmt.Errorf("must config 'RippleExtra'")
	}
	currency, err := data.NewCurrency(tokenCfg.RippleExtra.Currency)
	if err != nil {
		return fmt.Errorf("invalid currency '%v', %w", tokenCfg.RippleExtra.Currency, err)
	}
	currencyMap[tokenCfg.RippleExtra.Currency] = currency
	configedDecimals := *tokenCfg.Decimals
	if currency.IsNative() {
		if configedDecimals != 6 {
			return fmt.Errorf("invalid native decimals: want 6 but have %v", configedDecimals)
		}
		if tokenCfg.RippleExtra.Issuer != "" {
			return fmt.Errorf("must config empty 'RippleExtra.Issuer' for native")
		}
	} else {
		if tokenCfg.RippleExtra.Issuer == "" {
			return fmt.Errorf("must config 'RippleExtra.Issuer' for non native")
		}
		issuer, errf := data.NewAccountFromAddress(tokenCfg.RippleExtra.Issuer)
		if errf != nil {
			return fmt.Errorf("invalid Issuer '%v', %w", tokenCfg.RippleExtra.Issuer, errf)
		}
		issuerMap[tokenCfg.RippleExtra.Issuer] = issuer
	}
	if !b.IsValidAddress(tokenCfg.DcrmAddress) {
		return fmt.Errorf("invalid 'DcrmAddress' in token '%v' config", currency)
	}
	if b.IsSrc &&
		tokenCfg.DepositAddress != tokenCfg.DcrmAddress &&
		!b.IsValidAddress(tokenCfg.DepositAddress) {
		return fmt.Errorf("invalid 'DepositAddress' in token '%v' config", currency)
	}
	pubAddr, err := PublicKeyHexToAddress(tokenCfg.DcrmPubkey)
	if err != nil {
		return err
	}
	if pubAddr != tokenCfg.DcrmAddress {
		return fmt.Errorf("mismatch dcrm public key and address in token '%v' config, pubkey address is '%v', dcrm addrss is '%v'", currency, pubAddr, tokenCfg.DcrmAddress)
	}
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

// SetRPCRetryTimes set rpc retry times (used in cmd tools)
func SetRPCRetryTimes(times int) {
	rpcRetryTimes = times
}

// GetLatestBlockNumber gets latest block number
// For ripple, GetLatestBlockNumber returns current ledger version
func (b *Bridge) GetLatestBlockNumber() (num uint64, err error) {
	for i := 0; i < rpcRetryTimes; i++ {
		for _, r := range b.Remotes {
			resp, err1 := r.Ledger(nil, false)
			if err1 != nil || resp == nil {
				err = err1
				log.Warn("Try get latest block number failed", "error", err1)
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
	var err error
	r, exist := b.Remotes[apiAddress]
	if !exist {
		r, err = websockets.NewRemote(apiAddress)
		if err != nil {
			log.Warn("Cannot connect to remote", "error", err)
			return 0, err
		}
		defer r.Close()
	}
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
		for _, r := range b.Remotes {
			resp, err1 := r.Tx(*txhash256)
			if err1 != nil || resp == nil {
				log.Warn("Try get transaction failed", "error", err1)
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
func (b *Bridge) GetTransactionStatus(txHash string) (status *tokens.TxStatus, err error) {
	status = new(tokens.TxStatus)
	tx, err := b.GetTransaction(txHash)
	if err != nil {
		return nil, err
	}

	txres, ok := tx.(*websockets.TxResult)
	if !ok {
		// unexpected
		log.Warn("Ripple GetTransactionStatus", "error", errTxResultType)
		return nil, errTxResultType
	}

	// Check tx status
	if !txres.TransactionWithMetaData.MetaData.TransactionResult.Success() {
		log.Warn("Ripple tx status is not success", "result", txres.TransactionWithMetaData.MetaData.TransactionResult)
		return nil, tokens.ErrTxWithWrongStatus
	}

	status.Receipt = nil
	inledger := txres.LedgerSequence
	status.BlockHeight = uint64(inledger)

	if latest, err := b.GetLatestBlockNumber(); err == nil && latest > uint64(inledger) {
		status.Confirmations = latest - uint64(inledger)
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
				log.Warn("Try get block hash failed", "error", err1)
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
				log.Warn("Try get block tx ids failed", "error", err1)
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
	if err != nil || acct == nil {
		log.Warn("get balance failed", "account", accountAddress, "err", err)
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
				continue
			}
			return
		}
		time.Sleep(rpcRetryInterval)
	}
	return
}

// GetAccountLine get account line
func (b *Bridge) GetAccountLine(currency, issuer, accountAddress string) (*data.AccountLine, error) {
	account, err := data.NewAccountFromAddress(accountAddress)
	if err != nil {
		return nil, err
	}
	var acclRes *websockets.AccountLinesResult
	for i := 0; i < rpcRetryTimes; i++ {
		for _, r := range b.Remotes {
			acclRes, err = r.AccountLines(*account, nil)
			if err == nil && acclRes != nil {
				break
			}
		}
		time.Sleep(rpcRetryInterval)
	}
	if err != nil {
		return nil, err
	}
	for _, accl := range acclRes.Lines {
		asset := accl.Asset()
		if asset.Currency == currency && asset.Issuer == issuer {
			return &accl, nil
		}
	}
	return nil, fmt.Errorf("account line not found")
}
