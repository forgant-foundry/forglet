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
//	    kind: binary                   # cross-compile + upload artifacts; also set main:
//	    main: ./cmd/myapp              # Go main package path for binary builds (default: ".")
//	    builds:                        # additional Go build variants (binary kind, Go only)
//	      - tags: no_embeddings        #   build tags for this variant (space-separated)
//	        suffix: slim               #   artifact suffix (e.g. myapp_${SEM}_linux_amd64_slim.tar.gz)
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
		payload, err := json.Marshal(deliveryPayload(group, meta.Template, meta.Name, delivery.kind, delivery.mainPkg, delivery.versionVar, delivery.builds, ci))
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

		if vp := vergantPayload(delivery); len(vp) > 0 {
			vpayload, err := json.Marshal(vp)
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
	}

	return nil
}

// deliveryCfg holds delivery workflow and vergant versioning configuration
// parsed from the github.delivery section of .forglet.yml.
type deliveryCfg struct {
	enabled            bool
	kind               string
	mainPkg            string          // Go main package path for binary builds; defaults to "."
	versionVar         string          // Full -X linker path for version injection (e.g. pkg/commands.Version)
	builds             []buildVariant  // additional Go build variants; binary kind only
	majorVersion       int
	defaultBranch      string
	supportBranchRegEx string
	devBranchRegEx     string
	patchBranchRegEx   string
	mode               string
}

// buildVariant describes one additional set of platform artifacts produced alongside
// the standard binary builds. Each variant recompiles with the given build tags and
// appends the suffix to every artifact filename (e.g. suffix "slim" →
// myapp_${SEM}_linux_amd64_slim.tar.gz). suffix is required; entries without one are skipped.
type buildVariant struct {
	tags   string // Go build tags (e.g. "no_embeddings"); multiple tags are space-separated
	suffix string // artifact filename suffix (required to avoid name collision with the standard build)
}

func defaultDeliveryCfg() deliveryCfg {
	return deliveryCfg{kind: "library"}
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
			if m, ok := d["main"].(string); ok && m != "" {
				delivery.mainPkg = m
			}
			if vv, ok := d["versionVar"].(string); ok && vv != "" {
				delivery.versionVar = vv
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
			if bs, ok := d["builds"].([]any); ok {
				for _, b := range bs {
					if bm, ok := b.(map[string]any); ok {
						v := buildVariant{}
						if s, ok := bm["tags"].(string); ok {
							v.tags = s
						}
						if s, ok := bm["suffix"].(string); ok {
							v.suffix = s
						}
						delivery.builds = append(delivery.builds, v)
					}
				}
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

// ciSteps returns the test job steps for the given template group. It is shared
// between the standalone ci.yml and the test job embedded in delivery.yml when
// both ci and delivery are enabled.
func ciSteps(group, template string) []any {
	switch group {
	case "go":
		return []any{
			map[string]any{"uses": "actions/checkout@v4"},
			map[string]any{
				"uses": "actions/setup-go@v5",
				"with": map[string]any{"go-version-file": "go.mod", "cache": true},
			},
			map[string]any{"run": "go test ./..."},
		}
	case "java":
		return []any{
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
		return []any{
			map[string]any{"uses": "actions/checkout@v4"},
			map[string]any{
				"uses": "actions/setup-node@v4",
				"with": map[string]any{"node-version": "22", "cache": "npm"},
			},
			map[string]any{"run": "npm ci"},
			map[string]any{"run": runStep},
		}
	}
	return nil
}

func ciPayload(group, template, branch string) map[string]any {
	base := map[string]any{
		"name": "ci",
		"on": map[string]any{
			"push":         map[string]any{"branches": []any{branch}},
			"pull_request": map[string]any{"branches": []any{branch}},
		},
	}
	base["jobs"] = map[string]any{
		"test": map[string]any{"runs-on": "ubuntu-latest", "steps": ciSteps(group, template)},
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

func deliveryPayload(group, template, name, kind, mainPkg, versionVar string, builds []buildVariant, withTest bool) map[string]any {
	const (
		configureGit   = "git config user.email \"github-actions[bot]@users.noreply.github.com\"\ngit config user.name \"github-actions[bot]\""
		installVergant = "go install github.com/forgant-foundry/vergant/cmd/vergant@latest"
		applyVersion   = "echo \"tag=$(vergant new)\" >> $GITHUB_OUTPUT"
		ifRelease      = "startsWith(steps.version.outputs.tag, 'v')"
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
		"run":  "VERSION=\"${{ steps.version.outputs.tag }}\"\ngh release create \"$VERSION\" --title \"${VERSION#v}\" --generate-notes",
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
					"run":  goDeliveryBuildScript(name, mainPkg, versionVar, builds),
				},
				map[string]any{
					"env":  map[string]any{"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}"},
					"if":   ifRelease,
					"name": "Create release",
					"run":  "VERSION=\"${{ steps.version.outputs.tag }}\"\ngh release create \"$VERSION\" --title \"${VERSION#v}\" --generate-notes dist/*",
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
					"run":  "VERSION=\"${{ steps.version.outputs.tag }}\"\nSEM=\"${VERSION#v}\"\nnpm version \"$SEM\" --no-git-tag-version\nnpm publish",
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
					"run":  "VERSION=\"${{ steps.version.outputs.tag }}\"\nSEM=\"${VERSION#v}\"\nmvn --batch-mode versions:set -DnewVersion=\"$SEM\" -DgenerateBackupPoms=false\nmvn --batch-mode package -DskipTests",
				},
				map[string]any{
					"env":  map[string]any{"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}"},
					"if":   ifRelease,
					"name": "Create release",
					"run":  "VERSION=\"${{ steps.version.outputs.tag }}\"\ngh release create \"$VERSION\" --title \"${VERSION#v}\" --generate-notes target/*.jar",
				},
			)
		}
	}

	deliveryJob := map[string]any{
		"runs-on":     "ubuntu-latest",
		"permissions": map[string]any{"contents": "write"},
		"steps":       steps,
	}
	jobs := map[string]any{"delivery": deliveryJob}
	if withTest {
		jobs["test"] = map[string]any{"runs-on": "ubuntu-latest", "steps": ciSteps(group, template)}
		deliveryJob["needs"] = []any{"test"}
	}
	return map[string]any{
		"name": "delivery",
		"on":   map[string]any{"push": map[string]any{"branches": []any{"**"}}},
		"jobs": jobs,
	}
}

// vergantPayload returns a map containing only the vergant fields explicitly
// configured in .forglet.yml. Fields at their zero value are omitted so the
// generated .vergant.yml stays minimal and vergant's own defaults apply for
// anything not present.
func vergantPayload(cfg deliveryCfg) map[string]any {
	m := map[string]any{}
	if cfg.majorVersion != 0 {
		m["majorVersion"] = cfg.majorVersion
	}
	if cfg.defaultBranch != "" {
		m["defaultBranch"] = cfg.defaultBranch
	}
	if cfg.supportBranchRegEx != "" {
		m["supportBranchRegEx"] = cfg.supportBranchRegEx
	}
	if cfg.devBranchRegEx != "" {
		m["devBranchRegEx"] = cfg.devBranchRegEx
	}
	if cfg.patchBranchRegEx != "" {
		m["patchBranchRegEx"] = cfg.patchBranchRegEx
	}
	if cfg.mode != "" {
		m["mode"] = cfg.mode
	}
	return m
}

// platformBuildLines generates the go build + archive commands for one binary variant
// across all four target platforms. binName is the temporary executable name (no .exe);
// artifactSuffix is appended before the archive extension (e.g. "_slim"); tagFlag is
// the -tags flag string with a leading space (e.g. " -tags no_embeddings"), or "".
func platformBuildLines(name, mainPkg, binName, artifactSuffix, tagFlag string) string {
	return fmt.Sprintf(`

GOOS=linux  GOARCH=amd64 go build%[5]s -ldflags="$LDFLAGS" -o /tmp/%[3]s %[2]s
tar czf "dist/%[1]s_${SEM}_linux_amd64%[4]s.tar.gz" -C /tmp %[3]s

GOOS=darwin GOARCH=amd64 go build%[5]s -ldflags="$LDFLAGS" -o /tmp/%[3]s %[2]s
tar czf "dist/%[1]s_${SEM}_darwin_amd64%[4]s.tar.gz" -C /tmp %[3]s

GOOS=darwin GOARCH=arm64 go build%[5]s -ldflags="$LDFLAGS" -o /tmp/%[3]s %[2]s
tar czf "dist/%[1]s_${SEM}_darwin_arm64%[4]s.tar.gz" -C /tmp %[3]s

GOOS=windows GOARCH=amd64 go build%[5]s -ldflags="$LDFLAGS" -o /tmp/%[3]s.exe %[2]s
cd /tmp && zip "${GITHUB_WORKSPACE}/dist/%[1]s_${SEM}_windows_amd64%[4]s.zip" %[3]s.exe && cd -`,
		name, mainPkg, binName, artifactSuffix, tagFlag)
}

func goDeliveryBuildScript(name, mainPkg, versionVar string, builds []buildVariant) string {
	if mainPkg == "" {
		mainPkg = "."
	}
	ldflags := "-s -w"
	if versionVar != "" {
		ldflags = fmt.Sprintf("-s -w -X %s=${SEM}", versionVar)
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, `VERSION="${{ steps.version.outputs.tag }}"
SEM="${VERSION#v}"
LDFLAGS="%s"
mkdir -p dist`, ldflags)
	sb.WriteString(platformBuildLines(name, mainPkg, name, "", ""))
	for _, v := range builds {
		if v.suffix == "" {
			continue
		}
		tagFlag := ""
		if v.tags != "" {
			tagFlag = " -tags " + v.tags
		}
		sb.WriteString(platformBuildLines(name, mainPkg, name+"_"+v.suffix, "_"+v.suffix, tagFlag))
	}
	fmt.Fprintf(&sb, "\n\ncd dist && sha256sum *.tar.gz *.zip > \"%s_${SEM}_checksums.txt\"", name)
	return sb.String()
}

func (p *GitHubActionsPlugin) RCSchema() project.SchemaContribution {
	deliveryShape := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"kind": map[string]any{
				"type":        "string",
				"enum":        []any{"library", "binary"},
				"description": "library: vergant versioning + plain GitHub release, no artifacts (default). binary: cross-compile + upload artifacts.",
				"default":     "library",
			},
			"main": map[string]any{
				"type":        "string",
				"description": "Go main package path for binary builds (e.g. ./cmd/myapp). Default: \".\".",
			},
			"builds": map[string]any{
				"type":        "array",
				"description": "Additional Go build variants (binary kind only). Each produces a parallel set of platform artifacts with a suffix-qualified filename.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"tags": map[string]any{
							"type":        "string",
							"description": "Go build tags for this variant (e.g. no_embeddings). Multiple tags are space-separated.",
						},
						"suffix": map[string]any{
							"type":        "string",
							"description": "Artifact filename suffix (e.g. slim → myapp_${SEM}_linux_amd64_slim.tar.gz). Required.",
						},
					},
					"required": []any{"suffix"},
				},
			},
			"versionVar": map[string]any{
				"type":        "string",
				"description": "Full linker path for version injection (e.g. github.com/org/repo/cmd/app/commands.Version). Omit to skip version embedding.",
			},
			"majorVersion": map[string]any{
				"type":        "integer",
				"description": "Current major version written to .vergant.yml.",
			},
			"defaultBranch": map[string]any{
				"type":        "string",
				"description": "Trunk branch for .vergant.yml versioning. Default: main.",
			},
			"supportBranchRegEx": map[string]any{
				"type":        "string",
				"description": "Support/maintenance branch pattern for .vergant.yml (e.g. ^release/.*).",
			},
			"devBranchRegEx": map[string]any{
				"type":        "string",
				"description": "Dev/pre-release branch pattern for .vergant.yml (e.g. ^feature/(.+)$).",
			},
			"patchBranchRegEx": map[string]any{
				"type":        "string",
				"description": "Patch/hotfix branch pattern for .vergant.yml (e.g. ^hotfix/(.+)$).",
			},
			"mode": map[string]any{
				"type":        "string",
				"enum":        []any{"release", "candidate"},
				"description": "vergant versioning mode. Default: release.",
			},
		},
	}
	return project.SchemaContribution{
		Properties: map[string]any{
			"github": map[string]any{
				"description": "GitHub Actions workflow generation.",
				"oneOf": []any{
					map[string]any{"type": "boolean", "description": "true enables CI workflow only."},
					map[string]any{
						"type": "object",
						"properties": map[string]any{
							"ci":            map[string]any{"type": "boolean", "description": "Generate .github/workflows/ci.yml."},
							"release":       map[string]any{"type": "boolean", "description": "Generate .github/workflows/release.yml (goreleaser, v*-tags)."},
							"defaultBranch": map[string]any{"type": "string", "description": "CI trigger branch. Default: main."},
							"delivery": map[string]any{
								"description": "Generate .github/workflows/delivery.yml + .vergant.yml.",
								"oneOf":       []any{map[string]any{"type": "boolean"}, deliveryShape},
							},
						},
					},
				},
			},
		},
	}
}

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
