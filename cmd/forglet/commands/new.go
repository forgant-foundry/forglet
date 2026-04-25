package commands

import (
	"fmt"
	"os"
	"path/filepath"

	goproj "github.com/forgant-foundry/forglet/internal/domains/golang"
	java "github.com/forgant-foundry/forglet/internal/domains/java"
	"github.com/forgant-foundry/forglet/internal/domains/node"
	"github.com/forgant-foundry/forglet/internal/project"
	"github.com/spf13/cobra"
)

var synthesizers = map[string]project.Synthesizer{
	"go":                 goproj.NewFlat(),
	"go-workspace":       goproj.NewWorkspace(),
	"go-lambda":          goproj.NewLambda(),
	"go-knative":         goproj.NewKnativeFunc(),
	"java":               java.NewFlat(),
	"java-multimodule":   java.NewMultimodule(),
	"java-lambda":        java.NewLambda(),
	"java-spring":        java.NewSpring(),
	"node-js":            node.NewJavaScript(),
	"node-lambda":        node.NewLambda(),
	"node-ts":            node.NewTypeScript(),
}

var newCmd = &cobra.Command{
	Use:   "new <template> <name>",
	Short: "Create a new project from a template",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		template, name := args[0], args[1]

		s, ok := synthesizers[template]
		if !ok {
			return fmt.Errorf("unknown template %q — available: go, go-knative, go-lambda, go-workspace, java, java-lambda, java-multimodule, java-spring, node-js, node-lambda, node-ts", template)
		}

		dir := filepath.Join(".", name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		p := project.New(dir).WithPlugins(registeredPlugins...).WithValidators(registeredValidators...)
		if err := p.Init(project.Meta{Name: name, Template: template}, s); err != nil {
			return err
		}

		fmt.Printf("created %s project %q\n", template, name)
		return nil
	},
}
