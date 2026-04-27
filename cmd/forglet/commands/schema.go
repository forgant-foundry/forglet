package commands

import (
	"encoding/json"
	"os"
	"sort"

	"github.com/forgant-foundry/forglet/internal/project"
)

// RunSchema prints a JSON Schema document for .forglet.yml to stdout.
// Top-level properties come from plugins implementing project.Schemer.
// Template-specific properties come from synthesizers implementing project.Schemer,
// wrapped in if/then blocks keyed on the template name.
func RunSchema() {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(buildSchema())
}

func buildSchema() map[string]any {
	tmplNames := sortedSynthKeys()
	enumVals := make([]any, len(tmplNames))
	for i, n := range tmplNames {
		enumVals[i] = n
	}

	props := map[string]any{
		"name": map[string]any{
			"type":        "string",
			"description": "Project name.",
		},
		"template": map[string]any{
			"type":        "string",
			"description": "Template name.",
			"enum":        enumVals,
		},
	}

	for _, p := range registeredPlugins {
		if s, ok := p.(project.Schemer); ok {
			for k, v := range s.RCSchema().Properties {
				props[k] = v
			}
		}
	}

	var allOf []any
	for _, tmpl := range tmplNames {
		s, ok := synthesizers[tmpl].(project.Schemer)
		if !ok {
			continue
		}
		contrib := s.RCSchema()
		if len(contrib.Properties) == 0 {
			continue
		}
		allOf = append(allOf, map[string]any{
			"if": map[string]any{
				"properties": map[string]any{"template": map[string]any{"const": tmpl}},
				"required":   []any{"template"},
			},
			"then": map[string]any{"properties": contrib.Properties},
		})
	}

	out := map[string]any{
		"$schema":     "http://json-schema.org/draft-07/schema#",
		"title":       "forglet project configuration",
		"description": "Configuration for .forglet.yml — edit and run 'forglet synth' to apply.",
		"type":        "object",
		"properties":  props,
	}
	if len(allOf) > 0 {
		out["allOf"] = allOf
	}
	return out
}

func sortedSynthKeys() []string {
	keys := make([]string, 0, len(synthesizers))
	for k := range synthesizers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
