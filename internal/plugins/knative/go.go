package knative

import (
	"encoding/json"
	"path/filepath"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
)

const (
	knativeFuncSpecVersion = "0.35.0"
	knativeFuncRuntime     = "go"
)

// GoPlugin adds Go-specific Knative support on top of the generic Plugin:
// func.yaml (the Knative func CLI manifest) and a multi-stage Go Dockerfile,
// plus handler scaffolds written once during Init.
//
// Always register alongside New():
//
//	project.New(dir).WithPlugins(knative.New(), knative.NewGo())
type GoPlugin struct{}

func NewGo() *GoPlugin { return &GoPlugin{} }

func (p *GoPlugin) Weave(meta project.Meta, rc map[string]any, stream *project.EventStream) error {
	basePayload, err := json.Marshal(map[string]any{
		"specVersion": knativeFuncSpecVersion,
		"name":        meta.Name,
		"runtime":     knativeFuncRuntime,
		"registry":    "",
		"image":       "",
	})
	if err != nil {
		return err
	}
	stream.SetFormat("func.yaml", project.FormatYAML)
	stream.Append("func.yaml", eventing.Event{
		ID:      newID(),
		Type:    "knative.func.configured",
		Seq:     1,
		Payload: json.RawMessage(basePayload),
	})

	override := map[string]any{}
	if name, ok := rc["name"].(string); ok && name != "" {
		override["name"] = name
	}
	if registry, ok := rc["registry"].(string); ok && registry != "" {
		override["registry"] = registry
	}
	if len(override) > 0 {
		overridePayload, err := json.Marshal(override)
		if err != nil {
			return err
		}
		stream.Append("func.yaml", eventing.Event{
			ID:      newID(),
			Type:    "knative.func.rc.override",
			Seq:     2,
			Payload: json.RawMessage(overridePayload),
		})
	}

	return nil
}

func (p *GoPlugin) Scaffold(dir string, _ project.Meta) error {
	if err := scaffoldOnce(filepath.Join(dir, "Dockerfile"), goDockerfile()); err != nil {
		return err
	}
	if err := scaffoldOnce(filepath.Join(dir, "handle.go"), goHandleGo()); err != nil {
		return err
	}
	if err := scaffoldOnce(filepath.Join(dir, "main.go"), goMainGo()); err != nil {
		return err
	}
	return scaffoldOnce(filepath.Join(dir, "handle_test.go"), goHandleTestGo())
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

func goHandleGo() []byte {
	return []byte(`package main

import (
	"fmt"
	"net/http"
)

// Handle processes an incoming HTTP request.
func Handle(res http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(res, "Hello, World!")
}
`)
}

func goMainGo() []byte {
	return []byte(`package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/", Handle)
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
`)
}

func goHandleTestGo() []byte {
	return []byte(`package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandle(t *testing.T) {
	rec := httptest.NewRecorder()
	Handle(rec, httptest.NewRequest("GET", "/", nil))
	if !strings.Contains(rec.Body.String(), "Hello") {
		t.Errorf("unexpected body: %q", rec.Body.String())
	}
}
`)
}
