package main

import (
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/urfave/cli/v2"
)

var (
	blacklistCommand = &cli.Command{
		Action:    blacklist,
		Name:      "blacklist",
		Usage:     "admin blacklist",
		ArgsUsage: "<add|remove|query> <address> <pairID>",
		Description: `
admin blacklist
`,
		Flags: commonAdminFlags,
	}
)

func blacklist(ctx *cli.Context) error {
	utils.SetLogger(ctx)
	method := "blacklist"
	if ctx.NArg() != 3 {
		_ = cli.ShowCommandHelp(ctx, method)
		fmt.Println()
		return fmt.Errorf("invalid arguments: %q", ctx.Args())
	}

	err := prepare(ctx)
	if err != nil {
		return err
	}

	operation := ctx.Args().Get(0)
	address := ctx.Args().Get(1)
	pairID := ctx.Args().Get(2)

	switch operation {
	case "add", "remove", "query":
	default:
		return fmt.Errorf("unknown operation '%v'", operation)
	}

	log.Printf("admin blacklist: %v %v %v", operation, address, pairID)

	params := []string{operation, address, pairID}
	result, err := adminCall(method, params)

	log.Printf("result is '%v'", result)
	return err
}
