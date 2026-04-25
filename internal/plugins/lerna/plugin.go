package lerna

import (
	"crypto/rand"
	"encoding/json"
	"fmt"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
)

// LernaPlugin adds Lerna monorepo support to any node template.
// It contributes lerna.json (via the JSON cross-cutting file mechanism) and
// adds lerna as a devDependency in package.json.
//
// Register WorkspacesPlugin alongside this plugin — Lerna requires npm workspaces:
//
//	project.New(dir).WithPlugins(workspaces.New(), lerna.New())
type LernaPlugin struct{}

func New() *LernaPlugin { return &LernaPlugin{} }

func (p *LernaPlugin) Weave(meta project.Meta, rc map[string]any, stream *project.EventStream) error {
	depPayload, err := json.Marshal(map[string]any{
		"devDependencies": map[string]any{
			"lerna": "^9.0.0",
		},
	})
	if err != nil {
		return err
	}
	stream.Append("package.json", eventing.Event{
		ID:      newID(),
		Type:    "devDependency.added",
		Seq:     1,
		Payload: json.RawMessage(depPayload),
	})

	lernaPayload, err := json.Marshal(map[string]any{
		"$schema": "node_modules/lerna/schemas/lerna-schema.json",
		"version": "0.0.0",
	})
	if err != nil {
		return err
	}
	stream.SetFormat("lerna.json", project.FormatJSON)
	stream.Append("lerna.json", eventing.Event{
		ID:      newID(),
		Type:    "lerna.configured",
		Seq:     1,
		Payload: json.RawMessage(lernaPayload),
	})

	return nil
}

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
