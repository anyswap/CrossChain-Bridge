package utils

import (
	"os"
	"path/filepath"

	"github.com/anyswap/CrossChain-Bridge/params"
	"github.com/urfave/cli/v2"
)

var (
	clientIdentifier string
	gitCommit        string
)

// NewApp creates an app with sane defaults.
func NewApp(identifier, gitcommit, usage string) *cli.App {
	clientIdentifier = identifier
	gitCommit = gitcommit
	app := cli.NewApp()
	app.Name = filepath.Base(os.Args[0])
	app.Version = params.VersionWithMeta
	if len(gitCommit) >= 8 {
		app.Version += "-" + gitCommit[:8]
	}
	app.Usage = usage
	return app
}
