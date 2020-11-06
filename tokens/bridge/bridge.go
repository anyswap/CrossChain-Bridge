package bridge

import (
	"strings"

	"github.com/anyswap/CrossChain-Bridge/dcrm"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/block"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc"
	"github.com/anyswap/CrossChain-Bridge/tokens/eth"
	"github.com/anyswap/CrossChain-Bridge/tokens/fsn"
	"github.com/btcsuite/btcutil"
)

// NewCrossChainBridge new bridge according to chain name
func NewCrossChainBridge(id string, isSrc bool) tokens.CrossChainBridge {
	blockChainIden := strings.ToUpper(id)
	switch {
	case strings.HasPrefix(blockChainIden, "BITCOIN"):
		return btc.NewCrossChainBridge(isSrc)
	case strings.HasPrefix(blockChainIden, "ETHEREUM"):
		return eth.NewCrossChainBridge(isSrc)
	case strings.HasPrefix(blockChainIden, "FUSION"):
		return fsn.NewCrossChainBridge(isSrc)
	case blockChainIden == "BLOCK":
		return block.NewCrossChainBridge(isSrc)
	default:
		log.Fatalf("Unsupported block chain %v", id)
		return nil
	}
}

// InitCrossChainBridge init bridge
func InitCrossChainBridge(isServer bool) {
	cfg := params.GetConfig()
	srcChain := cfg.SrcChain
	dstChain := cfg.DestChain
	srcGateway := cfg.SrcGateway
	dstGateway := cfg.DestGateway

	srcID := srcChain.BlockChain
	dstID := dstChain.BlockChain
	srcNet := srcChain.NetID
	dstNet := dstChain.NetID

	tokens.SrcBridge = NewCrossChainBridge(srcID, true)
	tokens.DstBridge = NewCrossChainBridge(dstID, false)
	log.Info("New bridge finished", "source", srcID, "sourceNet", srcNet, "dest", dstID, "destNet", dstNet)

	tokens.SrcBridge.SetChainAndGateway(srcChain, srcGateway)
	log.Info("Init bridge source", "source", srcID, "gateway", srcGateway)

	tokens.DstBridge.SetChainAndGateway(dstChain, dstGateway)
	log.Info("Init bridge destation", "dest", dstID, "gateway", dstGateway)

	tokens.IsDcrmDisabled = cfg.Dcrm.Disable
	tokens.LoadTokenPairsConfig(true)

	initBtcWithExtra(cfg.BtcExtra)

	dcrm.Init(cfg.Dcrm, isServer)

	log.Info("Init bridge success", "isServer", isServer, "dcrmEnabled", !cfg.Dcrm.Disable)
}

func initBtcWithExtra(btcExtra *tokens.BtcExtraConfig) {
	if btc.BridgeInstance == nil {
		return
	}

	if len(tokens.GetTokenPairsConfig()) != 1 {
		log.Fatalf("Btc bridge does not support multiple tokens")
	}

	pairCfg, exist := tokens.GetTokenPairsConfig()[btc.PairID]
	if !exist {
		log.Fatalf("Btc bridge must have pairID %v", btc.PairID)
	}

	tokens.BtcFromPublicKey = pairCfg.SrcToken.DcrmPubkey
	_, err := btc.BridgeInstance.GetCompressedPublicKey(tokens.BtcFromPublicKey, true)
	if err != nil {
		log.Fatal("wrong btc dcrm public key", "err", err)
	}

	if btcExtra == nil {
		return
	}

	if btcExtra.MinRelayFee > 0 {
		tokens.BtcMinRelayFee = btcExtra.MinRelayFee
		maxMinRelayFee, _ := btcutil.NewAmount(0.001)
		minRelayFee := btcutil.Amount(tokens.BtcMinRelayFee)
		if minRelayFee > maxMinRelayFee {
			log.Fatal("BtcMinRelayFee is too large", "value", minRelayFee, "max", maxMinRelayFee)
		}
	}

	if btcExtra.RelayFeePerKb > 0 {
		tokens.BtcRelayFeePerKb = btcExtra.RelayFeePerKb
		maxRelayFeePerKb, _ := btcutil.NewAmount(0.01)
		relayFeePerKb := btcutil.Amount(tokens.BtcRelayFeePerKb)
		if relayFeePerKb > maxRelayFeePerKb {
			log.Fatal("BtcRelayFeePerKb is too large", "value", relayFeePerKb, "max", maxRelayFeePerKb)
		}
	}

	log.Info("Init Btc extra", "MinRelayFee", tokens.BtcMinRelayFee, "RelayFeePerKb", tokens.BtcRelayFeePerKb)

	if btcExtra.UtxoAggregateMinCount > 0 {
		tokens.BtcUtxoAggregateMinCount = btcExtra.UtxoAggregateMinCount
	}

	if btcExtra.UtxoAggregateMinValue > 0 {
		tokens.BtcUtxoAggregateMinValue = btcExtra.UtxoAggregateMinValue
	}

	tokens.BtcUtxoAggregateToAddress = btcExtra.UtxoAggregateToAddress
	if !btc.BridgeInstance.IsValidAddress(tokens.BtcUtxoAggregateToAddress) {
		log.Fatal("wrong utxo aggregate to address", "toAddress", tokens.BtcUtxoAggregateToAddress)
	}

	log.Info("Init Btc extra", "UtxoAggregateMinCount", tokens.BtcUtxoAggregateMinCount, "UtxoAggregateMinValue", tokens.BtcUtxoAggregateMinValue, "UtxoAggregateToAddress", tokens.BtcUtxoAggregateToAddress)
}
