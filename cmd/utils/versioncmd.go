package utils

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/urfave/cli/v2"
)

var (
	// VersionCommand version subcommand
	VersionCommand = &cli.Command{
		Action:    version,
		Name:      "version",
		Usage:     "Print version numbers",
		ArgsUsage: " ",
		Description: `
The output of this command is supposed to be machine-readable.
`,
	}
)

func version(ctx *cli.Context) error {
	fmt.Println(strings.Title(clientIdentifier))
	fmt.Println("Version:", params.VersionWithMeta)
	if gitCommit != "" {
		fmt.Println("Git Commit:", gitCommit)
	}
	if gitDate != "" {
		fmt.Println("Git Commit Date:", gitDate)
	}
	fmt.Println("Architecture:", runtime.GOARCH)
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("Operating System:", runtime.GOOS)
	fmt.Printf("GOPATH=%s\n", os.Getenv("GOPATH"))
	fmt.Printf("GOROOT=%s\n", runtime.GOROOT())
	return nil
}
