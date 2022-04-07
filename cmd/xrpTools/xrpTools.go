package main

import (
	"fmt"
	"os"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/eth"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/websockets"
	"github.com/urfave/cli/v2"
)

var (
	clientIdentifier = "xrptools"
	gitCommit        = ""
	gitDate          = ""
	app              = utils.NewApp(clientIdentifier, gitCommit, gitDate, "the xrptools command line interface")

	keyType    string
	prikey     string
	seed       string
	keyseq     uint
	to         string
	toTag      *uint32
	memo       string
	amount     string
	txfee      int64
	apiAddress string
	net        string
	b          *ripple.Bridge
	startScan  uint64

	keyTypeFlag = &cli.StringFlag{
		Name:  "keytype",
		Usage: "key type (eg. ecdsa, ed25519)",
		Value: "ecdsa",
	}
	keyFlag = &cli.StringFlag{
		Name:  "key",
		Usage: "private key",
	}
	seedFlag = &cli.StringFlag{
		Name:  "seed",
		Usage: "private key seed",
	}
	keyseqFlag = &cli.UintFlag{
		Name:  "keyseq",
		Usage: "private key sequence",
		Value: 0,
	}
	toFlag = &cli.StringFlag{
		Name:  "to",
		Usage: "send xrp to",
	}
	toTagFlag = &cli.UintFlag{
		Name:  "totag",
		Usage: "destination tag",
	}
	amountFlag = &cli.StringFlag{
		Name:  "amount",
		Usage: "send xrp amount (in drop)",
	}
	feeFlag = &cli.Int64Flag{
		Name:  "fee",
		Usage: "tx fee",
		Value: 10,
	}
	memoFlag = &cli.StringFlag{
		Name:  "memo",
		Usage: "swapin bind address",
	}
	netFlag = &cli.StringFlag{
		Name:  "net",
		Usage: "submit on network",
		Value: "testnet",
	}
	apiAddressFlag = &cli.StringFlag{
		Name:  "remote",
		Usage: "ripple api provider",
	}
	startScanFlag = &cli.Uint64Flag{
		Name:  "startscan",
		Usage: "start scan",
		Value: uint64(13880345),
	}

	sendXRPCommand = &cli.Command{
		Action: sendXrpAction,
		Name:   "sendxrp",
		Usage:  "sendxrp",
		Flags: []cli.Flag{
			keyTypeFlag,
			keyFlag,
			seedFlag,
			keyseqFlag,
			toFlag,
			toTagFlag,
			amountFlag,
			feeFlag,
			memoFlag,
			netFlag,
			apiAddressFlag,
		},
	}
	scanCommand = &cli.Command{
		Action: scanTxAction,
		Name:   "scan",
		Usage:  "scan ripple ledgers and txs",
		Flags: []cli.Flag{
			netFlag,
			apiAddressFlag,
			startScanFlag,
		},
	}
)

func main() {
	initApp()
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initApp() {
	app.Action = xrptools
	app.HideVersion = true
	app.Commands = []*cli.Command{
		pubkeyToAddressCommand,
		sendXRPCommand,
		scanCommand,
		checkTxCommand,
	}
	app.Flags = []cli.Flag{
		utils.VerbosityFlag,
		utils.JSONFormatFlag,
		utils.ColorFormatFlag,
	}
}

func initBridge() {
	tokens.DstBridge = eth.NewCrossChainBridge(false)
	b = ripple.NewCrossChainBridge(true)
	b.Remotes = make(map[string]*websockets.Remote)
	remote, err := websockets.NewRemote(apiAddress)
	if err != nil || remote == nil {
		log.Fatal("Cannot connect to ripple")
	}
	log.Printf("Connected to remote api %v", apiAddress)
	b.Remotes[apiAddress] = remote
}

func xrptools(ctx *cli.Context) error {
	utils.SetLogger(ctx)
	if ctx.NArg() > 0 {
		return fmt.Errorf("invalid command: %q", ctx.Args().Get(0))
	}

	_ = cli.ShowAppHelp(ctx)
	fmt.Println()
	return fmt.Errorf("please specify a sub command to run")
}
