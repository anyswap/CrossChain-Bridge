package params

import (
	"encoding/json"
	"os"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
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
	MongoDB     *MongoDBConfig   `toml:",omitempty" json:",omitempty"`
	APIServer   *APIServerConfig `toml:",omitempty" json:",omitempty"`
	SrcChain    *tokens.ChainConfig
	SrcGateway  *tokens.GatewayConfig
	DestChain   *tokens.ChainConfig
	DestGateway *tokens.GatewayConfig
	Dcrm        *DcrmConfig            `toml:",omitempty" json:",omitempty"`
	Oracle      *OracleConfig          `toml:",omitempty" json:",omitempty"`
	BtcExtra    *tokens.BtcExtraConfig `toml:",omitempty" json:",omitempty"`
	Admins      []string               `toml:",omitempty" json:",omitempty"`
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

// HasAdmin has admin
func HasAdmin() bool {
	return len(serverConfig.Admins) != 0
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
