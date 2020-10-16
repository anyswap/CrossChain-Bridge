package main

import (
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/urfave/cli/v2"
)

var (
	bigvalueCommand = &cli.Command{
		Action:    bigvalue,
		Name:      "bigvalue",
		Usage:     "admin bigvalue",
		ArgsUsage: "<passswapin|passswapout> <txid> <pairID> <bind>",
		Description: `
admin bigvalue swap
`,
		Flags: commonAdminFlags,
	}
)

func bigvalue(ctx *cli.Context) error {
	utils.SetLogger(ctx)
	method := "bigvalue"
	if ctx.NArg() != 4 {
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

	switch operation {
	case passSwapinOp, passSwapoutOp:
	default:
		return fmt.Errorf("unknown operation '%v'", operation)
	}

	log.Printf("admin bigvalue: %v %v %v %v", operation, txid, pairID, bind)

	params := []string{operation, txid, pairID, bind}
	result, err := adminCall(method, params)

	log.Printf("result is '%v'", result)
	return err
}
