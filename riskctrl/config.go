package riskctrl

import (
	"encoding/json"
	"errors"

	"github.com/BurntSushi/toml"
	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

var (
	riskConfig *RiskConfig
)

// RiskConfig risk config
type RiskConfig struct {
	SrcChain   *tokens.ChainConfig
	SrcToken   *tokens.TokenConfig
	SrcGateway *tokens.GatewayConfig

	DestChain   *tokens.ChainConfig
	DestToken   *tokens.TokenConfig
	DestGateway *tokens.GatewayConfig

	Email *EmailConfig

	InitialDiffValue         float64
	MaxAuditBalanceDiffValue float64
	MaxAuditSupplyDiffValue  float64
	MinWithdrawReserve       float64
}

// EmailConfig email config
type EmailConfig struct {
	Server   string
	Port     int
	From     string
	FromName string
	Password string `json:"-"`
	To       []string
	Cc       []string
}

// GetConfig get config
func GetConfig() *RiskConfig {
	return riskConfig
}

// SetConfig set config
func SetConfig(config *RiskConfig) {
	riskConfig = config
}

// CheckConfig check config
func CheckConfig() (err error) {
	err = checkTokenConfig()
	if err != nil {
		return err
	}
	config := GetConfig()
	if config.MaxAuditBalanceDiffValue <= 0 {
		return errors.New("server must config positive 'MaxAuditBalanceDiffValue'")
	}
	if config.MaxAuditSupplyDiffValue <= 0 {
		return errors.New("server must config positive 'MaxAuditSupplyDiffValue'")
	}
	return nil
}

func checkTokenConfig() (err error) {
	config := GetConfig()
	if config.SrcToken == nil {
		return errors.New("server must config 'SrcToken'")
	}
	if config.SrcGateway == nil {
		return errors.New("server must config 'SrcGateway'")
	}
	if config.DestToken == nil {
		return errors.New("server must config 'DestToken'")
	}
	if config.DestGateway == nil {
		return errors.New("server must config 'DestGateway'")
	}
	return nil
}

// LoadConfig load config
func LoadConfig(configFile string) *RiskConfig {
	log.Printf("Config file is '%v'", configFile)
	if !common.FileExist(configFile) {
		log.Fatalf("LoadConfig error: config file '%v' not exist", configFile)
	}
	config := &RiskConfig{}
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		log.Fatalf("LoadConfig error (toml DecodeFile): %v", err)
	}

	SetConfig(config)

	var bs []byte
	if log.JSONFormat {
		bs, _ = json.Marshal(config)
	} else {
		bs, _ = json.MarshalIndent(config, "", "  ")
	}
	log.Println("LoadConfig finished.", string(bs))

	if err := CheckConfig(); err != nil {
		log.Fatalf("Check config failed. %v", err)
	}

	return riskConfig
}
