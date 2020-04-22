package params

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/BurntSushi/toml"
	"github.com/fsn-dev/crossChain-Bridge/common"
)

const (
	defaultApiPort      = 11556
	defServerConfigFile = "server.toml"
)

type SwapServerConfig struct {
	SrcChainName     string
	SrcAssetSymbol   string
	SrcAssetDecimals uint8
	SrcDcrmAddress   string
	SrcRpcServer     string

	DestChainName       string
	DestAssetSymbol     string
	DestAssetDecimals   uint8
	DescContractAddress string
	DestRpcServer       string

	ApiPort int `toml:",omitempty" json:"-"`
}

type MongoDBConfig struct {
	DbURL    string
	DbName   string
	UserName string `json:"-"`
	Password string `json:"-"`
}

func (cfg *MongoDBConfig) GetURL() string {
	if cfg.UserName == "" && cfg.Password == "" {
		return cfg.DbURL
	}
	return fmt.Sprintf("%s:%s@%s", cfg.UserName, cfg.Password, cfg.DbURL)
}

type AllConfig struct {
	SwapServer *SwapServerConfig
	MongoDB    *MongoDBConfig
}

var allConfig *AllConfig

func ServerConfig() *AllConfig {
	return allConfig
}

func GetApiPort() int {
	apiPort := allConfig.SwapServer.ApiPort
	if apiPort == 0 {
		apiPort = defaultApiPort
	}
	return apiPort
}

func LoadConfig(configFile string) (*AllConfig, error) {
	if allConfig == nil {
		if configFile == "" {
			// find config file in the execute directory (default).
			dir, err := common.ExecuteDir()
			if err != nil {
				return nil, err
			}
			configFile = common.AbsolutePath(dir, defServerConfigFile)
		}
		log.Println("Config file is", configFile)
		if !common.FileExist(configFile) {
			return nil, fmt.Errorf("config file %v not exist", configFile)
		}
		allConfig = &AllConfig{}
		if _, err := toml.DecodeFile(configFile, &allConfig); err != nil {
			return nil, err
		}
	}
	bs, _ := json.MarshalIndent(allConfig, "", "  ")
	log.Println("LoadConfig finished.", string(bs))
	return allConfig, nil
}
