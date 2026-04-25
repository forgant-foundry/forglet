package lerna_test

// Spec tests for LernaPlugin.
// All tests here are intentionally failing until the plugin is implemented.
//
// Note: implementing these tests also requires the project layer to support
// JSON cross-cutting files (lerna.json is not owned by any synthesizer).
// See internal/project/project.go renderCrossCuttingFiles.
//
// Certified against: lerna v9.0.7 / node v22.14.0 / npm 11.7.0
// Reference projects:
//   - .scratch/my-node-js-lerna-repo  (npx lerna init on node-js)
//   - .scratch/my-node-ts-lerna-repo  (npx lerna init on node-ts)

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/domains/node"
	"github.com/forgant-foundry/forglet/internal/plugins/lerna"
	"github.com/forgant-foundry/forglet/internal/plugins/workspaces"
	"github.com/forgant-foundry/forglet/internal/project"
	"github.com/forgant-foundry/forglet/internal/testutil"
)

// lernaSpecVersion is the lerna devDependency range written to package.json.
// Certified against lerna CLI v9.0.7.
const lernaSpecVersion = "^9.0.0"

func lernaAgg(t *testing.T, events []eventing.Event) *eventing.Aggregate {
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

// TestLernaPlugin_Weave_AddsLernaDevDep verifies the plugin contributes lerna
// to devDependencies in the package.json stream.
func TestLernaPlugin_Weave_AddsLernaDevDep(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := lerna.New()
	if err := p.Weave(project.Meta{Name: "my-repo", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	agg := lernaAgg(t, stream.Events()["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	devDeps, _ := pkg["devDependencies"].(map[string]any)
	if devDeps["lerna"] != lernaSpecVersion {
		t.Errorf("devDependencies.lerna = %v, want %q", devDeps["lerna"], lernaSpecVersion)
	}
}

// TestLernaPlugin_Weave_ContributesToLernaJSON verifies the plugin appends events
// for lerna.json so the project layer can render it as a JSON cross-cutting file.
func TestLernaPlugin_Weave_ContributesToLernaJSON(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := lerna.New()
	if err := p.Weave(project.Meta{Name: "my-repo", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	if len(stream.Events()["lerna.json"]) == 0 {
		t.Error("no events for lerna.json; plugin must contribute to the lerna.json stream")
	}
}

// TestLernaPlugin_Weave_LernaJSONHasVersion verifies the lerna.json events encode
// version: "0.0.0" (independent versioning is configured separately via rc).
func TestLernaPlugin_Weave_LernaJSONHasVersion(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := lerna.New()
	if err := p.Weave(project.Meta{Name: "my-repo", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	agg := lernaAgg(t, stream.Events()["lerna.json"])
	b, _ := agg.ToJSON()
	var lernaJSON map[string]any
	json.Unmarshal(b, &lernaJSON)
	if lernaJSON["version"] != "0.0.0" {
		t.Errorf("lerna.json version = %v, want %q", lernaJSON["version"], "0.0.0")
	}
}

// --- Integration tests (full project.Init pipeline) ---
//
// These also exercise the JSON cross-cutting file infrastructure in project.go.
// They will fail until both the plugin AND the rendering infrastructure are implemented.

func lernaWithWorkspacesProject(dir string) *project.Project {
	return project.New(dir).WithPlugins(workspaces.New(), lerna.New())
}

// TestLernaPlugin_Integration_NodeTS_CreatesLernaJSON verifies lerna.json is
// written to disk after Init with the TypeScript synthesizer.
func TestLernaPlugin_Integration_NodeTS_CreatesLernaJSON(t *testing.T) {
	dir := testutil.TempDir(t)
	p := lernaWithWorkspacesProject(dir)
	if err := p.Init(project.Meta{Name: "my-repo", Template: "node-ts"}, node.NewTypeScript()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "lerna.json")); err != nil {
		t.Errorf("lerna.json not written: %v", err)
	}
}

// TestLernaPlugin_Integration_NodeTS_LernaJSONHasVersion checks lerna.json content.
func TestLernaPlugin_Integration_NodeTS_LernaJSONHasVersion(t *testing.T) {
	dir := testutil.TempDir(t)
	p := lernaWithWorkspacesProject(dir)
	if err := p.Init(project.Meta{Name: "my-repo", Template: "node-ts"}, node.NewTypeScript()); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "lerna.json"))
	if err != nil {
		t.Fatalf("lerna.json not written: %v", err)
	}
	var lernaJSON map[string]any
	if err := json.Unmarshal(b, &lernaJSON); err != nil {
		t.Fatalf("lerna.json is not valid JSON: %v", err)
	}
	if lernaJSON["version"] != "0.0.0" {
		t.Errorf("lerna.json version = %v, want %q", lernaJSON["version"], "0.0.0")
	}
}

// TestLernaPlugin_Integration_NodeTS_LernaJSONHasSchema checks the $schema field
// that lerna uses to validate the file.
func TestLernaPlugin_Integration_NodeTS_LernaJSONHasSchema(t *testing.T) {
	dir := testutil.TempDir(t)
	p := lernaWithWorkspacesProject(dir)
	if err := p.Init(project.Meta{Name: "my-repo", Template: "node-ts"}, node.NewTypeScript()); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "lerna.json"))
	if err != nil {
		t.Fatalf("lerna.json not written: %v", err)
	}
	var lernaJSON map[string]any
	json.Unmarshal(b, &lernaJSON)
	if lernaJSON["$schema"] == nil || lernaJSON["$schema"] == "" {
		t.Errorf("lerna.json missing $schema field; got %v", lernaJSON["$schema"])
	}
}

// TestLernaPlugin_Integration_NodeTS_PackageJSONHasLernaDep verifies lerna is in
// devDependencies of the synthesized package.json.
func TestLernaPlugin_Integration_NodeTS_PackageJSONHasLernaDep(t *testing.T) {
	dir := testutil.TempDir(t)
	p := lernaWithWorkspacesProject(dir)
	if err := p.Init(project.Meta{Name: "my-repo", Template: "node-ts"}, node.NewTypeScript()); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	devDeps, _ := pkg["devDependencies"].(map[string]any)
	if devDeps["lerna"] != lernaSpecVersion {
		t.Errorf("devDependencies.lerna = %v, want %q", devDeps["lerna"], lernaSpecVersion)
	}
}

// TestLernaPlugin_Integration_NodeTS_IsIdempotent verifies re-synth leaves
// both package.json and lerna.json unchanged.
func TestLernaPlugin_Integration_NodeTS_IsIdempotent(t *testing.T) {
	dir := testutil.TempDir(t)
	proj := lernaWithWorkspacesProject(dir)
	s := node.NewTypeScript()
	if err := proj.Init(project.Meta{Name: "my-repo", Template: "node-ts"}, s); err != nil {
		t.Fatal(err)
	}
	firstPkg, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	firstLerna, err := os.ReadFile(filepath.Join(dir, "lerna.json"))
	if err != nil {
		t.Fatal(err)
	}

	if err := proj.Synthesize(s); err != nil {
		t.Fatal(err)
	}
	secondPkg, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	secondLerna, _ := os.ReadFile(filepath.Join(dir, "lerna.json"))

	if string(firstPkg) != string(secondPkg) {
		t.Error("package.json differs between first and second synth")
	}
	if string(firstLerna) != string(secondLerna) {
		t.Error("lerna.json differs between first and second synth")
	}
}

// TestLernaPlugin_Integration_NodeJS_CreatesLernaJSON verifies the plugin also
// works with the (stub) node-js synthesizer once it is implemented.
func TestLernaPlugin_Integration_NodeJS_CreatesLernaJSON(t *testing.T) {
	dir := testutil.TempDir(t)
	p := lernaWithWorkspacesProject(dir)
	if err := p.Init(project.Meta{Name: "my-repo", Template: "node-js"}, node.NewJavaScript()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "lerna.json")); err != nil {
		t.Errorf("lerna.json not written: %v", err)
	}
}
