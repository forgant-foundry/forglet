package knative

import (
	"path/filepath"

	"github.com/forgant-foundry/forglet/internal/project"
)

// GoPlugin adds Go-specific Knative support on top of the generic Plugin:
// a multi-stage Go Dockerfile scaffolded once during Init.
//
// Always register alongside New():
//
//	project.New(dir).WithPlugins(knative.New(), knative.NewGo())
type GoPlugin struct{}

func NewGo() *GoPlugin { return &GoPlugin{} }

func (p *GoPlugin) Weave(meta project.Meta, rc map[string]any, stream *project.EventStream) error {
	return nil
}

func (p *GoPlugin) Scaffold(dir string, meta project.Meta) error {
	return scaffoldOnce(filepath.Join(dir, "Dockerfile"), goDockerfile())
}

func goDockerfile() []byte {
	return []byte(`FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
`)
}
