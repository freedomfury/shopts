package main

import (
	"fmt"
	"os"

	"github.com/freedomfury/shopts/pkg/shopts"
)

// version is set at build time via -ldflags "-X main.version=v..."
var version = "dev"

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-V") {
		fmt.Println(version)
		return
	}
	if err := shopts.Run(os.Args, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
