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
		ArgsUsage: "<passswapin|failswapin|passswapout|failswapout> <txid> <pairID> <bind> [memo]",
		Description: `
manual manage swap, pass or fail swap directly. memo is optional message for the reasons.
`,
		Flags: commonAdminFlags,
	}
)

func manual(ctx *cli.Context) error {
	utils.SetLogger(ctx)
	method := "manual"
	if !(ctx.NArg() == 4 || ctx.NArg() == 5) {
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
	pairID := ctx.Args().Get(2)
	bind := ctx.Args().Get(3)

	var memo string
	if ctx.NArg() > 4 {
		memo = ctx.Args().Get(4)
	}

	switch operation {
	case passSwapinOp, passSwapoutOp, failSwapinOp, failSwapoutOp:
	default:
		return fmt.Errorf("unknown operation '%v'", operation)
	}

	log.Printf("[admin] manual %v %v %v %v", operation, txid, pairID, bind)

	params := []string{operation, txid, pairID, bind}
	if memo != "" {
		params = append(params, memo)
	}
	result, err := adminCall(method, params)

	log.Printf("result is '%v'", result)
	return err
}
