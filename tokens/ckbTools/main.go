package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/nervosnetwork/ckb-sdk-go/rpc"
	"github.com/nervosnetwork/ckb-sdk-go/types"
)

func main() {
	client, err := rpc.DialWithIndexer("https://mainnet.ckb.dev/rpc", "https://mainnet.ckb.dev/indexer")
	if err != nil {
		log.Fatalf("create rpc client error: %v", err)
	}

	ctx := context.Background()

	// 获取块高
	tipheader, err := client.GetTip(ctx)
	checkError(err)
	fmt.Printf("Tip header: %+v\n", tipheader)

	// 查余额
	fmt.Printf("\n====== GetCellsCapacity ======\n")
	args, err := hex.DecodeString("12a66c725a720c0ee0204e70b3731cbb7cde01fb") // ckb1qyqp9fnvwfd8yrqwuqsyuu9nwvwtklx7q8aszywpu4 // 主网账户，注意安全！
	checkError(err)
	searchKey := &indexer.SearchKey{
		Script: &types.Script{
			CodeHash: types.HexToHash("0x9bd7e06f3ecf4be0f2fcd2188b23f1b9fcc88e5d4b65a8637b17723bbda3cce8"),
			HashType: types.HashTypeType,
			Args:     args,
		},
		ScriptType: indexer.ScriptTypeLock,
	}
	capacity, err := client.GetCellsCapacity(ctx, searchKey)
	checkError(err)
	fmt.Printf("Capacity: %+v\n", capacity)

	// 查交易
	fmt.Printf("\n====== GetTransaction ======\n")
	txhash := types.HexToHash("0xc00bdb92d9a29cfc78cba9becdc69ff8080ba27d1e54257205c49ff77c7439ee")
	txstatus, err := client.GetTransaction(ctx, txhash)
	checkError(err)
	fmt.Printf("Tx: %v\nStatus: %v\n", txstatus.Transaction, txstatus.TxStatus)

	// 获取 cell
	fmt.Printf("\n====== GetCells ======\n")
	var LiveCells = make(map[types.OutPoint]*indexer.LiveCell)
	order := indexer.SearchOrder("asc") // "desc"
	limit := uint64(10)
	cellCursor := "" // "0x409bd7e06f3ecf4be0f2fcd2188b23f1b9fcc88e5d4b65a8637b17723bbda3cce8011400000012a66c725a720c0ee0204e70b3731cbb7cde01fb00000000003c6dc50000000100000000"
	cells, err := client.GetCells(ctx, searchKey, order, limit, cellCursor)
	checkError(err)
	fmt.Printf("Cells: %+v\n", cells)
	for _, obj := range cells.Objects {
		fmt.Printf("Object: %+v\n", obj)
		fmt.Printf("OutPoint: %+v\n", obj.OutPoint)
		LiveCells[*obj.OutPoint] = obj
	}

	// get cells again alert when duplicate
	cells2, err := client.GetCells(ctx, searchKey, order, limit, cellCursor)
	checkError(err)
	for _, obj := range cells2.Objects {
		if LiveCells[*obj.OutPoint] != nil {
			fmt.Printf("!!! Duplicate cell: %+v\n", obj.OutPoint)
		}
	}

	// 搜索交易
	fmt.Printf("\n====== GetTransactions ======\n")
	txCursor := "" // 0x809bd7e06f3ecf4be0f2fcd2188b23f1b9fcc88e5d4b65a8637b17723bbda3cce8011400000012a66c725a720c0ee0204e70b3731cbb7cde01fb00000000003c6dc5000000010000000001
	txs, err := client.GetTransactions(ctx, searchKey, order, limit, txCursor)
	checkError(err)
	fmt.Printf("Transactions: %+v\n", txs)
	for _, obj := range txs.Objects {
		fmt.Printf("Transaction: %+v\n", obj)
	}
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
