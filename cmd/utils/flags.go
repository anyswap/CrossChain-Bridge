package utils

import (
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/urfave/cli/v2"
)

var (
	ConfigFileFlag = &cli.StringFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   "Specify config file",
	}
	VerbosityFlag = &cli.Uint64Flag{
		Name:    "verbosity",
		Aliases: []string{"v"},
		Usage:   "log verbosity (0:panic, 1:fatal, 2:error, 3:warn, 4:info, 5:debug, 6:trace)",
		Value:   4,
	}
	JsonFormatFlag = &cli.BoolFlag{
		Name:  "json",
		Usage: "output log in json format",
	}
	ColorFormatFlag = &cli.BoolFlag{
		Name:  "color",
		Usage: "output log in color text format",
		Value: true,
	}
)

func SetLogger(ctx *cli.Context) {
	logLevel := ctx.Uint64(VerbosityFlag.Name)
	jsonFormat := ctx.Bool(JsonFormatFlag.Name)
	colorFormat := ctx.Bool(ColorFormatFlag.Name)
	log.SetLogger(uint32(logLevel), jsonFormat, colorFormat)
}

func GetConfigFilePath(ctx *cli.Context) string {
	return ctx.String(ConfigFileFlag.Name)
}
