package ltc

import (
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/ltc/electrs"
)

// BridgeInstance ltc bridge instance
var BridgeInstance BridgeInterface

// BridgeInterface ltc bridge interface
type BridgeInterface interface {
	tokens.CrossChainBridge

	GetCompressedPublicKey(fromPublicKey string, needVerify bool) (cPkData []byte, err error)
	GetP2shAddress(bindAddr string) (p2shAddress string, redeemScript []byte, err error)
	VerifyP2shTransaction(pairID, txHash, bindAddress string, allowUnstable bool) (*tokens.TxSwapInfo, error)
	//VerifyAggregateMsgHash(msgHash []string, args *tokens.BuildTxArgs) error
	AggregateUtxos(addrs []string, utxos []*electrs.ElectUtxo) (string, error)
	FindUtxos(addr string) ([]*electrs.ElectUtxo, error)
	StartSwapHistoryScanJob()
}
