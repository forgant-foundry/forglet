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

func TestGoFlat_Synthesize_GoModReadOnly(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewFlat()
	if err := project.New(dir).Init(project.Meta{Name: "myapp", Template: "go"}, s); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0444 {
		t.Errorf("go.mod mode = %04o, want 0444", info.Mode().Perm())
	}
}

func TestGoFlat_Synthesize_GoModHasMarker(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewFlat()
	if err := project.New(dir).Init(project.Meta{Name: "myapp", Template: "go"}, s); err != nil {
		t.Fatal(err)
	}
	content := readFile(t, filepath.Join(dir, "go.mod"))
	if !strings.HasPrefix(content, "// ") {
		t.Errorf("go.mod does not start with comment marker, got: %q", content[:min(40, len(content))])
	}
	assertContains(t, content, "forglet")
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

func TestGoWorkspace_Synthesize_GoWorkReadOnly(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewWorkspace()
	if err := project.New(dir).Init(project.Meta{Name: "myws", Template: "go-workspace"}, s); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "go.work"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0444 {
		t.Errorf("go.work mode = %04o, want 0444", info.Mode().Perm())
	}
}

func TestGoWorkspace_Synthesize_GoWorkHasMarker(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewWorkspace()
	if err := project.New(dir).Init(project.Meta{Name: "myws", Template: "go-workspace"}, s); err != nil {
		t.Fatal(err)
	}
	content := readFile(t, filepath.Join(dir, "go.work"))
	if !strings.HasPrefix(content, "// ") {
		t.Errorf("go.work does not start with comment marker, got: %q", content[:min(40, len(content))])
	}
	assertContains(t, content, "forglet")
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

// ---- Lambda (go-lambda) -----------------------------------------------------------

func TestGoLambda_InitializeEvents_ModuleFromName(t *testing.T) {
	s := goproj.NewLambda()
	events, err := s.InitializeEvents("myhandler")
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(events)
	modNode, ok := aggs["go.mod"].Node("module")
	if !ok {
		t.Fatal("no module node")
	}
	if got := modNode.Value.(string); got != "myhandler" {
		t.Errorf("module = %q, want %q", got, "myhandler")
	}
}

func TestGoLambda_InitializeEvents_GoVersionSet(t *testing.T) {
	s := goproj.NewLambda()
	events, err := s.InitializeEvents("myhandler")
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

func TestGoLambda_InitializeEvents_RequiresLambdaGo(t *testing.T) {
	s := goproj.NewLambda()
	events, err := s.InitializeEvents("myhandler")
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(events)
	dir := testutil.TempDir(t)
	if err := s.Synthesize(dir, aggs); err != nil {
		t.Fatal(err)
	}
	content := readFile(t, filepath.Join(dir, "go.mod"))
	assertContains(t, content, "github.com/aws/aws-lambda-go")
}

func TestGoLambda_OverlayEvents_Empty(t *testing.T) {
	s := goproj.NewLambda()
	events, err := s.OverlayEvents(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if events != nil {
		t.Error("expected nil for empty rc")
	}
}

func TestGoLambda_OverlayEvents_ModuleOverride(t *testing.T) {
	s := goproj.NewLambda()
	rc := map[string]any{"module": "github.com/myorg/handler"}
	templateEvents, _ := s.InitializeEvents("myhandler")
	overlayEvents, err := s.OverlayEvents(rc)
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(templateEvents, overlayEvents)
	modNode, _ := aggs["go.mod"].Node("module")
	if got := modNode.Value.(string); got != "github.com/myorg/handler" {
		t.Errorf("module = %q, want %q", got, "github.com/myorg/handler")
	}
}

func TestGoLambda_OverlayEvents_GoVersionOverride(t *testing.T) {
	s := goproj.NewLambda()
	rc := map[string]any{"go": "1.22"}
	templateEvents, _ := s.InitializeEvents("myhandler")
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

func TestGoLambda_OverlayEvents_RequireAccumulates(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewLambda()
	rc := map[string]any{
		"require": map[string]any{"github.com/some/lib": "v1.0.0"},
	}
	templateEvents, _ := s.InitializeEvents("myhandler")
	overlayEvents, err := s.OverlayEvents(rc)
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(templateEvents, overlayEvents)
	if err := s.Synthesize(dir, aggs); err != nil {
		t.Fatal(err)
	}
	content := readFile(t, filepath.Join(dir, "go.mod"))
	assertContains(t, content, "github.com/aws/aws-lambda-go")
	assertContains(t, content, "github.com/some/lib v1.0.0")
}

func TestGoLambda_Synthesize_GoModContent(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewLambda()
	if err := project.New(dir).Init(project.Meta{Name: "myhandler", Template: "go-lambda"}, s); err != nil {
		t.Fatal(err)
	}
	content := readFile(t, filepath.Join(dir, "go.mod"))
	assertContains(t, content, "module myhandler")
	assertContains(t, content, "go ")
	assertContains(t, content, "github.com/aws/aws-lambda-go")
}

func TestGoLambda_Synthesize_MainGoScaffolded(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewLambda()
	if err := project.New(dir).Init(project.Meta{Name: "myhandler", Template: "go-lambda"}, s); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "main.go")); err != nil {
		t.Error("main.go not created")
	}
	content := readFile(t, filepath.Join(dir, "main.go"))
	assertContains(t, content, "lambda.Start")
	assertContains(t, content, "HandleRequest")
}

func TestGoLambda_Synthesize_MainGoNotOverwritten(t *testing.T) {
	dir := testutil.TempDir(t)
	existing := "package main\n\nfunc main() { panic(\"custom\") }\n"
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}
	s := goproj.NewLambda()
	if err := project.New(dir).Init(project.Meta{Name: "myhandler", Template: "go-lambda"}, s); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, "main.go")); got != existing {
		t.Errorf("main.go overwritten; got:\n%s", got)
	}
}

func TestGoLambda_Synthesize_MakefileScaffolded(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewLambda()
	if err := project.New(dir).Init(project.Meta{Name: "myhandler", Template: "go-lambda"}, s); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Makefile")); err != nil {
		t.Error("Makefile not created")
	}
	content := readFile(t, filepath.Join(dir, "Makefile"))
	assertContains(t, content, "GOOS=linux")
	assertContains(t, content, "GOARCH=amd64")
	assertContains(t, content, "CGO_ENABLED=0")
	assertContains(t, content, "-o bootstrap")
}

func TestGoLambda_Synthesize_MakefileNotOverwritten(t *testing.T) {
	dir := testutil.TempDir(t)
	existing := ".PHONY: build\n\nbuild:\n\techo custom\n"
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}
	s := goproj.NewLambda()
	if err := project.New(dir).Init(project.Meta{Name: "myhandler", Template: "go-lambda"}, s); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, "Makefile")); got != existing {
		t.Errorf("Makefile overwritten; got:\n%s", got)
	}
}

func TestGoLambda_Idempotent(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewLambda()
	p := project.New(dir)
	if err := p.Init(project.Meta{Name: "myhandler", Template: "go-lambda"}, s); err != nil {
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

func TestGoLambda_Synthesize_GoModReadOnly(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewLambda()
	if err := project.New(dir).Init(project.Meta{Name: "myhandler", Template: "go-lambda"}, s); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0444 {
		t.Errorf("go.mod mode = %04o, want 0444", info.Mode().Perm())
	}
}

func TestGoLambda_Synthesize_GoModHasMarker(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewLambda()
	if err := project.New(dir).Init(project.Meta{Name: "myhandler", Template: "go-lambda"}, s); err != nil {
		t.Fatal(err)
	}
	content := readFile(t, filepath.Join(dir, "go.mod"))
	if !strings.HasPrefix(content, "// ") {
		t.Errorf("go.mod does not start with comment marker, got: %q", content[:min(40, len(content))])
	}
	assertContains(t, content, "forglet")
}

// ---- KnativeFunc (go-knative) -----------------------------------------------------

func TestGoKnativeFunc_InitializeEvents_NameFromProject(t *testing.T) {
	s := goproj.NewKnativeFunc()
	events, err := s.InitializeEvents("myfunc")
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(events)
	nameNode, ok := aggs["func.yaml"].Node("name")
	if !ok {
		t.Fatal("no name node in func.yaml aggregate")
	}
	if got := nameNode.Value.(string); got != "myfunc" {
		t.Errorf("name = %q, want %q", got, "myfunc")
	}
}

func TestGoKnativeFunc_InitializeEvents_RuntimeIsGo(t *testing.T) {
	s := goproj.NewKnativeFunc()
	events, err := s.InitializeEvents("myfunc")
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(events)
	runtimeNode, ok := aggs["func.yaml"].Node("runtime")
	if !ok {
		t.Fatal("no runtime node in func.yaml aggregate")
	}
	if got := runtimeNode.Value.(string); got != "go" {
		t.Errorf("runtime = %q, want %q", got, "go")
	}
}

func TestGoKnativeFunc_InitializeEvents_SpecVersionSet(t *testing.T) {
	s := goproj.NewKnativeFunc()
	events, err := s.InitializeEvents("myfunc")
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(events)
	node, ok := aggs["func.yaml"].Node("specVersion")
	if !ok {
		t.Fatal("no specVersion node")
	}
	if node.Value.(string) == "" {
		t.Error("specVersion is empty")
	}
}

func TestGoKnativeFunc_InitializeEvents_GoModPresent(t *testing.T) {
	s := goproj.NewKnativeFunc()
	events, err := s.InitializeEvents("myfunc")
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(events)
	if _, ok := aggs["go.mod"]; !ok {
		t.Fatal("go.mod aggregate missing from InitializeEvents")
	}
}

func TestGoKnativeFunc_OverlayEvents_Empty(t *testing.T) {
	s := goproj.NewKnativeFunc()
	events, err := s.OverlayEvents(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if events != nil {
		t.Error("expected nil for empty rc")
	}
}

func TestGoKnativeFunc_OverlayEvents_NameOverride(t *testing.T) {
	s := goproj.NewKnativeFunc()
	rc := map[string]any{"name": "custom-func"}
	templateEvents, _ := s.InitializeEvents("myfunc")
	overlayEvents, err := s.OverlayEvents(rc)
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(templateEvents, overlayEvents)
	nameNode, _ := aggs["func.yaml"].Node("name")
	if got := nameNode.Value.(string); got != "custom-func" {
		t.Errorf("name = %q, want %q", got, "custom-func")
	}
}

func TestGoKnativeFunc_OverlayEvents_RegistryOverride(t *testing.T) {
	s := goproj.NewKnativeFunc()
	rc := map[string]any{"registry": "gcr.io/myproject"}
	templateEvents, _ := s.InitializeEvents("myfunc")
	overlayEvents, err := s.OverlayEvents(rc)
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(templateEvents, overlayEvents)
	regNode, _ := aggs["func.yaml"].Node("registry")
	if got := regNode.Value.(string); got != "gcr.io/myproject" {
		t.Errorf("registry = %q, want %q", got, "gcr.io/myproject")
	}
}

func TestGoKnativeFunc_OverlayEvents_GoVersionOverride(t *testing.T) {
	s := goproj.NewKnativeFunc()
	rc := map[string]any{"go": "1.22"}
	templateEvents, _ := s.InitializeEvents("myfunc")
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

func TestGoKnativeFunc_Synthesize_FuncYamlContent(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewKnativeFunc()
	if err := project.New(dir).Init(project.Meta{Name: "myfunc", Template: "go-knative"}, s); err != nil {
		t.Fatal(err)
	}
	content := readFile(t, filepath.Join(dir, "func.yaml"))
	assertContains(t, content, "name:")
	assertContains(t, content, "myfunc")
	assertContains(t, content, "runtime:")
	assertContains(t, content, "go")
	assertContains(t, content, "specVersion:")
}

func TestGoKnativeFunc_Synthesize_GoModContent(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewKnativeFunc()
	if err := project.New(dir).Init(project.Meta{Name: "myfunc", Template: "go-knative"}, s); err != nil {
		t.Fatal(err)
	}
	content := readFile(t, filepath.Join(dir, "go.mod"))
	assertContains(t, content, "module myfunc")
	assertContains(t, content, "go ")
}

func TestGoKnativeFunc_Synthesize_HandleGoScaffolded(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewKnativeFunc()
	if err := project.New(dir).Init(project.Meta{Name: "myfunc", Template: "go-knative"}, s); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "handle.go")); err != nil {
		t.Error("handle.go not created")
	}
	content := readFile(t, filepath.Join(dir, "handle.go"))
	assertContains(t, content, "func Handle")
	assertContains(t, content, "http.ResponseWriter")
}

func TestGoKnativeFunc_Synthesize_HandleGoNotOverwritten(t *testing.T) {
	dir := testutil.TempDir(t)
	existing := "package main\n\nimport \"net/http\"\n\nfunc Handle(res http.ResponseWriter, req *http.Request) {}\n"
	if err := os.WriteFile(filepath.Join(dir, "handle.go"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}
	s := goproj.NewKnativeFunc()
	if err := project.New(dir).Init(project.Meta{Name: "myfunc", Template: "go-knative"}, s); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, "handle.go")); got != existing {
		t.Errorf("handle.go overwritten; got:\n%s", got)
	}
}

func TestGoKnativeFunc_Synthesize_MainGoScaffolded(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewKnativeFunc()
	if err := project.New(dir).Init(project.Meta{Name: "myfunc", Template: "go-knative"}, s); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "main.go")); err != nil {
		t.Error("main.go not created")
	}
	content := readFile(t, filepath.Join(dir, "main.go"))
	assertContains(t, content, "http.HandleFunc")
	assertContains(t, content, "PORT")
}

func TestGoKnativeFunc_Synthesize_MainGoNotOverwritten(t *testing.T) {
	dir := testutil.TempDir(t)
	existing := "package main\n\nfunc main() { panic(\"custom\") }\n"
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}
	s := goproj.NewKnativeFunc()
	if err := project.New(dir).Init(project.Meta{Name: "myfunc", Template: "go-knative"}, s); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, "main.go")); got != existing {
		t.Errorf("main.go overwritten; got:\n%s", got)
	}
}

func TestGoKnativeFunc_Synthesize_HandleTestGoScaffolded(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewKnativeFunc()
	if err := project.New(dir).Init(project.Meta{Name: "myfunc", Template: "go-knative"}, s); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "handle_test.go")); err != nil {
		t.Error("handle_test.go not created")
	}
	content := readFile(t, filepath.Join(dir, "handle_test.go"))
	assertContains(t, content, "TestHandle")
	assertContains(t, content, "httptest")
}

func TestGoKnativeFunc_Idempotent(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewKnativeFunc()
	p := project.New(dir)
	if err := p.Init(project.Meta{Name: "myfunc", Template: "go-knative"}, s); err != nil {
		t.Fatal(err)
	}
	firstGoMod := readFile(t, filepath.Join(dir, "go.mod"))
	firstFuncYaml := readFile(t, filepath.Join(dir, "func.yaml"))

	if err := p.Synthesize(s); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, "go.mod")); got != firstGoMod {
		t.Errorf("go.mod changed after second synth")
	}
	if got := readFile(t, filepath.Join(dir, "func.yaml")); got != firstFuncYaml {
		t.Errorf("func.yaml changed after second synth")
	}
}

func TestGoKnativeFunc_Synthesize_FuncYamlReadOnly(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewKnativeFunc()
	if err := project.New(dir).Init(project.Meta{Name: "myfunc", Template: "go-knative"}, s); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "func.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0444 {
		t.Errorf("func.yaml mode = %04o, want 0444", info.Mode().Perm())
	}
}

func TestGoKnativeFunc_Synthesize_FuncYamlHasMarker(t *testing.T) {
	dir := testutil.TempDir(t)
	s := goproj.NewKnativeFunc()
	if err := project.New(dir).Init(project.Meta{Name: "myfunc", Template: "go-knative"}, s); err != nil {
		t.Fatal(err)
	}
	content := readFile(t, filepath.Join(dir, "func.yaml"))
	if !strings.HasPrefix(content, "# ") {
		t.Errorf("func.yaml does not start with comment marker, got: %q", content[:min(40, len(content))])
	}
	assertContains(t, content, "forglet")
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
