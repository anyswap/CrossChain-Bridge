package main

import (
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/urfave/cli/v2"
)

var (
	reverifyCommand = &cli.Command{
		Action:    reverify,
		Name:      "reverify",
		Usage:     "admin reverify",
		ArgsUsage: "<swapin|swapout> <txid> <pairID> <bind>",
		Description: `
admin reverify swap
`,
		Flags: commonAdminFlags,
	}
)

func reverify(ctx *cli.Context) error {
	utils.SetLogger(ctx)
	method := "reverify"
	if ctx.NArg() != 4 {
		_ = cli.ShowCommandHelp(ctx, method)
		fmt.Println()
		return fmt.Errorf("invalid arguments: %q", ctx.Args())
	}
	return reverifyOrReswap(ctx, method)
}
