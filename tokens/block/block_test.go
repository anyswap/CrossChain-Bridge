package block

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/wallet/txauthor"
)

var b *Bridge

func init() {
	b = NewCrossChainBridge(true)

	b.ChainConfig = &tokens.ChainConfig{
		BlockChain: "Block",
		NetID:      "mainnet",
	}

	b.GatewayConfig = &tokens.GatewayConfig{
		APIAddress: []string{"5.189.139.168:51515"},
		Extras: &tokens.GatewayExtras{
			BlockExtra: &tokens.BlockExtraArgs{
				CoreAPIs: []tokens.BlocknetCoreAPIArgs{
					{
						APIAddress:  "5.189.139.168:51515",
						RPCUser:     "xxmm",
						RPCPassword: "123456",
						DisableTLS:  true,
					},
				},
				UTXOAPIAddresses: []string{"https://plugin-dev.core.cloudchainsinc.com"},
			},
		},
	}
}

func checkError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf(err.Error())
	}
}

func TestNewAddressPubKeyHash(t *testing.T) {
	t.Logf("TestNewAddressPubKeyHash")

	wif := "PnPaj8UeZCziJx9EBTDsZfzuYDZVaGwvrtettNeTqnhH5Z3d6B41"
	privWif, err := btcutil.DecodeWIF(wif)
	checkError(t, err)

	pkdata := privWif.SerializePubKey()

	addr, err := b.NewAddressPubKeyHash(pkdata)
	checkError(t, err)
	t.Logf("TestNewAddressPubKeyHash: %s\n", addr)

	realaddr := "BXcz95EZfLBREpQrMDsKFnMJSaUYNRyhHU"

	if addr.String() != realaddr {
		err := fmt.Errorf("Block address error, got %s, should be: %s", addr.String(), realaddr)
		checkError(t, err)
	}
}

func TestGetLatestBlockNumberOf(t *testing.T) {
	t.Logf("TestGetLatestBlockNumberOf")
	num, err := b.GetLatestBlockNumberOf("5.189.139.168:51515")
	checkError(t, err)
	t.Logf("GetLatestBlockNumberOf: %v\n", num)
}

func TestGetLatestBlockNumber(t *testing.T) {
	t.Logf("TestGetLatestBlockNumber")
	num, err := b.GetLatestBlockNumber()
	checkError(t, err)
	t.Logf("TestGetLatestBlockNumber: %v\n", num)
}

func TestGetTransactionByHash(t *testing.T) {
	t.Logf("TestGetTransactionByHash")
	//tx, err := b.GetTransactionByHash("a3e8864b64391ad991d0f4376cc2d1539efb3dffba8f90870b230dde618e764e")
	tx, err := b.GetTransactionByHash("8250273a112ec0b8d91d12d1652af77495df6c7c0cc2a2b1d93c02a6c5cfaba5")
	checkError(t, err)
	t.Logf("TestGetTransactionByHash: %+v\n", tx)
	for _, vout := range tx.Vout {
		t.Logf("ScriptpubkeyType: %+v\n", *vout.ScriptpubkeyType)
		t.Logf("Vout: %+v\n", *vout.ScriptpubkeyAddress)
	}
}

func TestGetElectTransactionStatus(t *testing.T) {
	t.Logf("TestGetElectTransactionStatus")
	//status, err := b.GetElectTransactionStatus("a3e8864b64391ad991d0f4376cc2d1539efb3dffba8f90870b230dde618e764e")
	status, err := b.GetElectTransactionStatus("8250273a112ec0b8d91d12d1652af77495df6c7c0cc2a2b1d93c02a6c5cfaba5")
	checkError(t, err)
	t.Logf("TestGetElectTransactionStatus: %+v\n", status)
	t.Logf("Confirmed: %+v\n", *status.Confirmed)
	t.Logf("BlockHeight: %+v\n", *status.BlockHeight)
	t.Logf("BlockHash: %+v\n", *status.BlockHash)
	t.Logf("BlockTime: %+v\n", *status.BlockTime)
}

func TestFindUtxos(t *testing.T) {
	t.Logf("TestFindUtxos")
	//utxos, err := b.FindUtxos("BmCQZdXFUhGvDZkFNyy9fshkGnoPzNnTnY")
	utxos, err := b.FindUtxos("Ccg7idzpeABztNarqwpcsF5NjVLinQJZLa")
	checkError(t, err)
	t.Logf("TestFindUtxos: %+v\n", utxos)
}

func TestGetPoolTxidList(t *testing.T) {
	t.Logf("TestGetPoolTxidList")
	ids, err := b.GetPoolTxidList()
	checkError(t, err)
	t.Logf("TestGetPoolTxidList: %+v\n", ids)
}

func TestGetPoolTransactions(t *testing.T) {
	t.Logf("TestGetPoolTransactions")
	txs, err := b.GetPoolTransactions("BmCQZdXFUhGvDZkFNyy9fshkGnoPzNnTnY")
	checkError(t, err)
	t.Logf("TestGetPoolTransactions: %+v\n", txs)
}

func TestGetOutspend(t *testing.T) {
	t.Logf("TestGetOutspend")
	outspend, err := b.GetOutspend("a3e8864b64391ad991d0f4376cc2d1539efb3dffba8f90870b230dde618e764e", 0)
	checkError(t, err)
	t.Logf("TestGetOutspend: spent: %+v\n", *outspend.Spent)
}

func TestPostTransaction(t *testing.T) {
	t.Logf("TestPostTransaction")

	var authoredTx *txauthor.AuthoredTx
	var wif string
	var privkey *ecdsa.PrivateKey
	var fromAddress, toAddress, changeAddress string

	fromAddress = "Bp23BeXEKXCTd7oRWtrKv8nc1SKXvDH3Hq"
	changeAddress = fromAddress
	//toAddress = "BpeGP9ooFTGSsdcysmMMgbRCqYQgpjrCsE"
	toAddress = "Ccg7idzpeABztNarqwpcsF5NjVLinQJZLa"
	//wif = "PnLR......NtKcv"
	wif = ""
	if wif == "" {
		t.Logf("Private key wif is required to test PostTransaction")
		return
	}

	pkwif, err := btcutil.DecodeWIF(wif)
	checkError(t, err)
	privkey = pkwif.PrivKey.ToECDSA()

	// build tx
	utxos, err := b.FindUtxos(fromAddress)
	checkError(t, err)
	t.Logf("utxos: %+v", utxos[0])

	txOuts, err := b.getTxOutputs(toAddress, big.NewInt(1000000), "")
	checkError(t, err)
	t.Logf("txouts: %+v\n", txOuts)

	inputSource := func(target btcAmountType) (total btcAmountType, inputs []*wireTxInType, inputValues []btcAmountType, scripts [][]byte, err error) {
		return b.getUtxosFromElectUtxos(target, []string{fromAddress}, utxos)
	}

	changeSource := func() ([]byte, error) {
		return b.GetPayToAddrScript(changeAddress)
	}

	relayFeePerKb, _ := b.getRelayFeePerKb()
	t.Logf("relayFeePerKb: %+v\n", relayFeePerKb)

	authoredTx, err = b.NewUnsignedTransaction(txOuts, btcAmountType(relayFeePerKb), inputSource, changeSource, true)
	checkError(t, err)
	t.Logf("authoredTx: %+v\n", authoredTx)

	// signTx
	signedTx, _, err := b.SignTransactionWithPrivateKey(authoredTx, privkey)
	checkError(t, err)
	t.Logf("signedTx: %+v\n", signedTx)

	tx := signedTx.(*txauthor.AuthoredTx).Tx

	buf := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	err = tx.Serialize(buf)
	checkError(t, err)
	txHex := hex.EncodeToString(buf.Bytes())
	t.Logf("Bridge send tx, hash: %v", tx.TxHash())
	txHash, err := b.PostTransaction(txHex)
	checkError(t, err)
	t.Logf("TestPostTransaction: %+v\n", txHash)
}

func TestGetBlockHash(t *testing.T) {
	t.Logf("TestGetBlockHash")
	hash, err := b.GetBlockHash(1000000)
	checkError(t, err)
	t.Logf("TestGetBlockHash: %+v\n", hash) // c5e7f9bf6daee954be9844de0ac80f3c1e2c2592974e7733d9f192b4e30b9c40
}

func TestGetBlockTxids(t *testing.T) {
	t.Logf("TestGetBlockTxids")
	ids, err := b.GetBlockTxids("c5e7f9bf6daee954be9844de0ac80f3c1e2c2592974e7733d9f192b4e30b9c40")
	checkError(t, err)
	t.Logf("TestGetBlockTxids: %+v\n", ids)
}

func TestGetBlock(t *testing.T) {
	t.Logf("TestGetBlock")
	blk, err := b.GetBlock("c5e7f9bf6daee954be9844de0ac80f3c1e2c2592974e7733d9f192b4e30b9c40")
	checkError(t, err)
	t.Logf("TestGetBlock: %+v\n", blk)
}

func TestGetBlockTransactions(t *testing.T) {
	t.Logf("TestGetBlockTransactions")
	txs, err := b.GetBlockTransactions("c5e7f9bf6daee954be9844de0ac80f3c1e2c2592974e7733d9f192b4e30b9c40", 0)
	checkError(t, err)
	t.Logf("TestGetBlockTransactions: %+v\n", txs)
}

/*func TestEstimateFeePerKb(t *testing.T) {
	t.Logf("TestEstimateFeePerKb")
	fee, err := b.EstimateFeePerKb(3)
	checkError(t, err)
	t.Logf("TestEstimateFeePerKb: %+v\n", fee)
}*/

func TestGetBalance(t *testing.T) {}

func TestGetPayToAddrScript(t *testing.T) {
	t.Logf("TestGetPayToAddrScript")
	script, err := b.GetPayToAddrScript("Ccg7idzpeABztNarqwpcsF5NjVLinQJZLa")
	checkError(t, err)
	t.Logf("TestGetPayToAddrScript: %s\n", hex.EncodeToString(script))
}
