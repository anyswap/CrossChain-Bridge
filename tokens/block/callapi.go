package block

import (
	"fmt"
	"math/big"
	"sort"

	primaryclient "github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
)

// CoreClient extends btcd rpcclient
type CoreClient struct {
	*rpcclient.Client
	Address string
	Closer  func()
}

// Client struct
type Client struct {
	CClients         []CoreClient
	UTXOAPIAddresses []string
}

// GetClient returns new Client
func (b *Bridge) GetClient() *Client {
	cfg := b.GetGatewayConfig()
	if cfg.Extras == nil || cfg.Extras.BlockExtra == nil {
		return nil
	}

	cclis := make([]CoreClient, 0)
	for _, args := range cfg.Extras.BlockExtra.CoreAPIs {
		connCfg := &rpcclient.ConnConfig{
			Host:         args.APIAddress,
			User:         args.RPCUser,
			Pass:         args.RPCPassword,
			HTTPPostMode: true,            // Bitcoin core only supports HTTP POST mode
			DisableTLS:   args.DisableTLS, // Bitcoin core does not provide TLS by default
		}

		client, err := rpcclient.New(connCfg, nil)
		if err != nil {
			continue
		}

		ccli := CoreClient{
			Client:  client,
			Address: connCfg.Host,
			Closer:  client.Shutdown,
		}
		cclis = append(cclis, ccli)
	}

	return &Client{
		CClients:         cclis,
		UTXOAPIAddresses: cfg.Extras.BlockExtra.UTXOAPIAddresses,
	}
}

// GetLatestBlockNumberOf impl
func (b *Bridge) GetLatestBlockNumberOf(apiAddress string) (uint64, error) {
	cli := b.GetClient()
	for _, ccli := range cli.CClients {
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
	for _, ccli := range cli.CClients {
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
func (b *Bridge) GetTransactionByHash(txHash string) (etx *electrs.ElectTx, err error) {
	cli := b.GetClient()
	errs := make([]error, 0)
	hash, err := chainhash.NewHashFromStr(txHash)
	if err != nil {
		return
	}
	for _, ccli := range cli.CClients {
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
	for _, ccli := range cli.CClients {
		txraw, err0 := ccli.GetRawTransactionVerbose(hash)
		if err0 == nil {
			ccli.Closer()
			txstatus = TxStatus(txraw)
			return
		}
		errs = append(errs, err0)
		ccli.Closer()
	}
	err = fmt.Errorf("%+v", errs)
	return
}

// FindUtxos impl
func (b *Bridge) FindUtxos(addr string) (utxos []*electrs.ElectUtxo, err error) {
	// cloudchainsinc
	cli := b.GetClient()

	currentHeight, err := b.GetLatestBlockNumber()
	if err != nil {
		return
	}

	for _, url := range cli.UTXOAPIAddresses {
		req := primaryclient.NewRequest("getutxos", "BLOCK", `[\"BmCQZdXFUhGvDZkFNyy9fshkGnoPzNnTnY\"]`)
		res := struct {
			Utxos []CloudchainUtxo `json:"utxos"`
		}{}

		err0 := primaryclient.RPCPostRequest(url, req, &res)

		if err0 == nil {
			for _, cutxo := range res.Utxos {

				value := uint64(cutxo.Value * 1e8)

				status := &electrs.ElectTxStatus{
					BlockHeight: &cutxo.BlockNumber,
				}

				confirmed := false
				if currentHeight-cutxo.BlockNumber > 6 {
					confirmed = true
				}
				status.Confirmed = &confirmed

				if blkhash, err1 := b.GetBlockHash(cutxo.BlockNumber); err1 != nil {
					status.BlockHash = &blkhash
					if blk, err2 := b.GetBlock(blkhash); err2 != nil {
						status.BlockTime = new(uint64)
						*status.BlockTime = uint64(*blk.Timestamp)
					}
				}

				utxo := &electrs.ElectUtxo{
					Txid:   &cutxo.Address,
					Vout:   &cutxo.Vout,
					Value:  &value,
					Status: status,
				}
				utxos = append(utxos, utxo)
			}
			sort.Sort(electrs.SortableElectUtxoSlice(utxos))
			return
		}
	}
	return
}

// CloudchainUtxo struct
type CloudchainUtxo struct {
	Address     string  `json:address`
	Txhash      string  `json:Txhash`
	Vout        uint32  `json:Vout`
	BlockNumber uint64  `json:block_number`
	Value       float64 `json:value`
}

// GetPoolTxidList impl
func (b *Bridge) GetPoolTxidList() (txids []string, err error) {
	cli := b.GetClient()
	txids = make([]string, 0)
	errs := make([]error, 0)
	for _, ccli := range cli.CClients {
		hashes, err0 := ccli.GetRawMempool()
		if err0 == nil {
			ccli.Closer()
			for _, hash := range hashes {
				txids = append(txids, hash.String())
			}
			return
		}
		errs = append(errs, err0)
		ccli.Closer()
	}
	err = fmt.Errorf("%+v", errs)
	return electrs.GetPoolTxidList(b)
}

// GetPoolTransactions impl
func (b *Bridge) GetPoolTransactions(addr string) (txs []*electrs.ElectTx, err error) {
	txids, err := b.GetPoolTxidList()
	if err != nil {
		return
	}
	errs := make([]error, 0)
	for _, txid := range txids {
		tx, err0 := b.GetTransactionByHash(txid)
		if err0 != nil {
			errs = append(errs, err0)
			continue
		}
		if true {
			txs = append(txs, tx)
		}
	}
	err = fmt.Errorf("%+v", errs)
	return
}

// GetTransactionHistory impl
// lastSeenTxis 以后所有的交易
func (b *Bridge) GetTransactionHistory(addr, lastSeenTxid string) (etxs []*electrs.ElectTx, err error) {
	return
}

// GetOutspend impl
func (b *Bridge) GetOutspend(txHash string, vout uint32) (evout *electrs.ElectOutspend, err error) {
	/*cli := b.GetClient()
	errs := make([]error, 0)
	hash, err := chainhash.NewHashFromStr(txHash)
	if err != nil {
		return
	}
	for _, ccli := range cli.CClients {
		txraws, err0 := ccli.GetRawTransactionVerbose(hash)
		if err0 == nil {
			ccli.Closer()
			vout = TxOutspend(txraws, vout)
			return
		}
		errs = append(errs, err0)
		ccli.Closer()
	}
	err = fmt.Errorf("%+v", errs)
	return*/
	return
}

// PostTransaction impl
func (b *Bridge) PostTransaction(txHex string) (txHash string, err error) {
	cli := b.GetClient()
	errs := make([]error, 0)
	for _, ccli := range cli.CClients {
		msgtx := DecodeTxHex(txHex, 0, false)
		hash, err0 := ccli.SendRawTransaction(msgtx, true)
		if err0 == nil {
			txHash = hash.String()
			ccli.Closer()
			return
		}
		errs = append(errs, err0)
		ccli.Closer()
	}
	err = fmt.Errorf("%+v", errs)
	return
}

// GetBlockHash impl
func (b *Bridge) GetBlockHash(height uint64) (hash string, err error) {
	cli := b.GetClient()
	errs := make([]error, 0)
	for _, ccli := range cli.CClients {
		bh, err0 := ccli.GetBlockHash(int64(height))
		if err0 == nil {
			hash = bh.String()
			ccli.Closer()
			return
		}
		errs = append(errs, err0)
		ccli.Closer()
	}
	err = fmt.Errorf("%+v", errs)
	return
}

// GetBlockTxids impl
func (b *Bridge) GetBlockTxids(blockHash string) (txids []string, err error) {
	cli := b.GetClient()
	errs := make([]error, 0)
	for _, ccli := range cli.CClients {
		hash, err := chainhash.NewHashFromStr(blockHash)
		if err != nil {
			continue
		}
		block, err0 := ccli.GetBlockVerbose(hash)
		if err0 == nil {
			txids = block.Tx
			ccli.Closer()
			return txids, nil
		}
		errs = append(errs, err0)
		ccli.Closer()
	}
	err = fmt.Errorf("%+v", errs)
	return
}

// GetBlock impl
func (b *Bridge) GetBlock(blockHash string) (eblock *electrs.ElectBlock, err error) {
	cli := b.GetClient()
	errs := make([]error, 0)
	hash, err := chainhash.NewHashFromStr(blockHash)
	if err != nil {
		return
	}
	for _, ccli := range cli.CClients {
		block, err0 := ccli.GetBlockVerbose(hash)
		if err0 == nil {
			ccli.Closer()
			eblock = ConvertBlock(block)
			return eblock, nil
		}
		errs = append(errs, err0)
		ccli.Closer()
	}
	err = fmt.Errorf("%+v", errs)
	return
}

// GetBlockTransactions impl
func (b *Bridge) GetBlockTransactions(blockHash string, startIndex uint32) (etxs []*electrs.ElectTx, err error) {
	cli := b.GetClient()
	errs := make([]error, 0)
	hash, err := chainhash.NewHashFromStr(blockHash)
	if err != nil {
		return
	}
	for _, ccli := range cli.CClients {
		block, err0 := ccli.GetBlockVerbose(hash)
		if err0 == nil {
			txs := block.RawTx
			for _, tx := range txs {
				etx := ConvertTx(&tx)
				etxs = append(etxs, etx)
			}
			ccli.Closer()
			return
		}
		errs = append(errs, err0)
		ccli.Closer()
	}
	err = fmt.Errorf("%+v", errs)
	return
}

// EstimateFeePerKb impl
func (b *Bridge) EstimateFeePerKb(blocks int) (fee int64, err error) {
	//EstimateFee
	cli := b.GetClient()
	errs := make([]error, 0)
	for _, ccli := range cli.CClients {
		fee, err0 := ccli.EstimateFee(int64(blocks))
		if err0 == nil {
			ccli.Closer()
			return int64(fee), nil
		}
		errs = append(errs, err0)
		ccli.Closer()
	}
	err = fmt.Errorf("%+v", errs)
	return
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
