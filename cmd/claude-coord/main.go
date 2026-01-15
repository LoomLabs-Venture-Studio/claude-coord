package main

import (
	"os"

	"github.com/LoomLabs-Venture-Studio/claude-coord/internal/cli"
)

// Version is set by goreleaser via ldflags
var Version = "dev"

func main() {
	cli.SetVersion(Version)
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
