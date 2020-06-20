// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package build

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// DryRunFlag dry run flag
var DryRunFlag = flag.Bool("n", false, "dry run, don't execute commands")

// MustRun executes the given command and exits the host process for
// any error.
func MustRun(cmd *exec.Cmd) {
	fmt.Println(">>>", strings.Join(cmd.Args, " "))
	if !*DryRunFlag {
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}
	}
}

// MustRunCommand wrap MustRun
func MustRunCommand(cmd string, args ...string) {
	MustRun(exec.Command(cmd, args...))
}

var warnedAboutGit bool

// RunGit runs a git subcommand and returns its output.
// The command must complete successfully.
func RunGit(args ...string) string {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if e, ok := err.(*exec.Error); ok && e.Err == exec.ErrNotFound {
			if !warnedAboutGit {
				log.Println("Warning: can't find 'git' in PATH")
				warnedAboutGit = true
			}
			return ""
		}
		log.Fatal(strings.Join(cmd.Args, " "), ": ", err, "\n", stderr.String())
	}
	return strings.TrimSpace(stdout.String())
}

// readGitFile returns content of file in .git directory.
func readGitFile(file string) string {
	content, err := ioutil.ReadFile(path.Join(".git", file))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

// GoTool returns the command that runs a go tool. This uses go from GOROOT instead of PATH
// so that go commands executed by build use the same version of Go as the 'host' that runs
// build code. e.g.
//
//     /usr/lib/go-1.12.1/bin/go run build/ci.go ...
//
// runs using go 1.12.1 and invokes go 1.12.1 tools from the same GOROOT. This is also important
// because runtime.Version checks on the host should match the tools that are run.
func GoTool(tool string, args ...string) *exec.Cmd {
	args = append([]string{tool}, args...)
	return exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), args...) //nolint:gosec // any better way?
}
