package main

import (
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/urfave/cli/v2"
)

var (
	passSwapinOp  = "passswapin"
	passSwapoutOp = "passswapout"
	failSwapinOp  = "failswapin"
	failSwapoutOp = "failswapout"
)

var (
	manualCommand = &cli.Command{
		Action:    manual,
		Name:      "manual",
		Usage:     "manual manage swap",
		ArgsUsage: "<passswapin|failswapin|passswapout|failswapout> <txid> [memo]",
		Description: `
manual manage swap, pass or fail swap directly. memo is optional message for the reasons.
`,
		Flags: commonAdminFlags,
	}
)

func manual(ctx *cli.Context) error {
	utils.SetLogger(ctx)
	method := "manual"
	if !(ctx.NArg() == 2 || ctx.NArg() == 3) {
		_ = cli.ShowCommandHelp(ctx, method)
		fmt.Println()
		return fmt.Errorf("invalid arguments: %q", ctx.Args())
	}

	err := prepare(ctx)
	if err != nil {
		return err
	}

	operation := ctx.Args().Get(0)
	txid := ctx.Args().Get(1)

	var memo string
	if ctx.NArg() > 2 {
		memo = ctx.Args().Get(2)
	}

	switch operation {
	case passSwapinOp, passSwapoutOp, failSwapinOp, failSwapoutOp:
	default:
		return fmt.Errorf("unknown operation '%v'", operation)
	}

	log.Printf("[admin] manual %v %v", operation, txid)

	params := []string{operation, txid, memo}
	result, err := adminCall(method, params)

	log.Printf("result is '%v'", result)
	return err
}
