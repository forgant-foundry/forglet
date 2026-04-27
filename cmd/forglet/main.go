package main

import (
	"fmt"
	"os"

	"github.com/forgant-foundry/forglet/cmd/forglet/commands"
	npmpolicy "github.com/forgant-foundry/forglet/internal/baseunit/npm"
	gitplugin     "github.com/forgant-foundry/forglet/internal/plugins/git"
	githubplugin  "github.com/forgant-foundry/forglet/internal/plugins/github"
	licenseplugin "github.com/forgant-foundry/forglet/internal/plugins/license"
)

func main() {
	commands.RegisterPlugin(gitplugin.New())
	commands.RegisterPlugin(githubplugin.New())
	commands.RegisterPlugin(licenseplugin.New())
	commands.RegisterValidator(npmpolicy.New())
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
