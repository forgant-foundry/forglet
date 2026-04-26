package git

import (
	"crypto/rand"
	"encoding/json"
	"fmt"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
)

// patterns maps template names to the .gitignore patterns appropriate for that stack.
var patterns = map[string][]string{
	// Go
	"go":           {".DS_Store", "*.exe", "*.exe~", "*.out", "*.test", "coverage.out"},
	"go-workspace": {".DS_Store", "*.exe", "*.exe~", "*.out", "*.test", "coverage.out"},
	"go-lambda":    {".DS_Store", "*.exe", "*.exe~", "*.out", "*.test", "bootstrap", "coverage.out"},
	"go-knative":   {".DS_Store", "*.exe", "*.exe~", "*.out", "*.test", "coverage.out"},
	// Java
	"java":            {".DS_Store", ".idea/", "*.class", "*.iml", "target/"},
	"java-multimodule": {".DS_Store", ".idea/", "*.class", "*.iml", "target/"},
	"java-lambda":     {".DS_Store", ".idea/", "*.class", "*.iml", "target/"},
	"java-spring":     {".DS_Store", ".idea/", "*.class", "*.iml", "target/"},
	// Node
	"node-ts":     {".DS_Store", ".env", "dist/", "node_modules/"},
	"node-js":     {".DS_Store", ".env", "node_modules/"},
	"node-lambda": {".DS_Store", ".env", "dist/", "node_modules/"},
}

// Plugin adds a .gitignore file when git: true is set in .forglet.yml.
// Patterns are selected based on meta.Template. Additional patterns can be
// appended per-project via the gitignore key in .forglet.yml:
//
//	git: true
//	gitignore:
//	  - ".env.local"
//	  - "secrets.json"
type Plugin struct{}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) Weave(meta project.Meta, rc map[string]any, stream *project.EventStream) error {
	if enabled, _ := rc["git"].(bool); !enabled {
		return nil
	}

	ps, ok := patterns[meta.Template]
	if !ok {
		return nil
	}

	stream.SetFormat(".gitignore", project.FormatPattern)

	payload := make(map[string]any, len(ps))
	for _, pat := range ps {
		payload[pat] = ""
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("git plugin: marshal payload: %w", err)
	}
	stream.Append(".gitignore", eventing.Event{
		ID:      newID(),
		Type:    "git.ignore",
		Seq:     1,
		Payload: json.RawMessage(b),
	})

	if extra, ok := asStringSlice(rc["gitignore"]); ok {
		extraPayload := make(map[string]any, len(extra))
		for _, pat := range extra {
			extraPayload[pat] = ""
		}
		b, err := json.Marshal(extraPayload)
		if err != nil {
			return fmt.Errorf("git plugin: marshal custom payload: %w", err)
		}
		stream.Append(".gitignore", eventing.Event{
			ID:      newID(),
			Type:    "git.ignore.custom",
			Seq:     2,
			Payload: json.RawMessage(b),
		})
	}

	return nil
}

func asStringSlice(v any) ([]string, bool) {
	raw, ok := v.([]any)
	if !ok || len(raw) == 0 {
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

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
