package main

import (
	"os"

	"github.com/freedomfury/shopts/pkg/shopts"
)

// version is set at build time via -ldflags "-X main.version=v..."
var version = "dev"

func main() {
	os.Exit(shopts.Run(os.Args, os.Stdout, os.Stderr, version))
}
