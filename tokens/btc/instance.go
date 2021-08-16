package btc

import (
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc/electrs"
)

// BridgeInstance btc bridge instance
var BridgeInstance BridgeInterface

// BridgeInterface btc bridge interface
type BridgeInterface interface {
	tokens.CrossChainBridge

	GetCompressedPublicKey(fromPublicKey string, needVerify bool) (cPkData []byte, err error)
	GetP2shAddress(bindAddr string) (p2shAddress string, redeemScript []byte, err error)
	VerifyP2shTransaction(pairID, txHash, bindAddress string, allowUnstable bool) (*tokens.TxSwapInfo, error)
	VerifyAggregateMsgHash(msgHash []string, args *tokens.BuildTxArgs) error
	AggregateUtxos(addrs []string, utxos []*electrs.ElectUtxo) (string, error)
	FindUtxos(addr string) ([]*electrs.ElectUtxo, error)
	GetOutspend(txHash string, vout uint32) (*electrs.ElectOutspend, error)

	StartSwapHistoryScanJob()
	StartChainTransactionScanJob()
	StartPoolTransactionScanJob()

	ShouldAggregate(aggUtxoCount int, aggSumVal uint64) bool
}
