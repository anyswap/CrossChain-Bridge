package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/fsn-dev/crossChain-Bridge/cmd/utils"
	"github.com/fsn-dev/crossChain-Bridge/log"
	"github.com/urfave/cli/v2"
)

var (
	clientIdentifier = "swaporacle"
	// Git SHA1 commit hash of the release (set via linker flags)
	gitCommit = ""
	// The app that holds all commands and flags.
	app = utils.NewApp(clientIdentifier, gitCommit, "the swaporacle command line interface")
)

func init() {
	// Initialize the CLI app and start action
	app.Action = swaporacle
	app.HideVersion = true // we have a command to print the version
	app.Copyright = "Copyright 2017-2020 The crossChain-Bridge Authors"
	app.Commands = []*cli.Command{
		utils.LicenseCommand,
		utils.VersionCommand,
	}
	app.Flags = []cli.Flag{
		utils.ConfigFileFlag,
	}
	sort.Sort(cli.CommandsByName(app.Commands))
}

func main() {
	if err := app.Run(os.Args); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func swaporacle(ctx *cli.Context) error {
	if ctx.NArg() > 0 {
		return fmt.Errorf("invalid command: %q", ctx.Args().Get(0))
	}
	log.Println("swap oracle stub")
	return nil
}
