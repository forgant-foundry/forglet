package main

import (
	"fmt"
	"os"

	"github.com/forgant-foundry/forglet/cmd/forglet/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
