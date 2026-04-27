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
	ciFile       = ".github/workflows/ci.yml"
	releaseFile  = ".github/workflows/release.yml"
	deliveryFile = ".github/workflows/delivery.yml"
	vergantFile  = ".vergant.yml"
)

// GitHubActionsPlugin synthesizes GitHub Actions workflow files and a managed
// .vergant.yml when delivery is enabled.
// Activated via .forglet.yml:
//
//	github: true                       # CI workflow only
//	github:
//	  ci: true                         # .github/workflows/ci.yml
//	  release: true                    # .github/workflows/release.yml (goreleaser/v*-tags)
//	  delivery: true                   # .github/workflows/delivery.yml + .vergant.yml
//	  delivery:
//	    kind: library                  # vergant versioning + plain GitHub release, no artifacts
//	    majorVersion: 2                # .vergant.yml: major version (default: 1)
//	    defaultBranch: develop         # .vergant.yml: trunk branch (default: main)
//	    supportBranchRegEx: "^release/.*"  # .vergant.yml: support branch pattern
//	    devBranchRegEx: "^feature/(.+)$"  # .vergant.yml: dev branch pattern
//	    patchBranchRegEx: "^hotfix/(.+)$" # .vergant.yml: patch branch pattern
//	    mode: candidate                # .vergant.yml: "release" or "candidate"
//	  defaultBranch: "develop"         # CI trigger branch (default: main)
//
// Workflow content is template-aware: Go templates cross-compile binaries,
// Java builds and uploads JARs, Node publishes to npm. Unknown templates are ignored.
type GitHubActionsPlugin struct{}

func New() *GitHubActionsPlugin { return &GitHubActionsPlugin{} }

func (p *GitHubActionsPlugin) Weave(meta project.Meta, rc map[string]any, stream *project.EventStream) error {
	ci, release, delivery, branch := parseRC(rc)
	if !ci && !release && !delivery.enabled {
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

	if delivery.enabled {
		payload, err := json.Marshal(deliveryPayload(group, meta.Name, delivery.kind))
		if err != nil {
			return fmt.Errorf("github plugin: delivery payload: %w", err)
		}
		stream.SetFormat(deliveryFile, project.FormatYAML)
		stream.Append(deliveryFile, eventing.Event{
			ID:      newID(),
			Type:    "github.delivery.configured",
			Seq:     1,
			Payload: json.RawMessage(payload),
		})

		vpayload, err := json.Marshal(vergantPayload(delivery))
		if err != nil {
			return fmt.Errorf("github plugin: vergant payload: %w", err)
		}
		stream.SetFormat(vergantFile, project.FormatYAML)
		stream.Append(vergantFile, eventing.Event{
			ID:      newID(),
			Type:    "vergant.configured",
			Seq:     1,
			Payload: json.RawMessage(vpayload),
		})
	}

	return nil
}

// deliveryCfg holds delivery workflow and vergant versioning configuration
// parsed from the github.delivery section of .forglet.yml.
type deliveryCfg struct {
	enabled            bool
	kind               string
	majorVersion       int
	defaultBranch      string
	supportBranchRegEx string
	devBranchRegEx     string
	patchBranchRegEx   string
	mode               string
}

func defaultDeliveryCfg() deliveryCfg {
	return deliveryCfg{
		kind:               "binary",
		majorVersion:       1,
		defaultBranch:      "main",
		supportBranchRegEx: `^support\/.*`,
		devBranchRegEx:     `^dev\/(.+)$`,
		patchBranchRegEx:   `^patch\/(.+)$`,
		mode:               "release",
	}
}

func parseRC(rc map[string]any) (ci, release bool, delivery deliveryCfg, branch string) {
	branch = "main"
	delivery = defaultDeliveryCfg()
	switch v := rc["github"].(type) {
	case bool:
		ci = v
	case map[string]any:
		ci, _ = v["ci"].(bool)
		release, _ = v["release"].(bool)
		switch d := v["delivery"].(type) {
		case bool:
			delivery.enabled = d
		case map[string]any:
			delivery.enabled = true
			if k, ok := d["kind"].(string); ok && k != "" {
				delivery.kind = k
			}
			if mv, ok := d["majorVersion"].(int); ok {
				delivery.majorVersion = mv
			}
			if db, ok := d["defaultBranch"].(string); ok && db != "" {
				delivery.defaultBranch = db
			}
			if s, ok := d["supportBranchRegEx"].(string); ok && s != "" {
				delivery.supportBranchRegEx = s
			}
			if s, ok := d["devBranchRegEx"].(string); ok && s != "" {
				delivery.devBranchRegEx = s
			}
			if s, ok := d["patchBranchRegEx"].(string); ok && s != "" {
				delivery.patchBranchRegEx = s
			}
			if s, ok := d["mode"].(string); ok && s != "" {
				delivery.mode = s
			}
		}
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

func deliveryPayload(group, name, kind string) map[string]any {
	const (
		configureGit   = "git config user.email \"github-actions[bot]@users.noreply.github.com\"\ngit config user.name \"github-actions[bot]\""
		installVergant = "go install github.com/forgant-foundry/vergant/cmd/vergant@latest"
		applyVersion   = "echo \"tag=$(vergant new)\" >> $GITHUB_OUTPUT"
		ifRelease      = "startsWith(steps.version.outputs.tag, 'r')"
	)

	versionStep := map[string]any{
		"id":   "version",
		"name": "Apply version",
		"run":  applyVersion,
	}
	libraryReleaseStep := map[string]any{
		"env":  map[string]any{"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}"},
		"if":   ifRelease,
		"name": "Create release",
		"run":  "VERSION=\"${{ steps.version.outputs.tag }}\"\ngh release create \"$VERSION\" --title \"${VERSION#r}\" --generate-notes",
	}

	var steps []any
	switch group {
	case "go":
		steps = []any{
			map[string]any{"uses": "actions/checkout@v4", "with": map[string]any{"fetch-depth": 0}},
			map[string]any{"uses": "actions/setup-go@v5", "with": map[string]any{"go-version-file": "go.mod"}},
			map[string]any{"name": "Download dependencies", "run": "go mod download"},
			map[string]any{"name": "Configure git identity", "run": configureGit},
			map[string]any{"name": "Install vergant", "run": installVergant},
			versionStep,
		}
		if kind == "library" {
			steps = append(steps, libraryReleaseStep)
		} else {
			steps = append(steps,
				map[string]any{
					"if":   ifRelease,
					"name": "Build release binaries",
					"run":  goDeliveryBuildScript(name),
				},
				map[string]any{
					"env":  map[string]any{"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}"},
					"if":   ifRelease,
					"name": "Create release",
					"run":  "VERSION=\"${{ steps.version.outputs.tag }}\"\ngh release create \"$VERSION\" --title \"${VERSION#r}\" --generate-notes dist/*",
				},
			)
		}
	case "node":
		steps = []any{
			map[string]any{"uses": "actions/checkout@v4", "with": map[string]any{"fetch-depth": 0}},
			map[string]any{
				"uses": "actions/setup-node@v4",
				"with": map[string]any{"node-version": "22", "cache": "npm", "registry-url": "https://registry.npmjs.org"},
			},
			map[string]any{"uses": "actions/setup-go@v5", "with": map[string]any{"go-version": "stable"}},
			map[string]any{"name": "Install dependencies", "run": "npm ci"},
			map[string]any{"name": "Configure git identity", "run": configureGit},
			map[string]any{"name": "Install vergant", "run": installVergant},
			versionStep,
		}
		if kind == "library" {
			steps = append(steps, libraryReleaseStep)
		} else {
			steps = append(steps,
				map[string]any{"if": ifRelease, "name": "Build", "run": "npm run build"},
				map[string]any{
					"env":  map[string]any{"NODE_AUTH_TOKEN": "${{ secrets.NPM_TOKEN }}"},
					"if":   ifRelease,
					"name": "Publish",
					"run":  "VERSION=\"${{ steps.version.outputs.tag }}\"\nSEM=\"${VERSION#r}\"\nnpm version \"$SEM\" --no-git-tag-version\nnpm publish",
				},
			)
		}
	case "java":
		steps = []any{
			map[string]any{"uses": "actions/checkout@v4", "with": map[string]any{"fetch-depth": 0}},
			map[string]any{
				"uses": "actions/setup-java@v4",
				"with": map[string]any{"java-version": "21", "distribution": "temurin", "cache": "maven"},
			},
			map[string]any{"uses": "actions/setup-go@v5", "with": map[string]any{"go-version": "stable"}},
			map[string]any{"name": "Configure git identity", "run": configureGit},
			map[string]any{"name": "Install vergant", "run": installVergant},
			versionStep,
		}
		if kind == "library" {
			steps = append(steps, libraryReleaseStep)
		} else {
			steps = append(steps,
				map[string]any{
					"if":   ifRelease,
					"name": "Build",
					"run":  "VERSION=\"${{ steps.version.outputs.tag }}\"\nSEM=\"${VERSION#r}\"\nmvn --batch-mode versions:set -DnewVersion=\"$SEM\" -DgenerateBackupPoms=false\nmvn --batch-mode package -DskipTests",
				},
				map[string]any{
					"env":  map[string]any{"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}"},
					"if":   ifRelease,
					"name": "Create release",
					"run":  "VERSION=\"${{ steps.version.outputs.tag }}\"\ngh release create \"$VERSION\" --title \"${VERSION#r}\" --generate-notes target/*.jar",
				},
			)
		}
	}

	return map[string]any{
		"name": "delivery",
		"on":   map[string]any{"push": map[string]any{"branches": []any{"**"}}},
		"jobs": map[string]any{
			"delivery": map[string]any{
				"runs-on":     "ubuntu-latest",
				"permissions": map[string]any{"contents": "write"},
				"steps":       steps,
			},
		},
	}
}

func vergantPayload(cfg deliveryCfg) map[string]any {
	return map[string]any{
		"majorVersion":       cfg.majorVersion,
		"defaultBranch":      cfg.defaultBranch,
		"supportBranchRegEx": cfg.supportBranchRegEx,
		"devBranchRegEx":     cfg.devBranchRegEx,
		"patchBranchRegEx":   cfg.patchBranchRegEx,
		"mode":               cfg.mode,
	}
}

func goDeliveryBuildScript(name string) string {
	return fmt.Sprintf(
		`VERSION="${{ steps.version.outputs.tag }}"
SEM="${VERSION#r}"
LDFLAGS="-s -w"
mkdir -p dist

GOOS=linux  GOARCH=amd64 go build -ldflags="$LDFLAGS" -o /tmp/%[1]s .
tar czf "dist/%[1]s_${SEM}_linux_amd64.tar.gz" -C /tmp %[1]s

GOOS=darwin GOARCH=amd64 go build -ldflags="$LDFLAGS" -o /tmp/%[1]s .
tar czf "dist/%[1]s_${SEM}_darwin_amd64.tar.gz" -C /tmp %[1]s

GOOS=darwin GOARCH=arm64 go build -ldflags="$LDFLAGS" -o /tmp/%[1]s .
tar czf "dist/%[1]s_${SEM}_darwin_arm64.tar.gz" -C /tmp %[1]s

GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -o /tmp/%[1]s.exe .
cd /tmp && zip "${GITHUB_WORKSPACE}/dist/%[1]s_${SEM}_windows_amd64.zip" %[1]s.exe && cd -

cd dist && sha256sum *.tar.gz *.zip > "%[1]s_${SEM}_checksums.txt"`, name)
}

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
