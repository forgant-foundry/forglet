package commands

import (
	"fmt"

	"github.com/forgant-foundry/forglet/internal/project"
	"github.com/spf13/cobra"
)

var synthCmd = &cobra.Command{
	Use:   "synth",
	Short: "Regenerate project files from the event log",
	RunE: func(cmd *cobra.Command, args []string) error {
		p := project.New(".").WithPlugins(registeredPlugins...)
		meta, err := p.LoadMeta()
		if err != nil {
			return err
		}

		s, ok := synthesizers[meta.Template]
		if !ok {
			return fmt.Errorf("unknown template %q", meta.Template)
		}

		if err := p.Synthesize(s); err != nil {
			return err
		}

		fmt.Println("synthesized project files")
		return nil
	},
}
