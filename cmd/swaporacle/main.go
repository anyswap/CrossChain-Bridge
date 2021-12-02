// Command swaporacle start the oracle node.
package main

import (
	"fmt"
	"os"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/worker"
	"github.com/urfave/cli/v2"
)

var (
	clientIdentifier = "swaporacle"
	// Git SHA1 commit hash of the release (set via linker flags)
	gitCommit = ""
	gitDate   = ""
	// The app that holds all commands and flags.
	app = utils.NewApp(clientIdentifier, gitCommit, gitDate, "the swaporacle command line interface")
)

func initApp() {
	// Initialize the CLI app and start action
	app.Action = swaporacle
	app.HideVersion = true // we have a command to print the version
	app.Copyright = "Copyright 2017-2020 The CrossChain-Bridge Authors"
	app.Commands = []*cli.Command{
		utils.LicenseCommand,
		utils.VersionCommand,
	}
	app.Flags = []cli.Flag{
		utils.DataDirFlag,
		utils.ConfigFileFlag,
		utils.TokenPairsDirFlag,
		utils.LogFileFlag,
		utils.LogRotationFlag,
		utils.LogMaxAgeFlag,
		utils.VerbosityFlag,
		utils.JSONFormatFlag,
		utils.ColorFormatFlag,
	}
}

func main() {
	initApp()
	if err := app.Run(os.Args); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func swaporacle(ctx *cli.Context) error {
	utils.SetLogger(ctx)
	if ctx.NArg() > 0 {
		return fmt.Errorf("invalid command: %q", ctx.Args().Get(0))
	}
	params.SetDataDir(utils.GetDataDir(ctx))
	configFile := utils.GetConfigFilePath(ctx)
	params.LoadConfig(configFile, false)

	tokens.SetTokenPairsDir(utils.GetTokenPairsDir(ctx))

	worker.StartWork(false)

	utils.TopWaitGroup.Wait()
	log.Info("swaporacle exit normally")
	return nil
}
