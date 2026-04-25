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

// KnativePlugin adds Knative Serving support to any node template.
// It manages .knative/service.yaml (via the YAML cross-cutting file mechanism),
// contributes docker:build / docker:push / kn:deploy scripts to package.json,
// adds .dockerignore patterns, and scaffolds a Dockerfile once during Init.
type KnativePlugin struct{}

func New() *KnativePlugin { return &KnativePlugin{} }

func (p *KnativePlugin) Weave(meta project.Meta, rc map[string]any, stream *project.EventStream) error {
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

	scriptPayload, err := json.Marshal(map[string]any{
		"scripts": map[string]any{
			"docker:build": fmt.Sprintf("docker build -t %s:latest .", meta.Name),
			"docker:push":  fmt.Sprintf("docker push %s:latest", meta.Name),
			"kn:deploy":    fmt.Sprintf("kn service update %s --image %s:latest", meta.Name, meta.Name),
		},
	})
	if err != nil {
		return err
	}
	stream.Append("package.json", eventing.Event{
		ID:      newID(),
		Type:    "scripts.added",
		Seq:     1,
		Payload: json.RawMessage(scriptPayload),
	})

	dockerignorePayload, err := json.Marshal(map[string]any{
		".git":         true,
		"node_modules": true,
	})
	if err != nil {
		return err
	}
	stream.SetFormat(".dockerignore", project.FormatPattern)
	stream.Append(".dockerignore", eventing.Event{
		ID:      newID(),
		Type:    "dockerignore.configured",
		Seq:     1,
		Payload: json.RawMessage(dockerignorePayload),
	})

	return nil
}

func (p *KnativePlugin) Scaffold(dir string, meta project.Meta) error {
	return scaffoldOnce(filepath.Join(dir, "Dockerfile"), dockerfile())
}

func scaffoldOnce(path string, content []byte) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}

func dockerfile() []byte {
	return []byte(`FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:22-alpine
WORKDIR /app
COPY --from=builder /app/dist ./dist
COPY package*.json ./
RUN npm ci --omit=dev
EXPOSE 8080
CMD ["node", "dist/index.js"]
`)
}

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
