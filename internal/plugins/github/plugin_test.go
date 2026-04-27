package github_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forgant-foundry/eventing"
	githubplugin "github.com/forgant-foundry/forglet/internal/plugins/github"
	"github.com/forgant-foundry/forglet/internal/project"
	"github.com/forgant-foundry/forglet/internal/testutil"
)

type noopSynth struct{}

func (s *noopSynth) InitializeEvents(_ string) (map[string][]eventing.Event, error) {
	return nil, nil
}
func (s *noopSynth) OverlayEvents(_ map[string]any) (map[string][]eventing.Event, error) {
	return nil, nil
}
func (s *noopSynth) Synthesize(_ string, _ map[string]*eventing.Aggregate) error { return nil }

// ---- CI workflow ------------------------------------------------------------------

func TestGitHubActions_Go_CIWorkflowCreated(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".github", "workflows", "ci.yml")); err != nil {
		t.Errorf("ci.yml not created: %v", err)
	}
}

func TestGitHubActions_Go_CIWorkflowHasGoSetup(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "ci.yml"))
	assertContains(t, content, "setup-go")
	assertContains(t, content, "go test ./...")
}

func TestGitHubActions_GoLambda_CIWorkflowHasGoTest(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github: true\n")

	if err := p.Init(project.Meta{Name: "myhandler", Template: "go-lambda"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "ci.yml"))
	assertContains(t, content, "go test ./...")
}

func TestGitHubActions_Java_CIWorkflowHasMaven(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "java"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "ci.yml"))
	assertContains(t, content, "setup-java")
	assertContains(t, content, "mvn")
}

func TestGitHubActions_JavaSpring_CIWorkflowHasMaven(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github: true\n")

	if err := p.Init(project.Meta{Name: "myservice", Template: "java-spring"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "ci.yml"))
	assertContains(t, content, "mvn")
}

func TestGitHubActions_NodeTs_CIWorkflowHasNpmTest(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github: true\n")

	if err := p.Init(project.Meta{Name: "my-app", Template: "node-ts"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "ci.yml"))
	assertContains(t, content, "setup-node")
	assertContains(t, content, "npm ci")
	assertContains(t, content, "npm test")
}

func TestGitHubActions_NodeLambda_CIWorkflowHasBuild(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github: true\n")

	if err := p.Init(project.Meta{Name: "my-handler", Template: "node-lambda"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "ci.yml"))
	assertContains(t, content, "npm run build")
	if strings.Contains(content, "npm test") {
		t.Error("node-lambda CI should use 'npm run build', not 'npm test'")
	}
}

func TestGitHubActions_DefaultBranch_IsMain(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	assertContains(t, readFile(t, filepath.Join(dir, ".github", "workflows", "ci.yml")), "main")
}

func TestGitHubActions_CustomDefaultBranch(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  ci: true\n  defaultBranch: develop\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	assertContains(t, readFile(t, filepath.Join(dir, ".github", "workflows", "ci.yml")), "develop")
}

// ---- Release workflow -------------------------------------------------------------

func TestGitHubActions_Go_ReleaseWorkflowCreated(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  ci: true\n  release: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".github", "workflows", "release.yml")); err != nil {
		t.Errorf("release.yml not created: %v", err)
	}
}

func TestGitHubActions_Go_ReleaseWorkflowHasGoreleaser(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  ci: true\n  release: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "release.yml"))
	assertContains(t, content, "goreleaser")
	assertContains(t, content, "GITHUB_TOKEN")
}

func TestGitHubActions_Java_ReleaseWorkflowHasMavenPackage(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  release: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "java"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "release.yml"))
	assertContains(t, content, "mvn")
	assertContains(t, content, "package")
	assertContains(t, content, "action-gh-release")
}

func TestGitHubActions_Node_ReleaseWorkflowHasBuild(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  release: true\n")

	if err := p.Init(project.Meta{Name: "my-app", Template: "node-ts"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "release.yml"))
	assertContains(t, content, "npm run build")
	assertContains(t, content, "action-gh-release")
}

func TestGitHubActions_ReleaseOnly_NoCIWorkflow(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  release: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".github", "workflows", "ci.yml")); !os.IsNotExist(err) {
		t.Error("ci.yml should not be created when only release: true is set")
	}
}

// ---- General behaviour -----------------------------------------------------------

func TestGitHubActions_Disabled_NoWorkflows(t *testing.T) {
	dir, p := setup(t)

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".github")); !os.IsNotExist(err) {
		t.Error("expected no .github directory when github key absent")
	}
}

func TestGitHubActions_GitHubFalse_NoWorkflows(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github: false\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".github")); !os.IsNotExist(err) {
		t.Error("expected no .github directory when github: false")
	}
}

func TestGitHubActions_UnknownTemplate_NoWorkflows(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github: true\n")

	if err := p.Init(project.Meta{Name: "x", Template: "unknown"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".github")); !os.IsNotExist(err) {
		t.Error("expected no .github directory for unknown template")
	}
}

func TestGitHubActions_CIWorkflowReadOnly(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(dir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0444 {
		t.Errorf("ci.yml mode = %04o, want 0444", info.Mode().Perm())
	}
}

func TestGitHubActions_CIWorkflowHasMarker(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "ci.yml"))
	assertContains(t, content, "forglet")
}

func TestGitHubActions_Idempotent(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github: true\n")
	s := &noopSynth{}

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, s); err != nil {
		t.Fatal(err)
	}
	first := readFile(t, filepath.Join(dir, ".github", "workflows", "ci.yml"))

	if err := p.Synthesize(s); err != nil {
		t.Fatal(err)
	}
	second := readFile(t, filepath.Join(dir, ".github", "workflows", "ci.yml"))

	if first != second {
		t.Errorf("ci.yml changed after second synth:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

// ---- Delivery workflow -----------------------------------------------------------

func TestDelivery_Go_WorkflowCreated(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".github", "workflows", "delivery.yml")); err != nil {
		t.Errorf("delivery.yml not created: %v", err)
	}
}

func TestDelivery_Go_HasVergant(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "vergant")
}

func TestDelivery_Go_UsesBinaryName(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "mycli", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "mycli")
}

func TestDelivery_Go_HasCrossCompilation(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "linux")
	assertContains(t, content, "darwin")
	assertContains(t, content, "windows")
}

func TestDelivery_Go_TriggerOnAllBranches(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "**")
}

func TestDelivery_Node_WorkflowCreated(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "my-app", Template: "node-ts"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".github", "workflows", "delivery.yml")); err != nil {
		t.Errorf("delivery.yml not created: %v", err)
	}
}

func TestDelivery_Node_HasVergant(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "my-app", Template: "node-ts"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "vergant")
}

func TestDelivery_Node_HasNpmPublish(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "my-app", Template: "node-ts"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "npm publish")
	assertContains(t, content, "NPM_TOKEN")
}

func TestDelivery_Node_HasNpmVersion(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "my-app", Template: "node-ts"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "npm version")
}

func TestDelivery_Java_WorkflowCreated(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "myservice", Template: "java"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".github", "workflows", "delivery.yml")); err != nil {
		t.Errorf("delivery.yml not created: %v", err)
	}
}

func TestDelivery_Java_HasVergant(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "myservice", Template: "java"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "vergant")
}

func TestDelivery_Java_HasMavenBuild(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "myservice", Template: "java"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "mvn")
	assertContains(t, content, "package")
	assertContains(t, content, "versions:set")
}

func TestDelivery_DeliveryOnly_NoCIWorkflow(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".github", "workflows", "ci.yml")); !os.IsNotExist(err) {
		t.Error("ci.yml should not be created when only delivery: true is set")
	}
}

func TestDelivery_WorkflowReadOnly(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0444 {
		t.Errorf("delivery.yml mode = %04o, want 0444", info.Mode().Perm())
	}
}

func TestDelivery_WorkflowHasMarker(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "forglet")
}

func TestDelivery_GithubTrue_NoDeliveryWorkflow(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".github", "workflows", "delivery.yml")); !os.IsNotExist(err) {
		t.Error("delivery.yml should not be created by 'github: true' shorthand")
	}
}

// ---- Library kind ----------------------------------------------------------------

func TestDelivery_Go_Library_NoCrossCompilation(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    kind: library\n")

	if err := p.Init(project.Meta{Name: "mylib", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	if strings.Contains(content, "linux") {
		t.Error("library delivery should not contain 'linux'")
	}
	if strings.Contains(content, "GOOS") {
		t.Error("library delivery should not contain 'GOOS'")
	}
}

func TestDelivery_Go_Library_HasReleaseCreate(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    kind: library\n")

	if err := p.Init(project.Meta{Name: "mylib", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "gh release create")
	assertContains(t, content, "--generate-notes")
}

func TestDelivery_Go_Library_NoDistFiles(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    kind: library\n")

	if err := p.Init(project.Meta{Name: "mylib", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	if strings.Contains(content, "dist/") {
		t.Error("library delivery should not reference dist/")
	}
}

func TestDelivery_Node_Library_NoNpmPublish(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    kind: library\n")

	if err := p.Init(project.Meta{Name: "my-lib", Template: "node-ts"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	if strings.Contains(content, "npm publish") {
		t.Error("library delivery should not contain 'npm publish'")
	}
}

func TestDelivery_Node_Library_HasReleaseCreate(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    kind: library\n")

	if err := p.Init(project.Meta{Name: "my-lib", Template: "node-ts"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "gh release create")
}

func TestDelivery_Java_Library_NoMavenPackage(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    kind: library\n")

	if err := p.Init(project.Meta{Name: "mylib", Template: "java"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	if strings.Contains(content, "mvn") {
		t.Error("library delivery should not contain 'mvn'")
	}
}

func TestDelivery_Java_Library_HasReleaseCreate(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    kind: library\n")

	if err := p.Init(project.Meta{Name: "mylib", Template: "java"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "gh release create")
	assertContains(t, content, "--generate-notes")
}

func TestDelivery_BinaryTrue_DefaultsToKindBinary(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "mycli", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "linux")
	assertContains(t, content, "GOOS")
}

func TestDelivery_Idempotent(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery: true\n")
	s := &noopSynth{}

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, s); err != nil {
		t.Fatal(err)
	}
	first := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))

	if err := p.Synthesize(s); err != nil {
		t.Fatal(err)
	}
	second := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))

	if first != second {
		t.Errorf("delivery.yml changed after second synth:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

// ---- .vergant.yml ----------------------------------------------------------------

func TestVergant_AbsentWhenNoFieldsConfigured(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".vergant.yml")); !os.IsNotExist(err) {
		t.Error(".vergant.yml should not be created when no vergant fields are configured")
	}
}

func TestVergant_CreatedWhenFieldSet(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    majorVersion: 2\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".vergant.yml")); err != nil {
		t.Errorf(".vergant.yml not created: %v", err)
	}
}

func TestVergant_OnlyWritesSetFields(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    majorVersion: 2\n    mode: candidate\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".vergant.yml"))
	assertContains(t, content, "majorVersion: 2")
	assertContains(t, content, "mode: candidate")
	if strings.Contains(content, "defaultBranch") {
		t.Error(".vergant.yml should not contain defaultBranch when not set")
	}
	if strings.Contains(content, "BranchRegEx") {
		t.Error(".vergant.yml should not contain branch regexes when not set")
	}
}

func TestVergant_MajorVersionOverride(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    majorVersion: 3\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	assertContains(t, readFile(t, filepath.Join(dir, ".vergant.yml")), "majorVersion: 3")
}

func TestVergant_DefaultBranchOverride(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    defaultBranch: develop\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	assertContains(t, readFile(t, filepath.Join(dir, ".vergant.yml")), "defaultBranch: develop")
}

func TestVergant_ModeCandidate(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    mode: candidate\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	assertContains(t, readFile(t, filepath.Join(dir, ".vergant.yml")), "mode: candidate")
}

func TestVergant_CustomBranchRegex(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    supportBranchRegEx: \"^release/.*\"\n    devBranchRegEx: \"^feature/(.+)$\"\n    patchBranchRegEx: \"^hotfix/(.+)$\"\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".vergant.yml"))
	assertContains(t, content, "^release/")
	assertContains(t, content, "^feature/")
	assertContains(t, content, "^hotfix/")
}

func TestVergant_IsReadOnly(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    majorVersion: 1\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(dir, ".vergant.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0444 {
		t.Errorf(".vergant.yml mode = %04o, want 0444", info.Mode().Perm())
	}
}

func TestVergant_HasManagedMarker(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    majorVersion: 1\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	assertContains(t, readFile(t, filepath.Join(dir, ".vergant.yml")), "forglet")
}

func TestVergant_AbsentWithoutDelivery(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".vergant.yml")); !os.IsNotExist(err) {
		t.Error(".vergant.yml should not be created when delivery is not enabled")
	}
}

func TestVergant_Idempotent(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    majorVersion: 2\n    mode: candidate\n")
	s := &noopSynth{}

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, s); err != nil {
		t.Fatal(err)
	}
	first := readFile(t, filepath.Join(dir, ".vergant.yml"))

	if err := p.Synthesize(s); err != nil {
		t.Fatal(err)
	}
	second := readFile(t, filepath.Join(dir, ".vergant.yml"))

	if first != second {
		t.Errorf(".vergant.yml changed after second synth:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

// ---- helpers ---------------------------------------------------------------------

func setup(t *testing.T) (string, *project.Project) {
	t.Helper()
	dir := testutil.TempDir(t)
	p := project.New(dir).WithPlugins(githubplugin.New())
	return dir, p
}

func writeRC(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".forglet.yml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

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
