package git

import (
	"crypto/rand"
	"encoding/json"
	"fmt"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
)

// patterns maps template names to the .gitignore patterns appropriate for that stack.
// Add new template entries here as new domains are introduced.
var patterns = map[string][]string{
	"node-ts":      {".env", "dist/", "node_modules/"},
	"go":           {"*.exe", "*.out", "*.test"},
	"go-workspace": {"*.exe", "*.out", "*.test"},
}

// Plugin adds a .gitignore file when git: true is set in .forglet.yml.
// Patterns are selected based on meta.Template so this plugin works correctly
// across all supported templates without requiring synthesizer-level changes.
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
	return nil
}

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
