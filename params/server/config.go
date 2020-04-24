package server

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/log"
)

const (
	defaultApiPort      = 11556
	defServerConfigFile = "server.toml"
)

var (
	serverConfig      *ServerConfig
	loadConfigStarter sync.Once
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

type ServerConfig struct {
	SwapServer *SwapServerConfig
	MongoDB    *MongoDBConfig
}

func GetConfig() *ServerConfig {
	return serverConfig
}

func SetConfig(config *ServerConfig) {
	serverConfig = config
}

func GetApiPort() int {
	apiPort := GetConfig().SwapServer.ApiPort
	if apiPort == 0 {
		apiPort = defaultApiPort
	}
	return apiPort
}

func LoadConfig(configFile string) *ServerConfig {
	loadConfigStarter.Do(func() {
		if configFile == "" {
			// find config file in the execute directory (default).
			dir, err := common.ExecuteDir()
			if err != nil {
				panic(fmt.Sprintf("LoadConfig error (get ExecuteDir): %v", err))
			}
			configFile = common.AbsolutePath(dir, defServerConfigFile)
		}
		log.Println("Config file is", configFile)
		if !common.FileExist(configFile) {
			panic(fmt.Sprintf("LoadConfig error: config file %v not exist", configFile))
		}
		config := &ServerConfig{}
		if _, err := toml.DecodeFile(configFile, &config); err != nil {
			panic(fmt.Sprintf("LoadConfig error (toml DecodeFile): %v", err))
		}

		SetConfig(config)
		var bs []byte
		if log.JsonFormat {
			bs, _ = json.Marshal(config)
		} else {
			bs, _ = json.MarshalIndent(config, "", "  ")
		}
		log.Println("LoadConfig finished.", string(bs))
	})
	return serverConfig
}
