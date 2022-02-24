package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/crypto"
	"github.com/urfave/cli/v2"
)

func initArgsSendXrp(ctx *cli.Context) {
	prikey = ctx.String(keyFlag.Name)
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
			log.Fatalf("unrecognized network: %v", net)
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
	var key crypto.Key
	var sequence *uint32

	if seed != "" {
		key = ripple.ImportKeyFromSeed(seed, "ecdsa")
		seq := uint32(keyseq)
		sequence = &seq
	} else {
		key = crypto.NewECDSAKeyFromPrivKeyBytes(common.FromHex(prikey))
	}

	from := ripple.GetAddress(key, sequence)
	log.Printf("sender address is %v", from)

	txseq, err := b.GetSeq(nil, from)
	if err != nil {
		log.Fatal("get account sequence failed", "account", from, "err", err)
	}
	log.Info("start build tx", "from", from, "sequence", sequence, "txseq", txseq, "to", to, "amount", amount, "memo", memo)

	tx, _, _ := ripple.NewUnsignedPaymentTransaction(key, sequence, *txseq, to, amount, 10, memo, "", false, false, false)

	stx, _, err := b.SignTransactionWithRippleKey(tx, key, sequence)
	if err != nil {
		log.Fatal("sign transaction failed", "err", err)
	}
	fmt.Printf("%+v\n", stx)

	txhash, err := b.SendTransaction(stx)
	if err != nil {
		log.Fatal("send transaction failed", "err", err)
	}
	fmt.Printf("Submited tx: %v\n", txhash)
	return txhash
}
