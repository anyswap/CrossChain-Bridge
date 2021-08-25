// Package build provide customized methods to build project.
// It can add external infos (eg. gitCommit, gitDate) to the version sub command.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/internal/build"
)

var gobin, _ = filepath.Abs(filepath.Join("build", "bin"))

func main() {
	log.SetFlags(log.Lshortfile)

	if _, err := os.Stat(filepath.Join("build", "ci.go")); os.IsNotExist(err) {
		log.Fatal("this script must be run from the root of the repository")
	}
	if len(os.Args) < 2 {
		log.Fatal("need subcommand as first argument")
	}
	switch os.Args[1] {
	case "install":
		doInstall(os.Args[2:])
	default:
		log.Fatal("unknown command ", os.Args[1])
	}
}

// Compiling

func doInstall(cmdline []string) {
	_ = flag.CommandLine.Parse(cmdline)
	env := build.Env()

	// Check Go version. People regularly open issues about compilation
	// failure with outdated Go. This should save them the trouble.
	if !strings.Contains(runtime.Version(), "devel") {
		// Figure out the minor version number since we can't textually compare (1.10 < 1.9)
		var minor int
		_, _ = fmt.Sscanf(strings.TrimPrefix(runtime.Version(), "go1."), "%d", &minor)

		if minor < 12 {
			log.Println("You have Go version", runtime.Version())
			log.Println("requires at least Go version 1.12 and cannot")
			log.Println("be compiled with an earlier version. Please upgrade your Go installation.")
			os.Exit(1)
		}
	}
	// Compile packages given as arguments, or everything if there are no arguments.
	packages := []string{"./..."}
	if flag.NArg() > 0 {
		packages = flag.Args()
	}

	goinstall := goTool("install", buildFlags(env)...)
	if runtime.GOARCH == "arm64" {
		goinstall.Args = append(goinstall.Args, "-p", "1")
	}
	goinstall.Args = append(goinstall.Args, "-v")
	goinstall.Args = append(goinstall.Args, packages...)
	build.MustRun(goinstall)
}

func buildFlags(env *build.Environment) (flags []string) {
	var ld []string
	if env.Commit != "" {
		ld = append(ld,
			"-X", "main.gitCommit="+env.Commit,
			"-X", "main.gitDate="+env.Date,
		)
	}
	if runtime.GOOS == "darwin" {
		ld = append(ld, "-s")
	}

	if len(ld) > 0 {
		flags = append(flags, "-ldflags", strings.Join(ld, " "))
	}
	return flags
}

func goTool(subcmd string, args ...string) *exec.Cmd {
	return goToolArch(runtime.GOARCH, os.Getenv("CC"), subcmd, args...)
}

func goToolArch(arch, cc, subcmd string, args ...string) *exec.Cmd {
	cmd := build.GoTool(subcmd, args...)
	if arch == "" || arch == runtime.GOARCH {
		cmd.Env = append(cmd.Env, "GOBIN="+gobin)
	} else {
		cmd.Env = append(cmd.Env, "CGO_ENABLED=1", "GOARCH="+arch)
	}
	if cc != "" {
		cmd.Env = append(cmd.Env, "CC="+cc)
	}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GOBIN=") {
			continue
		}
		cmd.Env = append(cmd.Env, e)
	}
	return cmd
}
