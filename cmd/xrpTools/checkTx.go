package main

import (
	"fmt"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/data"
	"github.com/anyswap/CrossChain-Bridge/tokens/ripple/rubblelabs/ripple/websockets"
	"github.com/urfave/cli/v2"
)

var (
	checkTxCommand = &cli.Command{
		Action:    checkTxAction,
		Name:      "checkTx",
		Usage:     "check transaction",
		ArgsUsage: "<txHash>",
		Flags: []cli.Flag{
			netFlag,
			apiAddressFlag,
		},
	}
)

func checkTxAction(ctx *cli.Context) error {
	utils.SetLogger(ctx)
	txHash := ctx.Args().Get(0)
	if txHash == "" {
		return fmt.Errorf("empty txHash argument")
	}
	net = ctx.String(netFlag.Name)
	apiAddress = ctx.String(apiAddressFlag.Name)
	if apiAddress == "" {
		apiAddress = initDefaultAPIAddress(net)
	}

	initBridge()

	if ok := checkTx(txHash); !ok {
		return fmt.Errorf("check tx failed")
	}
	if ok := checkStatus(txHash); !ok {
		return fmt.Errorf("check tx status failed")
	}
	return nil
}

func checkTx(txHash string) bool {
	tx, err := b.GetTransaction(txHash)
	if err != nil {
		log.Printf("Get tx failed, %v", err)
		return false
	}

	txres, ok := tx.(*websockets.TxResult)
	if !ok {
		log.Printf("Tx res type error")
		return false
	}

	if !txres.TransactionWithMetaData.MetaData.TransactionResult.Success() {
		log.Printf("Tx result: %v", txres.TransactionWithMetaData.MetaData.TransactionResult)
		return false
	}

	payment, ok := txres.TransactionWithMetaData.Transaction.(*data.Payment)
	if !ok || payment.GetTransactionType() != data.PAYMENT {
		log.Printf("Not a payment transaction")
		return false
	}

	bind, ok := ripple.GetBindAddressFromMemos(payment)
	if !ok {
		log.Printf("Get bind address failed")
		return false
	}
	log.Printf("Bind address: %v", bind)

	log.Println("Tx success!")
	return true
}

func checkStatus(txHash string) bool {
	status, err := b.GetTransactionStatus(txHash)
	fmt.Printf("%+v\n%v\n", status, err)

	log.Println("Tx status success!")
	return true
}
