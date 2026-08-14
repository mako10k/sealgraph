package main

import (
	"os"

	"github.com/mako10k/sealgraph/internal/cli"
)

func main() {
	os.Exit(cli.RunGitPlugin(os.Args[1:], os.Stdout, os.Stderr))
}
