package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/tokens/xrp"
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
)

func init() {
	flag.StringVar(&seed, "seed", "", "seed")
	flag.UintVar(&keyseq, "keyseq", 0, "key sequence")
	flag.StringVar(&to, "to", "", "destination address")
	flag.StringVar(&amount, "amount", "", "xrp amount")
	flag.StringVar(&memo, "memo", "", "memo")
	flag.StringVar(&net, "net", "testnet", "network")
}

func main() {
	b := xrp.NewCrossChainBridge(true)

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
	defer remote.Close()
	b.Remotes = append(b.Remotes, remote)

	key := xrp.ImportKeyFromSeed(seed, "ecdsa")
	keyseq := uint32(keyseq)

	from := xrp.GetAddress(key, &keyseq)
	txseq, err := b.GetSeq(from)
	if err != nil {
		log.Fatal(err)
	}

	tx, _, _ := xrp.NewUnsignedPaymentTransaction(key, &keyseq, txseq, to, amount, 10, "0x6D263DE8b5f755Ae0F0Bc87a5359836f18276E8C", "", false, false, false)

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
}
