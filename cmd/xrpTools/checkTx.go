package main

import (
	"fmt"
	"log"

	"github.com/anyswap/CrossChain-Bridge/tokens/xrp"
	"github.com/rubblelabs/ripple/data"
	"github.com/rubblelabs/ripple/websockets"
)

func checkTx(txHash string) bool {
	tx, err := b.GetTransaction(txHash)
	if err != nil {
		log.Printf("Get tx failed, %v", err)
		return false
	}

	txres, ok := tx.(*websockets.TxResult)
	if !ok {
		// unexpected
		log.Printf("Tx res type error")
		return false
	}

	if txres.TransactionWithMetaData.MetaData.TransactionResult != 0 {
		log.Printf("Tx result: %v", txres.TransactionWithMetaData.MetaData.TransactionResult)
		return false
	}

	payment, ok := txres.TransactionWithMetaData.Transaction.(*data.Payment)
	if !ok || payment.GetTransactionType() != data.PAYMENT {
		log.Printf("Not a payment transaction")
		return false
	}

	bind, ok := xrp.GetBindAddressFromMemos(payment)
	if !ok {
		log.Printf("Get bind address failed")
		return false
	}
	log.Printf("Bind address: %v\n", bind)

	log.Println("Tx success!")
	return true
}

func checkStatus(txHash string) bool {
	status := b.GetTransactionStatus(txHash)
	fmt.Printf("%+v\n", status)

	return true
}
