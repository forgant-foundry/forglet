package node

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
)

// overlaySeqBase ensures overlay events always have higher seq than template
// events and therefore win for the same key in the aggregate.
const overlaySeqBase = int64(1000)

// TypeScript synthesizes a Node.js TypeScript project.
// Managed files: package.json, tsconfig.json
type TypeScript struct{}

func NewTypeScript() *TypeScript { return &TypeScript{} }

func (s *TypeScript) InitializeEvents(name string) (map[string][]eventing.Event, error) {
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
				"build":       "tsc",
				"start":       "node dist/index.js",
				"postinstall": "allow-scripts",
			},
		}},
		{"devDependency.added", map[string]any{
			"devDependencies": map[string]any{
				"typescript":              "^6.0.0",
				"ts-node":                 "^10.9.2",
				"@types/node":             "^25.0.0",
				"@lavamoat/allow-scripts": "^3.3.1",
			},
		}},
	})
	if err != nil {
		return nil, err
	}

	tsconfig, err := makeEvents(1, []evtDef{
		{"init", map[string]any{
			"compilerOptions": map[string]any{
				"esModuleInterop":            true,
				"module":                     "commonjs",
				"outDir":                     "dist",
				"rootDir":                    "src",
				"strict":                     true,
				"target":                     "ES2020",
				"sourceMap":                  true,
				"declaration":                true,
				"declarationMap":             true,
				"noUncheckedIndexedAccess":   true,
				"exactOptionalPropertyTypes": true,
				"skipLibCheck":               true,
			},
		}},
	})
	if err != nil {
		return nil, err
	}

	return map[string][]eventing.Event{
		"package.json":  pkg,
		"tsconfig.json": tsconfig,
	}, nil
}

// OverlayEvents translates .forglet.yml config into overlay events for the
// node-ts template. Supported keys:
//
//	dependencies    map[string]string  — added to package.json dependencies
//	devDependencies map[string]string  — added to package.json devDependencies
//	scripts         map[string]string  — added to package.json scripts
func (s *TypeScript) OverlayEvents(rc map[string]any) (map[string][]eventing.Event, error) {
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
	if allowScripts, ok := asBoolMap(rc["allowScripts"]); ok {
		defs = append(defs, evtDef{"allowScripts.configured", map[string]any{
			"lavamoat": map[string]any{"allowScripts": allowScripts},
		}})
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

func (s *TypeScript) Synthesize(dir string, aggregates map[string]*eventing.Aggregate) error {
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

	// Write src/index.ts only if it doesn't exist — it's a scaffold, not managed.
	indexPath := filepath.Join(dir, "src", "index.ts")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Join(dir, "src"), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(indexPath, []byte("export {};\n"), 0644); err != nil {
			return err
		}
	}

	return nil
}

func (s *TypeScript) RCSchema() project.SchemaContribution {
	props := nodePackageSchemaProps()
	props["allowScripts"] = map[string]any{
		"type":                 "object",
		"description":         "@lavamoat/allow-scripts configuration. Keys are package names.",
		"additionalProperties": map[string]any{"type": "boolean"},
	}
	return project.SchemaContribution{Properties: props}
}

func nodePackageSchemaProps() map[string]any {
	return map[string]any{
		"dependencies": map[string]any{
			"type":                 "object",
			"description":         "Production npm dependencies.",
			"additionalProperties": map[string]any{"type": "string"},
		},
		"devDependencies": map[string]any{
			"type":                 "object",
			"description":         "Development npm dependencies.",
			"additionalProperties": map[string]any{"type": "string"},
		},
		"scripts": map[string]any{
			"type":                 "object",
			"description":         "npm scripts.",
			"additionalProperties": map[string]any{"type": "string"},
		},
	}
}

type evtDef struct {
	typ     string
	payload any
}

// makeEvents builds events with seq numbers starting at base (base, base+1, base+2, …).
func makeEvents(base int64, defs []evtDef) ([]eventing.Event, error) {
	evts := make([]eventing.Event, len(defs))
	for i, d := range defs {
		b, err := json.Marshal(d.payload)
		if err != nil {
			return nil, err
		}
		evts[i] = eventing.Event{
			ID:      newID(),
			Type:    d.typ,
			Seq:     base + int64(i),
			Payload: json.RawMessage(b),
		}
	}
	return evts, nil
}

func prettyJSON(agg *eventing.Aggregate) ([]byte, error) {
	b, err := agg.ToJSON()
	if err != nil {
		return nil, err
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return json.MarshalIndent(v, "", "  ")
}

// asStringMap coerces a yaml-parsed value into map[string]string.
func asStringMap(v any) (map[string]string, bool) {
	raw, ok := v.(map[string]any)
	if !ok || len(raw) == 0 {
		return nil, false
	}
	out := make(map[string]string, len(raw))
	for k, val := range raw {
		s, ok := val.(string)
		if !ok {
			continue
		}
		out[k] = s
	}
	return out, len(out) > 0
}

// asBoolMap coerces a yaml-parsed value into map[string]bool.
func asBoolMap(v any) (map[string]bool, bool) {
	raw, ok := v.(map[string]any)
	if !ok || len(raw) == 0 {
		return nil, false
	}
	out := make(map[string]bool, len(raw))
	for k, val := range raw {
		b, ok := val.(bool)
		if !ok {
			continue
		}
		out[k] = b
	}
	return out, len(out) > 0
}

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
