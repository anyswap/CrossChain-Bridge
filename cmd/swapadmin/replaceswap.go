package main

import (
	"fmt"
	"math/big"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/urfave/cli/v2"
)

var (
	replaceswapCommand = &cli.Command{
		Action:    replaceswap,
		Name:      "replaceswap",
		Usage:     "admin replace swap",
		ArgsUsage: "<swapin|swapout> <txid> <pairID> <bind> [gasPrice]",
		Description: `
admin replace swap with higher gas price
`,
		Flags: commonAdminFlags,
	}
)

func replaceswap(ctx *cli.Context) error {
	utils.SetLogger(ctx)
	method := "replaceswap"
	if !(ctx.NArg() == 4 || ctx.NArg() == 5) {
		_ = cli.ShowCommandHelp(ctx, method)
		fmt.Println()
		return fmt.Errorf("invalid number arguments: %q", ctx.Args())
	}

	err := prepare(ctx)
	if err != nil {
		return err
	}

	operation := ctx.Args().Get(0)
	txid := ctx.Args().Get(1)
	pairID := ctx.Args().Get(2)
	bind := ctx.Args().Get(3)

	var gasPriceStr string
	if ctx.NArg() > 4 {
		gasPriceStr = ctx.Args().Get(4)
		gasPrice, ok := new(big.Int).SetString(gasPriceStr, 0)
		if !ok {
			return fmt.Errorf("wrong gas price: %v", gasPriceStr)
		}
		if gasPrice.Cmp(big.NewInt(1e13)) > 0 {
			return fmt.Errorf("gas price is too large (> 10000 gwei): %v", gasPriceStr)
		}
	}

	switch operation {
	case swapinOp, swapoutOp:
	default:
		return fmt.Errorf("unknown operation '%v'", operation)
	}

	params := []string{operation, txid, pairID, bind, gasPriceStr}
	log.Printf("admin %v: %v %v %v %v %v", method, operation, txid, pairID, bind, gasPriceStr)

	result, err := adminCall(method, params)

	log.Printf("result is '%v'", result)
	return err
}
