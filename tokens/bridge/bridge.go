package bridge

import (
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/params"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/tokens/btc"
	"github.com/fsn-dev/crossChain-Bridge/tokens/eth"
	"github.com/fsn-dev/crossChain-Bridge/tokens/fsn"
)

func NewCrossChainBridge(id string, isSrc bool) tokens.CrossChainBridge {
	switch id {
	case "Bitcoin":
		return btc.NewCrossChainBridge(isSrc)
	case "Ethereum":
		return eth.NewCrossChainBridge(isSrc)
	case "Fusion":
		return fsn.NewCrossChainBridge(isSrc)
	default:
		panic("Unsupported block chain " + id)
	}
	return nil
}

func InitCrossChainBridge() {
	cfg := params.GetConfig()
	srcToken := cfg.SrcToken
	dstToken := cfg.DestToken
	srcGateway := cfg.SrcGateway
	dstGateway := cfg.DestGateway

	srcID := *srcToken.BlockChain
	dstID := *dstToken.BlockChain
	srcNet := *srcToken.NetID
	dstNet := *dstToken.NetID

	tokens.SrcBridge = NewCrossChainBridge(srcID, true)
	tokens.DstBridge = NewCrossChainBridge(dstID, false)
	log.Info("New bridge finished", "source", srcID, "sourceNet", srcNet, "dest", dstID, "destNet", dstNet)

	tokens.SrcBridge.SetTokenAndGateway(srcToken, srcGateway)
	log.Info("Init bridge source", "token", srcToken.Symbol, "gateway", srcGateway)

	tokens.DstBridge.SetTokenAndGateway(dstToken, dstGateway)
	log.Info("Init bridge destation", "token", dstToken.Symbol, "gateway", dstGateway)
}
