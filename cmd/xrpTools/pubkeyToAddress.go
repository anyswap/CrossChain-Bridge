package main

import (
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple"
	"github.com/urfave/cli/v2"
)

var (
	pubkeyToAddressCommand = &cli.Command{
		Action:    pubkeyToAddress,
		Name:      "pubkeyToAddress",
		Usage:     "convert public key to address",
		ArgsUsage: "<pubkey>",
		Flags:     []cli.Flag{},
	}
)

func pubkeyToAddress(ctx *cli.Context) error {
	utils.SetLogger(ctx)
	pubkey := ctx.Args().Get(0)
	if pubkey == "" {
		return fmt.Errorf("empty public key argument")
	}
	addr, err := ripple.PublicKeyHexToAddress(pubkey)
	if err != nil {
		log.Fatalf("%v", err)
	}
	fmt.Printf("address: %v\n", addr)
	return nil
}
