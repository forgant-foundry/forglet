package workspaces_test

// Spec tests for WorkspacesPlugin.
// All tests here are intentionally failing until the plugin is implemented.
//
// Reference projects:
//   - .scratch/my-node-js-monorepo  (npm workspaces on plain node-js)
//   - .scratch/my-node-ts-monorepo  (npm workspaces on node-ts)

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/domains/node"
	"github.com/forgant-foundry/forglet/internal/plugins/workspaces"
	"github.com/forgant-foundry/forglet/internal/project"
	"github.com/forgant-foundry/forglet/internal/testutil"
)

func wsAgg(t *testing.T, events []eventing.Event) *eventing.Aggregate {
	t.Helper()
	agg := eventing.NewAggregate()
	for i := range events {
		if err := agg.Apply(&events[i]); err != nil {
			t.Fatal(err)
		}
	}
	return agg
}

// --- Weave unit tests ---

// TestWorkspacesPlugin_Weave_AddsPrivateTrue verifies the plugin marks the root
// package as private (required by npm workspaces).
func TestWorkspacesPlugin_Weave_AddsPrivateTrue(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := workspaces.New()
	if err := p.Weave(project.Meta{Name: "my-monorepo", Template: "node-js"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	agg := wsAgg(t, stream.Events()["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	if pkg["private"] != true {
		t.Errorf("private = %v, want true", pkg["private"])
	}
}

// TestWorkspacesPlugin_Weave_AddsWorkspacesGlob verifies the workspaces glob is set.
func TestWorkspacesPlugin_Weave_AddsWorkspacesGlob(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := workspaces.New()
	if err := p.Weave(project.Meta{Name: "my-monorepo", Template: "node-js"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	agg := wsAgg(t, stream.Events()["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	wss, _ := pkg["workspaces"].([]any)
	if len(wss) == 0 || wss[0] != "packages/*" {
		t.Errorf("workspaces = %v, want [\"packages/*\"]", wss)
	}
}

// TestWorkspacesPlugin_Weave_NodeTS_AddsWorkspacesGlob verifies the plugin is
// template-agnostic: it applies the same changes to node-ts as to node-js.
func TestWorkspacesPlugin_Weave_NodeTS_AddsWorkspacesGlob(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := workspaces.New()
	if err := p.Weave(project.Meta{Name: "my-monorepo", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	agg := wsAgg(t, stream.Events()["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	wss, _ := pkg["workspaces"].([]any)
	if len(wss) == 0 || wss[0] != "packages/*" {
		t.Errorf("workspaces = %v, want [\"packages/*\"]", wss)
	}
}

// --- Integration tests (full project.Init pipeline) ---

// TestWorkspacesPlugin_Integration_NodeTS_PackageJSONHasWorkspaces runs a full
// Init with the TypeScript synthesizer and checks the written package.json.
func TestWorkspacesPlugin_Integration_NodeTS_PackageJSONHasWorkspaces(t *testing.T) {
	dir := testutil.TempDir(t)
	p := project.New(dir).WithPlugins(workspaces.New())
	if err := p.Init(project.Meta{Name: "my-monorepo", Template: "node-ts"}, node.NewTypeScript()); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	wss, _ := pkg["workspaces"].([]any)
	if len(wss) == 0 || wss[0] != "packages/*" {
		t.Errorf("workspaces = %v, want [\"packages/*\"]", wss)
	}
}

// TestWorkspacesPlugin_Integration_NodeTS_IsIdempotent verifies re-synth produces
// the same package.json.
func TestWorkspacesPlugin_Integration_NodeTS_IsIdempotent(t *testing.T) {
	dir := testutil.TempDir(t)
	proj := project.New(dir).WithPlugins(workspaces.New())
	s := node.NewTypeScript()
	if err := proj.Init(project.Meta{Name: "my-monorepo", Template: "node-ts"}, s); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := proj.Synthesize(s); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	if string(first) != string(second) {
		t.Error("package.json differs between first and second synth")
	}
}

// TestWorkspacesPlugin_Integration_NodeJS_PackageJSONHasWorkspaces checks the
// plugin works with the (stub) node-js synthesizer once it is implemented.
func TestWorkspacesPlugin_Integration_NodeJS_PackageJSONHasWorkspaces(t *testing.T) {
	dir := testutil.TempDir(t)
	p := project.New(dir).WithPlugins(workspaces.New())
	if err := p.Init(project.Meta{Name: "my-monorepo", Template: "node-js"}, node.NewJavaScript()); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatalf("package.json not written: %v", err)
	}
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	wss, _ := pkg["workspaces"].([]any)
	if len(wss) == 0 || wss[0] != "packages/*" {
		t.Errorf("workspaces = %v, want [\"packages/*\"]", wss)
	}
}
