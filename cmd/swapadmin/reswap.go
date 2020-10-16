package main

import (
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/urfave/cli/v2"
)

const (
	swapinOp  = "swapin"
	swapoutOp = "swapout"
	forceFlag = "--force"
)

var (
	reswapCommand = &cli.Command{
		Action:    reswap,
		Name:      "reswap",
		Usage:     "admin reswap",
		ArgsUsage: "<swapin|swapout> <txid> <pairID> <bind> [--force]",
		Description: `
admin reswap swap
`,
		Flags: commonAdminFlags,
	}
)

func reswap(ctx *cli.Context) error {
	utils.SetLogger(ctx)
	method := "reswap"
	if !(ctx.NArg() == 4 || ctx.NArg() == 5) {
		_ = cli.ShowCommandHelp(ctx, method)
		fmt.Println()
		return fmt.Errorf("invalid arguments: %q", ctx.Args())
	}
	return reverifyOrReswap(ctx, method)
}

func reverifyOrReswap(ctx *cli.Context, method string) error {
	err := prepare(ctx)
	if err != nil {
		return err
	}

	operation := ctx.Args().Get(0)
	txid := ctx.Args().Get(1)
	pairID := ctx.Args().Get(2)
	bind := ctx.Args().Get(3)

	var forceOpt string
	if ctx.NArg() > 4 {
		forceOpt = ctx.Args().Get(4)
		if forceOpt != forceFlag {
			return fmt.Errorf("wrong force flag %v, must be %v", forceOpt, forceFlag)
		}
	}

	switch operation {
	case swapinOp, swapoutOp:
	default:
		return fmt.Errorf("unknown operation '%v'", operation)
	}

	params := []string{operation, txid, pairID, bind}
	if forceOpt != "" {
		params = append(params, forceOpt)
		log.Printf("admin %v: %v %v %v %v %v", method, operation, txid, pairID, bind, forceOpt)
	} else {
		log.Printf("admin %v: %v %v %v %v", method, operation, txid, pairID, bind)
	}

	result, err := adminCall(method, params)

	log.Printf("result is '%v'", result)
	return err
}
