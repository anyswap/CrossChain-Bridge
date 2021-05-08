package colx

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"sort"

	"github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
	"github.com/giangnamnabka/btcd/btcjson"
	"github.com/giangnamnabka/btcd/chaincfg/chainhash"
	"github.com/giangnamnabka/btcd/rpcclient"
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
	ChainzAPIKeys []string
	id               *int
}

// NextID returns next id for FindUtxo request
func (c *Client) NextID() int {
	if c.id == nil {
		c.id = new(int)
		*c.id = 1
	}
	return *c.id
}

// Closer close all rpc clients
func (c *Client) Closer() {
	for _, ccli := range c.CClients {
		ccli.Closer()
	}
}

var blockClient *Client

var cclis = make([]CoreClient, 0)

// GetClient returns new Client
func (b *Bridge) GetClient() *Client {
	if blockClient != nil {
		return blockClient
	}

	cfg := b.GetGatewayConfig()
	if cfg.Extras == nil || cfg.Extras.ColxExtra == nil {
		return nil
	}

	if len(cclis) == 0 {
		for _, args := range cfg.Extras.ColxExtra.FullnodeAPIs {
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
	}

	blockClient = &Client{
		CClients:         cclis,
		ChainzAPIKeys: cfg.Extras.ColxExtra.ChainzAPIKeys,
	}
	return blockClient
}

// GetLatestBlockNumberOf impl
func (b *Bridge) GetLatestBlockNumberOf(apiAddress string) (uint64, error) {
	cli := b.GetClient()
	//# defer cli.Closer()
	for _, ccli := range cli.CClients {
		if ccli.Address == apiAddress {
			number, err := ccli.GetBlockCount()
			return uint64(number), err
		}
	}
	return 0, nil
}

// GetLatestBlockNumber impl
func (b *Bridge) GetLatestBlockNumber() (blocknumber uint64, err error) {
	cli := b.GetClient()
	//# defer cli.Closer()
	errs := make([]error, 0)
	for _, ccli := range cli.CClients {
		number, err0 := ccli.GetBlockCount()
		if err0 == nil {
			return uint64(number), nil
		}
		errs = append(errs, err0)
	}
	err = fmt.Errorf("%+v", errs)
	return
}

// GetTransactionByHash impl
func (b *Bridge) GetTransactionByHash(txHash string) (etx *electrs.ElectTx, err error) {
	cli := b.GetClient()
	//# defer cli.Closer()
	errs := make([]error, 0)
	hash, err := chainhash.NewHashFromStr(txHash)
	if err != nil {
		return
	}
	for _, ccli := range cli.CClients {
		tx, err0 := ccli.GetRawTransactionVerbose(hash)
		if err0 == nil {
			etx = ConvertTx(tx)
			return
		}
		errs = append(errs, err0)
	}
	err = fmt.Errorf("%+v", errs)
	return
}

// GetElectTransactionStatus impl
func (b *Bridge) GetElectTransactionStatus(txHash string) (txstatus *electrs.ElectTxStatus, err error) {
	cli := b.GetClient()
	//# defer cli.Closer()
	errs := make([]error, 0)
	hash, err := chainhash.NewHashFromStr(txHash)
	if err != nil {
		return
	}
	for _, ccli := range cli.CClients {
		txraw, err0 := ccli.GetRawTransactionVerbose(hash)
		if err0 == nil {
			txstatus = TxStatus(txraw)
			if h := txstatus.BlockHash; h != nil {
				if blk, err1 := b.GetBlock(*h); err1 == nil {
					*txstatus.BlockHeight = uint64(*blk.Height)
				}
			}
			return
		}
		errs = append(errs, err0)
	}
	err = fmt.Errorf("%+v", errs)
	return
}

// FindUtxos impl
func (b *Bridge) FindUtxos(addr string) (utxos []*electrs.ElectUtxo, err error) {
	cli := b.GetClient()
	cfg := b.GetChainConfig()

	errs := make([]error, 0)
	for _, key := range cli.ChainzAPIKeys {
		cutxos, err0 := getChainzUtxo(addr, key)

		if err0 == nil {
			for _, cutxo := range cutxos.UnspentOutputs {
				status := &electrs.ElectTxStatus{
					Confirmed: new(bool),
				}
				*status.Confirmed = uint64(cutxo.Confirmations) > *cfg.Confirmations

				utxo := &electrs.ElectUtxo{
					Txid:   new(string),
					Vout:   new(uint32),
					Value:  new(uint64),
					Status: new(electrs.ElectTxStatus),
				}
				*utxo.Txid = cutxo.TxHash
				*utxo.Vout = uint32(cutxo.TxOutN)
				*utxo.Value = uint64(cutxo.Value)
				utxo.Status = status
				utxos = append(utxos, utxo)
			}
			sort.Sort(electrs.SortableElectUtxoSlice(utxos))
			return utxos, err
		}
		errs = append(errs, err0)
	}
	err = fmt.Errorf("%+v", errs)
	return utxos, err
}

// getChainzUtxo
func getChainzUtxo(addr, key string) (ChainzUtxos, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://chainz.cryptoid.info/colx/api.dws?q=unspent&active=%s&key=%s", addr, key), nil)
	if err != nil {
		return ChainzUtxos{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:81.0) Gecko/20100101 Firefox/81.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("TE", "Trailers")
	req.Header.Set("Cookie", "cf_clearance=b639cc1208c4a9a1fa3a3dd8447fb2c53a20a0ad-1620455942-0-150; _ga=GA1.2.643865580.1607584926; __cfduid=d1cf9da8870917d990cd918c3e076228e1620455942; cf_chl_2=5b343c42b8f7c7e; cf_chl_prog=x9")
	resp, err := client.Do(req)
	if err != nil {
		return ChainzUtxos{}, err
	}
	defer resp.Body.Close()
	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ChainzUtxos{}, err
	}
	cutxos := ChainzUtxos{}
	err = json.Unmarshal(bodyText, &cutxos)
	return cutxos, err
}

// ChainzUtxos struct
type ChainzUtxos struct {
	UnspentOutputs []ChainzUtxo `json:"unspent_outputs"`
}

// ChainzUtxo
type ChainzUtxo struct {
	TxHash string `json:"tx_hash"`
	TxOutN int64 `json:"tx_ouput_n"`
	Value int64 `json:"value"`
	Confirmations int64 `json:"confirmations"`
	Script string `json:"script"`
	Addr string `json:"addr"`
}

// GetPoolTxidList impl
func (b *Bridge) GetPoolTxidList() (txids []string, err error) {
	cli := b.GetClient()
	//# defer cli.Closer()
	txids = make([]string, 0)
	errs := make([]error, 0)
	for _, ccli := range cli.CClients {
		hashes, err0 := ccli.GetRawMempool()
		if err0 == nil {
			for _, hash := range hashes {
				txids = append(txids, hash.String())
			}
			return
		}
		errs = append(errs, err0)
	}
	err = fmt.Errorf("%+v", errs)
	return
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
	if len(errs) > 0 {
		err = fmt.Errorf("%+v", errs)
	}
	return
}

// GetTransactionHistory impl
// lastSeenTxis 以后所有的交易
func (b *Bridge) GetTransactionHistory(addr, lastSeenTxid string) (etxs []*electrs.ElectTx, err error) {
	return
}

// GetOutspend impl
// Only to find out if txout is spent, does not tell in which transactions it is spent.
func (b *Bridge) GetOutspend(txHash string, vout uint32) (evout *electrs.ElectOutspend, err error) {
	cli := b.GetClient()
	//# defer cli.Closer()
	errs := make([]error, 0)
	hash, err := chainhash.NewHashFromStr(txHash)
	if err != nil {
		return
	}
	for _, ccli := range cli.CClients {
		txout, err0 := ccli.GetTxOut(hash, vout, true)
		if err0 == nil {
			evout = TxOutspend(txout)
			return
		}
		errs = append(errs, err0)
	}
	err = fmt.Errorf("%+v", errs)
	return
}

// PostTransaction impl
func (b *Bridge) PostTransaction(txHex string) (txHash string, err error) {
	cli := b.GetClient()
	//# defer cli.Closer()
	errs := make([]error, 0)
	var success bool
	for _, ccli := range cli.CClients {
		msgtx := DecodeTxHex(txHex, 0, false)
		hash, err0 := ccli.SendRawTransaction(msgtx, true)
		if err0 == nil && !success {
			success = true
			txHash = hash.String()
		}
		errs = append(errs, err0)
	}
	if success {
		return txHash, nil
	}
	err = fmt.Errorf("%+v", errs)
	return
}

// GetBlockHash impl
func (b *Bridge) GetBlockHash(height uint64) (hash string, err error) {
	cli := b.GetClient()
	//# defer cli.Closer()
	errs := make([]error, 0)
	for _, ccli := range cli.CClients {
		bh, err0 := ccli.GetBlockHash(int64(height))
		if err0 == nil {
			hash = bh.String()
			return
		}
		errs = append(errs, err0)
	}
	err = fmt.Errorf("%+v", errs)
	return
}

// GetBlockTxids impl
func (b *Bridge) GetBlockTxids(blockHash string) (txids []string, err error) {
	cli := b.GetClient()
	//# defer cli.Closer()
	errs := make([]error, 0)
	for _, ccli := range cli.CClients {
		hash, errf := chainhash.NewHashFromStr(blockHash)
		if errf != nil {
			continue
		}
		block, err0 := ccli.GetBlockVerbose(hash)
		if err0 == nil {
			txids = block.Tx
			return txids, nil
		}
		errs = append(errs, err0)
	}
	err = fmt.Errorf("%+v", errs)
	return
}

// GetBlock impl
func (b *Bridge) GetBlock(blockHash string) (eblock *electrs.ElectBlock, err error) {
	cli := b.GetClient()
	//# defer cli.Closer()
	errs := make([]error, 0)
	hash, err := chainhash.NewHashFromStr(blockHash)
	if err != nil {
		return
	}
	for _, ccli := range cli.CClients {
		block, err0 := ccli.GetBlockVerbose(hash)
		if err0 == nil {
			eblock = ConvertBlock(block)
			return eblock, nil
		}
		errs = append(errs, err0)
	}
	err = fmt.Errorf("%+v", errs)
	return
}

// GetBlockTransactions impl
func (b *Bridge) GetBlockTransactions(blockHash string, startIndex uint32) (etxs []*electrs.ElectTx, err error) {
	cli := b.GetClient()
	//# defer cli.Closer()
	errs := make([]error, 0)
	hash, err := chainhash.NewHashFromStr(blockHash)
	if err != nil {
		return
	}
	for _, ccli := range cli.CClients {
		block, err0 := ccli.GetBlockVerbose(hash)
		if err0 == nil {
			txs := block.Tx
			for _, txid := range txs {
				etx, err1 := b.GetTransactionByHash(txid)
				if err1 != nil {
					continue
				}
				etxs = append(etxs, etx)
			}
			return
		}
		errs = append(errs, err0)
	}
	err = fmt.Errorf("%+v", errs)
	return
}

// EstimateFeePerKb impl
func (b *Bridge) EstimateFeePerKb(blocks int) (fee int64, err error) {
	cli := b.GetClient()
	//# defer cli.Closer()
	errs := make([]error, 0)
	for _, ccli := range cli.CClients {
		res, err0 := ccli.Client.EstimateSmartFee(int64(blocks), &btcjson.EstimateModeEconomical)
		if err0 == nil {
			if len(res.Errors) > 0 {
				errs = append(errs, fmt.Errorf("%+v", res.Errors))
				continue
			}
			return int64(*res.FeeRate * 1e8), nil
		}
		errs = append(errs, err0)
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
