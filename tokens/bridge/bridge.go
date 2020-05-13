package bridge

import (
	"fmt"
	"strings"
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

	srcID := srcToken.BlockChain
	dstID := dstToken.BlockChain
	srcNet := srcToken.NetID
	dstNet := dstToken.NetID

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
	log.Info("Init dcrm rpc adress", "rpcaddress", *dcrmConfig.RpcAddress)

	if isServer {
		dcrm.SetSignPubkey(*dcrmConfig.Pubkey)
		log.Info("Init dcrm pubkey", "pubkey", *dcrmConfig.Pubkey)
	}

	group := *dcrmConfig.GroupID
	neededOracles := *dcrmConfig.NeededOracles
	totalOracles := *dcrmConfig.TotalOracles
	threshold := fmt.Sprintf("%d/%d", neededOracles, totalOracles)
	mode := fmt.Sprintf("%d", dcrmConfig.Mode)
	dcrm.SetDcrmGroup(group, threshold, mode)
	log.Info("Init dcrm group", "group", group, "threshold", threshold, "mode", mode)

	signGroups := dcrmConfig.SignGroups
	log.Info("Init dcrm sign groups", "sigGgroups", signGroups)
	dcrm.SetSignGroups(signGroups)

	err := dcrm.LoadKeyStore(*dcrmConfig.KeystoreFile, *dcrmConfig.PasswordFile)
	if err != nil {
		panic(err)
	}
	log.Info("Init dcrm, load keystore success")

	// init selfEnode
	var selfEnode string
	for {
		selfEnode, err = dcrm.GetEnode()
		if err == nil {
			log.Info("get dcrm enode info success", "enode", selfEnode)
			break
		}
		log.Error("InitDcrm can't get enode info", "err", err)
		time.Sleep(3 * time.Second)
	}
	sepIndex := strings.Index(selfEnode, "@")
	if sepIndex == -1 {
		panic("wrong enode, has no '@' char")
	}

	// check after initing selfEnode
	checkExist := func(chekcedEnode string, enodes []string) bool {
		for _, enode := range enodes {
			if enode[:sepIndex] == chekcedEnode[:sepIndex] {
				return true
			}
		}
		return false
	}

	for {
		groupInfo, err := dcrm.GetGroupByID(group)
		if err == nil && uint32(groupInfo.Count) != totalOracles {
			panic(fmt.Sprintf("dcrm account group %v member count is not %v", group, totalOracles))
		}
		if err == nil && uint32(len(groupInfo.Enodes)) == totalOracles {
			log.Info("get dcrm group info success", "groupInfo", groupInfo)
			if !checkExist(selfEnode, groupInfo.Enodes) {
				panic(fmt.Sprintf("self enode %v not exist in group %v, groupInfo is %v\n", selfEnode, group, groupInfo))
			}
			break
		}
		if err != nil {
			log.Error("InitDcrm can't get right group info", "groupID", group, "groupInfo", groupInfo, "needCount", totalOracles, "err", err)
		} else {
			log.Error("InitDcrm can't get right group info", "groupID", group, "groupInfo", groupInfo, "needCount", totalOracles)
		}
		time.Sleep(3 * time.Second)
	}

	for _, signGroupID := range dcrm.SignGroups {
		if !isServer {
			break
		}
		for {
			signGroupInfo, err := dcrm.GetGroupByID(signGroupID)
			if err == nil && uint32(signGroupInfo.Count) != neededOracles {
				panic(fmt.Sprintf("sig group %v member count is not %v", signGroupID, neededOracles))
			}
			if err == nil && uint32(len(signGroupInfo.Enodes)) == neededOracles {
				log.Info("get dcrm sign group info success", "signGroupInfo", signGroupInfo)
				if !checkExist(selfEnode, signGroupInfo.Enodes) {
					panic(fmt.Sprintf("self enode %v not exist in group %v, signGroupInfo is %v\n", selfEnode, signGroupID, signGroupInfo))
				}
				break
			}
			if err != nil {
				log.Error("InitDcrm can't get right sign group info", "signGroupID", signGroupID, "signGroupInfo", signGroupInfo, "needCount", neededOracles, "err", err)
			} else {
				log.Error("InitDcrm can't get right sign group info", "signGroupID", signGroupID, "signGroupInfo", signGroupInfo, "needCount", neededOracles)
			}
			time.Sleep(3 * time.Second)
		}
	}
}
