package main

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/eth"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"
)

var (
	scanEthCommand = &cli.Command{
		Action:    scanEth,
		Name:      "scaneth",
		Usage:     "scan swap on eth",
		ArgsUsage: " ",
		Description: `
distribute rewards by liquidity
`,
		Flags: []cli.Flag{
			utils.GatewayFlag,
			utils.SwapServerFlag,
			utils.DcrmAddressFlag,
			utils.TokenAddressFlag,
			utils.StartHeightFlag,
			utils.EndHeightFlag,
			utils.StableHeightFlag,
			utils.JobsFlag,
		},
	}
)

type ethSwapScanner struct {
	gateway      string
	swapServer   string
	dcrmAddress  string
	tokenAddress string
	startHeight  uint64
	endHeight    uint64
	stableHeight uint64
	jobCount     uint64

	client *ethclient.Client
	ctx    context.Context

	rpcInterval   time.Duration
	rpcRetryCount int
}

func scanEth(ctx *cli.Context) error {
	utils.SetLogger(ctx)
	scanner := &ethSwapScanner{
		ctx:           context.Background(),
		rpcInterval:   3 * time.Second,
		rpcRetryCount: 3,
	}
	scanner.gateway = ctx.String(utils.GatewayFlag.Name)
	scanner.swapServer = ctx.String(utils.SwapServerFlag.Name)
	scanner.dcrmAddress = ctx.String(utils.DcrmAddressFlag.Name)
	scanner.tokenAddress = ctx.String(utils.TokenAddressFlag.Name)
	scanner.startHeight = ctx.Uint64(utils.StartHeightFlag.Name)
	scanner.endHeight = ctx.Uint64(utils.EndHeightFlag.Name)
	scanner.stableHeight = ctx.Uint64(utils.StableHeightFlag.Name)
	scanner.jobCount = ctx.Uint64(utils.JobsFlag.Name)

	log.Info("get argument success",
		"gateway", scanner.gateway,
		"swapServer", scanner.swapServer,
		"dcrmAddress", scanner.dcrmAddress,
		"tokenAddress", scanner.tokenAddress,
		"start", scanner.startHeight,
		"end", scanner.endHeight,
		"stable", scanner.stableHeight,
		"jobs", scanner.jobCount,
	)

	scanner.verifyOptions()
	scanner.run()
	return nil
}

func (scanner *ethSwapScanner) verifyOptions() {
	if scanner.dcrmAddress == "" {
		log.Fatal("must specify dcrm address")
	}
	if scanner.gateway == "" {
		log.Fatal("must specify gateway address")
	}
	if scanner.swapServer == "" {
		log.Fatal("must specify swap server address")
	}

	start := scanner.startHeight
	end := scanner.endHeight
	jobs := scanner.jobCount
	if end != 0 && start >= end {
		log.Fatalf("wrong scan range [%v, %v)", start, end)
	}
	if jobs == 0 {
		log.Fatal("zero jobs specified")
	}

	ethcli, err := ethclient.Dial(scanner.gateway)
	if err != nil {
		log.Fatal("ethclient.Dail failed", "gateway", scanner.gateway, "err", err)
	}
	scanner.client = ethcli

	var version string
	for i := 0; i < scanner.rpcRetryCount; i++ {
		err = client.RPCPost(&version, scanner.swapServer, "swap.GetVersionInfo")
		if err == nil {
			log.Info("get server version succeed", "version", version)
			break
		}
		log.Warn("get server version failed", "swapServer", scanner.swapServer, "err", err)
		time.Sleep(scanner.rpcInterval)
	}
	if version == "" {
		log.Fatal("get server version failed", "swapServer", scanner.swapServer)
	}
}

func (scanner *ethSwapScanner) run() {
	start := scanner.startHeight
	wend := scanner.endHeight
	if wend == 0 {
		wend = scanner.loopGetLatestBlockNumber()
	}
	if start == 0 {
		start = wend
	}

	scanner.doScanRangeJob(start, wend)

	if scanner.endHeight == 0 {
		scanner.scanLoop(wend)
	}
}

func (scanner *ethSwapScanner) doScanRangeJob(start, end uint64) {
	if start >= end {
		return
	}
	jobs := scanner.jobCount
	count := end - start
	step := count / jobs
	if step == 0 {
		jobs = 1
		step = count
	}
	wg := new(sync.WaitGroup)
	for i := uint64(0); i < jobs; i++ {
		from := start + i*step
		to := start + (i+1)*step
		if i+1 == jobs {
			to = end
		}
		wg.Add(1)
		go scanner.scanRange(i+1, from, to, wg)
	}
	if scanner.endHeight != 0 {
		wg.Wait()
	}
}

func (scanner *ethSwapScanner) scanRange(job, from, to uint64, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Info(fmt.Sprintf("[%v] start scan range", job), "from", from, "to", to)

	for h := from; h < to; h++ {
		scanner.scanBlock(job, h, false)
	}

	log.Info(fmt.Sprintf("[%v] scan range finish", job), "from", from, "to", to)
}

func (scanner *ethSwapScanner) scanLoop(from uint64) {
	stable := scanner.stableHeight
	log.Info("start scan loop", "from", from, "stable", stable)
	for {
		latest := scanner.loopGetLatestBlockNumber()
		for h := latest; h > from; h-- {
			scanner.scanBlock(0, h, true)
		}
		if from+stable < latest {
			from = latest - stable
		}
		time.Sleep(5 * time.Second)
	}
}

func (scanner *ethSwapScanner) loopGetLatestBlockNumber() uint64 {
	for {
		header, err := scanner.client.HeaderByNumber(scanner.ctx, nil)
		if err == nil {
			log.Info("get latest block number success", "height", header.Number)
			return header.Number.Uint64()
		}
		log.Warn("get latest block number failed", "err", err)
		time.Sleep(scanner.rpcInterval)
	}
}

func (scanner *ethSwapScanner) loopGetBlock(height uint64) *types.Block {
	blockNumber := new(big.Int).SetUint64(height)
	for {
		block, err := scanner.client.BlockByNumber(scanner.ctx, blockNumber)
		if err == nil {
			return block
		}
		log.Warn("get block failed", "height", height, "err", err)
		time.Sleep(scanner.rpcInterval)
	}
}

func (scanner *ethSwapScanner) scanBlock(job, height uint64, cache bool) {
	block := scanner.loopGetBlock(height)
	blockHash := block.Hash().String()
	if cache && cachedBlocks.isScanned(blockHash) {
		return
	}
	log.Info(fmt.Sprintf("[%v] scan block %v", job, height), "hash", blockHash)
	for _, tx := range block.Transactions() {
		scanner.scanTransaction(tx)
	}
	if cache {
		cachedBlocks.addBlock(blockHash)
	}
}

func (scanner *ethSwapScanner) scanTransaction(tx *types.Transaction) {
	var err error
	if scanner.tokenAddress != "" {
		err = scanner.verifyErc20SwapinTx(tx)
	} else {
		err = scanner.verifySwapinTx(tx)
	}
	if !tokens.ShouldRegisterSwapForError(err) {
		return
	}
	txid := tx.Hash().String()
	log.Info("post swapin register", "txid", txid)
	var result interface{}
	for i := 0; i < scanner.rpcRetryCount; i++ {
		err = client.RPCPost(&result, scanner.swapServer, "swap.Swapin", txid)
		if tokens.ShouldRegisterSwapForError(err) {
			break
		}
		if strings.Contains(err.Error(), "swap already exist") {
			break
		}
		log.Warn("post swapin register failed", "txid", txid, "err", err)
	}
}

func (scanner *ethSwapScanner) verifyErc20SwapinTx(tx *types.Transaction) error {
	if tx.To() == nil || !strings.EqualFold(tx.To().String(), scanner.tokenAddress) {
		return tokens.ErrTxWithWrongContract
	}

	input := tx.Data()
	_, to, value, err := eth.ParseErc20SwapinTxInput(&input)
	if err != nil {
		return tokens.ErrTxWithWrongInput
	}

	if !strings.EqualFold(to, scanner.dcrmAddress) {
		return tokens.ErrTxWithWrongReceiver
	}

	if value.Sign() <= 0 {
		return tokens.ErrTxWithWrongValue
	}

	return nil
}

func (scanner *ethSwapScanner) verifySwapinTx(tx *types.Transaction) error {
	if tx.To() == nil || !strings.EqualFold(tx.To().String(), scanner.dcrmAddress) {
		return tokens.ErrTxWithWrongReceiver
	}

	if tx.Value().Sign() <= 0 {
		return tokens.ErrTxWithWrongValue
	}

	return nil
}

type cachedSacnnedBlocks struct {
	capacity  int
	nextIndex int
	hashes    []string
}

var cachedBlocks = &cachedSacnnedBlocks{
	capacity:  100,
	nextIndex: 0,
	hashes:    make([]string, 100),
}

func (cache *cachedSacnnedBlocks) addBlock(blockHash string) {
	cache.hashes[cache.nextIndex] = blockHash
	cache.nextIndex = (cache.nextIndex + 1) % cache.capacity
}

func (cache *cachedSacnnedBlocks) isScanned(blockHash string) bool {
	for _, b := range cache.hashes {
		if b == blockHash {
			return true
		}
	}
	return false
}
