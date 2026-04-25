package node_test

// Spec tests for the node-js template (internal/domains/node.JavaScript).
// All tests here are intentionally failing until the synthesizer is implemented.
//
// Certified against: node v22.14.0 / npm 11.7.0
// Reference project: .scratch/my-node-js-app (npm init -y)

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/forgant-foundry/forglet/internal/domains/node"
	"github.com/forgant-foundry/forglet/internal/project"
	"github.com/forgant-foundry/forglet/internal/testutil"
)

const (
	jsSpecNodeVersion = ">=22.0.0" // node v22.14.0
	jsSpecNPMVersion  = ">=11.0.0" // npm 11.7.0
)

// --- InitializeEvents ---

func TestNodeJS_InitializeEvents_HasPackageJSON(t *testing.T) {
	s := node.NewJavaScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files["package.json"]; !ok {
		t.Error("no events for package.json")
	}
}

func TestNodeJS_InitializeEvents_DoesNotHaveTSConfig(t *testing.T) {
	s := node.NewJavaScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files["tsconfig.json"]; ok {
		t.Error("node-js must not produce tsconfig.json events")
	}
}

func TestNodeJS_InitializeEvents_NamePropagates(t *testing.T) {
	s := node.NewJavaScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	agg := buildAgg(t, files["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	if pkg["name"] != "my-app" {
		t.Errorf("name = %v, want %q", pkg["name"], "my-app")
	}
}

func TestNodeJS_InitializeEvents_PackageJSONHasMainIndexJS(t *testing.T) {
	s := node.NewJavaScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	agg := buildAgg(t, files["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	if pkg["main"] != "index.js" {
		t.Errorf("main = %v, want %q", pkg["main"], "index.js")
	}
}

func TestNodeJS_InitializeEvents_PackageJSONHasTypeCommonJS(t *testing.T) {
	s := node.NewJavaScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	agg := buildAgg(t, files["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	if pkg["type"] != "commonjs" {
		t.Errorf("type = %v, want %q", pkg["type"], "commonjs")
	}
}

func TestNodeJS_InitializeEvents_PackageJSONHasEnginesNode(t *testing.T) {
	s := node.NewJavaScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	agg := buildAgg(t, files["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	engines, _ := pkg["engines"].(map[string]any)
	if engines["node"] != jsSpecNodeVersion {
		t.Errorf("engines.node = %v, want %q", engines["node"], jsSpecNodeVersion)
	}
}

func TestNodeJS_InitializeEvents_PackageJSONHasEnginesNPM(t *testing.T) {
	s := node.NewJavaScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	agg := buildAgg(t, files["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	engines, _ := pkg["engines"].(map[string]any)
	if engines["npm"] != jsSpecNPMVersion {
		t.Errorf("engines.npm = %v, want %q", engines["npm"], jsSpecNPMVersion)
	}
}

// --- Synthesize ---

func TestNodeJS_Synthesize_WritesPackageJSON(t *testing.T) {
	dir := testutil.TempDir(t)
	s := node.NewJavaScript()
	files, _ := s.InitializeEvents("my-app")
	aggregates := project.BuildAggregates(files, nil)
	if err := s.Synthesize(dir, aggregates); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err != nil {
		t.Errorf("package.json not written: %v", err)
	}
}

func TestNodeJS_Synthesize_ScaffoldsIndexJS(t *testing.T) {
	dir := testutil.TempDir(t)
	s := node.NewJavaScript()
	files, _ := s.InitializeEvents("my-app")
	aggregates := project.BuildAggregates(files, nil)
	if err := s.Synthesize(dir, aggregates); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "index.js")); err != nil {
		t.Errorf("index.js not scaffolded at root: %v", err)
	}
}

func TestNodeJS_Synthesize_IndexJSNotOverwritten(t *testing.T) {
	dir := testutil.TempDir(t)
	s := node.NewJavaScript()
	files, _ := s.InitializeEvents("my-app")
	aggregates := project.BuildAggregates(files, nil)
	if err := s.Synthesize(dir, aggregates); err != nil {
		t.Fatal(err)
	}

	customContent := []byte("// custom\n")
	indexPath := filepath.Join(dir, "index.js")
	if err := os.WriteFile(indexPath, customContent, 0644); err != nil {
		t.Fatal(err)
	}

	if err := s.Synthesize(dir, aggregates); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(indexPath)
	if string(got) != string(customContent) {
		t.Error("index.js was overwritten on second synth; scaffold must be written once only")
	}
}

func TestNodeJS_Synthesize_DoesNotWriteTSConfig(t *testing.T) {
	dir := testutil.TempDir(t)
	s := node.NewJavaScript()
	files, _ := s.InitializeEvents("my-app")
	aggregates := project.BuildAggregates(files, nil)
	if err := s.Synthesize(dir, aggregates); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "tsconfig.json")); err == nil {
		t.Error("tsconfig.json must not be written by the node-js template")
	}
}

func TestNodeJS_Synthesize_IsIdempotent(t *testing.T) {
	dir := testutil.TempDir(t)
	s := node.NewJavaScript()
	files, _ := s.InitializeEvents("my-app")
	aggregates := project.BuildAggregates(files, nil)

	if err := s.Synthesize(dir, aggregates); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatalf("first synth: package.json not written: %v", err)
	}

	if err := s.Synthesize(dir, aggregates); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(filepath.Join(dir, "package.json"))

	if string(first) != string(second) {
		t.Error("package.json content differs between first and second synth")
	}
}
