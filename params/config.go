package params

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/BurntSushi/toml"
	"github.com/fsn-dev/crossChain-Bridge/common"
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

var (
	allConfig *AllConfig
)

const (
	configFileName = "config.toml"
)

func Config() *AllConfig {
	return allConfig
}

func LoadConfig() (*AllConfig, error) {
	if allConfig == nil {
		// find config file in the execute directory (default).
		dir, err := common.ExecuteDir()
		if err != nil {
			return nil, err
		}
		configFile := common.AbsolutePath(dir, configFileName)
		if !common.FileExist(configFile) {
			return nil, fmt.Errorf("config file %v not exist", configFile)
		}
		allConfig = &AllConfig{}
		if _, err := toml.DecodeFile(configFile, &allConfig); err != nil {
			return nil, err
		}
	}
	log.Println("LoadConfig finished.")
	bs, _ := json.MarshalIndent(allConfig, "", "  ")
	fmt.Println(string(bs))
	return allConfig, nil
}
