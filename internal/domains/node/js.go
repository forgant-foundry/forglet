package node

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
)

// JavaScript synthesizes a Node.js project (no TypeScript).
// Managed files: package.json
// Scaffolded once: index.js
type JavaScript struct{}

func NewJavaScript() *JavaScript { return &JavaScript{} }

func (s *JavaScript) InitializeEvents(name string) (map[string][]eventing.Event, error) {
	pkg, err := makeEvents(1, []evtDef{
		{"init", map[string]any{
			"name":    name,
			"version": "0.1.0",
			"main":    "index.js",
			"type":    "commonjs",
		}},
		{"engines.set", map[string]any{
			"engines": map[string]any{
				"node": ">=22.0.0",
				"npm":  ">=11.0.0",
			},
		}},
		{"scripts.added", map[string]any{
			"scripts": map[string]any{
				"start":       "node index.js",
				"postinstall": "allow-scripts",
			},
		}},
		{"devDependency.added", map[string]any{
			"devDependencies": map[string]any{
				"@lavamoat/allow-scripts": "^3.3.1",
			},
		}},
	})
	if err != nil {
		return nil, err
	}
	return map[string][]eventing.Event{"package.json": pkg}, nil
}

func (s *JavaScript) OverlayEvents(rc map[string]any) (map[string][]eventing.Event, error) {
	if len(rc) == 0 {
		return nil, nil
	}

	var defs []evtDef
	if deps, ok := asStringMap(rc["dependencies"]); ok {
		defs = append(defs, evtDef{"dependency.added", map[string]any{"dependencies": deps}})
	}
	if devDeps, ok := asStringMap(rc["devDependencies"]); ok {
		defs = append(defs, evtDef{"devDependency.added", map[string]any{"devDependencies": devDeps}})
	}
	if scripts, ok := asStringMap(rc["scripts"]); ok {
		defs = append(defs, evtDef{"scripts.added", map[string]any{"scripts": scripts}})
	}

	if len(defs) == 0 {
		return nil, nil
	}

	pkg, err := makeEvents(overlaySeqBase, defs)
	if err != nil {
		return nil, err
	}
	return map[string][]eventing.Event{"package.json": pkg}, nil
}

func (s *JavaScript) Synthesize(dir string, aggregates map[string]*eventing.Aggregate) error {
	agg, ok := aggregates["package.json"]
	if !ok {
		return nil
	}
	b, err := prettyJSON(agg)
	if err != nil {
		return fmt.Errorf("serialize package.json: %w", err)
	}
	b = project.AddJSONMarker(b)
	if err := project.WriteManaged(filepath.Join(dir, "package.json"), b); err != nil {
		return err
	}

	// Write index.js only if it doesn't exist — scaffold, not managed.
	indexPath := filepath.Join(dir, "index.js")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		if err := os.WriteFile(indexPath, []byte("console.log('Hello Node');\n"), 0644); err != nil {
			return err
		}
	}

	return nil
}
