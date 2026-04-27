package git

import (
	"crypto/rand"
	"encoding/json"
	"fmt"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
)

// systemCategory groups related gitignore patterns under a named label.
// Each category is applied when git: true and can be suppressed via
// gitignore.exclude in .forglet.yml.
type systemCategory struct {
	name     string
	patterns []string
}

// systemCategories are applied for every matching template when git: true.
// They cover cross-cutting concerns (forglet metadata, OS artifacts, IDEs)
// that are not specific to any stack.
var systemCategories = []systemCategory{
	{name: "forglet",   patterns: []string{".forglet/"}},
	{name: "macos",     patterns: []string{".AppleDouble", ".DS_Store", ".LSOverride", "Icon", "Thumbs.db"}},
	{name: "jetbrains", patterns: []string{".idea/", "*.iml", "*.ipr", "*.iws"}},
	{name: "vscode",    patterns: []string{".vscode/"}},
	{name: "eclipse",   patterns: []string{".classpath", ".project", ".settings/"}},
}

// templatePatterns maps template names to their stack-specific gitignore patterns.
// OS and IDE patterns are not repeated here — they live in systemCategories above.
var templatePatterns = map[string][]string{
	// Go
	"go":           {"*.exe", "*.exe~", "*.out", "*.test", "coverage.out"},
	"go-workspace": {"*.exe", "*.exe~", "*.out", "*.test", "coverage.out"},
	"go-lambda":    {"*.exe", "*.exe~", "*.out", "*.test", "bootstrap", "coverage.out"},
	"go-knative":   {"*.exe", "*.exe~", "*.out", "*.test", "coverage.out"},
	// Java
	"java":             {"*.class", "target/"},
	"java-multimodule": {"*.class", "target/"},
	"java-lambda":      {"*.class", "target/"},
	"java-spring":      {"*.class", "target/"},
	// Node
	"node-ts":     {".env", "dist/", "node_modules/"},
	"node-js":     {".env", "node_modules/"},
	"node-lambda": {".env", "dist/", "node_modules/"},
}

// Plugin adds a .gitignore file when git: true is set in .forglet.yml.
// System categories (forglet, macos, jetbrains, vscode, eclipse) are always
// included and can be suppressed individually via gitignore.exclude. Stack
// patterns are selected by meta.Template. Additional patterns can be appended
// per-project via gitignore.add (or the shorthand flat-list form):
//
//	git: true
//	gitignore:
//	  add:
//	    - ".env.local"
//	  exclude:
//	    - "eclipse"
//	    - "vscode"
type Plugin struct{}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) Weave(meta project.Meta, rc map[string]any, stream *project.EventStream) error {
	if enabled, _ := rc["git"].(bool); !enabled {
		return nil
	}

	ps, ok := templatePatterns[meta.Template]
	if !ok {
		return nil
	}

	giRC := parseGitignoreRC(rc["gitignore"])
	excluded := make(map[string]bool, len(giRC.exclude))
	for _, name := range giRC.exclude {
		excluded[name] = true
	}

	stream.SetFormat(".gitignore", project.FormatPattern)

	for _, cat := range systemCategories {
		if excluded[cat.name] {
			continue
		}
		if err := appendPatterns(stream, "git.ignore."+cat.name, cat.patterns); err != nil {
			return err
		}
	}

	if err := appendPatterns(stream, "git.ignore.template", ps); err != nil {
		return err
	}

	if len(giRC.add) > 0 {
		if err := appendPatterns(stream, "git.ignore.custom", giRC.add); err != nil {
			return err
		}
	}

	return nil
}

type gitignoreRC struct {
	add     []string
	exclude []string
}

// parseGitignoreRC handles both the map form {add: [...], exclude: [...]}
// and the flat-list form [...] (backward compatible shorthand for add).
func parseGitignoreRC(v any) gitignoreRC {
	switch v := v.(type) {
	case []any:
		add, _ := anySliceToStrings(v)
		return gitignoreRC{add: add}
	case map[string]any:
		add, _ := asStringSlice(v["add"])
		exclude, _ := asStringSlice(v["exclude"])
		return gitignoreRC{add: add, exclude: exclude}
	}
	return gitignoreRC{}
}

func appendPatterns(stream *project.EventStream, eventType string, patterns []string) error {
	payload := make(map[string]any, len(patterns))
	for _, pat := range patterns {
		payload[pat] = ""
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("git plugin: marshal %s: %w", eventType, err)
	}
	stream.Append(".gitignore", eventing.Event{
		ID:      newID(),
		Type:    eventType,
		Seq:     1,
		Payload: json.RawMessage(b),
	})
	return nil
}

func asStringSlice(v any) ([]string, bool) {
	raw, ok := v.([]any)
	if !ok {
		return nil, false
	}
	return anySliceToStrings(raw)
}

func anySliceToStrings(raw []any) ([]string, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out, len(out) > 0
}

func (p *Plugin) RCSchema() project.SchemaContribution {
	return project.SchemaContribution{
		Properties: map[string]any{
			"git": map[string]any{
				"type":        "boolean",
				"description": "Generate a managed .gitignore with patterns appropriate for the active template.",
			},
			"gitignore": map[string]any{
				"description": "Additional .gitignore configuration. Requires git: true.",
				"oneOf": []any{
					map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Patterns to append (shorthand for gitignore.add).",
					},
					map[string]any{
						"type": "object",
						"properties": map[string]any{
							"add": map[string]any{
								"type":        "array",
								"items":       map[string]any{"type": "string"},
								"description": "Patterns to append after template defaults.",
							},
							"exclude": map[string]any{
								"type":        "array",
								"items":       map[string]any{"type": "string"},
								"description": "System category names to suppress. Built-in: forglet, macos, jetbrains, vscode, eclipse.",
							},
						},
					},
				},
			},
		},
	}
}

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
