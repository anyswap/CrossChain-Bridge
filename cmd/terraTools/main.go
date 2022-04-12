package main

import (
	"fmt"
	"os"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/eth"
	"github.com/anyswap/CrossChain-Bridge/tokens/terra"
	"github.com/urfave/cli/v2"
)

var (
	clientIdentifier = "terraTools"
	gitCommit        = ""
	gitDate          = ""

	app = utils.NewApp(clientIdentifier, gitCommit, gitDate, "the terra tools command line interface")

	br *terra.Bridge
)

func main() {
	initApp()
	initBridge()
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initApp() {
	app.Action = terraTools
	app.HideVersion = true
	app.Commands = []*cli.Command{
		pubkeyToAddressCommand,
		utils.VersionCommand,
	}
	app.Flags = []cli.Flag{
		utils.VerbosityFlag,
		utils.JSONFormatFlag,
		utils.ColorFormatFlag,
	}
}

func initBridge() {
	br = terra.NewCrossChainBridge(true)
	tokens.SrcBridge = br
	tokens.DstBridge = eth.NewCrossChainBridge(false)
}

func terraTools(ctx *cli.Context) error {
	utils.SetLogger(ctx)
	if ctx.NArg() > 0 {
		return fmt.Errorf("invalid command: %q", ctx.Args().Get(0))
	}

	_ = cli.ShowAppHelp(ctx)
	fmt.Println()
	return fmt.Errorf("please specify a sub command to run")
}
