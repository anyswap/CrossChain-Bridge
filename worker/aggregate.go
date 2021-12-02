package worker

import (
	"time"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
)

var (
	utxoPageLimit = 100

	aggSumVal   uint64
	aggAddrs    []string
	aggUtxos    []*electrs.ElectUtxo
	aggOffset   int
	aggInterval = 10 * time.Minute
)

// StartAggregateJob aggregate job
func StartAggregateJob() {
	if btc.BridgeInstance == nil {
		return
	}

	mongodb.MgoWaitGroup.Add(1)
	go loopDoAggregateJob()
}

func loopDoAggregateJob() {
	defer mongodb.MgoWaitGroup.Done()
	for loop := 1; ; loop++ {
		if utils.IsCleanuping() {
			return
		}
		logWorker("aggregate", "start aggregate job", "loop", loop)
		doAggregateJob()
		logWorker("aggregate", "finish aggregate job", "loop", loop)
		time.Sleep(aggInterval)
	}
}

func doAggregateJob() {
	aggOffset = 0
	for {
		if utils.IsCleanuping() {
			return
		}
		p2shAddrs, err := mongodb.FindP2shAddresses(aggOffset, utxoPageLimit)
		if err != nil {
			logWorkerError("aggregate", "FindP2shAddresses failed", err, "offset", aggOffset, "limit", utxoPageLimit)
			time.Sleep(3 * time.Second)
			continue
		}
		for _, p2shAddr := range p2shAddrs {
			findUtxosAndAggregate(p2shAddr.P2shAddress)
		}
		if len(p2shAddrs) < utxoPageLimit {
			break
		}
		aggOffset += utxoPageLimit
	}
}

func findUtxosAndAggregate(addr string) {
	findUtxos, _ := btc.BridgeInstance.FindUtxos(addr)
	for _, utxo := range findUtxos {
		if utxo.Value == nil || *utxo.Value == 0 {
			continue
		}
		if isUtxoExist(utxo) {
			continue
		}
		outspend, err := btc.BridgeInstance.GetOutspend(*utxo.Txid, *utxo.Vout)
		if err != nil {
			logWorkerError("aggregate", "get out spend failed", err, "address", addr, "utxo", utxo.String())
			continue
		}
		if *outspend.Spent {
			logWorkerTrace("aggregate", "ignore spent utxo", "address", addr, "utxo", utxo.String(), "outspend", outspend.String())
			continue
		}

		logWorker("aggregate", "find utxo", "address", addr, "utxo", utxo.String())

		aggSumVal += *utxo.Value
		aggAddrs = append(aggAddrs, addr)
		aggUtxos = append(aggUtxos, utxo)

		if btc.BridgeInstance.ShouldAggregate(len(aggUtxos), aggSumVal) {
			aggregate()
		}
	}
}

func isUtxoExist(utxo *electrs.ElectUtxo) bool {
	for _, item := range aggUtxos {
		if *item.Txid == *utxo.Txid && *item.Vout == *utxo.Vout {
			return true
		}
	}
	return false
}

func aggregate() {
	txHash, err := btc.BridgeInstance.AggregateUtxos(aggAddrs, aggUtxos)
	if err != nil {
		logWorkerError("aggregate", "AggregateUtxos failed", err)
	} else {
		logWorker("aggregate", "AggregateUtxos succeed", "txHash", txHash, "utxos", len(aggUtxos), "sumVal", aggSumVal)
	}
	aggSumVal = 0
	aggAddrs = nil
	aggUtxos = nil
}
