package riskctrl

import (
	"strings"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/eth"
	"github.com/anyswap/CrossChain-Bridge/tokens/fsn"
	"github.com/anyswap/CrossChain-Bridge/tools"
)

var (
	srcBridge *eth.Bridge
	dstBridge *fsn.Bridge
)

// InitCrossChainBridge init bridge
func InitCrossChainBridge() {
	cfg := GetConfig()
	srcToken := cfg.SrcToken
	dstToken := cfg.DestToken
	srcGateway := cfg.SrcGateway
	dstGateway := cfg.DestGateway

	srcID := srcToken.BlockChain
	dstID := dstToken.BlockChain
	srcNet := srcToken.NetID
	dstNet := dstToken.NetID

	if !strings.EqualFold(srcID, "ETHEREUM") || !strings.EqualFold(dstID, "FUSION") {
		log.Fatal("risk control only support eth 2 fsn bridge at present!!!")
	}

	srcBridge = eth.NewCrossChainBridge(true)
	dstBridge = fsn.NewCrossChainBridge(false)
	log.Info("New bridge finished", "source", srcID, "sourceNet", srcNet, "dest", dstID, "destNet", dstNet)

	srcBridge.SetTokenAndGatewayWithoutCheck(srcToken, srcGateway)
	log.Info("Init bridge source", "token", srcToken.Symbol, "gateway", srcGateway)

	dstBridge.SetTokenAndGatewayWithoutCheck(dstToken, dstGateway)
	log.Info("Init bridge destation", "token", dstToken.Symbol, "gateway", dstGateway)

	eth.InitExtCodeParts()

	srcBridge.VerifyConfig()
	dstBridge.VerifyConfig()
}

// InitEmailConfig init email config
func InitEmailConfig() {
	if riskConfig.Email == nil {
		log.Info("no email is config, ignore it")
		return
	}
	server := riskConfig.Email.Server
	port := riskConfig.Email.Port
	from := riskConfig.Email.From
	name := riskConfig.Email.FromName
	password := riskConfig.Email.Password
	tools.InitEmailConfig(server, port, from, name, password)
	log.Info("init email config", "server", server, "port", port, "from", from, "name", name)
}
