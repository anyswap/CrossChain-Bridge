package main

import (
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
)

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

//var mainnetendpoint = "3.225.171.164:50051"
var mainnetendpoint = "grpc.trongrid.io:50051"
var testnetendpoint = "grpc.shasta.trongrid.io:50051"

func main() {
	timeout := time.Second * 15
	cli := client.NewGrpcClientWithTimeout(testnetendpoint, timeout)
	err := cli.Start(grpc.WithInsecure())
	checkError(err)
	defer cli.Stop()

	res, err := cli.GetNowBlock()
	checkError(err)
	fmt.Printf("Block number: %+v\n", res.BlockHeader.RawData.Number)
}
