package main

import (
	"time"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/tools"
	"github.com/urfave/cli/v2"
)

func initArgsScan(ctx *cli.Context) {
	net = ctx.String(netFlag.Name)
	apiAddress = ctx.String(apiAddressFlag.Name)
	startScan = ctx.Uint64(startScanFlag.Name)
	if apiAddress == "" {
		apiAddress = initDefaultAPIAddress(net)
	}
}

func scanTxAction(ctx *cli.Context) error {
	utils.SetLogger(ctx)
	initArgsScan(ctx)
	initBridge()
	scanTx(startScan)
	return nil
}

func scanTx(start uint64) {
	stable := start
	confirmations := uint64(0)
	errorSubject := "[scanchain] get XRP block failed"
	scanSubject := "[scanchain] scanned XRP block"
	for {
		latest := tools.LoopGetLatestBlockNumber(b)
		log.Info("Scan chain", "latest block number", latest)
		for h := stable + 1; h <= latest; {
			blockHash, err := b.GetBlockHash(h)
			if err != nil {
				log.Error(errorSubject, "height", h, "err", err)
				time.Sleep(time.Second * 3)
				continue
			}
			log.Info("Scan chain, get block hash", "", blockHash)
			txids, err := b.GetBlockTxids(h)
			if err != nil {
				log.Error(errorSubject, "height", h, "blockHash", blockHash, "ledger index", h, "err", err)
				time.Sleep(time.Second * 3)
				continue
			}
			log.Info("Scan chain, get tx ids", "", txids)
			for _, txid := range txids {
				log.Info("Check transaction", "txid", txid)
				tx, err := b.GetTransaction(txid)
				if err != nil {
					log.Warn("Check transaction failed", "error", err)
				}
				log.Info("Check transaction success", "tx", tx)
			}
			log.Info(scanSubject, "blockHash", blockHash, "height", h, "txs", len(txids))
			h++
		}
		if stable+confirmations < latest {
			stable = latest - confirmations
		}
		time.Sleep(time.Second * 3)
	}
}
