package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/crypto"
	"github.com/urfave/cli/v2"
)

func initArgsSendXrp(ctx *cli.Context) {
	keyType = ctx.String(keyTypeFlag.Name)
	if !isValidKeyType(keyType) {
		log.Fatalf("invalid key type %v", keyType)
	}
	prikey = ctx.String(keyFlag.Name)
	seed = ctx.String(seedFlag.Name)
	keyseq = ctx.Uint(keyseqFlag.Name)
	to = ctx.String(toFlag.Name)
	if ctx.IsSet(toTagFlag.Name) {
		tag := uint32(ctx.Uint(toTagFlag.Name))
		toTag = &tag
	}
	amount = ctx.String(amountFlag.Name)
	txfee = ctx.Int64(feeFlag.Name)
	memo = ctx.String(memoFlag.Name)
	net = ctx.String(netFlag.Name)
	apiAddress = ctx.String(apiAddressFlag.Name)
	if apiAddress == "" {
		apiAddress = initDefaultAPIAddress(net)
	}
}

func initDefaultAPIAddress(net string) string {
	switch strings.ToLower(net) {
	case "mainnet", "main":
		return "wss://s2.ripple.com:443/"
	case "testnet", "test":
		return "wss://s.altnet.rippletest.net:443/"
	case "devnet", "dev":
		return "wss://s.devnet.rippletest.net:443/"
	default:
		log.Fatalf("unknown network: %v", net)
	}
	return ""
}

func sendXrpAction(ctx *cli.Context) error {
	utils.SetLogger(ctx)
	initArgsSendXrp(ctx)
	initBridge()
	sendXRP()
	return nil
}

func isValidKeyType(ktype string) bool {
	switch ktype {
	case "ecdsa", "ed25519":
		return true
	default:
		return false
	}
}

func isECDSA(ktype string) bool {
	return ktype == "ecdsa"
}

func sendXRP() string {
	var key crypto.Key
	var sequence *uint32
	var err error

	switch {
	case seed != "":
		key, err = ripple.ImportKeyFromSeed(seed, keyType)
		if err != nil {
			log.Fatal("import key from seed failed", "err", err)
		}
		if isECDSA(keyType) {
			seq := uint32(keyseq)
			sequence = &seq
		}
	case prikey != "":
		if isECDSA(keyType) {
			key = crypto.NewECDSAKeyFromPrivKeyBytes(common.FromHex(prikey))
		} else {
			key = crypto.NewEd25519KeyFromPrivKeyBytes(common.FromHex(prikey))
		}
	default:
		log.Fatal("must specify seed or key")
	}

	from := ripple.GetAddress(key, sequence)
	log.Printf("sender address is %v", from)
	log.Printf("sender pubkey is %v", common.ToHex(key.Public(sequence)))

	txseq, err := b.GetSeq(nil, from)
	if err != nil {
		log.Fatal("get account sequence failed", "account", from, "err", err)
	}
	log.Info("start build tx", "from", from, "sequence", sequence, "txseq", txseq, "to", to, "amount", amount, "memo", memo)

	tx, _, _ := ripple.NewUnsignedPaymentTransaction(key, sequence, *txseq, to, toTag, amount, txfee, memo, "", false, false, false)

	stx, _, err := b.SignTransactionWithRippleKey(tx, key, sequence)
	if err != nil {
		log.Fatal("sign transaction failed", "err", err)
	}
	jsdata, _ := json.MarshalIndent(stx, "", " ")
	fmt.Printf("sign transaction success: %v\n", string(jsdata))

	txhash, err := b.SendTransaction(stx)
	if err != nil {
		log.Fatal("send transaction failed", "err", err)
	}
	fmt.Printf("Submited tx success: %v\n", txhash)
	return txhash
}
