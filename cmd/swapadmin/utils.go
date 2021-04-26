package main

import (
	"errors"

	"github.com/anyswap/CrossChain-Bridge/admin"
	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/urfave/cli/v2"
)

var (
	swapServer string

	commonAdminFlags = []cli.Flag{
		utils.SwapServerFlag,
		utils.KeystoreFileFlag,
		utils.PasswordFileFlag,
	}
)

func adminCall(method string, params []string) (result interface{}, err error) {
	rawTx, err := admin.Sign(method, params)
	if err != nil {
		return "", err
	}
	timeout := 300
	reqID := 1010
	err = client.RPCPostWithTimeoutAndID(&result, timeout, reqID, swapServer, "swap.AdminCall", rawTx)
	return result, err
}

func loadKeyStore(ctx *cli.Context) error {
	keyfile := ctx.String(utils.KeystoreFileFlag.Name)
	passfile := ctx.String(utils.PasswordFileFlag.Name)
	return admin.LoadKeyStore(keyfile, passfile)
}

func initSwapServer(ctx *cli.Context) error {
	swapServer = ctx.String(utils.SwapServerFlag.Name)
	if swapServer == "" {
		return errors.New("must specify swapserver")
	}
	return nil
}

func prepare(ctx *cli.Context) (err error) {
	err = loadKeyStore(ctx)
	if err != nil {
		return err
	}

	err = initSwapServer(ctx)
	if err != nil {
		return err
	}

	return nil
}
