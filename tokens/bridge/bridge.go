package bridge

import (
	"fmt"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/dcrm"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/block"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc"
	"github.com/anyswap/CrossChain-Bridge/tokens/etc"
	"github.com/anyswap/CrossChain-Bridge/tokens/eth"
	"github.com/anyswap/CrossChain-Bridge/tokens/fsn"
	"github.com/anyswap/CrossChain-Bridge/tokens/ltc"
	"github.com/anyswap/CrossChain-Bridge/tokens/zol"
)

// NewCrossChainBridge new bridge according to chain name
func NewCrossChainBridge(id string, isSrc bool) tokens.CrossChainBridge {
	blockChainIden := strings.ToUpper(id)
	switch {
	case strings.HasPrefix(blockChainIden, "BITCOIN"):
		return btc.NewCrossChainBridge(isSrc)
	case strings.HasPrefix(blockChainIden, "LITECOIN"):
		return ltc.NewCrossChainBridge(isSrc)
	case strings.HasPrefix(blockChainIden, "BLOCK"):
		return block.NewCrossChainBridge(isSrc)
	case strings.HasPrefix(blockChainIden, "ETHCLASSIC"):
		return etc.NewCrossChainBridge(isSrc)
	case strings.HasPrefix(blockChainIden, "ETHEREUM"):
		return eth.NewCrossChainBridge(isSrc)
	case strings.HasPrefix(blockChainIden, "FUSION"):
		return fsn.NewCrossChainBridge(isSrc)
	case strings.HasPrefix(blockChainIden, "ZEROLIMIT"):
		return zol.NewCrossChainBridge(isSrc)
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

	tokens.AggregateIdentifier = fmt.Sprintf("%s:%s", params.GetIdentifier(), tokens.AggregateIdentifier)

	tokens.SrcBridge = NewCrossChainBridge(srcID, true)
	tokens.DstBridge = NewCrossChainBridge(dstID, false)
	log.Info("New bridge finished", "source", srcID, "sourceNet", srcNet, "dest", dstID, "destNet", dstNet)

	tokens.SrcBridge.SetChainAndGateway(srcChain, srcGateway)
	log.Info("Init bridge source", "source", srcID, "gateway", srcGateway)

	tokens.DstBridge.SetChainAndGateway(dstChain, dstGateway)
	log.Info("Init bridge destation", "dest", dstID, "gateway", dstGateway)

	tokens.IsDcrmDisabled = cfg.Dcrm.Disable
	tokens.LoadTokenPairsConfig(true)

	BlockChain := strings.ToUpper(srcChain.BlockChain)
	switch BlockChain {
	case "BITCOIN":
		btc.Init(cfg.BtcExtra)
	case "LITECOIN":
		ltc.Init(cfg.BtcExtra)
	case "BLOCK":
		block.Init(cfg.BtcExtra)
	case "ZEROLIMIT":
		zol.Init(cfg.BtcExtra)
	}

	dcrm.Init(cfg.Dcrm, isServer)

	log.Info("Init bridge success", "isServer", isServer, "dcrmEnabled", !cfg.Dcrm.Disable)
}
