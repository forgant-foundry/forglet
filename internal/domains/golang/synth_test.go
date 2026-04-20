package golang_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	goproj "github.com/forgant-foundry/forglet/internal/domains/golang"
	"github.com/forgant-foundry/forglet/internal/project"
	"github.com/forgant-foundry/forglet/internal/testutil"
)

// ---- Flat (go.mod) ----------------------------------------------------------------

func TestGoFlat_InitializeEvents_ModuleFromName(t *testing.T) {
	s := goproj.NewFlat()
	events, err := s.InitializeEvents("myapp")
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(events)
	modNode, ok := aggs["go.mod"].Node("module")
	if !ok {
		t.Fatal("no module node")
	}
	if got := modNode.Value.(string); got != "myapp" {
		t.Errorf("module = %q, want %q", got, "myapp")
	}
}

func TestGoFlat_InitializeEvents_GoVersionSet(t *testing.T) {
	s := goproj.NewFlat()
	events, err := s.InitializeEvents("myapp")
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(events)
	goNode, ok := aggs["go.mod"].Node("go")
	if !ok {
		t.Fatal("no go node")
	}
	if goNode.Value.(string) == "" {
		t.Error("go version is empty")
	}
}

func TestGoFlat_OverlayEvents_Empty(t *testing.T) {
	s := goproj.NewFlat()
	events, err := s.OverlayEvents(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if events != nil {
		t.Error("expected nil for empty rc")
	}
}

func TestGoFlat_OverlayEvents_ModuleOverride(t *testing.T) {
	s := goproj.NewFlat()
	rc := map[string]any{"module": "github.com/myorg/myapp"}
	templateEvents, _ := s.InitializeEvents("myapp")
	overlayEvents, err := s.OverlayEvents(rc)
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(templateEvents, overlayEvents)
	modNode, _ := aggs["go.mod"].Node("module")
	if got := modNode.Value.(string); got != "github.com/myorg/myapp" {
		t.Errorf("module = %q, want %q", got, "github.com/myorg/myapp")
	}
}

func TestGoFlat_OverlayEvents_GoVersionOverride(t *testing.T) {
	s := goproj.NewFlat()
	rc := map[string]any{"go": "1.22"}
	templateEvents, _ := s.InitializeEvents("myapp")
	overlayEvents, err := s.OverlayEvents(rc)
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(templateEvents, overlayEvents)
	goNode, _ := aggs["go.mod"].Node("go")
	if got := goNode.Value.(string); got != "1.22" {
		t.Errorf("go = %q, want %q", got, "1.22")
	}
}

func TestGoFlat_OverlayEvents_RequireInGoMod(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewFlat()
	rc := map[string]any{
		"require": map[string]any{"github.com/spf13/cobra": "v1.8.0"},
	}
	templateEvents, _ := s.InitializeEvents("myapp")
	overlayEvents, err := s.OverlayEvents(rc)
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(templateEvents, overlayEvents)
	if err := s.Synthesize(dir, aggs); err != nil {
		t.Fatal(err)
	}
	content := readFile(t, filepath.Join(dir, "go.mod"))
	assertContains(t, content, "github.com/spf13/cobra v1.8.0")
}

func TestGoFlat_Synthesize_GoModContent(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewFlat()
	if err := project.New(dir).Init(project.Meta{Name: "myapp", Template: "go"}, s); err != nil {
		t.Fatal(err)
	}
	content := readFile(t, filepath.Join(dir, "go.mod"))
	assertContains(t, content, "module myapp")
	assertContains(t, content, "go ")
}

func TestGoFlat_Synthesize_MainGoScaffolded(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewFlat()
	if err := project.New(dir).Init(project.Meta{Name: "myapp", Template: "go"}, s); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "main.go")); err != nil {
		t.Error("main.go not created")
	}
}

func TestGoFlat_Synthesize_MainGoNotOverwritten(t *testing.T) {
	dir := testutil.TempDir(t)
	existing := "package main\n\nfunc main() { panic(\"custom\") }\n"
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}
	s := goproj.NewFlat()
	if err := project.New(dir).Init(project.Meta{Name: "myapp", Template: "go"}, s); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, "main.go")); got != existing {
		t.Errorf("main.go overwritten; got:\n%s", got)
	}
}

func TestGoFlat_Idempotent(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewFlat()
	p := project.New(dir)
	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, s); err != nil {
		t.Fatal(err)
	}
	first := readFile(t, filepath.Join(dir, "go.mod"))

	if err := p.Synthesize(s); err != nil {
		t.Fatal(err)
	}
	second := readFile(t, filepath.Join(dir, "go.mod"))

	if first != second {
		t.Errorf("go.mod changed after second synth:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestGoFlat_RequireAccumulates(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewFlat()
	rc := map[string]any{
		"require": map[string]any{
			"github.com/spf13/cobra":  "v1.8.0",
			"github.com/some/library": "v0.3.0",
		},
	}
	overlay, err := s.OverlayEvents(rc)
	if err != nil {
		t.Fatal(err)
	}
	template, _ := s.InitializeEvents("myapp")
	aggs := project.BuildAggregates(template, overlay)
	if err := s.Synthesize(dir, aggs); err != nil {
		t.Fatal(err)
	}
	content := readFile(t, filepath.Join(dir, "go.mod"))
	assertContains(t, content, "github.com/spf13/cobra v1.8.0")
	assertContains(t, content, "github.com/some/library v0.3.0")
}

// ---- Workspace (go.work) ----------------------------------------------------------

func TestGoWorkspace_InitializeEvents_GoVersionSet(t *testing.T) {
	s := goproj.NewWorkspace()
	events, err := s.InitializeEvents("myws")
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(events)
	goNode, ok := aggs["go.work"].Node("go")
	if !ok {
		t.Fatal("no go node")
	}
	if goNode.Value.(string) == "" {
		t.Error("go version is empty")
	}
}

func TestGoWorkspace_InitializeEvents_UseContainsName(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewWorkspace()
	events, err := s.InitializeEvents("myws")
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(events)
	if err := s.Synthesize(dir, aggs); err != nil {
		t.Fatal(err)
	}
	content := readFile(t, filepath.Join(dir, "go.work"))
	assertContains(t, content, "./myws")
}

func TestGoWorkspace_OverlayEvents_Empty(t *testing.T) {
	s := goproj.NewWorkspace()
	events, err := s.OverlayEvents(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if events != nil {
		t.Error("expected nil for empty rc")
	}
}

func TestGoWorkspace_OverlayEvents_GoVersionOverride(t *testing.T) {
	s := goproj.NewWorkspace()
	rc := map[string]any{"go": "1.22"}
	templateEvents, _ := s.InitializeEvents("myws")
	overlayEvents, err := s.OverlayEvents(rc)
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(templateEvents, overlayEvents)
	goNode, _ := aggs["go.work"].Node("go")
	if got := goNode.Value.(string); got != "1.22" {
		t.Errorf("go = %q, want %q", got, "1.22")
	}
}

func TestGoWorkspace_Synthesize_GoWorkContent(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewWorkspace()
	if err := project.New(dir).Init(project.Meta{Name: "myws", Template: "go-workspace"}, s); err != nil {
		t.Fatal(err)
	}
	content := readFile(t, filepath.Join(dir, "go.work"))
	assertContains(t, content, "go ")
	assertContains(t, content, "use (")
	assertContains(t, content, "./myws")
}

func TestGoWorkspace_Synthesize_ModuleScaffolded(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewWorkspace()
	if err := project.New(dir).Init(project.Meta{Name: "myws", Template: "go-workspace"}, s); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "myws", "go.mod")); err != nil {
		t.Error("myws/go.mod not scaffolded")
	}
	if _, err := os.Stat(filepath.Join(dir, "myws", "main.go")); err != nil {
		t.Error("myws/main.go not scaffolded")
	}
}

func TestGoWorkspace_Synthesize_ModuleNotOverwritten(t *testing.T) {
	dir := testutil.TempDir(t)
	modDir := filepath.Join(dir, "myws")
	if err := os.MkdirAll(modDir, 0755); err != nil {
		t.Fatal(err)
	}
	existing := "module custom\n\ngo 1.20\n"
	if err := os.WriteFile(filepath.Join(modDir, "go.mod"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	s := goproj.NewWorkspace()
	if err := project.New(dir).Init(project.Meta{Name: "myws", Template: "go-workspace"}, s); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(modDir, "go.mod")); got != existing {
		t.Errorf("myws/go.mod overwritten; got:\n%s", got)
	}
}

func TestGoWorkspace_Idempotent(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewWorkspace()
	p := project.New(dir)
	if err := p.Init(project.Meta{Name: "myws", Template: "go-workspace"}, s); err != nil {
		t.Fatal(err)
	}
	first := readFile(t, filepath.Join(dir, "go.work"))

	if err := p.Synthesize(s); err != nil {
		t.Fatal(err)
	}
	second := readFile(t, filepath.Join(dir, "go.work"))

	if first != second {
		t.Errorf("go.work changed after second synth:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestGoWorkspace_UseAccumulates(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewWorkspace()
	rc := map[string]any{
		"use": []any{"./api", "./worker"},
	}
	overlay, err := s.OverlayEvents(rc)
	if err != nil {
		t.Fatal(err)
	}
	template, _ := s.InitializeEvents("myws")
	aggs := project.BuildAggregates(template, overlay)
	if err := s.Synthesize(dir, aggs); err != nil {
		t.Fatal(err)
	}
	content := readFile(t, filepath.Join(dir, "go.work"))
	assertContains(t, content, "./myws")
	assertContains(t, content, "./api")
	assertContains(t, content, "./worker")
}

// ---- helpers ----------------------------------------------------------------------

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func assertContains(t *testing.T, content, want string) {
	t.Helper()
	if !strings.Contains(content, want) {
		t.Errorf("expected %q in:\n%s", want, content)
	}
}
