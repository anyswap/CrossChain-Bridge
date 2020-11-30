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
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
	"github.com/fsn-dev/fsn-go-sdk/efsn/common"
	"github.com/fsn-dev/fsn-go-sdk/efsn/core/types"
	"github.com/fsn-dev/fsn-go-sdk/efsn/ethclient"
	"github.com/urfave/cli/v2"
)

var (
	isSwapoutType2Flag = &cli.BoolFlag{
		Name:  "swapoutType2",
		Usage: "is swapout bind address string type",
	}

	scanEthCommand = &cli.Command{
		Action:    scanEth,
		Name:      "scaneth",
		Usage:     "scan swap on eth",
		ArgsUsage: " ",
		Description: `
scan swap on eth
`,
		Flags: []cli.Flag{
			utils.GatewayFlag,
			utils.SwapServerFlag,
			utils.SwapTypeFlag,
			utils.DepositAddressSliceFlag,
			utils.TokenAddressSliceFlag,
			utils.PairIDSliceFlag,
			utils.StartHeightFlag,
			utils.EndHeightFlag,
			utils.StableHeightFlag,
			utils.JobsFlag,
			isSwapoutType2Flag,
		},
	}
)

type ethSwapScanner struct {
	gateway          string
	swapServer       string
	swapType         string
	depositAddresses []string
	tokenAddresses   []string
	pairIDs          []string
	startHeight      uint64
	endHeight        uint64
	stableHeight     uint64
	jobCount         uint64
	isSwapoutType2   bool

	client *ethclient.Client
	ctx    context.Context

	rpcInterval   time.Duration
	rpcRetryCount int

	isSwapin bool
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
	scanner.swapType = ctx.String(utils.SwapTypeFlag.Name)
	scanner.depositAddresses = ctx.StringSlice(utils.DepositAddressSliceFlag.Name)
	scanner.tokenAddresses = ctx.StringSlice(utils.TokenAddressSliceFlag.Name)
	scanner.pairIDs = ctx.StringSlice(utils.PairIDSliceFlag.Name)
	scanner.startHeight = ctx.Uint64(utils.StartHeightFlag.Name)
	scanner.endHeight = ctx.Uint64(utils.EndHeightFlag.Name)
	scanner.stableHeight = ctx.Uint64(utils.StableHeightFlag.Name)
	scanner.jobCount = ctx.Uint64(utils.JobsFlag.Name)
	scanner.isSwapoutType2 = ctx.Bool(isSwapoutType2Flag.Name)

	switch strings.ToLower(scanner.swapType) {
	case "swapin":
		scanner.isSwapin = true
	case "swapout":
		scanner.isSwapin = false
	default:
		log.Fatalf("unknown swap type: '%v'", scanner.swapType)
	}

	log.Info("get argument success",
		"gateway", scanner.gateway,
		"swapServer", scanner.swapServer,
		"swapType", scanner.swapType,
		"depositAddress", scanner.depositAddresses,
		"tokenAddress", scanner.tokenAddresses,
		"pairID", scanner.pairIDs,
		"start", scanner.startHeight,
		"end", scanner.endHeight,
		"stable", scanner.stableHeight,
		"jobs", scanner.jobCount,
	)

	scanner.verifyOptions()
	scanner.init()
	scanner.run()
	return nil
}

func (scanner *ethSwapScanner) verifyOptions() {
	if scanner.isSwapin && len(scanner.depositAddresses) != len(scanner.pairIDs) {
		log.Fatalf("count of depositAddresses and pairIDs mismatch")
	}
	if len(scanner.tokenAddresses) != len(scanner.pairIDs) {
		log.Fatalf("count of tokenAddresses and pairIDs mismatch")
	}
	if !scanner.isSwapin && len(scanner.tokenAddresses) == 0 {
		log.Fatal("must sepcify token address for swapout scan")
	}
	for i, pairID := range scanner.pairIDs {
		if pairID == "" {
			log.Fatal("must specify pairid")
		}
		if scanner.isSwapin && !common.IsHexAddress(scanner.depositAddresses[i]) {
			log.Fatalf("invalid deposit address '%v'", scanner.depositAddresses[i])
		}
		if scanner.tokenAddresses[i] != "" && !common.IsHexAddress(scanner.tokenAddresses[i]) {
			log.Fatalf("invalid token address '%v'", scanner.tokenAddresses[i])
		}
		switch strings.ToLower(pairID) {
		case "btc", "ltc":
			scanner.isSwapoutType2 = true
		}
	}
	if scanner.gateway == "" {
		log.Fatal("must specify gateway address")
	}
	if scanner.swapServer == "" {
		log.Fatal("must specify swap server address")
	}
	scanner.verifyJobsOption()
}

func (scanner *ethSwapScanner) verifyJobsOption() {
	if scanner.endHeight != 0 && scanner.startHeight >= scanner.endHeight {
		log.Fatalf("wrong scan range [%v, %v)", scanner.startHeight, scanner.endHeight)
	}
	if scanner.jobCount == 0 {
		log.Fatal("zero jobs specified")
	}
}

func (scanner *ethSwapScanner) init() {
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

	eth.InitExtCodePartsWithFlag(scanner.isSwapoutType2)

	for _, tokenAddr := range scanner.tokenAddresses {
		if scanner.isSwapin && tokenAddr == "" {
			continue
		}
		var code []byte
		code, err = ethcli.CodeAt(scanner.ctx, common.HexToAddress(tokenAddr), nil)
		if err != nil {
			log.Fatalf("get contract code of '%v' failed, %v", tokenAddr, err)
		}
		if len(code) == 0 {
			log.Fatalf("'%v' is not contract address", tokenAddr)
		}
		if scanner.isSwapin {
			err = eth.VerifyErc20ContractCode(code)
			if err != nil {
				log.Warn("verify erc20 code failed. please ensure it's proxy contract of erc20", "contract", tokenAddr, "err", err)
				err = nil // do not exit
			}
		} else {
			err = eth.VerifySwapContractCode(code)
		}
		if err != nil {
			log.Fatalf("wrong contract address '%v', %v", tokenAddr, err)
		}
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

// nolint:dupl // in diff sub command
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
	log.Info(fmt.Sprintf("[%v] scan block %v", job, height), "hash", blockHash, "txs", len(block.Transactions()))
	for _, tx := range block.Transactions() {
		scanner.scanTransaction(tx)
	}
	if cache {
		cachedBlocks.addBlock(blockHash)
	}
}

func (scanner *ethSwapScanner) scanTransaction(tx *types.Transaction) {
	var err error
	for i, pairID := range scanner.pairIDs {
		tokenAddress := scanner.tokenAddresses[i]
		if scanner.isSwapin {
			depositAddress := scanner.depositAddresses[i]
			if tokenAddress != "" {
				err = scanner.verifyErc20SwapinTx(tx, tokenAddress, depositAddress)
			} else {
				err = scanner.verifySwapinTx(tx, depositAddress)
			}
		} else {
			err = scanner.verifySwapoutTx(tx, tokenAddress)
		}
		if !tokens.ShouldRegisterSwapForError(err) {
			continue
		}
		txid := tx.Hash().String()
		scanner.postSwap(txid, pairID)
		break
	}
}

func (scanner *ethSwapScanner) postSwap(txid, pairID string) {
	var subject, rpcMethod string
	if scanner.isSwapin {
		subject = "post swapin register"
		rpcMethod = "swap.Swapin"
	} else {
		subject = "post swapout register"
		rpcMethod = "swap.Swapout"
	}
	log.Info(subject, "txid", txid, "pairID", pairID)

	var result interface{}
	args := map[string]interface{}{
		"txid":   txid,
		"pairid": pairID,
	}
	for i := 0; i < scanner.rpcRetryCount; i++ {
		err := client.RPCPost(&result, scanner.swapServer, rpcMethod, args)
		if tokens.ShouldRegisterSwapForError(err) {
			break
		}
		if tools.IsSwapAlreadyExistRegisterError(err) {
			break
		}
		log.Warn(subject+" failed", "txid", txid, "pairID", pairID, "err", err)
	}
}

func (scanner *ethSwapScanner) verifyErc20SwapinTx(tx *types.Transaction, tokenAddress, depositAddress string) error {
	if tx.To() == nil || !strings.EqualFold(tx.To().String(), tokenAddress) {
		return tokens.ErrTxWithWrongContract
	}

	input := tx.Data()
	_, _, value, err := eth.ParseErc20SwapinTxInput(&input, depositAddress)
	if err != nil {
		return err
	}

	if value.Sign() <= 0 {
		return tokens.ErrTxWithWrongValue
	}

	return nil
}

func (scanner *ethSwapScanner) verifySwapinTx(tx *types.Transaction, depositAddress string) error {
	if tx.To() == nil || !strings.EqualFold(tx.To().String(), depositAddress) {
		return tokens.ErrTxWithWrongReceiver
	}

	if tx.Value().Sign() <= 0 {
		return tokens.ErrTxWithWrongValue
	}

	return nil
}

func (scanner *ethSwapScanner) verifySwapoutTx(tx *types.Transaction, tokenAddress string) error {
	if tx.To() == nil || !strings.EqualFold(tx.To().String(), tokenAddress) {
		return tokens.ErrTxWithWrongContract
	}

	input := tx.Data()
	_, value, err := eth.ParseSwapoutTxInput(&input)
	if err != nil {
		return err
	}

	if value.Sign() <= 0 {
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
