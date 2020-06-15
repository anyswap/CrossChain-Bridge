package worker

import (
	"time"

	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/tokens/btc"
	"github.com/fsn-dev/crossChain-Bridge/tokens/btc/electrs"
)

var (
	utxoChan              = make(chan *addrUxto, utxoPageLimit)
	utxoPageLimit         = 100
	utxoAggregateMinCount = 10
	utxoAggregateMinValue = uint64(100000)
)

type addrUxto struct {
	addr string
	utxo *electrs.ElectUtxo
}

// StartAggregateJob aggregate job
func StartAggregateJob() {
	if btc.BridgeInstance == nil {
		return
	}
	go doAggregate()
	for loop := 1; ; loop++ {
		logWorker("aggregate", "start aggregate job", "loop", loop)
		offset := 0
		for {
			p2shAddrs, err := mongodb.FindP2shAddresses(offset, utxoPageLimit)
			if err != nil {
				time.Sleep(3 * time.Second)
				continue
			}
			for _, p2shAddr := range p2shAddrs {
				utxos, _ := btc.BridgeInstance.FindUtxos(p2shAddr.P2shAddress)
				for _, utxo := range utxos {
					utxoChan <- &addrUxto{
						addr: p2shAddr.P2shAddress,
						utxo: utxo,
					}
				}
			}
			if len(p2shAddrs) < utxoPageLimit {
				break
			}
			offset += utxoPageLimit
		}
		logWorker("aggregate", "finish aggregate job", "loop", loop)
		time.Sleep(300 * time.Second)
	}
}

func doAggregate() {
	var (
		sumVal uint64
		addrs  []string
		utxos  []*electrs.ElectUtxo
	)
	for {
		addrutxo := <-utxoChan

		addr := addrutxo.addr
		utxo := addrutxo.utxo
		if utxo.Value == nil || *utxo.Value == 0 {
			continue
		}
		logWorker("aggregate", "find utxo", "address", addr, "utxo", utxo)

		sumVal += *utxo.Value
		addrs = append(addrs, addr)
		utxos = append(utxos, utxo)
		if len(utxos) == utxoAggregateMinCount || sumVal >= utxoAggregateMinValue {
			txHash, err := btc.BridgeInstance.AggregateUtxos(addrs, utxos)
			if err != nil {
				logWorkerError("aggregate", "aggregateUtxos failed", err)
			} else {
				logWorker("aggregate", "aggregateUtxos succeed", "txHash", txHash, "utxos", len(utxos), "sumVal", sumVal)
			}
			sumVal = 0
			addrs = nil
			utxos = nil
		}
	}
}
