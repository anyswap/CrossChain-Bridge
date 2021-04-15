package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"google.golang.org/grpc"
	"github.com/golang/protobuf/ptypes"

	tronaddress "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	proto "github.com/golang/protobuf/proto"
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
	// GetCode()

	// BuildSwapinTx()

	// ScanBlock()

	// GetTxArgs()

	// GetContractCode()

	GetTransaction()

	// MarshalUnmarshalTx()

	// GetSmartContractLog()
}

func GetTransaction() {
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

	divide()
	// f1ca51fac8b21527443068a56dd6b01a625d3f283534f10645a7932c73b1bae3 普通转账
	// c0391bd5fe5913df182282b4c07df0aa26f476c8286dc965eb4a780b5b690984 TRC20 转账
	//txinfo, err := cli.GetTransactionInfoByID("f1ca51fac8b21527443068a56dd6b01a625d3f283534f10645a7932c73b1bae3")
	txinfo, err := cli.GetTransactionInfoByID("c0391bd5fe5913df182282b4c07df0aa26f476c8286dc965eb4a780b5b690984")
	// txinfo, err := cli.GetTransactionInfoByID("aa1d7b84277097c3fe8657a663e01eec11f3b5cfcdf2dea41f5593784637fab7")
	//txinfo, err := cli.GetTransactionInfoByID("0802edce3c7bd11b4d995eac84a7b235594f12088493264f3fd4d9f4bd991b57")
	//txinfo, err := cli.GetTransactionInfoByID("b034937d9e2975170b73b6cc3f9f7857f813ab26f9560fd3785f9e8c1e7085ca")
	checkError(err)
	fmt.Printf("Transaction info: %+v\n", txinfo)
	fmt.Printf("Transaction info contract result: %v\n", new(big.Int).SetBytes(txinfo.GetContractResult()[0]))
	fmt.Printf("Transaction block: %+v\n", txinfo.BlockNumber)
	fmt.Printf("Transaction time: %+v\n", txinfo.BlockTimeStamp)
	fmt.Printf("Transaction contract result: %+v\n", txinfo.ContractResult)
	fmt.Printf("Transaction receipt: %+v\n", txinfo.Receipt)

	txlog := txinfo.GetLog()
	fmt.Printf("Transaction log length: %v\n", len(txlog))
	/*
	fmt.Printf("Transaction log: %v\n", txlog[0])
	fmt.Printf("Transaction log address: %v\n", tronaddress.Address(append([]byte{0x41}, txlog[0].GetAddress()...))) // 合约地址
	fmt.Printf("Transaction log topics: %X\n", txlog[0].GetTopics()[0]) // trc20TransferEventSignature DDF252AD1BE2C89B69C2B068FC378DAA952BA7F163C4A11628F55A4DF523B3EF
	fmt.Printf("Transaction log topics: %X\n", txlog[0].GetTopics()[1]) // from
	fmt.Printf("Transaction log topics: %X\n", txlog[0].GetTopics()[2]) // to
	fmt.Printf("Transaction log data: %X\n", txlog[0].GetData())
	*/

	divide()
	//tx, err := cli.GetTransactionByID("f1ca51fac8b21527443068a56dd6b01a625d3f283534f10645a7932c73b1bae3")
	//tx, err := cli.GetTransactionByID("c0391bd5fe5913df182282b4c07df0aa26f476c8286dc965eb4a780b5b690984")
	// tx, err := cli.GetTransactionByID("aa1d7b84277097c3fe8657a663e01eec11f3b5cfcdf2dea41f5593784637fab7")
	// tx, err := cli.GetTransactionByID("0802edce3c7bd11b4d995eac84a7b235594f12088493264f3fd4d9f4bd991b57")
	tx, err := cli.GetTransactionByID("b034937d9e2975170b73b6cc3f9f7857f813ab26f9560fd3785f9e8c1e7085ca")
	checkError(err)
	fmt.Printf("Transaction raw data: %+v\n", tx.GetRawData())
	fmt.Printf("Transaction expiration: %+v\n", tx.GetRawData().GetExpiration())
	fmt.Printf("Transaction fee limit: %v\n", tx.GetRawData().GetFeeLimit())
	fmt.Printf("Transaction: %+v\n", tx)
	fmt.Printf("Transaction Ret: %+v\n", tx.GetRet()[0])
	fmt.Printf("Transaction Ret Success: %+v\n", (tx.GetRet()[0].GetRet() == core.Transaction_Result_SUCESS))
	fmt.Printf("Contract Ret: %+v\n", tx.GetRet()[0].GetContractRet())
	fmt.Printf("Contract Ret Default: %+v\n", (tx.GetRet()[0].GetContractRet() == core.Transaction_Result_DEFAULT))
	fmt.Printf("Contract Ret Success: %+v\n", (tx.GetRet()[0].GetContractRet() == core.Transaction_Result_SUCCESS))
	fmt.Printf("Contract Ret Out Of Energy: %+v\n", (tx.GetRet()[0].GetContractRet() == core.Transaction_Result_OUT_OF_ENERGY))
	if len(tx.RawData.Contract) != 1 {
		checkError(fmt.Errorf("Invalid contract"))
	}
	contract := tx.RawData.Contract[0]
	switch contract.Type {
	case core.Transaction_Contract_TransferContract:
		// 普通转账
		var c core.TransferContract
		err = ptypes.UnmarshalAny(contract.GetParameter(), &c)
		checkError(err)
		fmt.Printf("Trigger smart contract: %+v\n", c)
		fmt.Printf("To address: %v\n", tronaddress.Address(c.ToAddress))
		fmt.Printf("From address: %v\n", tronaddress.Address(c.OwnerAddress))
		fmt.Printf("Transfer value: %v\n", big.NewInt(c.Amount))
	case core.Transaction_Contract_TransferAssetContract:
		// TRC10
		checkError(fmt.Errorf("TRC10 transfer not supported"))
	case core.Transaction_Contract_TriggerSmartContract:
		// TRC20
		var c core.TriggerSmartContract
		err = ptypes.UnmarshalAny(contract.GetParameter(), &c)
		checkError(err)
		fmt.Printf("Trigger smart contract: %+v\n", c)
		fmt.Printf("Contract address: %v\n", tronaddress.Address(c.ContractAddress))
		fmt.Printf("Data: %X\n", c.Data)

	default:
		return
	}

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

func ScanBlock() {
	cli := client.NewGrpcClientWithTimeout(testnetendpoint, timeout)
	//cli := client.NewGrpcClientWithTimeout(mainnetendpoint, timeout)
	err := cli.Start(grpc.WithInsecure())
	checkError(err)
	defer cli.Stop()

	divide()

	var step int = 10
	var start, end int64
	longSleep := time.Second * 2
	shortSleep := time.Millisecond * 400
	start = 13689800 // TODO Load latest scanned
	end = start + int64(step)
	for {
		res, err := cli.GetBlockByLimitNext(start, end)
		checkError(err)
		fmt.Printf("Blocks: %+v\n", len(res.Block))
		l := len(res.Block)
		if l > 0 {
			fmt.Printf("%v - %v\n", res.Block[0].BlockHeader.RawData.Number, res.Block[l-1].BlockHeader.RawData.Number)
		}
		// TODO process tx
		for _, block := range res.Block {
			txexts := make([]*api.TransactionExtention, 0)
			txexts = block.Transactions
			for _, txext := range txexts {
				var tx *core.Transaction
				tx = txext.Transaction
				fmt.Printf("tx: %+v/n", tx)
			}
		}

		// TODO Add latest scanned
		start = start + int64(len(res.Block))
		end = start + int64(step)
		if len(res.Block) < step {
			time.Sleep(longSleep)
		} else {
			time.Sleep(shortSleep)
		}
	}
}

func GetTxArgs() {
	divide()
	cli := client.NewGrpcClientWithTimeout(testnetendpoint, timeout)
	//cli := client.NewGrpcClientWithTimeout(mainnetendpoint, timeout)
	err := cli.Start(grpc.WithInsecure())
	checkError(err)
	defer cli.Stop()
	//tx, err := cli.GetTransactionByID("0xc0391bd5fe5913df182282b4c07df0aa26f476c8286dc965eb4a780b5b690984")
	//tx, err := cli.GetTransactionByID("c7effb1b0b86f4a22dcce26208027a21ef903685655a2d75e4819a63b903f0e7")
	//tx, err := cli.GetTransactionByID("013e79e7dd5229909dea401d498e2e474dacbe7cfa8fda57a19a17c610d22df9")
	tx, err := cli.GetTransactionByID("b034937d9e2975170b73b6cc3f9f7857f813ab26f9560fd3785f9e8c1e7085ca")
	checkError(err)
	fmt.Printf("Tx: %+v", tx)
	rawData, err := proto.Marshal(tx.GetRawData())
	checkError(err)
	h256h := sha256.New()
	h256h.Write(rawData)
	hash := h256h.Sum(nil)
	txhash := common.ToHex(hash)
	txhash = strings.TrimPrefix(txhash, "0x")

	fmt.Printf("Tx hash: %v\n", txhash)

	ret := tx.GetRet()
	crt := ret[0].ContractRet
	fmt.Printf("%v\n", crt)

	var contract core.TriggerSmartContract
	err = ptypes.UnmarshalAny(tx.GetRawData().GetContract()[0].GetParameter(), &contract)
	checkError(err)
	fmt.Printf("\nContract: %+v\n", contract)
	data := tx.GetRawData().GetData()
	fmt.Printf("\nData: %X\n", data)
	fmt.Printf("\nData: %X\n", contract.Data)
}

func GetContractCode() {
	divide()
	contractDesc, err := tronaddress.Base58ToAddress("TQCeH8Bc7zcJv6DjdCYWQuMX4Rzmc3gcs2")
	checkError(err)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cli := client.NewGrpcClientWithTimeout(testnetendpoint, timeout)
	//cli := client.NewGrpcClientWithTimeout(mainnetendpoint, timeout)
	err = cli.Start(grpc.WithInsecure())
	checkError(err)
	defer cli.Stop()
	sm, err := cli.Client.GetContract(ctx, client.GetMessageBytes(contractDesc))
	checkError(err)
	fmt.Printf("SM: %X", sm.GetBytecode())
}

func MarshalUnmarshalTx() {
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
	fmt.Printf("Tx: %+v\n", tx.Transaction)

	txmsg, err := proto.Marshal(tx.Transaction)
	checkError(err)
	fmt.Printf("txmsg: %X\n", txmsg)

	var decodedTx core.Transaction
	err = proto.Unmarshal(txmsg, &decodedTx)
	checkError(err)
	fmt.Printf("Decoded tx:\n%+v\n", &decodedTx)
}
