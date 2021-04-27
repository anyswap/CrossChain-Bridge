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
	gitDate          string
)

// NewApp creates an app with sane defaults.
func NewApp(identifier, gitcommit, gitdate, usage string) *cli.App {
	clientIdentifier = identifier
	gitCommit = gitcommit
	gitDate = gitdate
	app := cli.NewApp()
	app.Name = filepath.Base(os.Args[0])
	app.Version = params.VersionWithCommit(gitCommit, gitDate)
	app.Usage = usage
	return app
}
