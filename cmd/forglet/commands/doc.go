// Package commands provides the Cobra CLI commands for forglet and the
// registration API that custom forglet binaries use to wire in their plugins
// and validators before calling Execute.
//
// # Building a platform-specific binary
//
// Create a custom main.go in your own module that imports this package,
// registers your plugins and validators, then calls Execute:
//
//	package main
//
//	import (
//	    "fmt"
//	    "os"
//
//	    "github.com/forgant-foundry/forglet/cmd/forglet/commands"
//	    "github.com/mycompany/forglet-plugins/standards"
//	    "github.com/mycompany/forglet-validators/policy"
//	)
//
//	func main() {
//	    commands.RegisterPlugin(standards.New())
//	    commands.RegisterValidator(policy.New())
//	    if err := commands.Execute(); err != nil {
//	        fmt.Fprintln(os.Stderr, err)
//	        os.Exit(1)
//	    }
//	}
//
// Both [RegisterPlugin] and [RegisterValidator] are variadic — pass multiple
// values in one call or call them multiple times before Execute.
//
// # What the CLI exposes
//
// forglet new <template> <name>  — create a new project directory and run Init.
// forglet synth                  — re-synthesize managed files in the current directory.
// forglet schema                 — print JSON Schema for .forglet.yml (pipe to a file
//                                  for IDE YAML-language-server autocomplete).
// forglet help [topic]           — print implementation reference for plugin or domain authors.
//                                  Topics: plugin, domain.
//
// The schema and help commands are handled before the Cobra dispatcher so
// they work without Cobra and in custom binaries that replace Execute.
//
// Plugins registered via [RegisterPlugin] are passed to every [project.Project]
// created by both commands. Validators registered via [RegisterValidator]
// run at the end of every synthesis. Plugins and synthesizers that implement
// [project.Schemer] contribute their rc key definitions to [RunSchema].
package commands
