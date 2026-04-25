package commands

import "github.com/spf13/cobra"

// Version is set at build time via -ldflags. Defaults to "dev" for local builds.
var Version = "dev"

var root = &cobra.Command{
	Use:     "forglet",
	Short:   "Template-based project management with event sourcing",
	Version: Version,
}

func Execute() error {
	return root.Execute()
}

func init() {
	root.AddCommand(newCmd, synthCmd)
}
