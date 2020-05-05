package bridge

import (
	"fmt"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/dcrm"
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

func InitCrossChainBridge(isServer bool) {
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

	InitDcrm(cfg.Dcrm, isServer)
}

func InitDcrm(dcrmConfig *params.DcrmConfig, isServer bool) {
	dcrm.SetDcrmRpcAddress(*dcrmConfig.RpcAddress)
	if isServer {
		dcrm.SetSignPubkey(*dcrmConfig.Pubkey)
		log.Info("Init dcrm pubkey", "pubkey", *dcrmConfig.Pubkey)
	}
	group := *dcrmConfig.GroupID
	threshold := fmt.Sprintf("%d/%d", *dcrmConfig.NeededOracles, *dcrmConfig.TotalOracles)
	mode := fmt.Sprintf("%d", dcrmConfig.Mode)
	dcrm.SetDcrmGroup(group, threshold, mode)
	log.Info("Init dcrm rpc adress", "rpcaddress", *dcrmConfig.RpcAddress)
	log.Info("Init dcrm group", "group", group, "threshold", threshold, "mode", mode)

	err := dcrm.LoadKeyStore(*dcrmConfig.KeystoreFile, *dcrmConfig.PasswordFile)
	if err != nil {
		panic(err)
	}
	for {
		enode, err := dcrm.GetEnode()
		if err != nil {
			log.Error("InitDcrm can't get enode info", "err", err)
			time.Sleep(3 * time.Second)
		}
		log.Info("get dcrm enode info success", "enode", enode)
		break
	}
	log.Info("Init dcrm, load keystore success")
}
