package commands

import "github.com/spf13/cobra"

var root = &cobra.Command{
	Use:   "forglet",
	Short: "Template-based project management with event sourcing",
}

func Execute() error {
	return root.Execute()
}

func init() {
	root.AddCommand(newCmd, synthCmd)
}
