package main

import (
	"fmt"
	"os"

	"github.com/forgant-foundry/forglet/cmd/forglet/commands"
	gitplugin "github.com/forgant-foundry/forglet/internal/plugins/git"
)

func main() {
	commands.RegisterPlugin(gitplugin.New())
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
