package params

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/fsn-dev/crossChain-Bridge/common"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/tokens"
)

const (
	defaultApiPort      = 11556
	defServerConfigFile = "config.toml"
)

var (
	serverConfig      *ServerConfig
	loadConfigStarter sync.Once
)

type ServerConfig struct {
	Identifier  string
	MongoDB     *MongoDBConfig   `toml:",omitempty"`
	ApiServer   *ApiServerConfig `toml:",omitempty"`
	SrcToken    *tokens.TokenConfig
	SrcGateway  *tokens.GatewayConfig
	DestToken   *tokens.TokenConfig
	DestGateway *tokens.GatewayConfig
	Dcrm        *DcrmConfig
	Oracle      *OracleConfig
}

type DcrmConfig struct {
	RpcAddress    *string
	GroupID       *string
	SignGroups    []string
	NeededOracles *uint32
	TotalOracles  *uint32
	Mode          uint32  // 0:managed 1:private (default 0)
	Pubkey        *string `toml:",omitempty"`
	KeystoreFile  *string `toml:",omitempty"`
	PasswordFile  *string `toml:",omitempty"`
}

type OracleConfig struct {
	ServerApiAddress string
}

type ApiServerConfig struct {
	Port int
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

func GetApiPort() int {
	apiPort := GetConfig().ApiServer.Port
	if apiPort == 0 {
		apiPort = defaultApiPort
	}
	return apiPort
}

func GetIdentifier() string {
	return GetConfig().Identifier
}

func GetConfig() *ServerConfig {
	return serverConfig
}

func SetConfig(config *ServerConfig) {
	serverConfig = config
}

func CheckConfig(isServer bool) (err error) {
	config := GetConfig()
	if config.Identifier == "" {
		return errors.New("server must config non empty 'Identifier'")
	}
	if isServer {
		if config.MongoDB == nil {
			return errors.New("server must config 'MongoDB'")
		}
		if config.ApiServer == nil {
			return errors.New("server must config 'ApiServer'")
		}
	} else {
		if config.Oracle != nil {
			err = config.Oracle.CheckConfig(isServer)
			if err != nil {
				return err
			}
		}
	}
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
	if config.Dcrm == nil {
		return errors.New("server must config 'Dcrm'")
	}
	err = config.Dcrm.CheckConfig(isServer)
	if err != nil {
		return err
	}
	err = config.SrcToken.CheckConfig(true)
	if err != nil {
		return err
	}
	err = config.DestToken.CheckConfig(false)
	if err != nil {
		return err
	}
	return nil
}

func (c *DcrmConfig) CheckConfig(isServer bool) (err error) {
	if c.RpcAddress == nil {
		return errors.New("dcrm must config 'RpcAddress'")
	}
	if c.GroupID == nil {
		return errors.New("dcrm must config 'GroupID'")
	}
	if c.NeededOracles == nil {
		return errors.New("dcrm must config 'NeededOracles'")
	}
	if c.TotalOracles == nil {
		return errors.New("dcrm must config 'TotalOracles'")
	}
	if c.Mode != 0 {
		return errors.New("dcrm must config 'Mode' to 0 (managed)")
	}
	if isServer {
		if c.Pubkey == nil {
			return errors.New("swap server dcrm must config 'Pubkey'")
		}
		if len(c.SignGroups) == 0 {
			return errors.New("swap server dcrm must config 'SignGroups'")
		}
	}
	if c.KeystoreFile == nil {
		return errors.New("dcrm must config 'KeystoreFile'")
	}
	if c.PasswordFile == nil {
		return errors.New("dcrm must config 'PasswordFile'")
	}
	return nil
}

func (c *OracleConfig) CheckConfig(isServer bool) (err error) {
	return nil
}

func LoadConfig(configFile string, isServer bool) *ServerConfig {
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
		if err := CheckConfig(isServer); err != nil {
			panic(fmt.Sprintf("Check server config error: %v", err))
		}
	})
	return serverConfig
}
