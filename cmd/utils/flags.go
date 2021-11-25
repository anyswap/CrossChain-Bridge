package utils

import (
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/urfave/cli/v2"
)

var (
	// DataDirFlag --datadir
	DataDirFlag = &cli.StringFlag{
		Name:  "datadir",
		Usage: "data directory",
		Value: "",
	}
	// ConfigFileFlag -c|--config
	ConfigFileFlag = &cli.StringFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   "Specify config file",
	}
	// TokenPairsDirFlag --pairsdir
	TokenPairsDirFlag = &cli.StringFlag{
		Name:  "pairsdir",
		Usage: "Specify token pairs directory",
	}
	// LogFileFlag --log
	LogFileFlag = &cli.StringFlag{
		Name:  "log",
		Usage: "Specify log file, support rotate",
	}
	// LogRotationFlag --rotate
	LogRotationFlag = &cli.Uint64Flag{
		Name:  "rotate",
		Usage: "log rotation time (unit hour)",
		Value: 24,
	}
	// LogMaxAgeFlag --maxage
	LogMaxAgeFlag = &cli.Uint64Flag{
		Name:  "maxage",
		Usage: "log max age (unit hour)",
		Value: 7200,
	}
	// VerbosityFlag -v|--verbosity
	VerbosityFlag = &cli.Uint64Flag{
		Name:    "verbosity",
		Aliases: []string{"v"},
		Usage:   "log verbosity (0:panic, 1:fatal, 2:error, 3:warn, 4:info, 5:debug, 6:trace)",
		Value:   4,
	}
	// JSONFormatFlag --json
	JSONFormatFlag = &cli.BoolFlag{
		Name:  "json",
		Usage: "output log in json format",
	}
	// ColorFormatFlag --color
	ColorFormatFlag = &cli.BoolFlag{
		Name:  "color",
		Usage: "output log in color text format",
		Value: true,
	}
	// StartHeightFlag --start
	StartHeightFlag = &cli.Uint64Flag{
		Name:  "start",
		Usage: "start height (start inclusive)",
	}
	// EndHeightFlag --end
	EndHeightFlag = &cli.Uint64Flag{
		Name:  "end",
		Usage: "end height (end exclusive)",
	}
	// StableHeightFlag --stable
	StableHeightFlag = &cli.Uint64Flag{
		Name:  "stable",
		Usage: "stable height",
		Value: 5,
	}
	// JobsFlag --jobs
	JobsFlag = &cli.Uint64Flag{
		Name:  "jobs",
		Usage: "number of jobs",
		Value: 4,
	}
	// GatewayFlag --gateway
	GatewayFlag = &cli.StringFlag{
		Name:  "gateway",
		Usage: "gateway URL to connect",
	}
	// SwapServerFlag --swapserver
	SwapServerFlag = &cli.StringFlag{
		Name:  "swapserver",
		Usage: "swap server RPC address",
	}
	// DcrmAddressFlag --dcrmAddress
	DcrmAddressFlag = &cli.StringFlag{
		Name:  "dcrmAddress",
		Usage: "dcrm address",
	}
	// DepositAddressSliceFlag --deposit
	DepositAddressSliceFlag = &cli.StringSliceFlag{
		Name:  "deposit",
		Usage: "deposit address slice",
	}
	// DepositAddressFlag --deposit
	DepositAddressFlag = &cli.StringFlag{
		Name:  "deposit",
		Usage: "deposit address",
	}
	// TokenAddressSliceFlag --token
	TokenAddressSliceFlag = &cli.StringSliceFlag{
		Name:  "token",
		Usage: "token address slice",
	}
	// PairIDSliceFlag --pairid
	PairIDSliceFlag = &cli.StringSliceFlag{
		Name:  "pairid",
		Usage: "token pair id slice",
	}
	// KeystoreFileFlag --keystore
	KeystoreFileFlag = &cli.StringFlag{
		Name:  "keystore",
		Usage: "keystore file",
	}
	// PasswordFileFlag --password
	PasswordFileFlag = &cli.StringFlag{
		Name:  "password",
		Usage: "password file",
	}
	// SwapTypeFlag --swaptype
	SwapTypeFlag = &cli.StringFlag{
		Name:  "swaptype",
		Usage: "value can be swapin or swapout",
		Value: "swapin",
	}
)

// SetLogger set log level, json format, color, rotate ...
func SetLogger(ctx *cli.Context) {
	logLevel := ctx.Uint64(VerbosityFlag.Name)
	jsonFormat := ctx.Bool(JSONFormatFlag.Name)
	colorFormat := ctx.Bool(ColorFormatFlag.Name)
	log.SetLogger(uint32(logLevel), jsonFormat, colorFormat)

	logFile := ctx.String(LogFileFlag.Name)
	if logFile != "" {
		logRotation := ctx.Uint64(LogRotationFlag.Name)
		logMaxAge := ctx.Uint64(LogMaxAgeFlag.Name)
		log.SetLogFile(logFile, logRotation, logMaxAge)
	}
}

// GetDataDir specified by `--datadir`
func GetDataDir(ctx *cli.Context) string {
	return ctx.String(DataDirFlag.Name)
}

// GetConfigFilePath specified by `-c|--config`
func GetConfigFilePath(ctx *cli.Context) string {
	return ctx.String(ConfigFileFlag.Name)
}

// GetTokenPairsDir specified by `--pairsdir`
func GetTokenPairsDir(ctx *cli.Context) string {
	return ctx.String(TokenPairsDirFlag.Name)
}
