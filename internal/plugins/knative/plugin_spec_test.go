package knative_test

// Spec tests for the Knative plugin family.
//
// Three plugins, each with a single responsibility:
//
//   knative.New()      — generic base: .knative/service.yaml, .dockerignore (.git)
//   knative.NewNode()  — Node layer: package.json scripts, node_modules in .dockerignore, Node Dockerfile
//   knative.NewGo()    — Go layer:   Go Dockerfile
//
// Canonical compositions:
//   Node Knative: project.New(dir).WithPlugins(knative.New(), knative.NewNode())
//   Go Knative:   project.New(dir).WithPlugins(knative.New(), knative.NewGo())

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/domains/golang"
	"github.com/forgant-foundry/forglet/internal/domains/node"
	"github.com/forgant-foundry/forglet/internal/plugins/knative"
	"github.com/forgant-foundry/forglet/internal/project"
	"github.com/forgant-foundry/forglet/internal/testutil"
	"gopkg.in/yaml.v3"
)

// --- helpers ---

func applyEvents(t *testing.T, events []eventing.Event) *eventing.Aggregate {
	t.Helper()
	agg := eventing.NewAggregate()
	for i := range events {
		if err := agg.Apply(&events[i]); err != nil {
			t.Fatal(err)
		}
	}
	return agg
}

func nodeKnativeProject(dir string) *project.Project {
	return project.New(dir).WithPlugins(knative.New(), knative.NewNode())
}

func goKnativeProject(dir string) *project.Project {
	return project.New(dir).WithPlugins(knative.New(), knative.NewGo())
}

// ============================================================
// Plugin (generic base)
// ============================================================

func TestPlugin_Weave_ContributesToServiceYAML(t *testing.T) {
	stream := project.NewEventStream(nil)
	if err := knative.New().Weave(project.Meta{Name: "my-svc"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	if len(stream.Events()[".knative/service.yaml"]) == 0 {
		t.Error("expected events for .knative/service.yaml")
	}
}

func TestPlugin_Weave_ServiceYAMLAPIVersion(t *testing.T) {
	stream := project.NewEventStream(nil)
	knative.New().Weave(project.Meta{Name: "my-svc"}, nil, stream)
	agg := applyEvents(t, stream.Events()[".knative/service.yaml"])
	b, _ := agg.ToYAML()
	var svc map[string]any
	yaml.Unmarshal(b, &svc)
	if svc["apiVersion"] != "serving.knative.dev/v1" {
		t.Errorf("apiVersion = %v, want serving.knative.dev/v1", svc["apiVersion"])
	}
}

func TestPlugin_Weave_ServiceYAMLKind(t *testing.T) {
	stream := project.NewEventStream(nil)
	knative.New().Weave(project.Meta{Name: "my-svc"}, nil, stream)
	agg := applyEvents(t, stream.Events()[".knative/service.yaml"])
	b, _ := agg.ToYAML()
	var svc map[string]any
	yaml.Unmarshal(b, &svc)
	if svc["kind"] != "Service" {
		t.Errorf("kind = %v, want Service", svc["kind"])
	}
}

func TestPlugin_Weave_ServiceYAMLMetadataNameIsProjectName(t *testing.T) {
	stream := project.NewEventStream(nil)
	knative.New().Weave(project.Meta{Name: "my-svc"}, nil, stream)
	agg := applyEvents(t, stream.Events()[".knative/service.yaml"])
	b, _ := agg.ToYAML()
	var svc map[string]any
	yaml.Unmarshal(b, &svc)
	meta, _ := svc["metadata"].(map[string]any)
	if meta["name"] != "my-svc" {
		t.Errorf("metadata.name = %v, want my-svc", meta["name"])
	}
}

func TestPlugin_Weave_ServiceYAMLContainerImageUsesProjectName(t *testing.T) {
	stream := project.NewEventStream(nil)
	knative.New().Weave(project.Meta{Name: "my-svc"}, nil, stream)
	agg := applyEvents(t, stream.Events()[".knative/service.yaml"])
	b, _ := agg.ToYAML()
	if !strings.Contains(string(b), "my-svc:latest") {
		t.Error("service.yaml must reference <name>:latest as container image")
	}
}

func TestPlugin_Weave_DockerignoreHasGit(t *testing.T) {
	stream := project.NewEventStream(nil)
	knative.New().Weave(project.Meta{Name: "my-svc"}, nil, stream)
	agg := applyEvents(t, stream.Events()[".dockerignore"])
	found := false
	for _, n := range agg.Value {
		if n.Name == ".git" && n.Status == eventing.NodeActive {
			found = true
		}
	}
	if !found {
		t.Error(".dockerignore must contain .git pattern")
	}
}

func TestPlugin_Weave_DockerignoreDoesNotHaveNodeModules(t *testing.T) {
	stream := project.NewEventStream(nil)
	knative.New().Weave(project.Meta{Name: "my-svc"}, nil, stream)
	agg := applyEvents(t, stream.Events()[".dockerignore"])
	for _, n := range agg.Value {
		if n.Name == "node_modules" && n.Status == eventing.NodeActive {
			t.Error("generic Plugin must not add node_modules to .dockerignore — that belongs to NodePlugin")
		}
	}
}

// ============================================================
// NodePlugin
// ============================================================

func TestNodePlugin_Weave_AddsDockerBuildScript(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	knative.NewNode().Weave(project.Meta{Name: "my-app"}, nil, stream)
	agg := applyEvents(t, stream.Events()["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	scripts, _ := pkg["scripts"].(map[string]any)
	if scripts["docker:build"] == nil {
		t.Error("scripts.docker:build must be set")
	}
	if !strings.Contains(fmt.Sprint(scripts["docker:build"]), "my-app") {
		t.Errorf("scripts.docker:build = %v; must reference project name", scripts["docker:build"])
	}
}

func TestNodePlugin_Weave_AddsDockerPushScript(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	knative.NewNode().Weave(project.Meta{Name: "my-app"}, nil, stream)
	agg := applyEvents(t, stream.Events()["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	scripts, _ := pkg["scripts"].(map[string]any)
	if scripts["docker:push"] == nil {
		t.Error("scripts.docker:push must be set")
	}
}

func TestNodePlugin_Weave_AddsKnDeployScript(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	knative.NewNode().Weave(project.Meta{Name: "my-app"}, nil, stream)
	agg := applyEvents(t, stream.Events()["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	scripts, _ := pkg["scripts"].(map[string]any)
	if scripts["kn:deploy"] == nil {
		t.Error("scripts.kn:deploy must be set")
	}
	if !strings.Contains(fmt.Sprint(scripts["kn:deploy"]), "my-app") {
		t.Errorf("scripts.kn:deploy = %v; must reference project name", scripts["kn:deploy"])
	}
}

func TestNodePlugin_Weave_AddsNodeModulesToDockerignore(t *testing.T) {
	stream := project.NewEventStream(nil)
	knative.NewNode().Weave(project.Meta{Name: "my-app"}, nil, stream)
	agg := applyEvents(t, stream.Events()[".dockerignore"])
	found := false
	for _, n := range agg.Value {
		if n.Name == "node_modules" && n.Status == eventing.NodeActive {
			found = true
		}
	}
	if !found {
		t.Error("NodePlugin must add node_modules to .dockerignore")
	}
}

func TestNodePlugin_Scaffold_WritesDockerfile(t *testing.T) {
	dir := testutil.TempDir(t)
	if err := knative.NewNode().Scaffold(dir, project.Meta{Name: "my-app"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Dockerfile")); err != nil {
		t.Errorf("Dockerfile not created: %v", err)
	}
}

func TestNodePlugin_Scaffold_DockerfileUsesNodeImage(t *testing.T) {
	dir := testutil.TempDir(t)
	knative.NewNode().Scaffold(dir, project.Meta{Name: "my-app"})
	b, _ := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	if !strings.Contains(string(b), "node:22") {
		t.Error("Node Dockerfile must use node:22 base image")
	}
}

func TestNodePlugin_Scaffold_DockerfileNotOverwritten(t *testing.T) {
	dir := testutil.TempDir(t)
	p := knative.NewNode()
	p.Scaffold(dir, project.Meta{Name: "my-app"})
	custom := []byte("# custom\n")
	os.WriteFile(filepath.Join(dir, "Dockerfile"), custom, 0644)
	p.Scaffold(dir, project.Meta{Name: "my-app"})
	got, _ := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	if string(got) != string(custom) {
		t.Error("Dockerfile must not be overwritten on second Scaffold call")
	}
}

// ============================================================
// GoPlugin
// ============================================================

func TestGoPlugin_Weave_ContributesNothingToPackageJSON(t *testing.T) {
	stream := project.NewEventStream(nil)
	knative.NewGo().Weave(project.Meta{Name: "my-app"}, nil, stream)
	if len(stream.Events()["package.json"]) > 0 {
		t.Error("GoPlugin must not contribute to package.json")
	}
}

func TestGoPlugin_Scaffold_WritesDockerfile(t *testing.T) {
	dir := testutil.TempDir(t)
	if err := knative.NewGo().Scaffold(dir, project.Meta{Name: "my-app"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Dockerfile")); err != nil {
		t.Errorf("Dockerfile not created: %v", err)
	}
}

func TestGoPlugin_Scaffold_DockerfileUsesGoImage(t *testing.T) {
	dir := testutil.TempDir(t)
	knative.NewGo().Scaffold(dir, project.Meta{Name: "my-app"})
	b, _ := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	if !strings.Contains(string(b), "golang:") {
		t.Error("Go Dockerfile must use golang: base image")
	}
}

func TestGoPlugin_Scaffold_DockerfileNotOverwritten(t *testing.T) {
	dir := testutil.TempDir(t)
	p := knative.NewGo()
	p.Scaffold(dir, project.Meta{Name: "my-app"})
	custom := []byte("# custom\n")
	os.WriteFile(filepath.Join(dir, "Dockerfile"), custom, 0644)
	p.Scaffold(dir, project.Meta{Name: "my-app"})
	got, _ := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	if string(got) != string(custom) {
		t.Error("Dockerfile must not be overwritten on second Scaffold call")
	}
}

func TestGoPlugin_Weave_ContributesToFuncYAML(t *testing.T) {
	stream := project.NewEventStream(nil)
	knative.NewGo().Weave(project.Meta{Name: "my-func"}, nil, stream)
	if len(stream.Events()["func.yaml"]) == 0 {
		t.Error("expected events for func.yaml")
	}
}

func TestGoPlugin_Weave_FuncYAML_NameFromMeta(t *testing.T) {
	stream := project.NewEventStream(nil)
	knative.NewGo().Weave(project.Meta{Name: "my-func"}, nil, stream)
	agg := applyEvents(t, stream.Events()["func.yaml"])
	b, _ := agg.ToYAML()
	if !strings.Contains(string(b), "my-func") {
		t.Errorf("func.yaml must contain meta.Name; got:\n%s", b)
	}
}

func TestGoPlugin_Weave_FuncYAML_RuntimeIsGo(t *testing.T) {
	stream := project.NewEventStream(nil)
	knative.NewGo().Weave(project.Meta{Name: "my-func"}, nil, stream)
	agg := applyEvents(t, stream.Events()["func.yaml"])
	n, ok := agg.Node("runtime")
	if !ok {
		t.Fatal("no runtime node in func.yaml aggregate")
	}
	if n.Value.(string) != "go" {
		t.Errorf("runtime = %q, want go", n.Value)
	}
}

func TestGoPlugin_Weave_FuncYAML_SpecVersionSet(t *testing.T) {
	stream := project.NewEventStream(nil)
	knative.NewGo().Weave(project.Meta{Name: "my-func"}, nil, stream)
	agg := applyEvents(t, stream.Events()["func.yaml"])
	n, ok := agg.Node("specVersion")
	if !ok || n.Value.(string) == "" {
		t.Error("specVersion must be set in func.yaml")
	}
}

func TestGoPlugin_Weave_FuncYAML_RCNameOverride(t *testing.T) {
	stream := project.NewEventStream(nil)
	knative.NewGo().Weave(project.Meta{Name: "my-func"}, map[string]any{"name": "custom-name"}, stream)
	agg := applyEvents(t, stream.Events()["func.yaml"])
	n, _ := agg.Node("name")
	if n.Value.(string) != "custom-name" {
		t.Errorf("name = %q, want custom-name", n.Value)
	}
}

func TestGoPlugin_Weave_FuncYAML_RCRegistryOverride(t *testing.T) {
	stream := project.NewEventStream(nil)
	knative.NewGo().Weave(project.Meta{Name: "my-func"}, map[string]any{"registry": "gcr.io/myproject"}, stream)
	agg := applyEvents(t, stream.Events()["func.yaml"])
	n, _ := agg.Node("registry")
	if n.Value.(string) != "gcr.io/myproject" {
		t.Errorf("registry = %q, want gcr.io/myproject", n.Value)
	}
}

func TestGoPlugin_Scaffold_WritesHandleGo(t *testing.T) {
	dir := testutil.TempDir(t)
	knative.NewGo().Scaffold(dir, project.Meta{Name: "my-func"})
	content, err := os.ReadFile(filepath.Join(dir, "handle.go"))
	if err != nil {
		t.Fatalf("handle.go not created: %v", err)
	}
	if !strings.Contains(string(content), "func Handle") {
		t.Error("handle.go must define func Handle")
	}
}

func TestGoPlugin_Scaffold_WritesMainGo(t *testing.T) {
	dir := testutil.TempDir(t)
	knative.NewGo().Scaffold(dir, project.Meta{Name: "my-func"})
	content, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("main.go not created: %v", err)
	}
	if !strings.Contains(string(content), "http.HandleFunc") || !strings.Contains(string(content), "PORT") {
		t.Error("main.go must contain HTTP server wiring")
	}
}

func TestGoPlugin_Scaffold_WritesHandleTestGo(t *testing.T) {
	dir := testutil.TempDir(t)
	knative.NewGo().Scaffold(dir, project.Meta{Name: "my-func"})
	content, err := os.ReadFile(filepath.Join(dir, "handle_test.go"))
	if err != nil {
		t.Fatalf("handle_test.go not created: %v", err)
	}
	if !strings.Contains(string(content), "TestHandle") {
		t.Error("handle_test.go must contain TestHandle")
	}
}

func TestGoPlugin_Scaffold_HandleGoNotOverwritten(t *testing.T) {
	dir := testutil.TempDir(t)
	p := knative.NewGo()
	p.Scaffold(dir, project.Meta{Name: "my-func"})
	custom := []byte("// custom\n")
	os.WriteFile(filepath.Join(dir, "handle.go"), custom, 0644)
	p.Scaffold(dir, project.Meta{Name: "my-func"})
	got, _ := os.ReadFile(filepath.Join(dir, "handle.go"))
	if string(got) != string(custom) {
		t.Error("handle.go must not be overwritten on second Scaffold call")
	}
}

// ============================================================
// Integration — Go Knative func.yaml
// ============================================================

func TestGoKnative_Integration_FuncYamlExists(t *testing.T) {
	dir := testutil.TempDir(t)
	if err := goKnativeProject(dir).Init(project.Meta{Name: "my-func", Template: "go-knative"}, golang.NewGoKnative()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "func.yaml")); err != nil {
		t.Errorf("func.yaml not written: %v", err)
	}
}

func TestGoKnative_Integration_FuncYamlContent(t *testing.T) {
	dir := testutil.TempDir(t)
	goKnativeProject(dir).Init(project.Meta{Name: "my-func", Template: "go-knative"}, golang.NewGoKnative())
	b, _ := os.ReadFile(filepath.Join(dir, "func.yaml"))
	for _, want := range []string{"name:", "my-func", "runtime:", "go", "specVersion:"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("func.yaml missing %q", want)
		}
	}
}

func TestGoKnative_Integration_FuncYamlReadOnly(t *testing.T) {
	dir := testutil.TempDir(t)
	goKnativeProject(dir).Init(project.Meta{Name: "my-func", Template: "go-knative"}, golang.NewGoKnative())
	info, err := os.Stat(filepath.Join(dir, "func.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0444 {
		t.Errorf("func.yaml mode = %04o, want 0444", info.Mode().Perm())
	}
}

func TestGoKnative_Integration_FuncYamlHasMarker(t *testing.T) {
	dir := testutil.TempDir(t)
	goKnativeProject(dir).Init(project.Meta{Name: "my-func", Template: "go-knative"}, golang.NewGoKnative())
	b, _ := os.ReadFile(filepath.Join(dir, "func.yaml"))
	if !strings.HasPrefix(string(b), "# ") || !strings.Contains(string(b), "forglet") {
		t.Error("func.yaml must start with forglet managed comment")
	}
}

func TestGoKnative_Integration_HandleGoScaffolded(t *testing.T) {
	dir := testutil.TempDir(t)
	goKnativeProject(dir).Init(project.Meta{Name: "my-func", Template: "go-knative"}, golang.NewGoKnative())
	content, err := os.ReadFile(filepath.Join(dir, "handle.go"))
	if err != nil {
		t.Fatalf("handle.go not created: %v", err)
	}
	if !strings.Contains(string(content), "func Handle") {
		t.Error("handle.go must define func Handle")
	}
}

func TestGoKnative_Integration_MainGoHasHTTPServer(t *testing.T) {
	dir := testutil.TempDir(t)
	goKnativeProject(dir).Init(project.Meta{Name: "my-func", Template: "go-knative"}, golang.NewGoKnative())
	content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	if !strings.Contains(string(content), "http.HandleFunc") {
		t.Error("main.go must contain HTTP server wiring")
	}
}

// ============================================================
// Integration — Node Knative (New + NewNode)
// ============================================================

func TestNodeKnative_Integration_ServiceYAMLExists(t *testing.T) {
	dir := testutil.TempDir(t)
	if err := nodeKnativeProject(dir).Init(project.Meta{Name: "my-svc", Template: "node-ts"}, node.NewTypeScript()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".knative", "service.yaml")); err != nil {
		t.Errorf(".knative/service.yaml not written: %v", err)
	}
}

func TestNodeKnative_Integration_ServiceYAMLContent(t *testing.T) {
	dir := testutil.TempDir(t)
	nodeKnativeProject(dir).Init(project.Meta{Name: "my-svc", Template: "node-ts"}, node.NewTypeScript())
	b, _ := os.ReadFile(filepath.Join(dir, ".knative", "service.yaml"))
	for _, want := range []string{"serving.knative.dev/v1", "kind: Service", "name: my-svc", "my-svc:latest"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("service.yaml missing %q", want)
		}
	}
}

func TestNodeKnative_Integration_DockerfileExists(t *testing.T) {
	dir := testutil.TempDir(t)
	nodeKnativeProject(dir).Init(project.Meta{Name: "my-svc", Template: "node-ts"}, node.NewTypeScript())
	if _, err := os.Stat(filepath.Join(dir, "Dockerfile")); err != nil {
		t.Errorf("Dockerfile not created: %v", err)
	}
}

func TestNodeKnative_Integration_DockerfileIsNodeBased(t *testing.T) {
	dir := testutil.TempDir(t)
	nodeKnativeProject(dir).Init(project.Meta{Name: "my-svc", Template: "node-ts"}, node.NewTypeScript())
	b, _ := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	if !strings.Contains(string(b), "node:22") {
		t.Error("Node Knative Dockerfile must use node:22")
	}
}

func TestNodeKnative_Integration_DockerignoreHasGitAndNodeModules(t *testing.T) {
	dir := testutil.TempDir(t)
	nodeKnativeProject(dir).Init(project.Meta{Name: "my-svc", Template: "node-ts"}, node.NewTypeScript())
	b, _ := os.ReadFile(filepath.Join(dir, ".dockerignore"))
	for _, want := range []string{".git", "node_modules"} {
		if !strings.Contains(string(b), want) {
			t.Errorf(".dockerignore missing %q", want)
		}
	}
}

func TestNodeKnative_Integration_PackageJSONHasScripts(t *testing.T) {
	dir := testutil.TempDir(t)
	nodeKnativeProject(dir).Init(project.Meta{Name: "my-svc", Template: "node-ts"}, node.NewTypeScript())
	b, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	scripts, _ := pkg["scripts"].(map[string]any)
	for _, s := range []string{"docker:build", "docker:push", "kn:deploy"} {
		if scripts[s] == nil {
			t.Errorf("package.json missing scripts.%s", s)
		}
	}
}

func TestNodeKnative_Integration_Idempotent(t *testing.T) {
	dir := testutil.TempDir(t)
	proj := nodeKnativeProject(dir)
	s := node.NewTypeScript()
	proj.Init(project.Meta{Name: "my-svc", Template: "node-ts"}, s)
	svc1, _ := os.ReadFile(filepath.Join(dir, ".knative", "service.yaml"))
	pkg1, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	proj.Synthesize(s)
	svc2, _ := os.ReadFile(filepath.Join(dir, ".knative", "service.yaml"))
	pkg2, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	if string(svc1) != string(svc2) {
		t.Error("service.yaml changed on re-synth")
	}
	if string(pkg1) != string(pkg2) {
		t.Error("package.json changed on re-synth")
	}
}

func TestNodeKnative_Integration_DockerfileNotTouchedOnResynth(t *testing.T) {
	dir := testutil.TempDir(t)
	proj := nodeKnativeProject(dir)
	s := node.NewTypeScript()
	proj.Init(project.Meta{Name: "my-svc", Template: "node-ts"}, s)
	custom := []byte("# custom\n")
	os.WriteFile(filepath.Join(dir, "Dockerfile"), custom, 0644)
	proj.Synthesize(s)
	got, _ := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	if string(got) != string(custom) {
		t.Error("Dockerfile must not be modified by Synthesize")
	}
}

// ============================================================
// Integration — Go Knative (New + NewGo)
// ============================================================

func TestGoKnative_Integration_ServiceYAMLExists(t *testing.T) {
	dir := testutil.TempDir(t)
	if err := goKnativeProject(dir).Init(project.Meta{Name: "my-svc", Template: "go-knative"}, golang.NewGoKnative()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".knative", "service.yaml")); err != nil {
		t.Errorf(".knative/service.yaml not written: %v", err)
	}
}

func TestGoKnative_Integration_ServiceYAMLContent(t *testing.T) {
	dir := testutil.TempDir(t)
	goKnativeProject(dir).Init(project.Meta{Name: "my-svc", Template: "go-knative"}, golang.NewGoKnative())
	b, _ := os.ReadFile(filepath.Join(dir, ".knative", "service.yaml"))
	for _, want := range []string{"serving.knative.dev/v1", "kind: Service", "name: my-svc", "my-svc:latest"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("service.yaml missing %q", want)
		}
	}
}

func TestGoKnative_Integration_DockerfileExists(t *testing.T) {
	dir := testutil.TempDir(t)
	goKnativeProject(dir).Init(project.Meta{Name: "my-svc", Template: "go-knative"}, golang.NewGoKnative())
	if _, err := os.Stat(filepath.Join(dir, "Dockerfile")); err != nil {
		t.Errorf("Dockerfile not created: %v", err)
	}
}

func TestGoKnative_Integration_DockerfileIsGoBased(t *testing.T) {
	dir := testutil.TempDir(t)
	goKnativeProject(dir).Init(project.Meta{Name: "my-svc", Template: "go-knative"}, golang.NewGoKnative())
	b, _ := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	if !strings.Contains(string(b), "golang:") {
		t.Error("Go Knative Dockerfile must use golang: base image")
	}
}

func TestGoKnative_Integration_DockerignoreHasGit(t *testing.T) {
	dir := testutil.TempDir(t)
	goKnativeProject(dir).Init(project.Meta{Name: "my-svc", Template: "go-knative"}, golang.NewGoKnative())
	b, _ := os.ReadFile(filepath.Join(dir, ".dockerignore"))
	if !strings.Contains(string(b), ".git") {
		t.Error(".dockerignore must contain .git")
	}
}

func TestGoKnative_Integration_NoPackageJSON(t *testing.T) {
	dir := testutil.TempDir(t)
	goKnativeProject(dir).Init(project.Meta{Name: "my-svc", Template: "go-knative"}, golang.NewGoKnative())
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
		t.Error("Go Knative project must not have a package.json")
	}
}

func TestGoKnative_Integration_Idempotent(t *testing.T) {
	dir := testutil.TempDir(t)
	proj := goKnativeProject(dir)
	s := golang.NewGoKnative()
	proj.Init(project.Meta{Name: "my-svc", Template: "go-knative"}, s)
	svc1, _ := os.ReadFile(filepath.Join(dir, ".knative", "service.yaml"))
	funcYaml1, _ := os.ReadFile(filepath.Join(dir, "func.yaml"))
	proj.Synthesize(s)
	svc2, _ := os.ReadFile(filepath.Join(dir, ".knative", "service.yaml"))
	funcYaml2, _ := os.ReadFile(filepath.Join(dir, "func.yaml"))
	if string(svc1) != string(svc2) {
		t.Error("service.yaml changed on re-synth")
	}
	if string(funcYaml1) != string(funcYaml2) {
		t.Error("func.yaml changed on re-synth")
	}
}

func TestGoKnative_Integration_DockerfileNotTouchedOnResynth(t *testing.T) {
	dir := testutil.TempDir(t)
	proj := goKnativeProject(dir)
	s := golang.NewGoKnative()
	proj.Init(project.Meta{Name: "my-svc", Template: "go-knative"}, s)
	custom := []byte("# custom\n")
	os.WriteFile(filepath.Join(dir, "Dockerfile"), custom, 0644)
	proj.Synthesize(s)
	got, _ := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	if string(got) != string(custom) {
		t.Error("Dockerfile must not be modified by Synthesize")
	}
}
