package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	tronaddress "github.com/fbsobreira/gotron-sdk/pkg/address"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/tools/crypto"
)

/*

pubkey to address
./trontools pubkeyToAddress --pubkey 04d38309dfdfd9adf129287b68cf2e1f1124e0cbc40cc98f94e5f2d23c26712fa3b33d63280dd1448319a6a4f4111722d6b3a730ebe07652ed2b3770947b3de2e2
TXSxUhgSoHkHNLgip2kQRHXVT6BqoaqtvX

tron to eth
./trontools tronToEth --tron TXSxUhgSoHkHNLgip2kQRHXVT6BqoaqtvX
0xEB9A2A3502A78B27842DD2FB09E9514E3CE597ED

eth to tron
./trontools ethToTron --eth 0xEB9A2A3502A78B27842DD2FB09E9514E3CE597ED
TXSxUhgSoHkHNLgip2kQRHXVT6BqoaqtvX

*/

var app = cli.NewApp()

func initApp() {
	// Initialize the CLI app and start action
	app.Commands = []*cli.Command{
		ethToTronCommand,
		tronToEthCommand,
		pubkeyToAddressCommand,
	}
}

func main() {
	initApp()
	if err := app.Run(os.Args); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

var (
	ethFlag = &cli.StringFlag{
		Name:  "eth",
		Usage: "eth address",
		Value: "",
	}
	tronFlag = &cli.StringFlag{
		Name:  "tron",
		Usage: "tron address",
		Value: "",
	}
	pubkeyFlag = &cli.StringFlag{
		Name:  "pubkey",
		Usage: "pubkey hex",
		Value: "",
	}
)

var (
	ethToTronCommand = &cli.Command{
		Action:      ethToTron,
		Name:        "ethToTron",
		Usage:       "convert eth address to tron address",
		ArgsUsage:   " ",
		Description: ``,
		Flags: []cli.Flag{
			ethFlag,
		},
	}
	tronToEthCommand = &cli.Command{
		Action:      tronToEth,
		Name:        "tronToEth",
		Usage:       "convert tron address to eth address",
		ArgsUsage:   " ",
		Description: ``,
		Flags: []cli.Flag{
			tronFlag,
		},
	}
	pubkeyToAddressCommand = &cli.Command{
		Action:      pubkeyToAddress,
		Name:        "pubkeyToAddress",
		Usage:       "convert pubkey hex to tron address",
		ArgsUsage:   " ",
		Description: ``,
		Flags: []cli.Flag{
			pubkeyFlag,
		},
	}
)

func ethToTron(ctx *cli.Context) error {
	ethAddress := ctx.String(ethFlag.Name)

	tronaddr := tronaddress.Address(append([]byte{0x41}, common.HexToAddress(ethAddress).Bytes()...))
	fmt.Println(tronaddr.String())
	return nil
}

func tronToEth(ctx *cli.Context) error {
	tronAddress := ctx.String(tronFlag.Name)
	addr, err := tronaddress.Base58ToAddress(tronAddress)
	if err != nil {
		return err
	}
	ethaddr := common.BytesToAddress(addr.Bytes())
	fmt.Println(ethaddr.String())
	return nil
}

func pubkeyToAddress(ctx *cli.Context) error {
	pubkeyhex := ctx.String(pubkeyFlag.Name)
	pubkeyhex = strings.TrimPrefix(pubkeyhex, "0x")
	bz, err := hex.DecodeString(pubkeyhex)
	if err != nil {
		return err
	}
	ecdsaPub, err := crypto.UnmarshalPubkey(bz)
	if err != nil {
		return err
	}
	ethAddress := crypto.PubkeyToAddress(*ecdsaPub)
	fmt.Printf("%X\n", ethAddress)
	tronaddr := tronaddress.Address(append([]byte{0x41}, ethAddress.Bytes()...))
	fmt.Println(tronaddr)
	return nil
}
