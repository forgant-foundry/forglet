package main

import (
	"fmt"
	"os"

	"github.com/forgant-foundry/forglet/cmd/forglet/commands"
	npmpolicy "github.com/forgant-foundry/forglet/internal/baseunit/npm"
	gitplugin "github.com/forgant-foundry/forglet/internal/plugins/git"
)

func main() {
	commands.RegisterPlugin(gitplugin.New())
	commands.RegisterValidator(npmpolicy.New())
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
