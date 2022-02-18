package eth

import (
	"io/ioutil"
	"math/big"
	"net/http"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	ecrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/common/hexutil"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/types"
)

func TestSendTx(t *testing.T) {
	url := "https://rpc-testnet.rei.network"
	chainId := big.NewInt(12357)
	b, _ := hexutil.Decode("0x9e557ceb31535e269d53fcf9604efd799f383505edf08b64defcbf45a6c4a0db")
	mykey, _ := ecrypto.ToECDSA(b)
	mykey.Curve = btcec.S256()
	to := common.HexToAddress("0xCe15AA76A07E109deb359dA8a731Df0D640066C2")
	var nonce uint64 = 5
	amount, _ := new(big.Int).SetString("3450000000000000000000000000", 0)
	tx := types.NewTransaction(nonce, to, amount, 1000000, big.NewInt(1e9), nil)
	signer := types.NewEIP2930Signer(chainId)
	signedTx, signerr := types.SignTx(tx, signer, mykey)
	if signerr != nil {
		t.Error(signerr)
		return
	}
	t.Log(signedTx)
	rawtx := signedTx.RawStr()
	t.Log(rawtx)
	//rawtx = ""
	var result string
	err := client.RPCPost(&result, url, "eth_sendrawtransaction", rawtx)
	if err == nil {
		t.Log(result)
		return
	}
	t.Log(result, err)
}

func TestGetLatestBlockNumber(t *testing.T) {
	var result string
	err := client.RPCPost(&result, "https://mainnet.aurora.dev/5vwoRnBWvhXD2qjBsKX5UNGdLuaZonTiqJLs9BnrzXLs", "eth_blockNumber")
	//err := client.RPCPost(&result, "http://1.15.228.87:30003", "eth_blockNumber")
	if err == nil {
		t.Log(result)
		return
	}
	t.Error(err) // json-rpc error -32600, Invalid request
}

func TestGetLatestBlockNumber2(t *testing.T) {
	client := &http.Client{}
	var data = strings.NewReader(`{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`)
	req, err := http.NewRequest("POST", "https://mainnet.aurora.dev/5vwoRnBWvhXD2qjBsKX5UNGdLuaZonTiqJLs9BnrzXLs", data)
	//req, err := http.NewRequest("POST", "http://1.15.228.87:30003", data)
	if err != nil {
		t.Error(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)

	if err != nil {
		t.Error(err)
	}

	defer resp.Body.Close()

	bodyText, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		t.Error(err)
	}

	t.Logf("%s\n", bodyText) // {"jsonrpc":"2.0","id":38,"result":"0x34029c8"}
}
