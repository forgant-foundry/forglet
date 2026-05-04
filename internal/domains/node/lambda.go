package node

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
)

//go:embed scaffold/handler.ts
var lambdaHandlerTSScaffold []byte

// Lambda synthesizes a Node.js TypeScript Lambda monorepo.
// Managed files: package.json, tsconfig.json
// Scaffolded once: packages/handler/index.ts, packages/handler/package.json
//
// Register alongside WorkspacesPlugin to get workspaces: ["packages/*"]:
//
//	project.New(dir).WithPlugins(workspaces.New())
type Lambda struct{}

func NewLambda() *Lambda { return &Lambda{} }

func (s *Lambda) InitializeEvents(name string) (map[string][]eventing.Event, error) {
	pkg, err := makeEvents(1, []evtDef{
		{"init", map[string]any{
			"name":    name,
			"version": "0.1.0",
			"private": true,
		}},
		{"engines.set", map[string]any{
			"engines": map[string]any{
				"node": ">=22.0.0",
				"npm":  ">=11.0.0",
			},
		}},
		{"scripts.added", map[string]any{
			"scripts": map[string]any{
				"build":       "esbuild packages/handler/index.ts --bundle --platform=node --target=node22 --outfile=dist/handler.js",
				"postinstall": "allow-scripts",
			},
		}},
		{"devDependency.added", map[string]any{
			"devDependencies": map[string]any{
				"typescript":              "^6.0.0",
				"ts-node":                 "^10.9.2",
				"@types/node":             "^25.0.0",
				"esbuild":                 "^0.24.0",
				"@lavamoat/allow-scripts": "^3.3.1",
			},
		}},
	})
	if err != nil {
		return nil, err
	}

	// Reuse the TypeScript tsconfig — same compilation target.
	tsEvents, err := (&TypeScript{}).InitializeEvents(name)
	if err != nil {
		return nil, err
	}

	return map[string][]eventing.Event{
		"package.json":  pkg,
		"tsconfig.json": tsEvents["tsconfig.json"],
	}, nil
}

// OverlayEvents supports the same rc keys as TypeScript:
//
//	dependencies    map[string]string  — added to package.json dependencies
//	devDependencies map[string]string  — added to package.json devDependencies
//	scripts         map[string]string  — added to package.json scripts
//	allowScripts    map[string]bool    — lavamoat allowScripts config
func (s *Lambda) OverlayEvents(rc map[string]any) (map[string][]eventing.Event, error) {
	return (&TypeScript{}).OverlayEvents(rc)
}

func (s *Lambda) RCSchema() project.SchemaContribution {
	props := nodePackageSchemaProps()
	props["allowScripts"] = map[string]any{
		"type":                 "object",
		"description":         "@lavamoat/allow-scripts configuration. Keys are package names.",
		"additionalProperties": map[string]any{"type": "boolean"},
	}
	return project.SchemaContribution{Properties: props}
}

func (s *Lambda) Synthesize(dir string, aggregates map[string]*eventing.Aggregate) error {
	for _, filename := range []string{"package.json", "tsconfig.json"} {
		agg, ok := aggregates[filename]
		if !ok {
			continue
		}
		b, err := prettyJSON(agg)
		if err != nil {
			return fmt.Errorf("serialize %s: %w", filename, err)
		}
		b = project.AddJSONMarker(b)
		if err := project.WriteManaged(filepath.Join(dir, filename), b); err != nil {
			return err
		}
	}

	handlerDir := filepath.Join(dir, "packages", "handler")
	if err := os.MkdirAll(handlerDir, 0755); err != nil {
		return err
	}

	handlerIndex := filepath.Join(handlerDir, "index.ts")
	if _, err := os.Stat(handlerIndex); os.IsNotExist(err) {
		if err := os.WriteFile(handlerIndex, lambdaHandlerTSScaffold, 0644); err != nil {
			return err
		}
	}

	handlerPkg := filepath.Join(handlerDir, "package.json")
	if _, err := os.Stat(handlerPkg); os.IsNotExist(err) {
		b, err := json.MarshalIndent(map[string]any{"name": "handler", "version": "1.0.0"}, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(handlerPkg, append(b, '\n'), 0644); err != nil {
			return err
		}
	}

	return nil
}

