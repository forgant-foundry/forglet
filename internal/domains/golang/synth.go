package golang

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
)

const (
	defaultGoVersion   = "1.25"
	overlaySeqBase     = int64(1000)
	lambdaDepModule    = "github.com/aws/aws-lambda-go"
	lambdaDepVersion   = "v1.54.0"
)

// Flat synthesizes a flat single-module Go project.
// Managed files: go.mod
type Flat struct{}

func NewFlat() *Flat { return &Flat{} }

func (s *Flat) InitializeEvents(name string) (map[string][]eventing.Event, error) {
	events, err := makeEvents(1, []evtDef{
		{"init", map[string]any{
			"module": name,
			"go":     defaultGoVersion,
		}},
	})
	if err != nil {
		return nil, err
	}
	return map[string][]eventing.Event{"go.mod": events}, nil
}

// OverlayEvents translates .forglet.yml config into overlay events for the go template.
// Supported keys:
//
//	module  string              — overrides the module path
//	go      string              — overrides the Go version
//	require map[string]string   — added to the require block
func (s *Flat) OverlayEvents(rc map[string]any) (map[string][]eventing.Event, error) {
	if len(rc) == 0 {
		return nil, nil
	}

	var defs []evtDef
	if modPath, ok := rc["module"].(string); ok && modPath != "" {
		defs = append(defs, evtDef{"module.set", map[string]any{"module": modPath}})
	}
	if goVersion, ok := rc["go"].(string); ok && goVersion != "" {
		defs = append(defs, evtDef{"go.set", map[string]any{"go": goVersion}})
	}
	if require, ok := asStringMap(rc["require"]); ok {
		defs = append(defs, evtDef{"require.added", map[string]any{"require": require}})
	}

	if len(defs) == 0 {
		return nil, nil
	}
	events, err := makeEvents(overlaySeqBase, defs)
	if err != nil {
		return nil, err
	}
	return map[string][]eventing.Event{"go.mod": events}, nil
}

func (s *Flat) Synthesize(dir string, aggregates map[string]*eventing.Aggregate) error {
	if agg, ok := aggregates["go.mod"]; ok {
		b, err := renderGoMod(agg)
		if err != nil {
			return fmt.Errorf("render go.mod: %w", err)
		}
		b = project.AddTextMarker(b, "//")
		if err := project.WriteManaged(filepath.Join(dir, "go.mod"), b); err != nil {
			return err
		}
	}
	return nil
}

func (s *Flat) Scaffold(dir string, _ project.Meta) error {
	return scaffoldFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"))
}

func (s *Flat) PostSynthesize(dir string) error { return goModTidy(dir) }

func (s *Flat) RCSchema() project.SchemaContribution {
	return project.SchemaContribution{Properties: goModSchemaProps()}
}

// GoKnative synthesizes a Knative func-style Go project.
// It manages only go.mod — func.yaml and Go handler scaffolds are contributed
// by the knative plugin (knative.New() + knative.NewGo()).
//
// Canonical registration in a custom binary:
//
//	project.New(dir).WithPlugins(knative.New(), knative.NewGo())
//	// synthesizer:
//	golang.NewGoKnative()
type GoKnative struct{}

func NewGoKnative() *GoKnative { return &GoKnative{} }

func (s *GoKnative) InitializeEvents(name string) (map[string][]eventing.Event, error) {
	return (&Flat{}).InitializeEvents(name)
}

func (s *GoKnative) OverlayEvents(rc map[string]any) (map[string][]eventing.Event, error) {
	return (&Flat{}).OverlayEvents(rc)
}

func (s *GoKnative) Synthesize(dir string, aggregates map[string]*eventing.Aggregate) error {
	return (&Flat{}).Synthesize(dir, aggregates)
}

func (s *GoKnative) PostSynthesize(dir string) error { return goModTidy(dir) }

func (s *GoKnative) RCSchema() project.SchemaContribution {
	return project.SchemaContribution{Properties: goModSchemaProps()}
}

// Workspace synthesizes a Go workspace (go.work + per-module scaffolds).
// Managed files: go.work
type Workspace struct{}

func NewWorkspace() *Workspace { return &Workspace{} }

func (s *Workspace) InitializeEvents(name string) (map[string][]eventing.Event, error) {
	events, err := makeEvents(1, []evtDef{
		{"init", map[string]any{
			"go":  defaultGoVersion,
			"use": map[string]any{"./" + name: ""},
		}},
	})
	if err != nil {
		return nil, err
	}
	return map[string][]eventing.Event{"go.work": events}, nil
}

// OverlayEvents translates .forglet.yml config into overlay events for the go-workspace template.
// Supported keys:
//
//	go   string   — overrides the Go version
//	use  []string — additional module paths to add to the use directive
func (s *Workspace) OverlayEvents(rc map[string]any) (map[string][]eventing.Event, error) {
	if len(rc) == 0 {
		return nil, nil
	}

	var defs []evtDef
	if goVersion, ok := rc["go"].(string); ok && goVersion != "" {
		defs = append(defs, evtDef{"go.set", map[string]any{"go": goVersion}})
	}
	if paths, ok := asStringSlice(rc["use"]); ok {
		pathMap := make(map[string]any, len(paths))
		for _, p := range paths {
			pathMap[p] = ""
		}
		defs = append(defs, evtDef{"use.added", map[string]any{"use": pathMap}})
	}

	if len(defs) == 0 {
		return nil, nil
	}
	events, err := makeEvents(overlaySeqBase, defs)
	if err != nil {
		return nil, err
	}
	return map[string][]eventing.Event{"go.work": events}, nil
}

func (s *Workspace) Synthesize(dir string, aggregates map[string]*eventing.Aggregate) error {
	agg, ok := aggregates["go.work"]
	if !ok {
		return nil
	}

	b, err := renderGoWork(agg)
	if err != nil {
		return fmt.Errorf("render go.work: %w", err)
	}
	b = project.AddTextMarker(b, "//")
	if err := project.WriteManaged(filepath.Join(dir, "go.work"), b); err != nil {
		return err
	}

	// Scaffold a go.mod and main.go in each use directory if not already present.
	useNode, ok := agg.Node("use")
	if !ok {
		return nil
	}
	useObj, ok := useNode.Value.(eventing.Object)
	if !ok {
		return nil
	}
	for _, n := range activeNodes(useObj) {
		modDir := filepath.Join(dir, filepath.FromSlash(n.Name))
		if err := scaffoldModule(modDir, filepath.Base(n.Name)); err != nil {
			return err
		}
	}

	return nil
}

func (s *Workspace) PostSynthesize(dir string) error { return goWorkSync(dir) }

func (s *Workspace) RCSchema() project.SchemaContribution {
	return project.SchemaContribution{
		Properties: map[string]any{
			"go": map[string]any{
				"type":        "string",
				"description": "Go version (e.g. \"1.23\").",
			},
			"use": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Additional workspace members to add to go.work beyond the default.",
			},
		},
	}
}

// Lambda synthesizes a Go AWS Lambda project.
// Managed files: go.mod
type Lambda struct{}

func NewLambda() *Lambda { return &Lambda{} }

func (s *Lambda) InitializeEvents(name string) (map[string][]eventing.Event, error) {
	events, err := makeEvents(1, []evtDef{
		{"init", map[string]any{
			"module": name,
			"go":     defaultGoVersion,
			"require": map[string]any{
				lambdaDepModule: lambdaDepVersion,
			},
		}},
	})
	if err != nil {
		return nil, err
	}
	return map[string][]eventing.Event{"go.mod": events}, nil
}

// OverlayEvents supports the same rc keys as Flat: module, go, require.
func (s *Lambda) OverlayEvents(rc map[string]any) (map[string][]eventing.Event, error) {
	return (&Flat{}).OverlayEvents(rc)
}

func (s *Lambda) Synthesize(dir string, aggregates map[string]*eventing.Aggregate) error {
	if agg, ok := aggregates["go.mod"]; ok {
		b, err := renderGoMod(agg)
		if err != nil {
			return fmt.Errorf("render go.mod: %w", err)
		}
		b = project.AddTextMarker(b, "//")
		if err := project.WriteManaged(filepath.Join(dir, "go.mod"), b); err != nil {
			return err
		}
	}

	mainPath := filepath.Join(dir, "main.go")
	if _, err := os.Stat(mainPath); os.IsNotExist(err) {
		if err := os.WriteFile(mainPath, lambdaMainGo(), 0644); err != nil {
			return err
		}
	}

	makePath := filepath.Join(dir, "Makefile")
	if _, err := os.Stat(makePath); os.IsNotExist(err) {
		if err := os.WriteFile(makePath, lambdaMakefile(), 0644); err != nil {
			return err
		}
	}

	return nil
}

func (s *Lambda) PostSynthesize(dir string) error { return goModTidy(dir) }

func (s *Lambda) RCSchema() project.SchemaContribution {
	return project.SchemaContribution{Properties: goModSchemaProps()}
}

func lambdaMainGo() []byte {
	// Cannot use raw string literal — struct tags contain backticks.
	return []byte("package main\n\nimport (\n\t\"context\"\n\n\t\"github.com/aws/aws-lambda-go/lambda\"\n)\n\ntype Request struct {\n\tName string `json:\"name\"`\n}\n\ntype Response struct {\n\tMessage string `json:\"message\"`\n}\n\nfunc HandleRequest(_ context.Context, req Request) (Response, error) {\n\treturn Response{Message: \"Hello, \" + req.Name}, nil\n}\n\nfunc main() {\n\tlambda.Start(HandleRequest)\n}\n")
}

func lambdaMakefile() []byte {
	return []byte(".PHONY: build\n\nbuild:\n\tGOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap main.go\n")
}


func goModSchemaProps() map[string]any {
	return map[string]any{
		"module": map[string]any{
			"type":        "string",
			"description": "Go module path (e.g. github.com/myorg/myapp).",
		},
		"go": map[string]any{
			"type":        "string",
			"description": "Go version (e.g. \"1.23\").",
		},
		"require": map[string]any{
			"type":                 "object",
			"description":         "Additional go.mod dependencies.",
			"additionalProperties": map[string]any{"type": "string"},
			"examples":            []any{map[string]any{"github.com/spf13/cobra": "v1.8.0"}},
		},
	}
}

// scaffoldFile writes content to path only if the file does not already exist.
func scaffoldFile(path string, content []byte) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return os.WriteFile(path, content, 0644)
}

// renderGoMod renders a go.mod file from an aggregate.
// Output order: module, go, require block.
// Require entries are emitted in the order the aggregate stores them (alphabetical).
func renderGoMod(agg *eventing.Aggregate) ([]byte, error) {
	var buf bytes.Buffer

	modNode, ok := agg.Node("module")
	if !ok {
		return nil, fmt.Errorf("missing module field")
	}
	fmt.Fprintf(&buf, "module %s\n", nodeString(modNode))

	goNode, ok := agg.Node("go")
	if !ok {
		return nil, fmt.Errorf("missing go field")
	}
	fmt.Fprintf(&buf, "\ngo %s\n", nodeString(goNode))

	reqNode, ok := agg.Node("require")
	if ok {
		if reqObj, ok := reqNode.Value.(eventing.Object); ok {
			if active := activeNodes(reqObj); len(active) > 0 {
				fmt.Fprintf(&buf, "\nrequire (\n")
				for _, n := range active {
					fmt.Fprintf(&buf, "\t%s %s\n", n.Name, nodeString(n))
				}
				fmt.Fprintf(&buf, ")\n")
			}
		}
	}

	return buf.Bytes(), nil
}

// renderGoWork renders a go.work file from an aggregate.
// Output order: go, use block.
// Use paths are emitted in the order the aggregate stores them (alphabetical).
func renderGoWork(agg *eventing.Aggregate) ([]byte, error) {
	var buf bytes.Buffer

	goNode, ok := agg.Node("go")
	if !ok {
		return nil, fmt.Errorf("missing go field")
	}
	fmt.Fprintf(&buf, "go %s\n", nodeString(goNode))

	useNode, ok := agg.Node("use")
	if ok {
		if useObj, ok := useNode.Value.(eventing.Object); ok {
			if active := activeNodes(useObj); len(active) > 0 {
				fmt.Fprintf(&buf, "\nuse (\n")
				for _, n := range active {
					fmt.Fprintf(&buf, "\t%s\n", n.Name)
				}
				fmt.Fprintf(&buf, ")\n")
			}
		}
	}

	return buf.Bytes(), nil
}

// scaffoldModule writes go.mod and main.go into dir only if they do not exist.
func scaffoldModule(dir, name string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	modPath := filepath.Join(dir, "go.mod")
	if _, err := os.Stat(modPath); os.IsNotExist(err) {
		content := fmt.Sprintf("module %s\n\ngo %s\n", name, defaultGoVersion)
		if err := os.WriteFile(modPath, []byte(content), 0644); err != nil {
			return err
		}
	}

	mainPath := filepath.Join(dir, "main.go")
	if _, err := os.Stat(mainPath); os.IsNotExist(err) {
		if err := os.WriteFile(mainPath, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
			return err
		}
	}

	return nil
}

func activeNodes(obj eventing.Object) []*eventing.Node {
	out := make([]*eventing.Node, 0, len(obj))
	for _, n := range obj {
		if n.Status == eventing.NodeActive {
			out = append(out, n)
		}
	}
	return out
}

func nodeString(n *eventing.Node) string {
	s, _ := n.Value.(string)
	return s
}

type evtDef struct {
	typ     string
	payload any
}

func makeEvents(base int64, defs []evtDef) ([]eventing.Event, error) {
	evts := make([]eventing.Event, len(defs))
	for i, d := range defs {
		b, err := json.Marshal(d.payload)
		if err != nil {
			return nil, err
		}
		evts[i] = eventing.Event{
			ID:      newID(),
			Type:    d.typ,
			Seq:     base + int64(i),
			Payload: json.RawMessage(b),
		}
	}
	return evts, nil
}

func asStringMap(v any) (map[string]string, bool) {
	raw, ok := v.(map[string]any)
	if !ok || len(raw) == 0 {
		return nil, false
	}
	out := make(map[string]string, len(raw))
	for k, val := range raw {
		s, ok := val.(string)
		if !ok {
			continue
		}
		out[k] = s
	}
	return out, len(out) > 0
}

func asStringSlice(v any) ([]string, bool) {
	raw, ok := v.([]any)
	if !ok || len(raw) == 0 {
		return nil, false
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out, len(out) > 0
}

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// goModTidy runs 'go mod tidy' in dir, resolving indirect dependencies and
// updating go.sum. go.mod is written read-only by WriteManaged, so it is made
// writable before the command runs and restored to read-only after.
func goModTidy(dir string) error {
	modPath := filepath.Join(dir, "go.mod")
	if err := os.Chmod(modPath, 0644); err != nil {
		return fmt.Errorf("go mod tidy: chmod go.mod: %w", err)
	}
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go mod tidy: %w\n%s", err, out)
	}
	return os.Chmod(modPath, 0444)
}

// goWorkSync runs 'go work sync' in dir, syncing the workspace build list back
// to each member module's go.mod. Used in place of goModTidy for workspace
// projects where there is no go.mod at the root.
func goWorkSync(dir string) error {
	workPath := filepath.Join(dir, "go.work")
	if err := os.Chmod(workPath, 0644); err != nil {
		return fmt.Errorf("go work sync: chmod go.work: %w", err)
	}
	cmd := exec.Command("go", "work", "sync")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go work sync: %w\n%s", err, out)
	}
	return os.Chmod(workPath, 0444)
}
