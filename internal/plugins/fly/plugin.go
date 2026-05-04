package fly

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
	"gopkg.in/yaml.v3"
)

const (
	flyToml    = "fly.toml"
	deployFile = ".github/workflows/deploy.yml"
)

// Plugin synthesizes Fly.io deployment files. Register it in a custom forglet binary:
//
//	commands.RegisterPlugin(fly.New())
type Plugin struct{}

func New() *Plugin { return &Plugin{} }

type cfg struct {
	region      string
	port        int64
	memory      string
	cpus        int64
	healthPath  string
	main        string
	branch      string
}

func defaults() cfg {
	return cfg{
		region:     "iad",
		port:       8080,
		memory:     "256mb",
		cpus:       1,
		healthPath: "/healthz",
		main:       ".",
		branch:     "main",
	}
}

func parseRC(rc map[string]any) (cfg, bool) {
	c := defaults()
	switch v := rc["fly"].(type) {
	case bool:
		return c, v
	case map[string]any:
		if s, ok := v["region"].(string); ok && s != "" {
			c.region = s
		}
		if n, ok := v["port"].(int64); ok {
			c.port = n
		}
		if s, ok := v["memory"].(string); ok && s != "" {
			c.memory = s
		}
		if n, ok := v["cpus"].(int64); ok {
			c.cpus = n
		}
		if s, ok := v["healthPath"].(string); ok && s != "" {
			c.healthPath = s
		}
		if s, ok := v["main"].(string); ok && s != "" {
			c.main = s
		}
		if s, ok := v["branch"].(string); ok && s != "" {
			c.branch = s
		}
		return c, true
	default:
		return c, false
	}
}

func (p *Plugin) Weave(meta project.Meta, rc map[string]any, stream *project.EventStream) error {
	c, enabled := parseRC(rc)
	if !enabled {
		return nil
	}

	if err := p.weaveFlyToml(meta, c, stream); err != nil {
		return err
	}
	return p.weaveDeployWorkflow(c, stream)
}

func (p *Plugin) weaveFlyToml(meta project.Meta, c cfg, stream *project.EventStream) error {
	payload, err := json.Marshal(map[string]any{
		"app":            meta.Name,
		"primary_region": c.region,
		"http_service": map[string]any{
			"internal_port":      c.port,
			"force_https":        true,
			"auto_stop_machines": "stop",
			"auto_start_machines": true,
			"min_machines_running": int64(0),
		},
		"checks": map[string]any{
			"health": map[string]any{
				"port":         c.port,
				"type":         "http",
				"interval":     "30s",
				"timeout":      "2s",
				"grace_period": "5s",
				"method":       "get",
				"path":         c.healthPath,
			},
		},
		"vm": []any{
			map[string]any{
				"memory":    c.memory,
				"cpus":      c.cpus,
				"memory_mb": memoryMB(c.memory),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("fly plugin: fly.toml payload: %w", err)
	}
	stream.SetFormat(flyToml, project.FormatTOML)
	stream.Append(flyToml, eventing.Event{
		ID:      newID(),
		Type:    "fly.configured",
		Seq:     1,
		Payload: json.RawMessage(payload),
	})
	return nil
}

func (p *Plugin) weaveDeployWorkflow(c cfg, stream *project.EventStream) error {
	payload, err := json.Marshal(map[string]any{
		"name": "Deploy",
		"on": map[string]any{
			"push": map[string]any{
				"branches": []any{c.branch},
			},
		},
		"jobs": map[string]any{
			"deploy": map[string]any{
				"name":        "Deploy to Fly.io",
				"runs-on":     "ubuntu-latest",
				"concurrency": "deploy-group",
				"steps": []any{
					map[string]any{"uses": "actions/checkout@v4"},
					map[string]any{"uses": "superfly/flyctl-actions/setup-flyctl@master"},
					map[string]any{
						"run": "flyctl deploy --remote-only",
						"env": map[string]any{
							"FLY_API_TOKEN": "${{ secrets.FLY_API_TOKEN }}",
						},
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("fly plugin: deploy workflow payload: %w", err)
	}
	stream.SetFormat(deployFile, project.FormatYAML)
	stream.Append(deployFile, eventing.Event{
		ID:      newID(),
		Type:    "fly.deploy.configured",
		Seq:     1,
		Payload: json.RawMessage(payload),
	})
	return nil
}

func (p *Plugin) Scaffold(dir string, meta project.Meta) error {
	c := defaults()
	enabled := false
	if b, err := os.ReadFile(filepath.Join(dir, ".forglet.yml")); err == nil {
		var rc map[string]any
		if yaml.Unmarshal(b, &rc) == nil {
			c, enabled = parseRC(rc)
		}
	}
	if !enabled {
		return nil
	}
	return scaffoldOnce(filepath.Join(dir, "Dockerfile"), dockerfileForTemplate(meta.Template, c.main))
}

// memoryMB converts memory strings like "256mb" or "1gb" to an integer megabyte value.
// Returns 0 for unrecognised formats (fly.io will use its own default).
func memoryMB(memory string) int64 {
	var n int64
	var unit string
	fmt.Sscanf(memory, "%d%s", &n, &unit)
	switch unit {
	case "mb", "MB":
		return n
	case "gb", "GB":
		return n * 1024
	}
	return 0
}

func dockerfileForTemplate(template, mainPkg string) []byte {
	switch {
	case strings.HasPrefix(template, "java"):
		return dockerfileJava()
	case template == "node-js":
		return dockerfileNodeJS()
	case strings.HasPrefix(template, "node"):
		return dockerfileNodeTS()
	default:
		return dockerfileGo(mainPkg)
	}
}

func dockerfileGo(mainPkg string) []byte {
	return []byte(fmt.Sprintf(`FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/server %s

FROM alpine:3.21

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /bin/server /bin/server

EXPOSE 8080

CMD ["/bin/server"]
`, mainPkg))
}

func dockerfileJava() []byte {
	return []byte(`FROM maven:3.9-eclipse-temurin-21 AS builder

WORKDIR /app
COPY pom.xml .
RUN mvn dependency:go-offline -q
COPY src ./src
RUN mvn --batch-mode package -DskipTests

FROM eclipse-temurin:21-jre-alpine

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/target/*.jar /app/app.jar

EXPOSE 8080

CMD ["java", "-jar", "/app/app.jar"]
`)
}

func dockerfileNodeTS() []byte {
	return []byte(`FROM node:22-alpine AS builder

WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:22-alpine

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
COPY package*.json ./

EXPOSE 8080

CMD ["node", "dist/index.js"]
`)
}

func dockerfileNodeJS() []byte {
	return []byte(`FROM node:22-alpine

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY package*.json ./
RUN npm ci --omit=dev
COPY . .

EXPOSE 8080

CMD ["node", "src/index.js"]
`)
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

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
