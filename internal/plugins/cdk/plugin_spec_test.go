package cdk_test

// Spec tests for CDKPlugin.
// All tests here are intentionally failing until the plugin is implemented.
//
// Model: per-function packages (npm workspaces + NodejsFunction per Lambda).
// Register WorkspacesPlugin alongside CDKPlugin.
//
// Certified against: aws-cdk-lib ^2.0.0 / constructs ^10.0.0 / esbuild ^0.25.0
// node v22.14.0 / npm 11.7.0

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/domains/node"
	"github.com/forgant-foundry/forglet/internal/plugins/cdk"
	"github.com/forgant-foundry/forglet/internal/plugins/workspaces"
	"github.com/forgant-foundry/forglet/internal/project"
	"github.com/forgant-foundry/forglet/internal/testutil"
)

const (
	cdkLibVersion     = "^2.0.0"  // aws-cdk-lib
	constructsVersion = "^10.0.0" // constructs
	cdkCLIVersion     = "^2.0.0"  // aws-cdk CLI
	esbuildVersion    = "^0.25.0" // esbuild (used internally by NodejsFunction)
	cdkAppEntry       = "npx ts-node --prefer-ts-exts bin/app.ts"
)

func cdkAgg(t *testing.T, events []eventing.Event) *eventing.Aggregate {
	t.Helper()
	agg := eventing.NewAggregate()
	for i := range events {
		if err := agg.Apply(&events[i]); err != nil {
			t.Fatal(err)
		}
	}
	return agg
}

func cdkProject(dir string) *project.Project {
	return project.New(dir).WithPlugins(workspaces.New(), cdk.New())
}

// --- Weave: package.json devDependencies ---

func TestCDKPlugin_Weave_AddsAWSCDKLib(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := cdk.New()
	if err := p.Weave(project.Meta{Name: "my-app", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	agg := cdkAgg(t, stream.Events()["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	devDeps, _ := pkg["devDependencies"].(map[string]any)
	if devDeps["aws-cdk-lib"] != cdkLibVersion {
		t.Errorf("devDependencies.aws-cdk-lib = %v, want %q", devDeps["aws-cdk-lib"], cdkLibVersion)
	}
}

func TestCDKPlugin_Weave_AddsConstructs(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := cdk.New()
	if err := p.Weave(project.Meta{Name: "my-app", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	agg := cdkAgg(t, stream.Events()["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	devDeps, _ := pkg["devDependencies"].(map[string]any)
	if devDeps["constructs"] != constructsVersion {
		t.Errorf("devDependencies.constructs = %v, want %q", devDeps["constructs"], constructsVersion)
	}
}

func TestCDKPlugin_Weave_AddsAWSCDKCLI(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := cdk.New()
	if err := p.Weave(project.Meta{Name: "my-app", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	agg := cdkAgg(t, stream.Events()["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	devDeps, _ := pkg["devDependencies"].(map[string]any)
	if devDeps["aws-cdk"] != cdkCLIVersion {
		t.Errorf("devDependencies.aws-cdk = %v, want %q", devDeps["aws-cdk"], cdkCLIVersion)
	}
}

func TestCDKPlugin_Weave_AddsEsbuild(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := cdk.New()
	if err := p.Weave(project.Meta{Name: "my-app", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	agg := cdkAgg(t, stream.Events()["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	devDeps, _ := pkg["devDependencies"].(map[string]any)
	if devDeps["esbuild"] != esbuildVersion {
		t.Errorf("devDependencies.esbuild = %v, want %q", devDeps["esbuild"], esbuildVersion)
	}
}

// --- Weave: package.json scripts ---

func TestCDKPlugin_Weave_AddsCDKScript(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := cdk.New()
	if err := p.Weave(project.Meta{Name: "my-app", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	agg := cdkAgg(t, stream.Events()["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	scripts, _ := pkg["scripts"].(map[string]any)
	if scripts["deploy"] != "cdk deploy" {
		t.Errorf("scripts.deploy = %v, want %q", scripts["deploy"], "cdk deploy")
	}
	if scripts["destroy"] != "cdk destroy" {
		t.Errorf("scripts.destroy = %v, want %q", scripts["destroy"], "cdk destroy")
	}
	if scripts["synth"] != "cdk synth" {
		t.Errorf("scripts.synth = %v, want %q", scripts["synth"], "cdk synth")
	}
}

// --- Weave: cdk.json ---

func TestCDKPlugin_Weave_ContributesToCDKJSON(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := cdk.New()
	if err := p.Weave(project.Meta{Name: "my-app", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	if len(stream.Events()["cdk.json"]) == 0 {
		t.Error("no events for cdk.json; plugin must contribute to the cdk.json stream")
	}
}

func TestCDKPlugin_Weave_CDKJSONHasAppField(t *testing.T) {
	stream := project.NewEventStream(map[string][]eventing.Event{"package.json": {}})
	p := cdk.New()
	if err := p.Weave(project.Meta{Name: "my-app", Template: "node-ts"}, nil, stream); err != nil {
		t.Fatal(err)
	}
	agg := cdkAgg(t, stream.Events()["cdk.json"])
	b, _ := agg.ToJSON()
	var cdkJSON map[string]any
	json.Unmarshal(b, &cdkJSON)
	if cdkJSON["app"] != cdkAppEntry {
		t.Errorf("cdk.json app = %v, want %q", cdkJSON["app"], cdkAppEntry)
	}
}

// --- Scaffold ---

func TestCDKPlugin_Scaffold_WritesBinAppTS(t *testing.T) {
	dir := testutil.TempDir(t)
	p := cdk.New()
	if err := p.Scaffold(dir, project.Meta{Name: "my-app", Template: "node-ts"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "bin", "app.ts")); err != nil {
		t.Errorf("bin/app.ts not created: %v", err)
	}
}

func TestCDKPlugin_Scaffold_WritesLibStackTS(t *testing.T) {
	dir := testutil.TempDir(t)
	p := cdk.New()
	if err := p.Scaffold(dir, project.Meta{Name: "my-app", Template: "node-ts"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "lib", "stack.ts")); err != nil {
		t.Errorf("lib/stack.ts not created: %v", err)
	}
}

// TestCDKPlugin_Scaffold_LibStackTSReferencesNodejsFunction verifies the scaffold
// uses NodejsFunction — the per-function Lambda pattern.
func TestCDKPlugin_Scaffold_LibStackTSReferencesNodejsFunction(t *testing.T) {
	dir := testutil.TempDir(t)
	p := cdk.New()
	if err := p.Scaffold(dir, project.Meta{Name: "my-app", Template: "node-ts"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "lib", "stack.ts"))
	if err != nil {
		t.Fatalf("lib/stack.ts not created: %v", err)
	}
	if !containsString(b, "NodejsFunction") {
		t.Error("lib/stack.ts must reference NodejsFunction for per-function Lambda pattern")
	}
}

func TestCDKPlugin_Scaffold_BinAppTSNotOverwritten(t *testing.T) {
	dir := testutil.TempDir(t)
	p := cdk.New()
	if err := p.Scaffold(dir, project.Meta{Name: "my-app", Template: "node-ts"}); err != nil {
		t.Fatal(err)
	}
	binPath := filepath.Join(dir, "bin", "app.ts")
	customContent := []byte("// custom\n")
	if err := os.WriteFile(binPath, customContent, 0644); err != nil {
		t.Fatal(err)
	}
	if err := p.Scaffold(dir, project.Meta{Name: "my-app", Template: "node-ts"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(binPath)
	if string(got) != string(customContent) {
		t.Error("bin/app.ts was overwritten on second Scaffold call; scaffold files must be written once only")
	}
}

func TestCDKPlugin_Scaffold_LibStackTSNotOverwritten(t *testing.T) {
	dir := testutil.TempDir(t)
	p := cdk.New()
	if err := p.Scaffold(dir, project.Meta{Name: "my-app", Template: "node-ts"}); err != nil {
		t.Fatal(err)
	}
	stackPath := filepath.Join(dir, "lib", "stack.ts")
	customContent := []byte("// custom\n")
	if err := os.WriteFile(stackPath, customContent, 0644); err != nil {
		t.Fatal(err)
	}
	if err := p.Scaffold(dir, project.Meta{Name: "my-app", Template: "node-ts"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(stackPath)
	if string(got) != string(customContent) {
		t.Error("lib/stack.ts was overwritten on second Scaffold call; scaffold files must be written once only")
	}
}

// --- Integration (full project.Init pipeline) ---

func TestCDKPlugin_Integration_NodeTS_CDKJSONExists(t *testing.T) {
	dir := testutil.TempDir(t)
	if err := cdkProject(dir).Init(project.Meta{Name: "my-app", Template: "node-ts"}, node.NewTypeScript()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "cdk.json")); err != nil {
		t.Errorf("cdk.json not written: %v", err)
	}
}

func TestCDKPlugin_Integration_NodeTS_CDKJSONHasApp(t *testing.T) {
	dir := testutil.TempDir(t)
	if err := cdkProject(dir).Init(project.Meta{Name: "my-app", Template: "node-ts"}, node.NewTypeScript()); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "cdk.json"))
	if err != nil {
		t.Fatalf("cdk.json not written: %v", err)
	}
	var cdkJSON map[string]any
	json.Unmarshal(b, &cdkJSON)
	if cdkJSON["app"] != cdkAppEntry {
		t.Errorf("cdk.json app = %v, want %q", cdkJSON["app"], cdkAppEntry)
	}
}

func TestCDKPlugin_Integration_NodeTS_PackageJSONHasCDKDeps(t *testing.T) {
	dir := testutil.TempDir(t)
	if err := cdkProject(dir).Init(project.Meta{Name: "my-app", Template: "node-ts"}, node.NewTypeScript()); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	devDeps, _ := pkg["devDependencies"].(map[string]any)
	for dep, want := range map[string]string{
		"aws-cdk-lib": cdkLibVersion,
		"constructs":  constructsVersion,
		"aws-cdk":     cdkCLIVersion,
		"esbuild":     esbuildVersion,
	} {
		if devDeps[dep] != want {
			t.Errorf("devDependencies.%s = %v, want %q", dep, devDeps[dep], want)
		}
	}
}

func TestCDKPlugin_Integration_NodeTS_BinAppTSExists(t *testing.T) {
	dir := testutil.TempDir(t)
	if err := cdkProject(dir).Init(project.Meta{Name: "my-app", Template: "node-ts"}, node.NewTypeScript()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "bin", "app.ts")); err != nil {
		t.Errorf("bin/app.ts not created: %v", err)
	}
}

func TestCDKPlugin_Integration_NodeTS_LibStackTSExists(t *testing.T) {
	dir := testutil.TempDir(t)
	if err := cdkProject(dir).Init(project.Meta{Name: "my-app", Template: "node-ts"}, node.NewTypeScript()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "lib", "stack.ts")); err != nil {
		t.Errorf("lib/stack.ts not created: %v", err)
	}
}

// TestCDKPlugin_Integration_NodeTS_ScaffoldNotTouchedOnResynth verifies that
// re-running Synthesize never modifies scaffold files.
func TestCDKPlugin_Integration_NodeTS_ScaffoldNotTouchedOnResynth(t *testing.T) {
	dir := testutil.TempDir(t)
	proj := cdkProject(dir)
	s := node.NewTypeScript()
	if err := proj.Init(project.Meta{Name: "my-app", Template: "node-ts"}, s); err != nil {
		t.Fatal(err)
	}

	binPath := filepath.Join(dir, "bin", "app.ts")
	customContent := []byte("// custom\n")
	if err := os.WriteFile(binPath, customContent, 0644); err != nil {
		t.Fatal(err)
	}

	if err := proj.Synthesize(s); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(binPath)
	if string(got) != string(customContent) {
		t.Error("bin/app.ts was modified by Synthesize; scaffold files must be immutable after Init")
	}
}

func containsString(b []byte, s string) bool {
	return strings.Contains(string(b), s)
}
