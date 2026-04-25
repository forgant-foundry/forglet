package knative_test

// Spec tests for KnativePlugin.
//
// KnativePlugin manages:
//   - .knative/service.yaml  — YAML cross-cutting file (Knative Serving manifest)
//   - package.json scripts   — docker:build, docker:push, kn:deploy
//   - .dockerignore          — pattern cross-cutting file
//   - Dockerfile             — scaffolded once during Init
//
// Compatible with node-ts and node-js templates; template-agnostic.
// Certified against: Knative Serving v1 API / node:22-alpine base image

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/domains/node"
	"github.com/forgant-foundry/forglet/internal/plugins/knative"
	"github.com/forgant-foundry/forglet/internal/project"
	"github.com/forgant-foundry/forglet/internal/testutil"
	"gopkg.in/yaml.v3"
)

func knativeAgg(t *testing.T, events []eventing.Event) *eventing.Aggregate {
	t.Helper()
	agg := eventing.NewAggregate()
	for i := range events {
		if err := agg.Apply(&events[i]); err != nil {
			t.Fatal(err)
		}
	}
	return agg
}

func knativeProject(dir string) *project.Project {
	return project.New(dir).WithPlugins(knative.New())
}

// --- Weave: .knative/service.yaml ---

func TestKnativePlugin_Weave_ContributesToServiceYAML(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := knative.New()
	if err := p.Weave(project.Meta{Name: "my-app", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	if len(stream.Events()[".knative/service.yaml"]) == 0 {
		t.Error("no events for .knative/service.yaml; plugin must contribute to the service.yaml stream")
	}
}

func TestKnativePlugin_Weave_ServiceYAMLHasAPIVersion(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := knative.New()
	if err := p.Weave(project.Meta{Name: "my-app", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	agg := knativeAgg(t, stream.Events()[".knative/service.yaml"])
	b, _ := agg.ToYAML()
	var svc map[string]any
	yaml.Unmarshal(b, &svc)
	if svc["apiVersion"] != "serving.knative.dev/v1" {
		t.Errorf("apiVersion = %v, want %q", svc["apiVersion"], "serving.knative.dev/v1")
	}
}

func TestKnativePlugin_Weave_ServiceYAMLHasKind(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := knative.New()
	if err := p.Weave(project.Meta{Name: "my-app", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	agg := knativeAgg(t, stream.Events()[".knative/service.yaml"])
	b, _ := agg.ToYAML()
	var svc map[string]any
	yaml.Unmarshal(b, &svc)
	if svc["kind"] != "Service" {
		t.Errorf("kind = %v, want %q", svc["kind"], "Service")
	}
}

func TestKnativePlugin_Weave_ServiceYAMLMetadataNameIsProjectName(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := knative.New()
	if err := p.Weave(project.Meta{Name: "my-svc", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	agg := knativeAgg(t, stream.Events()[".knative/service.yaml"])
	b, _ := agg.ToYAML()
	var svc map[string]any
	yaml.Unmarshal(b, &svc)
	meta, _ := svc["metadata"].(map[string]any)
	if meta["name"] != "my-svc" {
		t.Errorf("metadata.name = %v, want %q", meta["name"], "my-svc")
	}
}

func TestKnativePlugin_Weave_ServiceYAMLContainerImageUsesProjectName(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := knative.New()
	if err := p.Weave(project.Meta{Name: "my-svc", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	agg := knativeAgg(t, stream.Events()[".knative/service.yaml"])
	b, _ := agg.ToYAML()
	if !strings.Contains(string(b), "my-svc:latest") {
		t.Error("service.yaml must reference <project-name>:latest as the container image")
	}
}

// --- Weave: package.json scripts ---

func TestKnativePlugin_Weave_AddsDockerBuildScript(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := knative.New()
	if err := p.Weave(project.Meta{Name: "my-app", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	agg := knativeAgg(t, stream.Events()["package.json"])
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

func TestKnativePlugin_Weave_AddsDockerPushScript(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := knative.New()
	if err := p.Weave(project.Meta{Name: "my-app", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	agg := knativeAgg(t, stream.Events()["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	scripts, _ := pkg["scripts"].(map[string]any)
	if scripts["docker:push"] == nil {
		t.Error("scripts.docker:push must be set")
	}
}

func TestKnativePlugin_Weave_AddsKnDeployScript(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := knative.New()
	if err := p.Weave(project.Meta{Name: "my-app", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	agg := knativeAgg(t, stream.Events()["package.json"])
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

// --- Weave: .dockerignore ---

func TestKnativePlugin_Weave_AddsDockerignorePatterns(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := knative.New()
	if err := p.Weave(project.Meta{Name: "my-app", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	if len(stream.Events()[".dockerignore"]) == 0 {
		t.Error("no events for .dockerignore; plugin must contribute dockerignore patterns")
	}
}

func TestKnativePlugin_Weave_DockerignoreHasNodeModules(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := knative.New()
	if err := p.Weave(project.Meta{Name: "my-app", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	agg := knativeAgg(t, stream.Events()[".dockerignore"])
	found := false
	for _, n := range agg.Value {
		if n.Name == "node_modules" && n.Status == eventing.NodeActive {
			found = true
			break
		}
	}
	if !found {
		t.Error(".dockerignore aggregate must contain node_modules pattern")
	}
}

func TestKnativePlugin_Weave_DockerignoreHasGit(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := knative.New()
	if err := p.Weave(project.Meta{Name: "my-app", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	agg := knativeAgg(t, stream.Events()[".dockerignore"])
	found := false
	for _, n := range agg.Value {
		if n.Name == ".git" && n.Status == eventing.NodeActive {
			found = true
			break
		}
	}
	if !found {
		t.Error(".dockerignore aggregate must contain .git pattern")
	}
}

// --- Scaffold ---

func TestKnativePlugin_Scaffold_WritesDockerfile(t *testing.T) {
	dir := testutil.TempDir(t)
	p := knative.New()
	if err := p.Scaffold(dir, project.Meta{Name: "my-app", Template: "node-ts"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Dockerfile")); err != nil {
		t.Errorf("Dockerfile not created: %v", err)
	}
}

func TestKnativePlugin_Scaffold_DockerfileReferencesNodeImage(t *testing.T) {
	dir := testutil.TempDir(t)
	p := knative.New()
	if err := p.Scaffold(dir, project.Meta{Name: "my-app", Template: "node-ts"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	if err != nil {
		t.Fatalf("Dockerfile not created: %v", err)
	}
	if !strings.Contains(string(b), "node:22") {
		t.Error("Dockerfile must use node:22 base image")
	}
}

func TestKnativePlugin_Scaffold_DockerfileNotOverwritten(t *testing.T) {
	dir := testutil.TempDir(t)
	p := knative.New()
	if err := p.Scaffold(dir, project.Meta{Name: "my-app", Template: "node-ts"}); err != nil {
		t.Fatal(err)
	}
	dockerfilePath := filepath.Join(dir, "Dockerfile")
	customContent := []byte("# custom\n")
	if err := os.WriteFile(dockerfilePath, customContent, 0644); err != nil {
		t.Fatal(err)
	}
	if err := p.Scaffold(dir, project.Meta{Name: "my-app", Template: "node-ts"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dockerfilePath)
	if string(got) != string(customContent) {
		t.Error("Dockerfile was overwritten on second Scaffold call; scaffold files must be written once only")
	}
}

// --- Integration (full project.Init pipeline) ---

func TestKnativePlugin_Integration_NodeTS_ServiceYAMLExists(t *testing.T) {
	dir := testutil.TempDir(t)
	if err := knativeProject(dir).Init(project.Meta{Name: "my-svc", Template: "node-ts"}, node.NewTypeScript()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".knative", "service.yaml")); err != nil {
		t.Errorf(".knative/service.yaml not written: %v", err)
	}
}

func TestKnativePlugin_Integration_NodeTS_ServiceYAMLContent(t *testing.T) {
	dir := testutil.TempDir(t)
	if err := knativeProject(dir).Init(project.Meta{Name: "my-svc", Template: "node-ts"}, node.NewTypeScript()); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, ".knative", "service.yaml"))
	if err != nil {
		t.Fatalf(".knative/service.yaml not written: %v", err)
	}
	content := string(b)
	for _, want := range []string{
		"serving.knative.dev/v1",
		"kind: Service",
		"name: my-svc",
		"my-svc:latest",
	} {
		if !strings.Contains(content, want) {
			t.Errorf(".knative/service.yaml missing %q", want)
		}
	}
}

func TestKnativePlugin_Integration_NodeTS_DockerfileExists(t *testing.T) {
	dir := testutil.TempDir(t)
	if err := knativeProject(dir).Init(project.Meta{Name: "my-svc", Template: "node-ts"}, node.NewTypeScript()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Dockerfile")); err != nil {
		t.Errorf("Dockerfile not created: %v", err)
	}
}

func TestKnativePlugin_Integration_NodeTS_DockerignoreExists(t *testing.T) {
	dir := testutil.TempDir(t)
	if err := knativeProject(dir).Init(project.Meta{Name: "my-svc", Template: "node-ts"}, node.NewTypeScript()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".dockerignore")); err != nil {
		t.Errorf(".dockerignore not written: %v", err)
	}
}

func TestKnativePlugin_Integration_NodeTS_DockerignoreHasPatterns(t *testing.T) {
	dir := testutil.TempDir(t)
	if err := knativeProject(dir).Init(project.Meta{Name: "my-svc", Template: "node-ts"}, node.NewTypeScript()); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, ".dockerignore"))
	if err != nil {
		t.Fatalf(".dockerignore not written: %v", err)
	}
	for _, pattern := range []string{"node_modules", ".git"} {
		if !strings.Contains(string(b), pattern) {
			t.Errorf(".dockerignore missing pattern %q", pattern)
		}
	}
}

func TestKnativePlugin_Integration_NodeTS_PackageJSONHasScripts(t *testing.T) {
	dir := testutil.TempDir(t)
	if err := knativeProject(dir).Init(project.Meta{Name: "my-svc", Template: "node-ts"}, node.NewTypeScript()); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	scripts, _ := pkg["scripts"].(map[string]any)
	for _, script := range []string{"docker:build", "docker:push", "kn:deploy"} {
		if scripts[script] == nil {
			t.Errorf("package.json scripts.%s is missing", script)
		}
	}
}

// TestKnativePlugin_Integration_Idempotent verifies re-synth produces the same files.
func TestKnativePlugin_Integration_Idempotent(t *testing.T) {
	dir := testutil.TempDir(t)
	proj := knativeProject(dir)
	s := node.NewTypeScript()
	if err := proj.Init(project.Meta{Name: "my-svc", Template: "node-ts"}, s); err != nil {
		t.Fatal(err)
	}

	b1, _ := os.ReadFile(filepath.Join(dir, ".knative", "service.yaml"))
	pkgB1, _ := os.ReadFile(filepath.Join(dir, "package.json"))

	if err := proj.Synthesize(s); err != nil {
		t.Fatal(err)
	}

	b2, _ := os.ReadFile(filepath.Join(dir, ".knative", "service.yaml"))
	pkgB2, _ := os.ReadFile(filepath.Join(dir, "package.json"))

	if string(b1) != string(b2) {
		t.Error(".knative/service.yaml changed on re-synth; synthesis must be idempotent")
	}
	if string(pkgB1) != string(pkgB2) {
		t.Error("package.json changed on re-synth; synthesis must be idempotent")
	}
}

// TestKnativePlugin_Integration_DockerfileNotTouchedOnResynth verifies that
// re-running Synthesize never modifies the scaffolded Dockerfile.
func TestKnativePlugin_Integration_DockerfileNotTouchedOnResynth(t *testing.T) {
	dir := testutil.TempDir(t)
	proj := knativeProject(dir)
	s := node.NewTypeScript()
	if err := proj.Init(project.Meta{Name: "my-svc", Template: "node-ts"}, s); err != nil {
		t.Fatal(err)
	}

	dockerfilePath := filepath.Join(dir, "Dockerfile")
	customContent := []byte("# custom\n")
	if err := os.WriteFile(dockerfilePath, customContent, 0644); err != nil {
		t.Fatal(err)
	}

	if err := proj.Synthesize(s); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dockerfilePath)
	if string(got) != string(customContent) {
		t.Error("Dockerfile was modified by Synthesize; scaffold files must be immutable after Init")
	}
}
