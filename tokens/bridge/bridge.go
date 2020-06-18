package bridge

import (
	"fmt"
	"strings"
	"time"

	"github.com/btcsuite/btcutil"
	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/dcrm"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/params"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
	"github.com/fsn-dev/crossChain-Bridge/tokens/btc"
	"github.com/fsn-dev/crossChain-Bridge/tokens/eth"
	"github.com/fsn-dev/crossChain-Bridge/tokens/fsn"
)

// NewCrossChainBridge new bridge according to chain name
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
}

// InitCrossChainBridge init bridge
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

	initBtcExtra(cfg.BtcExtra)

	initDcrm(cfg.Dcrm, isServer)
}

func initBtcExtra(btcExtra *tokens.BtcExtraConfig) {
	if btc.BridgeInstance == nil || btcExtra == nil {
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
		maxRelayFeePerKb, _ := btcutil.NewAmount(0.001)
		relayFeePerKb := btcutil.Amount(tokens.BtcRelayFeePerKb)
		if relayFeePerKb > maxRelayFeePerKb {
			log.Fatal("BtcRelayFeePerKb is too large", "value", relayFeePerKb, "max", maxRelayFeePerKb)
		}
	}

	log.Info("Init Btc extra", "MinRelayFee", tokens.BtcMinRelayFee, "RelayFeePerKb", tokens.BtcRelayFeePerKb)

	if btcExtra.FromPublicKey != "" {
		tokens.BtcFromPublicKey = btcExtra.FromPublicKey
		pk := common.FromHex(tokens.BtcFromPublicKey)
		address, _ := btcutil.NewAddressPubKeyHash(btcutil.Hash160(pk), btc.BridgeInstance.GetChainConfig())
		pubkeyAddress := address.EncodeAddress()
		log.Info("Init Btc extra", "FromPublicKey", tokens.BtcFromPublicKey, "address", pubkeyAddress)

		btcDcrmAddress := btc.BridgeInstance.TokenConfig.DcrmAddress
		if pubkeyAddress != btcDcrmAddress {
			log.Fatal("BtcFromPublicKey's address mismatch dcrm address", "pubkeyAddress", pubkeyAddress, "dcrmAddress", btcDcrmAddress)
		}
	}

	if btcExtra.UtxoAggregateMinCount > 0 {
		tokens.BtcUtxoAggregateMinCount = btcExtra.UtxoAggregateMinCount
	}

	if btcExtra.UtxoAggregateMinValue > 0 {
		tokens.BtcUtxoAggregateMinValue = btcExtra.UtxoAggregateMinValue
	}

	log.Info("Init Btc extra", "UtxoAggregateMinCount", tokens.BtcUtxoAggregateMinCount, "UtxoAggregateMinValue", tokens.BtcUtxoAggregateMinValue)
}

func initDcrm(dcrmConfig *params.DcrmConfig, isServer bool) {
	dcrm.SetDcrmRPCAddress(*dcrmConfig.RPCAddress)
	log.Info("Init dcrm rpc adress", "rpcaddress", *dcrmConfig.RPCAddress)

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

	dcrm.ServerDcrmUser = common.HexToAddress(dcrmConfig.ServerAccount)
	log.Info("Init server dcrm user success", "ServerDcrmUser", dcrm.ServerDcrmUser.String())

	if isServer && !dcrm.IsSwapServer() {
		log.Error("wrong dcrm user for server", "have", dcrm.GetDcrmUser().String(), "want", dcrm.ServerDcrmUser.String())
		panic("wrong dcrm user for server")
	}

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

	for _, signGroupID := range dcrm.GetSignGroups() {
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
