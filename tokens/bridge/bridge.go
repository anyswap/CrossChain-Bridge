package bridge

import (
	"fmt"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/dcrm"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/btc"
	"github.com/anyswap/CrossChain-Bridge/tokens/eth"
	"github.com/anyswap/CrossChain-Bridge/tokens/fsn"
	"github.com/btcsuite/btcutil"
)

// NewCrossChainBridge new bridge according to chain name
func NewCrossChainBridge(id string, isSrc bool) tokens.CrossChainBridge {
	switch strings.ToUpper(id) {
	case "BITCOIN":
		return btc.NewCrossChainBridge(isSrc)
	case "ETHEREUM":
		return eth.NewCrossChainBridge(isSrc)
	case "FUSION":
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

	eth.InitExtCodeParts()
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
	log.Info("Init dcrm rpc address", "rpcaddress", *dcrmConfig.RPCAddress)

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

	selfEnode := initSelfEnode()

	verifyGroupInfo(group, totalOracles, selfEnode)

	if isServer {
		for _, signGroupID := range dcrm.GetSignGroups() {
			verifyGroupInfo(signGroupID, neededOracles, selfEnode)
		}
	}
}

func initSelfEnode() string {
	var (
		selfEnode string
		err       error
	)
	for {
		selfEnode, err = dcrm.GetEnode()
		if err == nil {
			log.Info("get dcrm enode info success", "enode", selfEnode)
			break
		}
		log.Error("InitDcrm can't get enode info", "err", err)
		time.Sleep(3 * time.Second)
	}
	return selfEnode
}

func checkExist(chekcedEnode string, enodes []string) bool {
	sepIndex := strings.Index(chekcedEnode, "@")
	if sepIndex == -1 {
		panic("wrong enode, has no '@' char")
	}
	for _, enode := range enodes {
		if enode[:sepIndex] == chekcedEnode[:sepIndex] {
			return true
		}
	}
	return false
}

func verifyGroupInfo(groupID string, memberCount uint32, selfEnode string) {
	for {
		groupInfo, err := dcrm.GetGroupByID(groupID)
		if err == nil && uint32(groupInfo.Count) != memberCount {
			panic(fmt.Sprintf("dcrm group %v member count is not %v", groupID, memberCount))
		}
		if err == nil && uint32(len(groupInfo.Enodes)) == memberCount {
			log.Info("get dcrm group info success", "groupInfo", groupInfo)
			if !checkExist(selfEnode, groupInfo.Enodes) {
				panic(fmt.Sprintf("self enode %v not exist in group %v, groupInfo is %v\n", selfEnode, groupID, groupInfo))
			}
			break
		}
		if err != nil {
			log.Error("InitDcrm get group info failed", "groupID", groupID, "groupInfo", groupInfo, "needCount", memberCount, "err", err)
		} else {
			log.Error("InitDcrm get group info with wrong number of enodes", "groupID", groupID, "groupInfo", groupInfo, "needCount", memberCount)
		}
		time.Sleep(3 * time.Second)
	}
}
