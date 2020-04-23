package utils

import (
	"os"
	"path/filepath"

	"github.com/fsn-dev/crossChain-Bridge/params"
	"github.com/urfave/cli/v2"
)

var (
	clientIdentifier string
	gitCommit        string
)

// NewApp creates an app with sane defaults.
func NewApp(clientIdentifier_, gitCommit_, usage string) *cli.App {
	clientIdentifier = clientIdentifier_
	gitCommit = gitCommit_
	app := cli.NewApp()
	app.Name = filepath.Base(os.Args[0])
	app.Version = params.VersionWithMeta
	if len(gitCommit) >= 8 {
		app.Version += "-" + gitCommit[:8]
	}
	app.Usage = usage
	return app
}
