//nolint
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens/terra"
)

func main() {
	req := &terra.SimulateRequest{TxBytes: []byte("1234")}
	fmt.Printf("%s\n", req)
}

// GetBlockResult get block result
type GetBlockResult struct {
	Block *Block `json:"block"`
}

// Block block
type Block struct {
	Header *Header `json:"header"`
}

// Header header
type Header struct {
	ChainID string    `json:"chain_id"`
	Height  string    `json:"height"`
	Time    time.Time `json:"time"`
}

var rpcTimeout = 60

func main1() {
	block, err := GetLatestBlock("https://lcd.terra.dev/")
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	fmt.Printf("%+v\n", block)
	fmt.Printf("%+v\n", block.Header)
	height, err := common.GetUint64FromStr(block.Header.Height)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	fmt.Printf("height : %v\n", height)
}

func joinURLPath(url, path string) string {
	url = strings.TrimSuffix(url, "/")
	if !strings.HasPrefix(path, "/") {
		url += "/"
	}
	return url + path
}

func GetLatestBlock(url string) (*Block, error) {
	path := "/blocks/latest"
	var result GetBlockResult
	err := client.RPCGetWithTimeout(&result, joinURLPath(url, path), rpcTimeout)
	if err != nil {
		return nil, err
	}
	return result.Block, err
}
