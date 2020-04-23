package main

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/cmd/utils"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/fsn-dev/crossChain-Bridge/mongodb"
	"github.com/fsn-dev/crossChain-Bridge/params"
	"github.com/fsn-dev/crossChain-Bridge/rpc/server"
	"github.com/fsn-dev/crossChain-Bridge/worker"
	"github.com/urfave/cli/v2"
)

var (
	clientIdentifier = "swapserver"
	// Git SHA1 commit hash of the release (set via linker flags)
	gitCommit = ""
	// The app that holds all commands and flags.
	app = utils.NewApp(clientIdentifier, gitCommit, "the swapserver command line interface")
)

func init() {
	// Initialize the CLI app and start action
	app.Action = swapserver
	app.HideVersion = true // we have a command to print the version
	app.Copyright = "Copyright 2017-2020 The crossChain-Bridge Authors"
	app.Commands = []*cli.Command{
		utils.LicenseCommand,
		utils.VersionCommand,
	}
	app.Flags = []cli.Flag{
		utils.ConfigFileFlag,
		utils.VerbosityFlag,
		utils.JsonFormatFlag,
		utils.ColorFormatFlag,
	}
	sort.Sort(cli.CommandsByName(app.Commands))
}

func main() {
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
	configFile := utils.GetConfigFilePath(ctx)
	config, err := params.LoadConfig(configFile)
	if err != nil {
		panic(err)
	}

	dbConfig := config.MongoDB
	mongoURL := dbConfig.GetURL()
	dbName := dbConfig.DbName
	mongodb.MongoServerInit(mongoURL, dbName)

	worker.StartWork()

	time.Sleep(100 * time.Millisecond)

	for {
		server.StartAPIServer()
		time.Sleep(time.Duration(60) * time.Second)
		log.Println("restart API server")
	}
}
