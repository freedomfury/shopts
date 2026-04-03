package main

import (
	"fmt"
	"os"

	"github.com/freedomfury/shopts/pkg/shopts"
)

func main() {
	if err := shopts.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
