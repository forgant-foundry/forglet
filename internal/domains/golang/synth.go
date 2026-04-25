package golang

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
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

	mainPath := filepath.Join(dir, "main.go")
	if _, err := os.Stat(mainPath); os.IsNotExist(err) {
		if err := os.WriteFile(mainPath, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
			return err
		}
	}

	return nil
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

func lambdaMainGo() []byte {
	// Cannot use raw string literal — struct tags contain backticks.
	return []byte("package main\n\nimport (\n\t\"context\"\n\n\t\"github.com/aws/aws-lambda-go/lambda\"\n)\n\ntype Request struct {\n\tName string `json:\"name\"`\n}\n\ntype Response struct {\n\tMessage string `json:\"message\"`\n}\n\nfunc HandleRequest(_ context.Context, req Request) (Response, error) {\n\treturn Response{Message: \"Hello, \" + req.Name}, nil\n}\n\nfunc main() {\n\tlambda.Start(HandleRequest)\n}\n")
}

func lambdaMakefile() []byte {
	return []byte(".PHONY: build\n\nbuild:\n\tGOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap main.go\n")
}

// KnativeFunc synthesizes a Knative func-style Go project.
// Managed files: go.mod, func.yaml
type KnativeFunc struct{}

func NewKnativeFunc() *KnativeFunc { return &KnativeFunc{} }

const (
	knativeFuncSpecVersion = "0.35.0"
	knativeFuncRuntime     = "go"
)

func (s *KnativeFunc) InitializeEvents(name string) (map[string][]eventing.Event, error) {
	goModEvents, err := (&Flat{}).InitializeEvents(name)
	if err != nil {
		return nil, err
	}

	funcEvents, err := makeEvents(1, []evtDef{
		{"init", map[string]any{
			"specVersion": knativeFuncSpecVersion,
			"name":        name,
			"runtime":     knativeFuncRuntime,
			"registry":    "",
			"image":       "",
		}},
	})
	if err != nil {
		return nil, err
	}

	result := make(map[string][]eventing.Event, len(goModEvents)+1)
	for k, v := range goModEvents {
		result[k] = v
	}
	result["func.yaml"] = funcEvents
	return result, nil
}

// OverlayEvents supports go.mod keys (module, go, require) plus func.yaml keys:
//
//	name      string — overrides the function name in func.yaml
//	registry  string — sets the container registry
func (s *KnativeFunc) OverlayEvents(rc map[string]any) (map[string][]eventing.Event, error) {
	result := map[string][]eventing.Event{}

	goModOverlays, err := (&Flat{}).OverlayEvents(rc)
	if err != nil {
		return nil, err
	}
	for k, v := range goModOverlays {
		result[k] = v
	}

	var defs []evtDef
	if name, ok := rc["name"].(string); ok && name != "" {
		defs = append(defs, evtDef{"name.set", map[string]any{"name": name}})
	}
	if registry, ok := rc["registry"].(string); ok && registry != "" {
		defs = append(defs, evtDef{"registry.set", map[string]any{"registry": registry}})
	}
	if len(defs) > 0 {
		funcEvents, err := makeEvents(overlaySeqBase, defs)
		if err != nil {
			return nil, err
		}
		result["func.yaml"] = funcEvents
	}

	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func (s *KnativeFunc) Synthesize(dir string, aggregates map[string]*eventing.Aggregate) error {
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

	if agg, ok := aggregates["func.yaml"]; ok {
		b, err := agg.ToYAML()
		if err != nil {
			return fmt.Errorf("render func.yaml: %w", err)
		}
		b = project.AddTextMarker(b, "#")
		if err := project.WriteManaged(filepath.Join(dir, "func.yaml"), b); err != nil {
			return err
		}
	}

	for path, content := range map[string][]byte{
		"handle.go":      knativeFuncHandleGo(),
		"main.go":        knativeFuncMainGo(),
		"handle_test.go": knativeFuncHandleTestGo(),
	} {
		if err := scaffoldFile(filepath.Join(dir, path), content); err != nil {
			return err
		}
	}

	return nil
}

func knativeFuncHandleGo() []byte {
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

func knativeFuncMainGo() []byte {
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

func knativeFuncHandleTestGo() []byte {
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
