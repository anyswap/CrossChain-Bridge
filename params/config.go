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
	locDataDir        string
	bridgeConfig      *BridgeConfig
	loadConfigStarter sync.Once

	// IsSwapServer if true then it's swap server, otherwise it's swap oracle
	IsSwapServer bool

	// ServerAPIAddress server api address
	ServerAPIAddress string

	// GetBalanceBlockNumberOpt pending or latest
	GetBalanceBlockNumberOpt = "latest"
)

// variables used for testing
var (
	// ChanIn channel to receive input arguments
	ChanIn = make(chan map[string]string)
	// ChanOut channel to send output result
	ChanOut = make(chan string)
)

// BridgeConfig config items (decode from toml file)
type BridgeConfig struct {
	Identifier  string
	SrcChain    *tokens.ChainConfig
	SrcGateway  *tokens.GatewayConfig
	DestChain   *tokens.ChainConfig
	DestGateway *tokens.GatewayConfig
	TokenPrice  *tokens.TokenPriceConfig
	Server      *ServerConfig          `toml:",omitempty" json:",omitempty"`
	Oracle      *OracleConfig          `toml:",omitempty" json:",omitempty"`
	BtcExtra    *tokens.BtcExtraConfig `toml:",omitempty" json:",omitempty"`
	Extra       *ExtraConfig           `toml:",omitempty" json:",omitempty"`
	Dcrm        *DcrmConfig            `toml:",omitempty" json:",omitempty"`
}

// ServerConfig swap server config
type ServerConfig struct {
	MongoDB    *MongoDBConfig   `toml:",omitempty" json:",omitempty"`
	APIServer  *APIServerConfig `toml:",omitempty" json:",omitempty"`
	Admins     []string         `toml:",omitempty" json:",omitempty"`
	Assistants []string         `toml:",omitempty" json:",omitempty"`

	SendTxLoopCount    int `toml:",omitempty" json:",omitempty"`
	SendTxLoopInterval int `toml:",omitempty" json:",omitempty"`
}

// DcrmConfig dcrm related config
type DcrmConfig struct {
	Disable     bool
	SignType    string // ECDSA, ED25519 etc.
	APIPrefix   string
	RPCTimeout  uint64
	SignTimeout uint64

	VerifySignatureInAccept bool `toml:",omitempty" json:",omitempty"`

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
	ServerAPIAddress      string
	GetAcceptListInterval uint64
	PendingInvalidAccept  bool `toml:",omitempty" json:",omitempty"`
}

// APIServerConfig api service config
type APIServerConfig struct {
	Port             int
	AllowedOrigins   []string
	MaxRequestsLimit int
}

// MongoDBConfig mongodb config
type MongoDBConfig struct {
	DBURL    string   `toml:",omitempty" json:",omitempty"`
	DBURLs   []string `toml:",omitempty" json:",omitempty"`
	DBName   string
	UserName string `json:"-"`
	Password string `json:"-"`
}

// ExtraConfig extra config
type ExtraConfig struct {
	IsTestMode               bool `toml:",omitempty" json:",omitempty"`
	IsDebugMode              bool `toml:",omitempty" json:",omitempty"`
	MustRegisterAccount      bool
	IsSwapoutToStringAddress bool `toml:",omitempty" json:",omitempty"`
	EnableCheckBlockFork     bool
	IsNullSwapoutNativeMemo  bool `toml:",omitempty" json:",omitempty"`
	UsePendingBalance        bool `toml:",omitempty" json:",omitempty"`
	CheckBindAddrIsContract  bool `toml:",omitempty" json:",omitempty"`
}

// GetAPIPort get api service port
func GetAPIPort() int {
	apiPort := GetServerConfig().APIServer.Port
	if apiPort == 0 {
		apiPort = defaultAPIPort
	}
	return apiPort
}

// GetIdentifier get identifier (to distiguish in dcrm accept)
func GetIdentifier() string {
	return GetConfig().Identifier
}

// GetReplaceIdentifier get identifier (to distiguish in dcrm accept)
func GetReplaceIdentifier() string {
	return GetConfig().Identifier + ":replaceswap"
}

// MustRegisterAccount flag
func MustRegisterAccount() bool {
	return GetExtraConfig() != nil && GetExtraConfig().MustRegisterAccount
}

// IsSwapoutToStringAddress swapout to string address (eg. btc)
func IsSwapoutToStringAddress() bool {
	return GetExtraConfig() != nil && GetExtraConfig().IsSwapoutToStringAddress
}

// EnableCheckBlockFork enable check block fork
func EnableCheckBlockFork() bool {
	return GetExtraConfig() != nil && GetExtraConfig().EnableCheckBlockFork
}

// IsNullSwapoutNativeMemo set no unlock memo in building swapout tx
func IsNullSwapoutNativeMemo() bool {
	return GetExtraConfig() != nil && GetExtraConfig().IsNullSwapoutNativeMemo
}

// IsTestMode is test mode (get rid of business related components: MPC, DB, etc.)
func IsTestMode() bool {
	return GetExtraConfig() != nil && GetExtraConfig().IsTestMode
}

// IsDebugMode is debug mode, add more debugging log infos
func IsDebugMode() bool {
	return GetExtraConfig() != nil && GetExtraConfig().IsDebugMode
}

// CheckBindAddrIsContract check if bind address is contract
func CheckBindAddrIsContract() bool {
	return GetExtraConfig() != nil && GetExtraConfig().CheckBindAddrIsContract
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

// GetConfig get bridge config
func GetConfig() *BridgeConfig {
	return bridgeConfig
}

// SetConfig set bridge config
func SetConfig(config *BridgeConfig) {
	bridgeConfig = config
	tokens.TokenPriceCfg = config.TokenPrice
}

// GetServerConfig get server config
func GetServerConfig() *ServerConfig {
	return GetConfig().Server
}

// GetOracleConfig get oracle config
func GetOracleConfig() *OracleConfig {
	return GetConfig().Oracle
}

// GetExtraConfig get extra config
func GetExtraConfig() *ExtraConfig {
	return GetConfig().Extra
}

// GetTokenPriceConfig get token price config
func GetTokenPriceConfig() *tokens.TokenPriceConfig {
	return GetConfig().TokenPrice
}

// LoadConfig load config
func LoadConfig(configFile string, isServer bool) *BridgeConfig {
	loadConfigStarter.Do(func() {
		if configFile == "" {
			log.Fatalf("LoadConfig error: no config file specified")
		}
		log.Println("Config file is", configFile)
		if !common.FileExist(configFile) {
			log.Fatalf("LoadConfig error: config file %v not exist", configFile)
		}
		config := &BridgeConfig{}
		if _, err := toml.DecodeFile(configFile, &config); err != nil {
			log.Fatalf("LoadConfig error (toml DecodeFile): %v", err)
		}

		if isServer {
			config.Oracle = nil
		} else {
			config.Server = nil
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
		log.Info("Check config success", "isServer", isServer, "configFile", configFile)
	})
	return bridgeConfig
}

// HasAdmin has admin
func HasAdmin() bool {
	return len(GetServerConfig().Admins) != 0
}

// IsAdmin is admin
func IsAdmin(account string) bool {
	for _, admin := range GetServerConfig().Admins {
		if strings.EqualFold(account, admin) {
			return true
		}
	}
	return false
}

// IsAssistant is assistant
func IsAssistant(account string) bool {
	for _, assistant := range GetServerConfig().Assistants {
		if strings.EqualFold(account, assistant) {
			return true
		}
	}
	return false
}

// SetDataDir set data dir
func SetDataDir(dir string) {
	if dir == "" {
		if !IsSwapServer {
			log.Warn("suggest specify '--datadir' to enhance accept job")
		}
		return
	}
	currDir, err := common.CurrentDir()
	if err != nil {
		log.Fatal("get current dir failed", "err", err)
	}
	locDataDir = common.AbsolutePath(currDir, dir)
	log.Info("set data dir success", "datadir", locDataDir)
}

// GetDataDir get data dir
func GetDataDir() string {
	return locDataDir
}
