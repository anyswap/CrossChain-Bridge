package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsn-dev/crossChain-Bridge/params"
	"github.com/urfave/cli/v2"
)

type logWriter struct {
}

func (writer logWriter) Write(bytes []byte) (int, error) {
	return fmt.Print(time.Now().UTC().Format("2006-01-02T15:04:05.999Z") + " " + string(bytes))
}

func InitLogger() {
	log.SetFlags(0)
	log.SetOutput(new(logWriter))
}

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
