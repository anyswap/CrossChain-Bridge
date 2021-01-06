package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	elog "github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/eth"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
	"github.com/anyswap/CrossChain-Bridge/tokens/xrp"
	"github.com/rubblelabs/ripple/data"
	"github.com/rubblelabs/ripple/websockets"
)

var (
	seed       string
	keyseq     uint
	to         string
	memo       string
	amount     string
	apiAddress string
	net        string
	b          *xrp.Bridge
)

func init() {
	flag.StringVar(&seed, "seed", "", "seed")
	flag.UintVar(&keyseq, "keyseq", 0, "key sequence")
	flag.StringVar(&to, "to", "", "destination address")
	flag.StringVar(&amount, "amount", "", "xrp amount")
	flag.StringVar(&memo, "memo", "", "memo")
	flag.StringVar(&net, "net", "testnet", "network")
}

func initBridge() func() {
	tokens.DstBridge = eth.NewCrossChainBridge(false)
	b = xrp.NewCrossChainBridge(true)

	flag.Parse()
	switch strings.ToLower(net) {
	case "mainnet", "main":
		apiAddress = "wss://s2.ripple.com:443/"
	case "testnet", "test":
		apiAddress = "wss://s.altnet.rippletest.net:443/"
	default:
		log.Fatal(fmt.Errorf("unknown network: %v", net))
	}

	remote, err := websockets.NewRemote(apiAddress)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(apiAddress)
	b.Remotes[apiAddress] = remote
	return remote.Close
}

func main() {
	close := initBridge()
	defer close()
	/*txhash := sendXRP()
	time.Sleep(time.Second * 5)
	checkTx(txhash)
	checkStatus(txhash)*/
	//checkTx("707EB888A528EEE20615585DB82535E5A8F54E6446A400940FD8F9B3C643CD37")
	//checkStatus("FFE78C8707031799A8EEFA526D670511DF16EB19C911B700ABB625F8D0C46EEE")
	scanTx()
}

func sendXRP() string {
	key := xrp.ImportKeyFromSeed(seed, "ecdsa")
	keyseq := uint32(keyseq)

	from := xrp.GetAddress(key, &keyseq)
	txseq, err := b.GetSeq(from)
	if err != nil {
		log.Fatal(err)
	}

	tx, _, _ := xrp.NewUnsignedPaymentTransaction(key, &keyseq, txseq, to, amount, 10, memo, "", false, false, false)

	/*privData := key.Private(&keyseq)
	priv, _ := btcec.PrivKeyFromBytes(btcec.S256(), privData)

	stx, _, err := b.SignTransactionWithPrivateKey(tx, priv.ToECDSA())
	if err != nil {
		log.Fatal(err)
	}*/
	stx, _, err := b.SignTransactionWithRippleKey(tx, key, &keyseq)
	if err != nil {
		log.Fatal(fmt.Errorf("Sign transaction failed, %v", err))
	}
	fmt.Printf("%+v\n", stx)

	txhash, err := b.SendTransaction(stx)
	if err != nil {
		log.Fatal(fmt.Errorf("Send transaction failed, %v", err))
	}
	fmt.Printf("Submited tx: %v\n", txhash)
	return txhash
}

func checkTx(txHash string) bool {
	tx, err := b.GetTransaction(txHash)
	if err != nil {
		log.Printf("Get tx failed, %v", err)
		return false
	}

	txres, ok := tx.(*websockets.TxResult)
	if !ok {
		// unexpected
		log.Printf("Tx res type error")
		return false
	}

	if txres.TransactionWithMetaData.MetaData.TransactionResult != 0 {
		log.Printf("Tx result: %v", txres.TransactionWithMetaData.MetaData.TransactionResult)
		return false
	}

	payment, ok := txres.TransactionWithMetaData.Transaction.(*data.Payment)
	if !ok || payment.GetTransactionType() != data.PAYMENT {
		log.Printf("Not a payment transaction")
		return false
	}

	bind, ok := xrp.GetBindAddressFromMemos(payment)
	if !ok {
		log.Printf("Get bind address failed")
		return false
	}
	log.Printf("Bind address: %v\n", bind)

	log.Println("Tx success!")
	return true
}

func checkStatus(txHash string) bool {
	status := b.GetTransactionStatus(txHash)
	fmt.Printf("%+v\n", status)

	return true
}

func scanTx() {
	start := uint64(13794220)
	stable := start
	confirmations := uint64(0)
	errorSubject := "[scanchain] get XRP block failed"
	scanSubject := "[scanchain] scanned XRP block"
	for {
		latest := tools.LoopGetLatestBlockNumber(b)
		elog.Info("Scan chain", "latest block number", latest)
		for h := stable + 1; h <= latest; {
			blockHash, err := b.GetBlockHash(h)
			if err != nil {
				elog.Error(errorSubject, "height", h, "err", err)
				time.Sleep(time.Second * 3)
				continue
			}
			elog.Info("Scan chain, get block hash", "", blockHash)
			txids, err := b.GetBlockTxids(h)
			if err != nil {
				elog.Error(errorSubject, "height", h, "blockHash", blockHash, "ledger index", h, "err", err)
				time.Sleep(time.Second * 3)
				continue
			}
			elog.Info("Scan chain, get tx ids", "", txids)
			for _, txid := range txids {
				elog.Info("Check transaction", "txid", txid)
				tx, err := b.GetTransaction(txid)
				if err != nil {
					elog.Warn("Check transaction failed", "error", err)
				}
				elog.Info("Check transaction success", "tx", tx)
			}
			elog.Info(scanSubject, "blockHash", blockHash, "height", h, "txs", len(txids))
			h++
		}
		if stable+confirmations < latest {
			stable = latest - confirmations
		}
		time.Sleep(time.Second * 3)
	}
}
