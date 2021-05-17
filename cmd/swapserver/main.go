package main

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/params"
	rpcserver "github.com/anyswap/CrossChain-Bridge/rpc/server"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/worker"
	"github.com/urfave/cli/v2"
)

var (
	clientIdentifier = "swapserver"
	// Git SHA1 commit hash of the release (set via linker flags)
	gitCommit = ""
	gitDate   = ""
	// The app that holds all commands and flags.
	app = utils.NewApp(clientIdentifier, gitCommit, gitDate, "the swapserver command line interface")
)

func initApp() {
	// Initialize the CLI app and start action
	app.Action = swapserver
	app.HideVersion = true // we have a command to print the version
	app.Copyright = "Copyright 2017-2020 The CrossChain-Bridge Authors"
	app.Commands = []*cli.Command{
		utils.LicenseCommand,
		utils.VersionCommand,
	}
	app.Flags = []cli.Flag{
		utils.ConfigFileFlag,
		utils.TokenPairsDirFlag,
		utils.LogFileFlag,
		utils.LogRotationFlag,
		utils.LogMaxAgeFlag,
		utils.VerbosityFlag,
		utils.JSONFormatFlag,
		utils.ColorFormatFlag,
	}
	sort.Sort(cli.CommandsByName(app.Commands))
}

func main() {
	initApp()
	if err := app.Run(os.Args); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func swapserver(ctx *cli.Context) error {
	utils.SetLogger(ctx)
	if ctx.NArg() > 0 {
		return fmt.Errorf("invalid command: %q", ctx.Args().Get(0))
	}
	exitCh := make(chan struct{})
	params.IsSwapServer = true
	configFile := utils.GetConfigFilePath(ctx)
	config := params.LoadConfig(configFile, true)

	tokens.SetTokenPairsDir(utils.GetTokenPairsDir(ctx))

	dbConfig := config.MongoDB
	mongodb.MongoServerInit([]string{dbConfig.DBURL}, dbConfig.DBName, dbConfig.UserName, dbConfig.Password)

	worker.StartWork(true)
	time.Sleep(100 * time.Millisecond)
	rpcserver.StartAPIServer()

	<-exitCh
	return nil
}
