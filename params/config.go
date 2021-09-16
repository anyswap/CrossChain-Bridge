package params

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
)

const (
	defaultAPIPort = 11556
)

var (
	serverConfig      *ServerConfig
	loadConfigStarter sync.Once

	// ServerAPIAddress server api address
	ServerAPIAddress string
	// IsSwapServer swap server flag
	IsSwapServer bool
)

// ServerConfig config items (decode from toml file)
type ServerConfig struct {
	Identifier          string
	MustRegisterAccount bool             `toml:",omitempty" json:",omitempty"`
	MongoDB             *MongoDBConfig   `toml:",omitempty" json:",omitempty"`
	APIServer           *APIServerConfig `toml:",omitempty" json:",omitempty"`
	SrcChain            *tokens.ChainConfig
	SrcGateway          *tokens.GatewayConfig
	DestChain           *tokens.ChainConfig
	DestGateway         *tokens.GatewayConfig
	Dcrm                *DcrmConfig            `toml:",omitempty" json:",omitempty"`
	Oracle              *OracleConfig          `toml:",omitempty" json:",omitempty"`
	BtcExtra            *tokens.BtcExtraConfig `toml:",omitempty" json:",omitempty"`
	Extra               *ExtraConfig           `toml:",omitempty" json:",omitempty"`
	Admins              []string               `toml:",omitempty" json:",omitempty"`
}

// DcrmConfig dcrm related config
type DcrmConfig struct {
	Disable       bool
	APIPrefix     string
	RPCTimeout    uint64
	SignTimeout   uint64
	GroupID       *string
	NeededOracles *uint32
	TotalOracles  *uint32
	Mode          uint32 // 0:managed 1:private (default 0)
	Initiators    []string
	DefaultNode   *DcrmNodeConfig
	OtherNodes    []*DcrmNodeConfig `toml:",omitempty" json:",omitempty"`
}

// DcrmNodeConfig dcrm node config
type DcrmNodeConfig struct {
	RPCAddress   *string
	SignGroups   []string `toml:",omitempty" json:",omitempty"`
	KeystoreFile *string  `json:"-"`
	PasswordFile *string  `json:"-"`
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

// ExtraConfig extra config
type ExtraConfig struct {
	IsDebugMode              bool `toml:",omitempty" json:",omitempty"`
	IsSwapoutToStringAddress bool `toml:",omitempty" json:",omitempty"`

	MinReserveFee string
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

// MustRegisterAccount flag
func MustRegisterAccount() bool {
	return GetConfig().MustRegisterAccount
}

// IsSwapoutToStringAddress swapout to string address (eg. btc)
func IsSwapoutToStringAddress() bool {
	return GetExtraConfig() != nil && GetExtraConfig().IsSwapoutToStringAddress
}

// IsDebugMode is debug mode, add more debugging log infos
func IsDebugMode() bool {
	return GetExtraConfig() != nil && GetExtraConfig().IsDebugMode
}

// IsDcrmEnabled is dcrm enabled (for dcrm sign)
func IsDcrmEnabled() bool {
	return !GetConfig().Dcrm.Disable
}

// IsDcrmInitiator is initiator of dcrm sign
func IsDcrmInitiator(account string) bool {
	for _, initiator := range GetConfig().Dcrm.Initiators {
		if strings.EqualFold(account, initiator) {
			return true
		}
	}
	return false
}

// GetConfig get config items structure
func GetConfig() *ServerConfig {
	return serverConfig
}

// SetConfig set config items
func SetConfig(config *ServerConfig) {
	serverConfig = config
}

// GetExtraConfig get extra config
func GetExtraConfig() *ExtraConfig {
	return GetConfig().Extra
}

// LoadConfig load config
func LoadConfig(configFile string, isServer bool) *ServerConfig {
	loadConfigStarter.Do(func() {
		log.Println("Config file is", configFile)
		if !common.FileExist(configFile) {
			log.Fatalf("LoadConfig error: config file %v not exist", configFile)
		}
		config := &ServerConfig{}
		if _, err := toml.DecodeFile(configFile, &config); err != nil {
			log.Fatalf("LoadConfig error (toml DecodeFile): %v", err)
		}

		IsSwapServer = isServer
		if isServer {
			config.Oracle = nil
		} else {
			config.MongoDB = nil
			config.APIServer = nil
			config.Admins = nil
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
