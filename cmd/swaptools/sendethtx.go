package main

import (
	"math/big"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/eth"
	"github.com/anyswap/CrossChain-Bridge/tools"
	"github.com/anyswap/CrossChain-Bridge/tools/keystore"
	"github.com/anyswap/CrossChain-Bridge/types"
	"github.com/urfave/cli/v2"
)

var (
	// nolint:lll // allow long line of example
	sendEthTxCommand = &cli.Command{
		Action:    sendEthTx,
		Name:      "sendethtx",
		Usage:     "send eth transaction",
		ArgsUsage: " ",
		Description: `
send eth tx command, sign tx with keystore and password file.

Example:

./swaptools sendethtx --gateway http://1.2.3.4:5555 --keystore ./UTC.json --password ./password.txt --from 0x1111111111111111111111111111111111111111 --to 0x2222222222222222222222222222222222222222 --value 1000000000000000000 --input 0x0123456789 --dryrun
`,
		Flags: []cli.Flag{
			utils.GatewayFlag,
			utils.KeystoreFileFlag,
			utils.PasswordFileFlag,
			senderFlag,
			receiverFlag,
			valueFlag,
			inputDataFlag,
			gasLimitFlag,
			gasPriceFlag,
			accountNonceFlag,
			dryRunFlag,
		},
	}
)

type ethTxSender struct {
	gateway      string
	keystoreFile string
	passwordFile string
	sender       string
	receiver     string
	dryRun       bool

	value      *big.Int
	input      []byte
	keyWrapper *keystore.Key
}

var (
	ethBridge *eth.Bridge
	ethSender = &ethTxSender{}
	ethExtra  = &tokens.EthExtraArgs{}
)

func (ets *ethTxSender) initArgs(ctx *cli.Context) {
	ets.gateway = ctx.String(utils.GatewayFlag.Name)
	ets.keystoreFile = ctx.String(utils.KeystoreFileFlag.Name)
	ets.passwordFile = ctx.String(utils.PasswordFileFlag.Name)
	ets.sender = ctx.String(senderFlag.Name)
	ets.receiver = ctx.String(receiverFlag.Name)
	ets.dryRun = ctx.Bool(dryRunFlag.Name)

	if ets.keystoreFile == "" || ets.passwordFile == "" {
		log.Fatal("must specify '-keystore' and '-password' flag")
	}
	if ets.sender == "" {
		log.Fatal("must specify '-from' flag")
	}

	if ctx.IsSet(valueFlag.Name) {
		value, err := common.GetBigIntFromStr(ctx.String(valueFlag.Name))
		if err != nil {
			log.Fatalf("wrong value. %v", err)
		}
		ets.value = value
	}
	if ctx.IsSet(inputDataFlag.Name) {
		ets.input = common.FromHex(ctx.String(inputDataFlag.Name))
	}

	if ctx.IsSet(gasLimitFlag.Name) {
		gasLimitValue := ctx.Uint64(gasLimitFlag.Name)
		ethExtra.Gas = &gasLimitValue
		log.Printf("gas limit is set to %v", gasLimitValue)
	}
	if ctx.IsSet(gasPriceFlag.Name) {
		gasPriceValue, err := common.GetBigIntFromStr(ctx.String(gasPriceFlag.Name))
		if err != nil {
			log.Fatalf("wrong gas price. %v", err)
		}
		ethExtra.GasPrice = gasPriceValue
		log.Printf("gas price is set to %v", gasPriceValue)
	}
	if ctx.IsSet(accountNonceFlag.Name) {
		nonceValue := ctx.Uint64(accountNonceFlag.Name)
		ethExtra.Nonce = &nonceValue
		log.Printf("account nonce is set to %v", nonceValue)
	}

	log.Info("initArgs finished", "gateway", ets.gateway,
		"from", ets.sender, "to", ets.receiver, "value", ets.value,
		"input", common.ToHex(ets.input), "dryRun", ets.dryRun)
}

func (ets *ethTxSender) doInit() {
	var err error
	ets.keyWrapper, err = tools.LoadKeyStore(ets.keystoreFile, ets.passwordFile)
	if err != nil {
		log.Fatal("load keystore failed", "err", err)
	}

	keyAddr := ets.keyWrapper.Address.String()
	if !strings.EqualFold(keyAddr, ets.sender) {
		log.Fatal("sender mismatch", "sender", ets.sender, "keyAddr", keyAddr)
	}
	log.Info("load keystore success", "address", keyAddr)

	ets.initBridge()
}

func (ets *ethTxSender) initBridge() {
	ethBridge = eth.NewCrossChainBridge(true)
	ethBridge.ChainConfig = &tokens.ChainConfig{
		BlockChain: "Ethereum",
		NetID:      "custom",
	}
	ethBridge.GatewayConfig = &tokens.GatewayConfig{
		APIAddress: []string{ets.gateway},
	}
	ethBridge.VerifyChainID()
}

func (ets *ethTxSender) buildTx() (rawTx interface{}, err error) {
	args := &tokens.BuildTxArgs{
		From:  ets.sender,
		To:    ets.receiver,
		Value: ets.value,
		Input: &ets.input,
		Extra: &tokens.AllExtras{
			EthExtra: ethExtra,
		},
	}
	return ethBridge.BuildRawTransaction(args)
}

func sendEthTx(ctx *cli.Context) error {
	utils.SetLogger(ctx)
	ethSender.initArgs(ctx)

	ethSender.doInit()

	rawTx, err := ethSender.buildTx()
	if err != nil {
		log.Fatal("BuildRawTransaction error", "err", err)
	}

	signedTx, txHash, err := ethBridge.SignTransactionWithPrivateKey(rawTx, ethSender.keyWrapper.PrivateKey)
	if err != nil {
		log.Fatal("SignTransaction failed", "err", err)
	}
	log.Info("SignTransaction success", "txHash", txHash)

	tx, _ := signedTx.(*types.Transaction)
	tx.PrintPretty()

	if !ethSender.dryRun {
		_, err = ethBridge.SendTransaction(signedTx)
		if err != nil {
			log.Error("SendTransaction failed", "err", err)
		}
	} else {
		log.Info("------------ dry run, does not sendtx -------------")
	}
	return nil
}
