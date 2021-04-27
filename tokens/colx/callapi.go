package colx

import (
	"fmt"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
	"github.com/giangnamnabka/btcd/rpcclient"
	"github.com/giangnamnabka/btcd/chaincfg/chainhash"
)

// FullnodeClient is a fullnode client
type FullnodeClient struct {
	*rpcclient.Client
	Address string
	Closer  func()
}

type Client struct {
	FClients         []FullnodeClient
}

// GetClient returns new Client
func (b *Bridge) GetClient() *Client {
	cfg := b.GetGatewayConfig()
	if cfg.Extras == nil || cfg.Extras.ColxExtra == nil {
		return nil
	}

	clis := make([]FullnodeClient, 0)
	for _, args := range cfg.Extras.ColxExtra.FullnodeAPIs {
		connCfg := &rpcclient.ConnConfig{
			Host:         args.APIAddress,
			User:         args.RPCUser,
			Pass:         args.RPCPassword,
			HTTPPostMode: args.HTTPPostMode,            // Bitcoin core only supports HTTP POST mode
			DisableTLS:   args.DisableTLS, // Bitcoin core does not provide TLS by default
		}

		client, err := rpcclient.New(connCfg, nil)
		if err != nil {
			continue
		}

		cli := FullnodeClient{
			Client:  client,
			Address: connCfg.Host,
			Closer:  client.Shutdown,
		}
		clis = append(clis, cli)
	}

	return &Client{
		FClients:         cclis,
	}
}

// GetLatestBlockNumberOf impl
func (b *Bridge) GetLatestBlockNumberOf(apiAddress string) (uint64, error) {
	cli := b.GetClient()
	for _, ccli := range cli.FClients {
		if ccli.Address == apiAddress {
			number, err := ccli.GetBlockCount()
			ccli.Closer()
			return uint64(number), err
		}
		ccli.Closer()
	}
	return 0, nil
}

// GetLatestBlockNumber impl
func (b *Bridge) GetLatestBlockNumber() (blocknumber uint64, err error) {
	cli := b.GetClient()
	errs := make([]error, 0)
	for _, ccli := range cli.FClients {
		number, err0 := ccli.GetBlockCount()
		if err0 == nil {
			ccli.Closer()
			return uint64(number), nil
		}
		errs = append(errs, err0)
		ccli.Closer()
	}
	err = fmt.Errorf("%+v", errs)
	return
}

// GetTransactionByHash impl
func (b *Bridge) GetTransactionByHash(txHash string) (*electrs.ElectTx, error) {
	cli := b.GetClient()
	errs := make([]error, 0)
	hash, err := chainhash.NewHashFromStr(txHash)
	if err != nil {
		return
	}
	for _, ccli := range cli.FClients {
		tx, err0 := ccli.GetRawTransactionVerbose(hash)
		if err0 == nil {
			ccli.Closer()
			etx = ConvertTx(tx)
			return
		}
		errs = append(errs, err0)
		ccli.Closer()
	}
	err = fmt.Errorf("%+v", errs)
	return
}

// GetElectTransactionStatus impl
func (b *Bridge) GetElectTransactionStatus(txHash string) (txstatus *electrs.ElectTxStatus, err error) {
	cli := b.GetClient()
	errs := make([]error, 0)
	hash, err := chainhash.NewHashFromStr(txHash)
	if err != nil {
		return
	}
	for _, ccli := range cli.FClients {
		txraw, err0 := ccli.GetRawTransactionVerbose(hash)
		if err0 == nil {
			ccli.Closer()
			txstatus = TxStatus(txraw)
			if h := txstatus.BlockHash; h != nil {
				if blk, err1 := b.GetBlock(*h); err1 == nil {
					*txstatus.BlockHeight = uint64(*blk.Height)
				}
			}
			return
		}
		errs = append(errs, err0)
		ccli.Closer()
	}
	err = fmt.Errorf("%+v", errs)
	return
}

// FindUtxos impl
func (b *Bridge) FindUtxos(addr string) ([]*electrs.ElectUtxo, error) {
	// ListUnspent
	btcaddr, cvterr := b.ConvertCOLXAddress(addr, "")
	if cvterr == nil {
		addr = btcaddr.String()
	}
	return electrs.FindUtxos(b, addr)
}

// GetPoolTxidList impl
func (b *Bridge) GetPoolTxidList() ([]string, error) {
	return electrs.GetPoolTxidList(b)
}

// GetPoolTransactions impl
func (b *Bridge) GetPoolTransactions(addr string) ([]*electrs.ElectTx, error) {
	btcaddr, cvterr := b.ConvertCOLXAddress(addr, "")
	if cvterr == nil {
		addr = btcaddr.String()
	}
	results, err := electrs.GetPoolTransactions(b, addr)
	if err == nil {
		for _, result := range results {
			*result = *b.ToCOLXTx(result)
		}
	}
	return results, err
}

// GetTransactionHistory impl
func (b *Bridge) GetTransactionHistory(addr, lastSeenTxid string) ([]*electrs.ElectTx, error) {
	btcaddr, cvterr := b.ConvertCOLXAddress(addr, "")
	if cvterr == nil {
		addr = btcaddr.String()
	}
	results, err := electrs.GetTransactionHistory(b, addr, lastSeenTxid)
	if err == nil {
		for _, result := range results {
			*result = *b.ToCOLXTx(result)
		}
	}
	return results, err
}

// GetOutspend impl
func (b *Bridge) GetOutspend(txHash string, vout uint32) (*electrs.ElectOutspend, error) {
	return electrs.GetOutspend(b, txHash, vout)
}

// PostTransaction impl
func (b *Bridge) PostTransaction(txHex string) (txHash string, err error) {
	return electrs.PostTransaction(b, txHex)
}

// GetBlockHash impl
func (b *Bridge) GetBlockHash(height uint64) (string, error) {
	return electrs.GetBlockHash(b, height)
}

// GetBlockTxids impl
func (b *Bridge) GetBlockTxids(blockHash string) ([]string, error) {
	return electrs.GetBlockTxids(b, blockHash)
}

// GetBlock impl
func (b *Bridge) GetBlock(blockHash string) (*electrs.ElectBlock, error) {
	return electrs.GetBlock(b, blockHash)
}

// GetBlockTransactions impl
func (b *Bridge) GetBlockTransactions(blockHash string, startIndex uint32) ([]*electrs.ElectTx, error) {
	results, err := electrs.GetBlockTransactions(b, blockHash, startIndex)
	if err == nil {
		for _, result := range results {
			*result = *b.ToCOLXTx(result)
		}
	}
	return results, err
}

// EstimateFeePerKb impl
func (b *Bridge) EstimateFeePerKb(blocks int) (int64, error) {
	return electrs.EstimateFeePerKb(b, blocks)
}

// GetBalance impl
func (b *Bridge) GetBalance(account string) (*big.Int, error) {
	utxos, err := b.FindUtxos(account)
	if err != nil {
		return nil, err
	}
	var balance uint64
	for _, utxo := range utxos {
		balance += *utxo.Value
	}
	return new(big.Int).SetUint64(balance), nil
}

// GetTokenBalance impl
func (b *Bridge) GetTokenBalance(tokenType, tokenAddress, accountAddress string) (*big.Int, error) {
	return nil, fmt.Errorf("[%v] can not get token balance of token with type '%v'", b.ChainConfig.BlockChain, tokenType)
}

// GetTokenSupply impl
func (b *Bridge) GetTokenSupply(tokenType, tokenAddress string) (*big.Int, error) {
	return nil, fmt.Errorf("[%v] can not get token supply of token with type '%v'", b.ChainConfig.BlockChain, tokenType)
}
