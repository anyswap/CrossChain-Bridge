package main

import (
	"fmt"
	"time"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/rpc/rpcapi"
	"github.com/urfave/cli/v2"
)

var (
	blacklistCommand = &cli.Command{
		Action:    blacklist,
		Name:      "blacklist",
		Usage:     "admin blacklist",
		ArgsUsage: "<add|remove|query> <address>",
		Description: `
admin blacklist
`,
		Flags: []cli.Flag{
			utils.SwapServerFlag,
			utils.KeystoreFileFlag,
			utils.PasswordFileFlag,
		},
	}
)

func blacklist(ctx *cli.Context) error {
	utils.SetLogger(ctx)
	method := "blacklist"
	if ctx.NArg() != 2 {
		_ = cli.ShowCommandHelp(ctx, method)
		fmt.Println()
		return fmt.Errorf("invalid arguments: %q", ctx.Args())
	}

	err := loadKeyStore(ctx)
	if err != nil {
		return err
	}

	err = initSwapServer(ctx)
	if err != nil {
		return err
	}

	operation := ctx.Args().Get(0)
	address := ctx.Args().Get(1)

	switch operation {
	case "add", "remove", "query":
	default:
		return fmt.Errorf("unknown operation '%v'", operation)
	}

	log.Printf("admin blacklist: %v %v", operation, address)

	args := &rpcapi.AdminCallArg{
		Method:    method,
		Params:    []string{operation, address},
		Timestamp: time.Now().Unix(),
	}
	result, err := adminCall(args)

	log.Printf("result is '%v'", result)
	return err
}
