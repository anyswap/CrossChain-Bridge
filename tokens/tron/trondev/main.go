package main

import (
	"context"
	"fmt"
	"log"
	//"math/big"
	"time"

	"google.golang.org/grpc"

	tronaddress "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	//"github.com/fbsobreira/gotron-sdk/pkg/common"
	//"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"

)

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func divide() {
	fmt.Printf("\n\n============================================================\n\n")
}

//var mainnetendpoint = "3.225.171.164:50051"
var mainnetendpoint = "grpc.trongrid.io:50051"
var testnetendpoint = "grpc.shasta.trongrid.io:50051"
var timeout = time.Second * 15

func main() {
	cli := client.NewGrpcClientWithTimeout(testnetendpoint, timeout)
	err := cli.Start(grpc.WithInsecure())
	checkError(err)
	defer cli.Stop()

	/*
		res, err := cli.GetNowBlock()
		checkError(err)
		fmt.Printf("Block number: %+v\n", res.BlockHeader.RawData.Number)

		divide()
	*/
	/*
		Test private key
		d8ea0b60ec7585c5b42742102e3d7b19eddbd54b0d538aca86e81d3e00886795
		ETH address
		0x35eE5830f802FD21780514A33420cA2c500d2232
		Tron address
		TEtNLh69XnK9Fs8suCogK3sRrWJbQHah4k
	*/

	/*
		acct, err := cli.GetAccount("TEtNLh69XnK9Fs8suCogK3sRrWJbQHah4k")
		checkError(err)
		fmt.Printf("Account: %+v\n", acct)
		fmt.Printf("Account balance: %v\n", acct.Balance)
		fmt.Printf("Account votes: %v\n", acct.Votes)
		fmt.Printf("Account frozen: %v\n", acct.Frozen)
		fmt.Printf("Account account resource:: %v\n", acct.AccountResource)
	*/
	/*
		查询未激活的地址会报 error: account not found
		向未激活的账户转账会多扣除发送者 0.1 trx ($ 0.008)
	*/
	/*
		unit = sun
		1 trx = 1e6 sun
		balance 是可用余额，不包括 frozen 部分
	*/
	/*
		所有的交易都需要带宽，系统分配的带宽每天只有 5000, 冻结 trx 获得更多带宽
		操作合约需要能量，冻结 trx 获得能量
		也可以租赁，17 trx = 1000 trx * 7 天 (贵爆了？)
		冻结 trx 获得票数，可以投票
	*/

	/*
		TRC10
		TRC20 就是 ERC20
		ERC20 test token
		TN3EZa6J6XekTgVcP4N4R43dTZmFE7zDud
		TRC20 test token
		TQCeH8Bc7zcJv6DjdCYWQuMX4Rzmc3gcs2
	*/
	/*
		divide()
		contractAddress := "TQCeH8Bc7zcJv6DjdCYWQuMX4Rzmc3gcs2"
		balance, err := cli.TRC20ContractBalance("TEtNLh69XnK9Fs8suCogK3sRrWJbQHah4k", contractAddress)
		checkError(err)
		fmt.Printf("Token balance: %+v\n", balance)

		divide()

		totalSupplyMethodSignature := "0x18160ddd"
		result, err := cli.TRC20Call("", contractAddress, totalSupplyMethodSignature, true, 0) // true, 0 表示 read
		checkError(err)
		totalSupply := new(big.Int).SetBytes(result.GetConstantResult()[0])
		fmt.Printf("Token total supply: %v\n", totalSupply)
	*/

	/*
		divide()
		tx, err := cli.GetTransactionInfoByID("c0391bd5fe5913df182282b4c07df0aa26f476c8286dc965eb4a780b5b690984")
		//tx, err := cli.GetTransactionInfoByID("f1ca51fac8b21527443068a56dd6b01a625d3f283534f10645a7932c73b1bae3")
		checkError(err)
		fmt.Printf("Transaction block: %+v\n", tx.BlockNumber)
		fmt.Printf("Transaction time: %+v\n", tx.BlockTimeStamp)
		fmt.Printf("Transaction contract result: %+v\n", tx.ContractResult)
		fmt.Printf("Transaction receipt: %+v\n", tx.Receipt)
		fmt.Printf("Transaction log: %+v\n", tx.Log)
	*/

	// 构造交易
	// TransferContract 普通转账
	/*
	from := "TEtNLh69XnK9Fs8suCogK3sRrWJbQHah4k"
	to := "TXexdzdZv3z5mJP1yzTeEc4LZwcBvenmZc"
	divide()
	contract := &core.TransferContract{}
	contract.OwnerAddress, err = common.DecodeCheck(from)
	checkError(err)
	contract.ToAddress, err = common.DecodeCheck(to)
	checkError(err)
	contract.Amount = 1000
	fmt.Printf("Contract: %+v\n", contract)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	tx, err := cli.Client.CreateTransaction2(ctx, contract)
	checkError(err)
	fmt.Printf("Tx: %+v\n", tx)
	fmt.Printf("Txid: %X\n", tx.Txid)
	*/

	// TRC20 转账
	// TriggerSmartContract 合约交易
	/*
	divide()
	tokenAddress := "TQCeH8Bc7zcJv6DjdCYWQuMX4Rzmc3gcs2"
	trc20tx, err := cli.TRC20Send(from, to, tokenAddress, big.NewInt(1000), 0)
	checkError(err)
	fmt.Printf("TRC20: %+v\n", trc20tx)
	fmt.Printf("TRC20 txid: %X\n", trc20tx.Txid)
	*/

	// GetCode()

	BuildSwapinTx()
}

func BuildSwapinTx() {
	cli := client.NewGrpcClientWithTimeout(testnetendpoint, timeout)
	err := cli.Start(grpc.WithInsecure())
	checkError(err)
	defer cli.Stop()

	divide()

	from := "TEtNLh69XnK9Fs8suCogK3sRrWJbQHah4k"
	contract := "TBssYqEV8BxJJDhGsf7pUkfPZxGbt2JU2M"
	method := "Swapin"
	param := `[{"string":"0xbeea0dfefc66107a3b1922f75a67ddd1d577a36ed0099e84a381e7a71774501e"},{"address":"TEtNLh69XnK9Fs8suCogK3sRrWJbQHah4k"},{"uint256":"1"}]`

	tx, err := cli.TriggerConstantContract(from, contract, method, param)
	checkError(err)
	fmt.Printf("Tx: %+v\n", tx)
}

func GetCode() {
	cli := client.NewGrpcClientWithTimeout(testnetendpoint, timeout)
	err := cli.Start(grpc.WithInsecure())
	checkError(err)
	defer cli.Stop()

	divide()

	contractDesc, err := tronaddress.Base58ToAddress("TQCeH8Bc7zcJv6DjdCYWQuMX4Rzmc3gcs2")
	checkError(err)
	message := new(api.BytesMessage)
	message.Value = contractDesc
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	sm, err := cli.Client.GetContract(ctx, message)
	checkError(err)
	fmt.Printf("Bytecode: %X\n", sm.Bytecode)
}