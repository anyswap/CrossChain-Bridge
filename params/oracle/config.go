package oracle

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/log"
)

const (
	defOracleConfigFile = "oracle.toml"
)

var (
	oracleConfig      *OracleConfig
	loadConfigStarter sync.Once
)

type SwapOracleConfig struct {
}

type OracleConfig struct {
	SwapOracle *SwapOracleConfig
}

func GetConfig() *OracleConfig {
	return oracleConfig
}

func SetConfig(config *OracleConfig) {
	oracleConfig = config
}

func LoadConfig(configFile string) *OracleConfig {
	loadConfigStarter.Do(func() {
		if configFile == "" {
			// find config file in the execute directory (default).
			dir, err := common.ExecuteDir()
			if err != nil {
				panic(fmt.Sprintf("LoadConfig error (get ExecuteDir): %v", err))
			}
			configFile = common.AbsolutePath(dir, defOracleConfigFile)
		}
		log.Println("Config file is", configFile)
		if !common.FileExist(configFile) {
			panic(fmt.Sprintf("LoadConfig error: config file %v not exist", configFile))
		}
		config := &OracleConfig{}
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
	return oracleConfig
}
