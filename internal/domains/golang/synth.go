package golang

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/forgant-foundry/eventing"
)

const (
	defaultGoVersion = "1.25"
	overlaySeqBase   = int64(1000)
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
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), b, 0644); err != nil {
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
	if err := os.WriteFile(filepath.Join(dir, "go.work"), b, 0644); err != nil {
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
