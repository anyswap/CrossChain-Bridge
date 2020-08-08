package main

import (
	"encoding/json"
	"errors"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/rpc/rpcapi"
	"github.com/anyswap/CrossChain-Bridge/tools"
	"github.com/anyswap/CrossChain-Bridge/tools/crypto"
	"github.com/anyswap/CrossChain-Bridge/tools/keystore"
	"github.com/urfave/cli/v2"
)

var (
	swapServer string
	keyWrapper *keystore.Key
)

func adminCall(args *rpcapi.AdminCallArg) (result interface{}, err error) {
	data, _ := json.Marshal(args)
	sigHash := common.Keccak256Hash(data).Bytes()
	signature, err := crypto.Sign(sigHash, keyWrapper.PrivateKey)
	if err != nil {
		return "", err
	}
	args.Signature = signature
	err = client.RPCPost(&result, swapServer, "swap.AdminCall", args)
	return result, err
}

func loadKeyStore(ctx *cli.Context) error {
	keyfile := ctx.String(utils.KeystoreFileFlag.Name)
	passfile := ctx.String(utils.PasswordFileFlag.Name)
	key, err := tools.LoadKeyStore(keyfile, passfile)
	if err != nil {
		return err
	}
	keyWrapper = key
	log.Info("load keystore success", "address", keyWrapper.Address.String())
	return nil
}

func initSwapServer(ctx *cli.Context) error {
	swapServer = ctx.String(utils.SwapServerFlag.Name)
	if swapServer == "" {
		return errors.New("must specify swapserver")
	}
	return nil
}
