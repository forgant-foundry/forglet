package github

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
)

const (
	ciFile      = ".github/workflows/ci.yml"
	releaseFile = ".github/workflows/release.yml"
)

// GitHubActionsPlugin synthesizes GitHub Actions workflow files.
// Activated via .forglet.yml:
//
//	github: true                       # CI workflow only
//	github:
//	  ci: true                         # CI workflow
//	  release: true                    # release workflow
//	  defaultBranch: "develop"         # branch CI triggers on (default: main)
//
// Workflow content is template-aware: Go templates use goreleaser, Java uses
// Maven, Node uses npm. Unknown templates are ignored.
type GitHubActionsPlugin struct{}

func New() *GitHubActionsPlugin { return &GitHubActionsPlugin{} }

func (p *GitHubActionsPlugin) Weave(meta project.Meta, rc map[string]any, stream *project.EventStream) error {
	ci, release, branch := parseRC(rc)
	if !ci && !release {
		return nil
	}

	group := templateGroup(meta.Template)
	if group == "" {
		return nil
	}

	if ci {
		payload, err := json.Marshal(ciPayload(group, meta.Template, branch))
		if err != nil {
			return fmt.Errorf("github plugin: ci payload: %w", err)
		}
		stream.SetFormat(ciFile, project.FormatYAML)
		stream.Append(ciFile, eventing.Event{
			ID:      newID(),
			Type:    "github.ci.configured",
			Seq:     1,
			Payload: json.RawMessage(payload),
		})
	}

	if release {
		payload, err := json.Marshal(releasePayload(group, meta.Template))
		if err != nil {
			return fmt.Errorf("github plugin: release payload: %w", err)
		}
		stream.SetFormat(releaseFile, project.FormatYAML)
		stream.Append(releaseFile, eventing.Event{
			ID:      newID(),
			Type:    "github.release.configured",
			Seq:     1,
			Payload: json.RawMessage(payload),
		})
	}

	return nil
}

func parseRC(rc map[string]any) (ci, release bool, branch string) {
	branch = "main"
	switch v := rc["github"].(type) {
	case bool:
		ci = v
	case map[string]any:
		ci, _ = v["ci"].(bool)
		release, _ = v["release"].(bool)
		if b, ok := v["defaultBranch"].(string); ok && b != "" {
			branch = b
		}
	}
	return
}

func templateGroup(template string) string {
	switch {
	case strings.HasPrefix(template, "go"):
		return "go"
	case strings.HasPrefix(template, "java"):
		return "java"
	case strings.HasPrefix(template, "node"):
		return "node"
	default:
		return ""
	}
}

func ciPayload(group, template, branch string) map[string]any {
	base := map[string]any{
		"name": "ci",
		"on": map[string]any{
			"push":         map[string]any{"branches": []any{branch}},
			"pull_request": map[string]any{"branches": []any{branch}},
		},
	}

	var steps []any
	switch group {
	case "go":
		steps = []any{
			map[string]any{"uses": "actions/checkout@v4"},
			map[string]any{
				"uses": "actions/setup-go@v5",
				"with": map[string]any{"go-version-file": "go.mod", "cache": true},
			},
			map[string]any{"run": "go test ./..."},
		}
	case "java":
		steps = []any{
			map[string]any{"uses": "actions/checkout@v4"},
			map[string]any{
				"uses": "actions/setup-java@v4",
				"with": map[string]any{"java-version": "21", "distribution": "temurin", "cache": "maven"},
			},
			map[string]any{"run": "mvn --batch-mode test"},
		}
	case "node":
		// node-lambda has no test script; run build instead
		runStep := "npm test"
		if template == "node-lambda" {
			runStep = "npm run build"
		}
		steps = []any{
			map[string]any{"uses": "actions/checkout@v4"},
			map[string]any{
				"uses": "actions/setup-node@v4",
				"with": map[string]any{"node-version": "22", "cache": "npm"},
			},
			map[string]any{"run": "npm ci"},
			map[string]any{"run": runStep},
		}
	}

	base["jobs"] = map[string]any{
		"test": map[string]any{"runs-on": "ubuntu-latest", "steps": steps},
	}
	return base
}

func releasePayload(group, template string) map[string]any {
	base := map[string]any{
		"name":        "release",
		"on":          map[string]any{"push": map[string]any{"tags": []any{"v*"}}},
		"permissions": map[string]any{"contents": "write"},
	}

	var steps []any
	switch group {
	case "go":
		steps = []any{
			map[string]any{
				"uses": "actions/checkout@v4",
				"with": map[string]any{"fetch-depth": 0},
			},
			map[string]any{
				"uses": "actions/setup-go@v5",
				"with": map[string]any{"go-version-file": "go.mod", "cache": true},
			},
			map[string]any{
				"uses": "goreleaser/goreleaser-action@v6",
				"with": map[string]any{"version": "~> v2", "args": "release --clean"},
				"env":  map[string]any{"GITHUB_TOKEN": "${{ secrets.GITHUB_TOKEN }}"},
			},
		}
	case "java":
		// java-lambda: fat JAR lives at target/*-shaded.jar; others at target/*.jar
		files := "target/*.jar"
		steps = []any{
			map[string]any{"uses": "actions/checkout@v4"},
			map[string]any{
				"uses": "actions/setup-java@v4",
				"with": map[string]any{"java-version": "21", "distribution": "temurin", "cache": "maven"},
			},
			map[string]any{"run": "mvn --batch-mode package -DskipTests"},
			map[string]any{
				"uses": "softprops/action-gh-release@v2",
				"with": map[string]any{"files": files},
				"env":  map[string]any{"GITHUB_TOKEN": "${{ secrets.GITHUB_TOKEN }}"},
			},
		}
	case "node":
		buildStep := "npm run build"
		steps = []any{
			map[string]any{"uses": "actions/checkout@v4"},
			map[string]any{
				"uses": "actions/setup-node@v4",
				"with": map[string]any{"node-version": "22", "cache": "npm"},
			},
			map[string]any{"run": "npm ci"},
			map[string]any{"run": buildStep},
			map[string]any{
				"uses": "softprops/action-gh-release@v2",
				"env":  map[string]any{"GITHUB_TOKEN": "${{ secrets.GITHUB_TOKEN }}"},
			},
		}
	}

	_ = template // reserved for future per-template release customisation
	base["jobs"] = map[string]any{
		"release": map[string]any{"runs-on": "ubuntu-latest", "steps": steps},
	}
	return base
}

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
