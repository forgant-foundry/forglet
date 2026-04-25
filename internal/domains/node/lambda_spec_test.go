package node_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forgant-foundry/forglet/internal/domains/node"
	"github.com/forgant-foundry/forglet/internal/project"
	"github.com/forgant-foundry/forglet/internal/testutil"
)

// ---- InitializeEvents ---

func TestNodeLambda_InitializeEvents_TargetFiles(t *testing.T) {
	s := node.NewLambda()
	files, err := s.InitializeEvents("my-lambda")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"package.json", "tsconfig.json"} {
		if _, ok := files[want]; !ok {
			t.Errorf("missing events for %s", want)
		}
	}
}

func TestNodeLambda_InitializeEvents_NamePropagates(t *testing.T) {
	s := node.NewLambda()
	files, err := s.InitializeEvents("my-lambda")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range files["package.json"] {
		if e.Type != "init" {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(e.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload["name"] != "my-lambda" {
			t.Errorf("init payload name = %q, want %q", payload["name"], "my-lambda")
		}
	}
}

func TestNodeLambda_InitializeEvents_IsPrivate(t *testing.T) {
	s := node.NewLambda()
	files, err := s.InitializeEvents("my-lambda")
	if err != nil {
		t.Fatal(err)
	}
	agg := buildAgg(t, files["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	if pkg["private"] != true {
		t.Errorf("private = %v, want true", pkg["private"])
	}
}

func TestNodeLambda_InitializeEvents_HasEsbuild(t *testing.T) {
	s := node.NewLambda()
	files, err := s.InitializeEvents("my-lambda")
	if err != nil {
		t.Fatal(err)
	}
	agg := buildAgg(t, files["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	devDeps, _ := pkg["devDependencies"].(map[string]any)
	if devDeps["esbuild"] == nil {
		t.Error("esbuild missing from devDependencies")
	}
}

func TestNodeLambda_InitializeEvents_BuildScriptUsesEsbuild(t *testing.T) {
	s := node.NewLambda()
	files, err := s.InitializeEvents("my-lambda")
	if err != nil {
		t.Fatal(err)
	}
	agg := buildAgg(t, files["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	scripts, _ := pkg["scripts"].(map[string]any)
	build, _ := scripts["build"].(string)
	if !strings.Contains(build, "esbuild") {
		t.Errorf("build script %q does not reference esbuild", build)
	}
	if !strings.Contains(build, "packages/handler") {
		t.Errorf("build script %q does not reference packages/handler", build)
	}
}

func TestNodeLambda_InitializeEvents_NoStartScript(t *testing.T) {
	s := node.NewLambda()
	files, err := s.InitializeEvents("my-lambda")
	if err != nil {
		t.Fatal(err)
	}
	agg := buildAgg(t, files["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	scripts, _ := pkg["scripts"].(map[string]any)
	if _, ok := scripts["start"]; ok {
		t.Error("start script should not be present in node-lambda template")
	}
}

// ---- OverlayEvents ---

func TestNodeLambda_OverlayEvents_Empty(t *testing.T) {
	s := node.NewLambda()
	events, err := s.OverlayEvents(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Errorf("expected no overlay events for empty rc, got %d files", len(events))
	}
}

func TestNodeLambda_OverlayEvents_DevDependencies(t *testing.T) {
	s := node.NewLambda()
	rc := map[string]any{
		"devDependencies": map[string]any{"prettier": "^3.0.0"},
	}
	files, err := s.OverlayEvents(rc)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files["package.json"]; !ok {
		t.Fatal("expected overlay events for package.json")
	}
}

func TestNodeLambda_OverlayEvents_DevDepsAccumulate(t *testing.T) {
	s := node.NewLambda()
	rc := map[string]any{
		"devDependencies": map[string]any{"prettier": "^3.0.0"},
	}
	templateEvents, _ := s.InitializeEvents("my-lambda")
	overlayEvents, err := s.OverlayEvents(rc)
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(templateEvents, overlayEvents)
	b, _ := aggs["package.json"].ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	devDeps, _ := pkg["devDependencies"].(map[string]any)
	if devDeps["esbuild"] == nil {
		t.Error("esbuild missing after overlay — devDeps did not accumulate")
	}
	if devDeps["prettier"] == nil {
		t.Error("prettier missing after overlay")
	}
}

// ---- Synthesize ---

func TestNodeLambda_Synthesize_PackageJsonWritten(t *testing.T) {
	dir := testutil.TempDir(t)
	s := node.NewLambda()
	if err := project.New(dir).Init(project.Meta{Name: "my-lambda", Template: "node-lambda"}, s); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err != nil {
		t.Error("package.json not created")
	}
}

func TestNodeLambda_Synthesize_TsconfigWritten(t *testing.T) {
	dir := testutil.TempDir(t)
	s := node.NewLambda()
	if err := project.New(dir).Init(project.Meta{Name: "my-lambda", Template: "node-lambda"}, s); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "tsconfig.json")); err != nil {
		t.Error("tsconfig.json not created")
	}
}

func TestNodeLambda_Synthesize_HandlerScaffolded(t *testing.T) {
	dir := testutil.TempDir(t)
	s := node.NewLambda()
	if err := project.New(dir).Init(project.Meta{Name: "my-lambda", Template: "node-lambda"}, s); err != nil {
		t.Fatal(err)
	}
	handlerTs := filepath.Join(dir, "packages", "handler", "index.ts")
	if _, err := os.Stat(handlerTs); err != nil {
		t.Error("packages/handler/index.ts not created")
	}
	b, _ := os.ReadFile(handlerTs)
	if !strings.Contains(string(b), "handler") {
		t.Error("packages/handler/index.ts does not export a handler function")
	}
}

func TestNodeLambda_Synthesize_HandlerPackageJsonScaffolded(t *testing.T) {
	dir := testutil.TempDir(t)
	s := node.NewLambda()
	if err := project.New(dir).Init(project.Meta{Name: "my-lambda", Template: "node-lambda"}, s); err != nil {
		t.Fatal(err)
	}
	handlerPkg := filepath.Join(dir, "packages", "handler", "package.json")
	if _, err := os.Stat(handlerPkg); err != nil {
		t.Error("packages/handler/package.json not created")
	}
	b, _ := os.ReadFile(handlerPkg)
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	if pkg["name"] != "handler" {
		t.Errorf("handler package.json name = %q, want %q", pkg["name"], "handler")
	}
}

func TestNodeLambda_Synthesize_HandlerNotOverwritten(t *testing.T) {
	dir := testutil.TempDir(t)
	handlerDir := filepath.Join(dir, "packages", "handler")
	if err := os.MkdirAll(handlerDir, 0755); err != nil {
		t.Fatal(err)
	}
	existing := "export const handler = async () => ({ statusCode: 418 });\n"
	if err := os.WriteFile(filepath.Join(handlerDir, "index.ts"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	s := node.NewLambda()
	if err := project.New(dir).Init(project.Meta{Name: "my-lambda", Template: "node-lambda"}, s); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(handlerDir, "index.ts"))
	if string(b) != existing {
		t.Errorf("packages/handler/index.ts overwritten; got:\n%s", string(b))
	}
}

func TestNodeLambda_Synthesize_PackageJsonReadOnly(t *testing.T) {
	dir := testutil.TempDir(t)
	s := node.NewLambda()
	if err := project.New(dir).Init(project.Meta{Name: "my-lambda", Template: "node-lambda"}, s); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0444 {
		t.Errorf("package.json mode = %04o, want 0444", info.Mode().Perm())
	}
}

func TestNodeLambda_Synthesize_PackageJsonHasMarker(t *testing.T) {
	dir := testutil.TempDir(t)
	s := node.NewLambda()
	if err := project.New(dir).Init(project.Meta{Name: "my-lambda", Template: "node-lambda"}, s); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	if !strings.Contains(string(b), "forglet") {
		t.Error("package.json missing managed marker")
	}
}

func TestNodeLambda_Idempotent(t *testing.T) {
	dir := testutil.TempDir(t)
	s := node.NewLambda()
	p := project.New(dir)
	if err := p.Init(project.Meta{Name: "my-lambda", Template: "node-lambda"}, s); err != nil {
		t.Fatal(err)
	}
	firstPkg, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	firstTs, _ := os.ReadFile(filepath.Join(dir, "tsconfig.json"))

	if err := p.Synthesize(s); err != nil {
		t.Fatal(err)
	}
	secondPkg, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	secondTs, _ := os.ReadFile(filepath.Join(dir, "tsconfig.json"))

	if string(firstPkg) != string(secondPkg) {
		t.Error("package.json changed after second synth")
	}
	if string(firstTs) != string(secondTs) {
		t.Error("tsconfig.json changed after second synth")
	}
}
