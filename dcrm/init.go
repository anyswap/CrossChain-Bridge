// Package dcrm is a client of dcrm server, doing the sign and accept tasks.
package dcrm

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tools"
	"github.com/anyswap/CrossChain-Bridge/tools/keystore"
	"github.com/anyswap/CrossChain-Bridge/types"
)

const (
	dcrmToAddress       = "0x00000000000000000000000000000000000000dc"
	dcrmWalletServiceID = 30400
)

var (
	dcrmSigner = types.MakeSigner("EIP155", big.NewInt(dcrmWalletServiceID))
	dcrmToAddr = common.HexToAddress(dcrmToAddress)

	dcrmAPIPrefix     = "dcrm_" // default prefix
	dcrmGroupID       string
	dcrmThreshold     string
	dcrmMode          string
	dcrmNeededOracles uint32
	dcrmTotalOracles  uint32

	dcrmRPCTimeout  = 10                // default to 10 seconds
	dcrmSignTimeout = 120 * time.Second // default to 120 seconds

	defaultDcrmNode   *NodeInfo
	allInitiatorNodes []*NodeInfo // server only

	selfEnode string
	allEnodes []string
)

// NodeInfo dcrm node info
type NodeInfo struct {
	keyWrapper     *keystore.Key
	dcrmUser       common.Address
	dcrmRPCAddress string
	signGroups     []string // sub groups for sign
}

// Init init dcrm
func Init(dcrmConfig *params.DcrmConfig, isServer bool) {
	if dcrmConfig.Disable {
		return
	}

	if dcrmConfig.APIPrefix != "" {
		dcrmAPIPrefix = dcrmConfig.APIPrefix
	}

	if dcrmConfig.RPCTimeout > 0 {
		dcrmRPCTimeout = int(dcrmConfig.RPCTimeout)
	}
	if dcrmConfig.SignTimeout > 0 {
		dcrmSignTimeout = time.Duration(dcrmConfig.SignTimeout * uint64(time.Second))
	}

	setDcrmGroup(*dcrmConfig.GroupID, dcrmConfig.Mode, *dcrmConfig.NeededOracles, *dcrmConfig.TotalOracles)
	setDefaultDcrmNodeInfo(initDcrmNodeInfo(dcrmConfig.DefaultNode, isServer))

	if isServer {
		for _, nodeCfg := range dcrmConfig.OtherNodes {
			initDcrmNodeInfo(nodeCfg, isServer)
		}
	}

	initSelfEnode()
	initAllEnodes()

	verifyInitiators(dcrmConfig.Initiators)
	log.Info("init dcrm success", "signTimeout", dcrmSignTimeout.String(), "isServer", isServer)
}

// setDefaultDcrmNodeInfo set default dcrm node info
func setDefaultDcrmNodeInfo(nodeInfo *NodeInfo) {
	defaultDcrmNode = nodeInfo
}

// GetAllInitiatorNodes get all initiator dcrm node info
func GetAllInitiatorNodes() []*NodeInfo {
	return allInitiatorNodes
}

// addInitiatorNode add initiator dcrm node info
func addInitiatorNode(nodeInfo *NodeInfo) {
	if nodeInfo.dcrmRPCAddress == "" {
		log.Fatal("initiator: empty dcrm rpc address")
	}
	if nodeInfo.dcrmUser == (common.Address{}) {
		log.Fatal("initiator: empty dcrm user")
	}
	if len(nodeInfo.signGroups) == 0 {
		log.Fatal("initiator: empty sign groups")
	}
	for _, oldNode := range allInitiatorNodes {
		if oldNode.dcrmRPCAddress == nodeInfo.dcrmRPCAddress ||
			oldNode.dcrmUser == nodeInfo.dcrmUser {
			log.Fatal("duplicate initiator", "user", nodeInfo.dcrmUser, "rpcAddr", nodeInfo.dcrmRPCAddress)
		}
	}
	allInitiatorNodes = append(allInitiatorNodes, nodeInfo)
}

// IsSwapServer returns if this dcrm user is the swap server
func IsSwapServer() bool {
	return len(allInitiatorNodes) > 0
}

// setDcrmGroup set dcrm group
func setDcrmGroup(group string, mode, neededOracles, totalOracles uint32) {
	dcrmGroupID = group
	dcrmNeededOracles = neededOracles
	dcrmTotalOracles = totalOracles
	dcrmThreshold = fmt.Sprintf("%d/%d", neededOracles, totalOracles)
	dcrmMode = fmt.Sprintf("%d", mode)
	log.Info("Init dcrm group", "group", dcrmGroupID, "threshold", dcrmThreshold, "mode", dcrmMode)
}

// GetGroupID return dcrm group id
func GetGroupID() string {
	return dcrmGroupID
}

// GetSelfEnode get self enode
func GetSelfEnode() string {
	return selfEnode
}

// GetAllEnodes get all enodes
func GetAllEnodes() []string {
	return allEnodes
}

// setDcrmRPCAddress set dcrm node rpc address
func (ni *NodeInfo) setDcrmRPCAddress(url string) {
	ni.dcrmRPCAddress = url
}

// GetDcrmRPCAddress get dcrm node rpc address
func (ni *NodeInfo) GetDcrmRPCAddress() string {
	return ni.dcrmRPCAddress
}

// setSignGroups set sign subgroups
func (ni *NodeInfo) setSignGroups(groups []string) {
	ni.signGroups = groups
}

// GetSignGroups get sign subgroups
func (ni *NodeInfo) GetSignGroups() []string {
	return ni.signGroups
}

// GetDcrmUser returns the dcrm user of specified keystore
func (ni *NodeInfo) GetDcrmUser() common.Address {
	return ni.dcrmUser
}

// LoadKeyStore load keystore
func (ni *NodeInfo) LoadKeyStore(keyfile, passfile string) (common.Address, error) {
	key, err := tools.LoadKeyStore(keyfile, passfile)
	if err != nil {
		return common.Address{}, err
	}
	ni.keyWrapper = key
	ni.dcrmUser = ni.keyWrapper.Address
	return ni.dcrmUser, nil
}

func initSelfEnode() {
	for {
		enode, err := GetEnode(defaultDcrmNode.dcrmRPCAddress)
		if err == nil {
			selfEnode = enode
			log.Info("get dcrm enode info success", "enode", enode)
			return
		}
		log.Error("can't get enode info", "rpcAddr", defaultDcrmNode.dcrmRPCAddress, "err", err)
		time.Sleep(10 * time.Second)
	}
}

func isEnodeExistIn(enode string, enodes []string) bool {
	sepIndex := strings.Index(enode, "@")
	if sepIndex == -1 {
		log.Fatal("wrong self enode, has no '@' char", "enode", enode)
	}
	cmpStr := enode[:sepIndex]
	for _, item := range enodes {
		if item[:sepIndex] == cmpStr {
			return true
		}
	}
	return false
}

func initAllEnodes() {
	allEnodes = verifySignGroupInfo(defaultDcrmNode.dcrmRPCAddress, dcrmGroupID, false, true)
}

func verifySignGroupInfo(rpcAddr, groupID string, isSignGroup, includeSelf bool) []string {
	memberCount := dcrmTotalOracles
	if isSignGroup {
		memberCount = dcrmNeededOracles
	}
	for {
		groupInfo, err := GetGroupByID(groupID, rpcAddr)
		if err != nil {
			log.Error("get group info failed", "groupID", groupID, "err", err)
			time.Sleep(10 * time.Second)
			continue
		}
		log.Info("get dcrm group info success", "groupInfo", groupInfo)
		if uint32(groupInfo.Count) != memberCount {
			log.Fatal("dcrm group member count mismatch", "groupID", dcrmGroupID, "have", groupInfo.Count, "want", memberCount)
		}
		if uint32(len(groupInfo.Enodes)) != memberCount {
			log.Fatal("get group info enodes count mismatch", "groupID", groupID, "have", len(groupInfo.Enodes), "want", memberCount)
		}
		exist := isEnodeExistIn(selfEnode, groupInfo.Enodes)
		if exist != includeSelf {
			log.Fatal("self enode's existence in group mismatch", "groupID", groupID, "groupInfo", groupInfo, "want", includeSelf, "have", exist)
		}
		if isSignGroup {
			for _, enode := range groupInfo.Enodes {
				if !isEnodeExistIn(enode, allEnodes) {
					log.Fatal("sign group has unrelated enode", "groupID", groupID, "enode", enode)
				}
			}
		}
		return groupInfo.Enodes
	}
}

func verifyInitiators(initiators []string) {
	if len(allInitiatorNodes) == 0 {
		return
	}
	if len(initiators) != len(allInitiatorNodes) {
		log.Fatal("initiators count mismatch", "initiators", len(initiators), "initiatorNodes", len(allInitiatorNodes))
	}

	isInGroup := true
	for _, dcrmNodeInfo := range allInitiatorNodes {
		exist := false
		dcrmUser := dcrmNodeInfo.dcrmUser.String()
		for _, initiator := range initiators {
			if strings.EqualFold(initiator, dcrmUser) {
				exist = true
			}
		}
		if !exist {
			log.Fatal("initiator misatch", "user", dcrmUser)
		}
		for _, signGroupID := range dcrmNodeInfo.GetSignGroups() {
			verifySignGroupInfo(dcrmNodeInfo.dcrmRPCAddress, signGroupID, true, isInGroup)
		}
		isInGroup = false
	}
}

func initDcrmNodeInfo(dcrmNodeCfg *params.DcrmNodeConfig, isServer bool) *NodeInfo {
	dcrmNodeInfo := &NodeInfo{}
	dcrmNodeInfo.setDcrmRPCAddress(*dcrmNodeCfg.RPCAddress)
	log.Info("Init dcrm rpc address", "rpcaddress", *dcrmNodeCfg.RPCAddress)

	dcrmUser, err := dcrmNodeInfo.LoadKeyStore(*dcrmNodeCfg.KeystoreFile, *dcrmNodeCfg.PasswordFile)
	if err != nil {
		log.Fatalf("load keystore error %v", err)
	}
	log.Info("Init dcrm, load keystore success", "user", dcrmUser.String())

	if isServer {
		if !params.IsDcrmInitiator(dcrmUser.String()) {
			log.Fatalf("server dcrm user %v is not in configed initiators", dcrmUser.String())
		}

		signGroups := dcrmNodeCfg.SignGroups
		log.Info("Init dcrm sign groups", "signGroups", signGroups)
		dcrmNodeInfo.setSignGroups(signGroups)
		addInitiatorNode(dcrmNodeInfo)
	}

	return dcrmNodeInfo
}
