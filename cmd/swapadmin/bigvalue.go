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
		ArgsUsage: "<passswapin|passswapout> <txid>",
		Description: `
admin bigvalue
`,
		Flags: []cli.Flag{
			utils.SwapServerFlag,
			utils.KeystoreFileFlag,
			utils.PasswordFileFlag,
		},
	}
)

func bigvalue(ctx *cli.Context) error {
	utils.SetLogger(ctx)
	method := "bigvalue"
	if ctx.NArg() != 2 {
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

	switch operation {
	case "passswapin", "passswapout":
	default:
		return fmt.Errorf("unknown operation '%v'", operation)
	}

	log.Printf("admin bigvalue: %v %v", operation, txid)

	params := []string{operation, txid}
	result, err := adminCall(method, params)

	log.Printf("result is '%v'", result)
	return err
}
