package knative

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
)

// NodePlugin adds Node.js-specific Knative support on top of the generic Plugin:
// docker:build / docker:push / kn:deploy scripts in package.json, node_modules
// in .dockerignore, and a multi-stage Node Dockerfile scaffolded once during Init.
//
// Always register alongside New():
//
//	project.New(dir).WithPlugins(knative.New(), knative.NewNode())
type NodePlugin struct{}

func NewNode() *NodePlugin { return &NodePlugin{} }

func (p *NodePlugin) Weave(meta project.Meta, rc map[string]any, stream *project.EventStream) error {
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

	nodeModulesPayload, err := json.Marshal(map[string]any{"node_modules": true})
	if err != nil {
		return err
	}
	stream.Append(".dockerignore", eventing.Event{
		ID:      newID(),
		Type:    "dockerignore.node",
		Seq:     1,
		Payload: json.RawMessage(nodeModulesPayload),
	})

	return nil
}

func (p *NodePlugin) Scaffold(dir string, meta project.Meta) error {
	return scaffoldOnce(filepath.Join(dir, "Dockerfile"), nodeDockerfile())
}

func nodeDockerfile() []byte {
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
