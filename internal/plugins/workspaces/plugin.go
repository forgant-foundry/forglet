package workspaces

import (
	"crypto/rand"
	"encoding/json"
	"fmt"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
)

// WorkspacesPlugin adds npm workspaces support to any node template.
// It contributes private: true and workspaces: ["packages/*"] to package.json.
// Compatible with node-js and node-ts templates.
type WorkspacesPlugin struct{}

func New() *WorkspacesPlugin { return &WorkspacesPlugin{} }

func (p *WorkspacesPlugin) Weave(meta project.Meta, rc map[string]any, stream *project.EventStream) error {
	payload, err := json.Marshal(map[string]any{
		"private":    true,
		"workspaces": []string{"packages/*"},
	})
	if err != nil {
		return err
	}
	stream.Append("package.json", eventing.Event{
		ID:      newID(),
		Type:    "workspaces.configured",
		Seq:     1,
		Payload: json.RawMessage(payload),
	})
	return nil
}

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
