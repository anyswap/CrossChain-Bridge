package params

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/rpc/client"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

const (
	defaultAPIPort      = 11556
	defServerConfigFile = "config.toml"
)

var (
	serverConfig      *ServerConfig
	loadConfigStarter sync.Once

	// ServerAPIAddress server api address
	ServerAPIAddress string

	// DataDir datadir
	DataDir = "datadir"
)

// ServerConfig config items (decode from toml file)
type ServerConfig struct {
	Identifier  string
	MongoDB     *MongoDBConfig   `toml:",omitempty"`
	APIServer   *APIServerConfig `toml:",omitempty"`
	SrcToken    *tokens.TokenConfig
	SrcGateway  *tokens.GatewayConfig
	DestToken   *tokens.TokenConfig
	DestGateway *tokens.GatewayConfig
	Dcrm        *DcrmConfig
	Oracle      *OracleConfig          `toml:",omitempty"`
	BtcExtra    *tokens.BtcExtraConfig `toml:",omitempty"`
	Admins      []string
}

// DcrmConfig dcrm related config
type DcrmConfig struct {
	ServerAccount string
	RPCAddress    *string
	GroupID       *string
	SignGroups    []string
	NeededOracles *uint32
	TotalOracles  *uint32
	Mode          uint32  // 0:managed 1:private (default 0)
	Pubkey        *string `toml:",omitempty"`
	KeystoreFile  *string `toml:",omitempty"`
	PasswordFile  *string `toml:",omitempty"`
}

// OracleConfig oracle config
type OracleConfig struct {
	ServerAPIAddress string
}

// APIServerConfig api service config
type APIServerConfig struct {
	Port           int
	AllowedOrigins []string
}

// MongoDBConfig mongodb config
type MongoDBConfig struct {
	DBURL    string
	DBName   string
	UserName string `json:"-"`
	Password string `json:"-"`
}

// GetAPIPort get api service port
func GetAPIPort() int {
	apiPort := GetConfig().APIServer.Port
	if apiPort == 0 {
		apiPort = defaultAPIPort
	}
	return apiPort
}

// GetIdentifier get identifier (to distiguish in dcrm accept)
func GetIdentifier() string {
	return GetConfig().Identifier
}

// GetServerDcrmUser get server dcrm user (initiator of dcrm sign)
func GetServerDcrmUser() string {
	return GetConfig().Dcrm.ServerAccount
}

// GetConfig get config items structure
func GetConfig() *ServerConfig {
	return serverConfig
}

// SetConfig set config items
func SetConfig(config *ServerConfig) {
	serverConfig = config
}

// CheckConfig check config
func CheckConfig(isServer bool) (err error) {
	config := GetConfig()
	if config.Identifier == "" {
		return errors.New("server must config non empty 'Identifier'")
	}
	if isServer {
		if config.MongoDB == nil {
			return errors.New("server must config 'MongoDB'")
		}
		if config.APIServer == nil {
			return errors.New("server must config 'APIServer'")
		}
	} else {
		if config.Oracle == nil {
			return errors.New("oracle must config 'Oracle'")
		}
		err = config.Oracle.CheckConfig()
		if err != nil {
			return err
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

// CheckConfig check dcrm config
func (c *DcrmConfig) CheckConfig(isServer bool) (err error) {
	if c.RPCAddress == nil {
		return errors.New("dcrm must config 'RPCAddress'")
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
	if c.ServerAccount == "" {
		return errors.New("dcrm must config 'ServerAccount'")
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

// CheckConfig check oracle config
func (c *OracleConfig) CheckConfig() (err error) {
	ServerAPIAddress = c.ServerAPIAddress
	if ServerAPIAddress == "" {
		return errors.New("oracle must config 'ServerAPIAddress'")
	}
	var version string
	for {
		err = client.RPCPost(&version, ServerAPIAddress, "swap.GetVersionInfo")
		if err == nil {
			log.Info("oracle get server version info succeed", "version", version)
			break
		}
		log.Warn("oracle connect ServerAPIAddress failed", "ServerAPIAddress", ServerAPIAddress, "err", err)
		time.Sleep(3 * time.Second)
	}
	return err
}

// LoadConfig load config
func LoadConfig(configFile string, isServer bool) *ServerConfig {
	loadConfigStarter.Do(func() {
		if configFile == "" {
			// find config file in the execute directory (default).
			dir, err := common.ExecuteDir()
			if err != nil {
				log.Fatalf("LoadConfig error (get ExecuteDir): %v", err)
			}
			configFile = common.AbsolutePath(dir, defServerConfigFile)
		}
		log.Println("Config file is", configFile)
		if !common.FileExist(configFile) {
			log.Fatalf("LoadConfig error: config file %v not exist", configFile)
		}
		config := &ServerConfig{}
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
		if err := CheckConfig(isServer); err != nil {
			log.Fatalf("Check config failed. %v", err)
		}
	})
	return serverConfig
}

// SetDataDir set data dir
func SetDataDir(datadir string) {
	if datadir != "" {
		DataDir = datadir
	}
	_ = os.MkdirAll(DataDir, os.ModePerm)
}

// IsAdmin is admin
func IsAdmin(account string) bool {
	for _, admin := range serverConfig.Admins {
		if strings.EqualFold(account, admin) {
			return true
		}
	}
	return false
}
