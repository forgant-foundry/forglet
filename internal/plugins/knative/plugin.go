package knative

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
)

// Plugin is the generic Knative base. It manages .knative/service.yaml and
// contributes .git to .dockerignore. It is language-agnostic: register it
// alongside a language-specific plugin (NewNode or NewGo).
//
//	project.New(dir).WithPlugins(knative.New(), knative.NewNode()) // Node
//	project.New(dir).WithPlugins(knative.New(), knative.NewGo())   // Go
type Plugin struct{}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) Weave(meta project.Meta, rc map[string]any, stream *project.EventStream) error {
	svcPayload, err := json.Marshal(map[string]any{
		"apiVersion": "serving.knative.dev/v1",
		"kind":       "Service",
		"metadata":   map[string]any{"name": meta.Name},
		"spec": map[string]any{
			"template": map[string]any{
				"spec": map[string]any{
					"containers": []any{
						map[string]any{"image": meta.Name + ":latest"},
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}
	stream.SetFormat(".knative/service.yaml", project.FormatYAML)
	stream.Append(".knative/service.yaml", eventing.Event{
		ID:      newID(),
		Type:    "knative.service.configured",
		Seq:     1,
		Payload: json.RawMessage(svcPayload),
	})

	gitPayload, err := json.Marshal(map[string]any{".git": true})
	if err != nil {
		return err
	}
	stream.SetFormat(".dockerignore", project.FormatPattern)
	stream.Append(".dockerignore", eventing.Event{
		ID:      newID(),
		Type:    "dockerignore.base",
		Seq:     1,
		Payload: json.RawMessage(gitPayload),
	})

	return nil
}

// scaffoldOnce writes content to path only if the file does not already exist.
func scaffoldOnce(path string, content []byte) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
