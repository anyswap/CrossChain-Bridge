package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/base58"
	"github.com/urfave/cli/v2"
)

var (
	// nolint:lll // allow long line of example
	sendBtcCommand = &cli.Command{
		Action:    sendBtc,
		Name:      "sendbtc",
		Usage:     "send btc",
		ArgsUsage: " ",
		Description: `
send btc command, sign tx with WIF or private key.

Example:

./swaptools sendbtc --gateway http://1.2.3.4:5555 --net testnet3 --wif ./wif.txt --from maiApsjjnceZ7Cx1UMj344JRU3R8A2Say6 --to mtc4xaZgJJZpN6BdoWk7pHFho1GTUnd5aP --value 10000 --to mfwanCuht2b4Lvb5XTds4Rvzy3jZ2ZWraL --value 20000 --memo "test send btc" --dryrun
`,
		Flags: []cli.Flag{
			utils.GatewayFlag,
			networkFlag,
			wifFileFlag,
			priKeyFileFlag,
			senderFlag,
			receiverSliceFlag,
			valueSliceFlag,
			memoFlag,
			relayFeePerKbFlag,
			dryRunFlag,
		},
	}

	networkFlag = &cli.StringFlag{
		Name:  "net",
		Usage: "network identifier, ie. mainnet, testnet3",
		Value: "testnet3",
	}
	wifFileFlag = &cli.StringFlag{
		Name:  "wif",
		Usage: "WIF file",
	}
	priKeyFileFlag = &cli.StringFlag{
		Name:  "pri",
		Usage: "private key file",
	}
	senderFlag = &cli.StringFlag{
		Name:  "from",
		Usage: "from address",
	}
	receiverSliceFlag = &cli.StringSliceFlag{
		Name:  "to",
		Usage: "to address slice",
	}
	valueSliceFlag = &cli.Int64SliceFlag{
		Name:  "value",
		Usage: "satoshi value slice",
	}
	memoFlag = &cli.StringFlag{
		Name:  "memo",
		Usage: "tx memo",
	}
	relayFeePerKbFlag = &cli.Int64Flag{
		Name:  "fee",
		Usage: "relay fee per kilo bytes",
		Value: 2000,
	}
	dryRunFlag = &cli.BoolFlag{
		Name:  "dryrun",
		Usage: "dry run",
	}
)

var (
	btcBridge *btc.Bridge

	gateway       string
	netID         string
	wifFile       string
	priFile       string
	sender        string
	receivers     []string
	amounts       []int64
	memo          string
	relayFeePerKb int64
	dryRun        bool
)

func initArgs(ctx *cli.Context) {
	gateway = ctx.String(utils.GatewayFlag.Name)
	netID = ctx.String(networkFlag.Name)
	wifFile = ctx.String(wifFileFlag.Name)
	priFile = ctx.String(priKeyFileFlag.Name)
	sender = ctx.String(senderFlag.Name)
	receivers = ctx.StringSlice(receiverSliceFlag.Name)
	amounts = ctx.Int64Slice(valueSliceFlag.Name)
	memo = ctx.String(memoFlag.Name)
	relayFeePerKb = ctx.Int64(relayFeePerKbFlag.Name)
	dryRun = ctx.Bool(dryRunFlag.Name)

	if netID == "" {
		log.Fatal("must specify '-net' flag")
	}
	if wifFile == "" && priFile == "" {
		log.Fatal("must specify '-wif' or '-pri' flag")
	}
	if sender == "" {
		log.Fatal("must specify '-from' flag")
	}
	if len(receivers) == 0 {
		log.Fatal("must specify '-to' flag")
	}
	if len(amounts) == 0 {
		log.Fatal("must specify '-value' flag")
	}
	if len(receivers) != len(amounts) {
		log.Fatal("count of receivers and values are not equal")
	}
}

func sendBtc(ctx *cli.Context) error {
	utils.SetLogger(ctx)
	initArgs(ctx)

	initBridge()

	wifStr := loadWIFForAddress()

	rawTx, err := btcBridge.BuildTransaction(sender, receivers, amounts, memo, relayFeePerKb)
	if err != nil {
		log.Fatal("BuildRawTransaction error", "err", err)
	}

	signedTx, txHash, err := btcBridge.SignTransactionWithWIF(rawTx, wifStr)
	if err != nil {
		log.Fatal("SignTransaction failed", "err", err)
	}
	log.Info("SignTransaction success", "txHash", txHash)

	fmt.Println(btc.AuthoredTxToString(signedTx, true))

	if !dryRun {
		_, err = btcBridge.SendTransaction(signedTx)
		if err != nil {
			log.Error("SendTransaction failed", "err", err)
		}
	} else {
		log.Info("------------ dry run, does not sendtx -------------")
	}
	return nil
}

func initBridge() {
	btcBridge = btc.NewCrossChainBridge(true)
	btcBridge.ChainConfig = &tokens.ChainConfig{
		BlockChain: "Bitcoin",
		NetID:      netID,
	}
	btcBridge.GatewayConfig = &tokens.GatewayConfig{
		APIAddress: []string{gateway},
	}
}

func loadWIFForAddress() string {
	var wifStr string
	if wifFile != "" {
		wifdata, err := ioutil.ReadFile(wifFile)
		if err != nil {
			log.Fatal("Read WIF file failed", "err", err)
		}
		wifStr = strings.TrimSpace(string(wifdata))
	} else {
		pridata, err := ioutil.ReadFile(priFile)
		if err != nil {
			log.Fatal("Read private key file failed", "err", err)
		}
		priKey := strings.TrimSpace(string(pridata))
		var pribs []byte
		if common.IsHex(priKey) {
			pribs, err = hex.DecodeString(priKey)
			if err != nil {
				log.Fatal("failed to decode hex private key string")
			}
		} else {
			pribs, _, err = base58.CheckDecode(priKey)
			if err != nil {
				pribs = base58.Decode(priKey)
			}
		}
		pri, _ := btcec.PrivKeyFromBytes(btcec.S256(), pribs)
		wif, err := btcutil.NewWIF(pri, btcBridge.GetChainParams(), true)
		if err != nil {
			log.Fatal("failed to parse private key")
		}
		wifStr = wif.String()
	}
	wif, err := btcutil.DecodeWIF(wifStr)
	if err != nil {
		log.Fatal("failed to decode WIF to verify")
	}
	pkdata := wif.SerializePubKey()
	pkaddr, _ := btcutil.NewAddressPubKeyHash(btcutil.Hash160(pkdata), btcBridge.GetChainParams())
	if pkaddr.EncodeAddress() != sender {
		log.Fatal("address mismatch", "decoded", pkaddr.EncodeAddress(), "from", sender)
	}
	return wifStr
}
