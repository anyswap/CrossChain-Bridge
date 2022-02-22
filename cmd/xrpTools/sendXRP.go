package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/tokens/ripple"
	"github.com/urfave/cli/v2"
)

func initArgsSendXrp(ctx *cli.Context) {
	seed = ctx.String(seedFlag.Name)
	keyseq = ctx.Uint(keyseqFlag.Name)
	to = ctx.String(toFlag.Name)
	amount = ctx.String(amountFlag.Name)
	memo = ctx.String(memoFlag.Name)
	net = ctx.String(netFlag.Name)
	apiAddress = ctx.String(apiAddressFlag.Name)
	if apiAddress == "" {
		switch strings.ToLower(net) {
		case "mainnet", "main":
			apiAddress = "wss://s2.ripple.com:443/"
		case "testnet", "test":
			apiAddress = "wss://s.altnet.rippletest.net:443/"
		case "devnet", "dev":
			apiAddress = "wss://s.devnet.rippletest.net:443/"
		default:
			log.Fatal(fmt.Errorf("unrecognized network: %v", net))
		}
	}
}

func sendXrpAction(ctx *cli.Context) error {
	initArgsSendXrp(ctx)
	initBridge()
	txhash := sendXRP()
	time.Sleep(time.Second * 5)
	checkTx(txhash)
	checkStatus(txhash)
	return nil
}

func sendXRP() string {
	key := ripple.ImportKeyFromSeed(seed, "ecdsa")
	keyseq := uint32(keyseq)

	from := ripple.GetAddress(key, &keyseq)
	txseq, err := b.GetSeq(nil, from)
	if err != nil {
		log.Fatal(err)
	}

	tx, _, _ := ripple.NewUnsignedPaymentTransaction(key, &keyseq, *txseq, to, amount, 10, memo, "", false, false, false)

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
