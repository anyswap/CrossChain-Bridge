package main

import (
	"github.com/urfave/cli/v2"
)

var (
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

	receiverFlag = &cli.StringFlag{
		Name:  "to",
		Usage: "to address",
	}
	valueFlag = &cli.StringFlag{
		Name:  "value",
		Usage: "value of unit wei",
	}
	inputDataFlag = &cli.StringFlag{
		Name:  "input",
		Usage: "tx input data",
	}

	gasLimitFlag = &cli.Uint64Flag{
		Name:  "gasLimit",
		Usage: "gas limit in transaction",
	}
	gasPriceFlag = &cli.StringFlag{
		Name:  "gasPrice",
		Usage: "gas price in transaction",
	}
	accountNonceFlag = &cli.Uint64Flag{
		Name:  "nonce",
		Usage: "nonce in transaction",
	}
)
