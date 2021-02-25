package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
	"github.com/anyswap/CrossChain-Bridge/tokens/eth"
	"github.com/anyswap/CrossChain-Bridge/tokens/ltc"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
	"github.com/urfave/cli/v2"
)

var (
	scanLtcCommand = &cli.Command{
		Action:    scanLtc,
		Name:      "scanltc",
		Usage:     "scan swap on ltc",
		ArgsUsage: " ",
		Description: `
scan swap on ltc
`,
		Flags: []cli.Flag{
			testnetFlag,
			mongoURLFlag,
			dbNameFlag,
			dbUserFlag,
			dbPassFlag,
			utils.GatewayFlag,
			utils.SwapServerFlag,
			utils.DepositAddressFlag,
			utils.StartHeightFlag,
			utils.EndHeightFlag,
			utils.StableHeightFlag,
			utils.JobsFlag,
		},
	}
)

type ltcSwapScanner struct {
	useTestnet     bool
	gateway        string
	swapServer     string
	depositAddress string
	startHeight    uint64
	endHeight      uint64
	stableHeight   uint64
	jobCount       uint64

	rpcInterval   time.Duration
	rpcRetryCount int

	bridge *ltc.Bridge
}

func scanLtc(ctx *cli.Context) error {
	utils.SetLogger(ctx)
	scanner := &ltcSwapScanner{
		rpcInterval:   1 * time.Second,
		rpcRetryCount: 3,
	}
	scanner.useTestnet = ctx.Bool(testnetFlag.Name)
	scanner.gateway = ctx.String(utils.GatewayFlag.Name)
	scanner.swapServer = ctx.String(utils.SwapServerFlag.Name)
	scanner.depositAddress = ctx.String(utils.DepositAddressFlag.Name)
	scanner.startHeight = ctx.Uint64(utils.StartHeightFlag.Name)
	scanner.endHeight = ctx.Uint64(utils.EndHeightFlag.Name)
	scanner.stableHeight = ctx.Uint64(utils.StableHeightFlag.Name)
	scanner.jobCount = ctx.Uint64(utils.JobsFlag.Name)

	log.Info("get argument success",
		"testnet", scanner.useTestnet,
		"gateway", scanner.gateway,
		"swapServer", scanner.swapServer,
		"depositAddress", scanner.depositAddress,
		"start", scanner.startHeight,
		"end", scanner.endHeight,
		"stable", scanner.stableHeight,
		"jobs", scanner.jobCount,
	)

	scanner.initMongodb(ctx)
	scanner.initBridge()
	scanner.verifyOptions()
	scanner.run()
	return nil
}

func (scanner *ltcSwapScanner) verifyOptions() {
	if !scanner.bridge.IsValidAddress(scanner.depositAddress) {
		log.Fatalf("invalid deposit address '%v'", scanner.depositAddress)
	}
	if scanner.gateway == "" {
		log.Fatal("must specify gateway address")
	}
	if scanner.swapServer == "" {
		log.Fatal("must specify swap server address")
	}

	oracle := params.OracleConfig{
		ServerAPIAddress: scanner.swapServer,
	}
	err := oracle.CheckConfig()
	if err != nil {
		log.Fatalf("check swap server failed. %v", err)
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
}

func (scanner *ltcSwapScanner) initMongodb(ctx *cli.Context) {
	dbURL := ctx.String(mongoURLFlag.Name)
	dbName := ctx.String(dbNameFlag.Name)
	userName := ctx.String(dbUserFlag.Name)
	passwd := ctx.String(dbPassFlag.Name)
	if dbName != "" {
		mongodb.MongoServerInit([]string{dbURL}, dbName, userName, passwd)
	}
}

func (scanner *ltcSwapScanner) initBridge() {
	scanner.bridge = ltc.NewCrossChainBridge(true)
	scanner.bridge.GatewayConfig = &tokens.GatewayConfig{
		APIAddress: []string{scanner.gateway},
	}
	ltcDecimals := uint8(8)
	netID := "Mainnet"
	if scanner.useTestnet {
		netID = "TestNet3"
	}
	scanner.bridge.ChainConfig = &tokens.ChainConfig{
		BlockChain:    "Bitcoin",
		NetID:         netID,
		Confirmations: &scanner.stableHeight,
	}
	pairConfig := &tokens.TokenPairConfig{
		PairID: ltc.PairID,
		SrcToken: &tokens.TokenConfig{
			ID:             "LTC",
			Name:           "LTC",
			Symbol:         "LTC",
			Decimals:       &ltcDecimals,
			DepositAddress: scanner.depositAddress,
		},
	}
	pairsConfig := make(map[string]*tokens.TokenPairConfig)
	pairsConfig[ltc.PairID] = pairConfig
	tokens.SetTokenPairsConfig(pairsConfig, false)
	tokens.SrcBridge = scanner.bridge
	tokens.DstBridge = eth.NewCrossChainBridge(false)
}

func (scanner *ltcSwapScanner) run() {
	start := scanner.startHeight
	wend := scanner.endHeight
	if wend == 0 {
		wend = tools.LoopGetLatestBlockNumber(scanner.bridge)
	}
	if start == 0 {
		start = wend
	}

	scanner.doScanRangeJob(start, wend)

	if scanner.endHeight == 0 {
		go scanner.scanPool()
		scanner.scanLoop(wend)
	}
}

// nolint:dupl // in diff sub command
func (scanner *ltcSwapScanner) doScanRangeJob(start, end uint64) {
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

func (scanner *ltcSwapScanner) scanRange(job, from, to uint64, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Info(fmt.Sprintf("[%v] start scan range", job), "from", from, "to", to)

	for h := from; h < to; h++ {
		scanner.scanBlock(job, h, false)
	}

	log.Info(fmt.Sprintf("[%v] scan range finish", job), "from", from, "to", to)
}

func (scanner *ltcSwapScanner) scanPool() {
	scanner.bridge.StartPoolTransactionScanJob()
}

func (scanner *ltcSwapScanner) scanLoop(from uint64) {
	stable := scanner.stableHeight
	log.Info("start scan loop", "from", from, "stable", stable)
	for {
		latest := tools.LoopGetLatestBlockNumber(scanner.bridge)
		for h := latest; h > from; h-- {
			scanner.scanBlock(0, h, true)
		}
		if from+stable < latest {
			from = latest - stable
		}
		time.Sleep(5 * time.Second)
	}
}

func (scanner *ltcSwapScanner) loopGetBlockHash(height uint64) string {
	for {
		blockHash, err := scanner.bridge.GetBlockHash(height)
		if err == nil {
			return blockHash
		}
		log.Warn("get block hash failed", "height", height, "err", err)
		time.Sleep(scanner.rpcInterval)
	}
}

func (scanner *ltcSwapScanner) scanBlock(job, height uint64, cache bool) {
	blockHash := scanner.loopGetBlockHash(height)
	if cache && ltcCachedBlocks.isScanned(blockHash) {
		return
	}
	block, err := scanner.bridge.GetBlock(blockHash)
	if err != nil {
		log.Warn("get block failed", "height", height, "hash", blockHash, "err", err)
		return
	}
	txCount := *block.TxCount
	log.Info(fmt.Sprintf("[%v] scan block %v start", job, height), "hash", blockHash, "txs", txCount)

	startIndex := uint32(0)
	for startIndex < txCount {
		var txs []*electrs.ElectTx
		for i := 0; i < scanner.rpcRetryCount; i++ {
			txs, err = scanner.bridge.GetBlockTransactions(blockHash, startIndex)
			if err == nil {
				break
			}
			log.Warn("get block txs failed", "height", height, "startIndex", startIndex, "err", err)
			time.Sleep(scanner.rpcInterval)
		}
		for i, tx := range txs {
			log.Trace(fmt.Sprintf("[%v] scan block %v process tx", job, height), "txid", *tx.Txid, "index", startIndex+uint32(i))
			scanner.processTx(tx)
		}
		log.Trace(fmt.Sprintf("[%v] scan block %v process txs", job, height), "startIndex", startIndex, "total", txCount)
		startIndex += 25 // 25 is elctrs API defined
	}

	if cache {
		ltcCachedBlocks.addBlock(blockHash)
	}
	log.Info(fmt.Sprintf("[%v] scan block %v finish", job, height))
}

func (scanner *ltcSwapScanner) processTx(tx *electrs.ElectTx) {
	txid := *tx.Txid
	p2shBindAddrs, err := scanner.bridge.CheckSwapinTxType(tx)
	if err != nil {
		return
	}
	if len(p2shBindAddrs) > 0 {
		for _, p2shBindAddr := range p2shBindAddrs {
			log.Info("post p2sh swapin register", "txid", txid, "bind", p2shBindAddr)
			args := map[string]interface{}{
				"txid": txid,
				"bind": p2shBindAddr,
			}
			var result interface{}
			for i := 0; i < scanner.rpcRetryCount; i++ {
				err = client.RPCPost(&result, scanner.swapServer, "swap.P2shSwapin", args)
				if tokens.ShouldRegisterSwapForError(err) {
					break
				}
				if tools.IsSwapAlreadyExistRegisterError(err) {
					break
				}
				log.Warn("post p2sh swapin register failed", "txid", txid, "bind", p2shBindAddr, "err", err)
			}
		}
	} else {
		value, memoScript, rightReceiver := scanner.bridge.GetReceivedValue(tx.Vout, scanner.depositAddress, "p2pkh")
		if !rightReceiver || value == 0 {
			return
		}
		bindAddress, bindOk := ltc.GetBindAddressFromMemoScipt(memoScript)
		if !bindOk {
			return
		}
		log.Info("post swapin register", "txid", txid, "pairid", ltc.PairID, "bind", bindAddress)
		args := map[string]interface{}{
			"txid":   txid,
			"pairid": ltc.PairID,
		}
		var result interface{}
		for i := 0; i < scanner.rpcRetryCount; i++ {
			err = client.RPCPost(&result, scanner.swapServer, "swap.Swapin", args)
			if tokens.ShouldRegisterSwapForError(err) {
				break
			}
			if tools.IsSwapAlreadyExistRegisterError(err) {
				break
			}
			log.Warn("post swapin register failed", "txid", txid, "bind", bindAddress, "err", err)
		}
	}
}

var ltcCachedBlocks = &cachedSacnnedBlocks{
	capacity:  100,
	nextIndex: 0,
	hashes:    make([]string, 100),
}
