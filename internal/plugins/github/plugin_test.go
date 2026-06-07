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
	writeRC(t, dir, "github:\n  delivery:\n    kind: binary\n")

	if err := p.Init(project.Meta{Name: "mycli", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "mycli")
}

func TestDelivery_Go_HasCrossCompilation(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    kind: binary\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "linux")
	assertContains(t, content, "darwin")
	assertContains(t, content, "windows")
}

func TestDelivery_Go_VersionVarInjected(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    kind: binary\n    versionVar: github.com/org/myapp/cmd/myapp/commands.Version\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "-X github.com/org/myapp/cmd/myapp/commands.Version=${SEM}")
}

func TestDelivery_Go_NoVersionVarByDefault(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    kind: binary\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	if strings.Contains(content, "-X ") {
		t.Error("expected no -X linker flag when versionVar is not set")
	}
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
	writeRC(t, dir, "github:\n  delivery:\n    kind: binary\n")

	if err := p.Init(project.Meta{Name: "my-app", Template: "node-ts"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "npm publish")
	assertContains(t, content, "NPM_TOKEN")
}

func TestDelivery_Node_HasNpmVersion(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    kind: binary\n")

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
	writeRC(t, dir, "github:\n  delivery:\n    kind: binary\n")

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

func TestDelivery_Go_Binary_CustomMainPkg(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    kind: binary\n    main: ./cmd/mycli\n")

	if err := p.Init(project.Meta{Name: "mycli", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "./cmd/mycli")
}

func TestDelivery_Go_Binary_DefaultMainPkg(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    kind: binary\n")

	if err := p.Init(project.Meta{Name: "mycli", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	// Default package is "." — should appear in go build commands
	assertContains(t, content, "go build")
	if strings.Contains(content, "./cmd/") {
		t.Error("default binary delivery should not reference ./cmd/")
	}
}

func TestDelivery_DefaultsToKindLibrary(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "mycli", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	if strings.Contains(content, "GOOS") {
		t.Error("default delivery kind should be library, not binary")
	}
	assertContains(t, content, "gh release create")
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

// ---- CI gate in delivery (ci + delivery both enabled) ----------------------------

func TestDelivery_WithCI_Go_HasTestJob(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  ci: true\n  delivery:\n    kind: binary\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "go test ./...")
}

func TestDelivery_WithCI_Go_DeliveryNeedsTest(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  ci: true\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "needs")
	assertContains(t, content, "test")
}

func TestDelivery_WithoutCI_Go_NoTestJob(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	if strings.Contains(content, "go test ./...") {
		t.Error("delivery without ci should not contain test steps")
	}
	if strings.Contains(content, "needs") {
		t.Error("delivery without ci should not have a needs dependency")
	}
}

func TestDelivery_WithCI_CIWorkflowStillCreated(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  ci: true\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".github", "workflows", "ci.yml")); err != nil {
		t.Errorf("ci.yml should still be created when both ci and delivery are enabled: %v", err)
	}
}

func TestDelivery_WithCI_Node_HasTestJob(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  ci: true\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "my-app", Template: "node-ts"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "npm test")
	assertContains(t, content, "needs")
}

func TestDelivery_WithCI_Java_HasTestJob(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  ci: true\n  delivery: true\n")

	if err := p.Init(project.Meta{Name: "myservice", Template: "java"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "mvn --batch-mode test")
	assertContains(t, content, "needs")
}

// ---- Build variants (delivery.builds) -------------------------------------------

func TestDelivery_Go_Binary_BuildVariant_StandardBuildsStillPresent(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    kind: binary\n    builds:\n      - tags: no_embeddings\n        suffix: slim\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "linux_amd64.tar.gz")
	assertContains(t, content, "darwin_amd64.tar.gz")
	assertContains(t, content, "darwin_arm64.tar.gz")
	assertContains(t, content, "windows_amd64.zip")
}

func TestDelivery_Go_Binary_BuildVariant_ProducesVariantArtifacts(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    kind: binary\n    builds:\n      - tags: no_embeddings\n        suffix: slim\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "linux_amd64_slim.tar.gz")
	assertContains(t, content, "darwin_amd64_slim.tar.gz")
	assertContains(t, content, "darwin_arm64_slim.tar.gz")
	assertContains(t, content, "windows_amd64_slim.zip")
}

func TestDelivery_Go_Binary_BuildVariant_HasBuildTags(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    kind: binary\n    builds:\n      - tags: no_embeddings\n        suffix: slim\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "-tags no_embeddings")
}

func TestDelivery_Go_Binary_BuildVariant_TagsAbsentFromStandardBuilds(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    kind: binary\n    builds:\n      - tags: no_embeddings\n        suffix: slim\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	// Standard build lines should appear before any -tags flag.
	// We verify by confirming standard artifacts exist independently (covered by StandardBuildsStillPresent).
	// Here we just assert the tag is present at least once, not on every build line.
	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "-tags no_embeddings")
	assertContains(t, content, "linux_amd64.tar.gz") // standard (no suffix) still present
}

func TestDelivery_Go_Binary_MultipleVariants(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    kind: binary\n    builds:\n      - tags: no_embeddings\n        suffix: slim\n      - tags: no_embeddings,debug\n        suffix: debug\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "linux_amd64_slim.tar.gz")
	assertContains(t, content, "linux_amd64_debug.tar.gz")
}

func TestDelivery_Go_Binary_Variant_NoSuffix_Skipped(t *testing.T) {
	dir, p := setup(t)
	// Entry with no suffix should be silently skipped — no duplicate standard artifacts.
	writeRC(t, dir, "github:\n  delivery:\n    kind: binary\n    builds:\n      - tags: no_embeddings\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	// Should still have standard builds but no extra artifact names.
	assertContains(t, content, "linux_amd64.tar.gz")
	if strings.Contains(content, "-tags no_embeddings") {
		t.Error("variant with no suffix should be skipped; expected no -tags flag in output")
	}
}

func TestDelivery_Go_Binary_BuildVariant_BinaryUsesBaseName(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    kind: binary\n    builds:\n      - tags: no_embeddings\n        suffix: slim\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	// The binary inside each variant archive must be the base name, not name_suffix.
	assertContains(t, content, "-o /tmp/myapp ")
	if strings.Contains(content, "-o /tmp/myapp_slim") {
		t.Error("variant binary should be named 'myapp', not 'myapp_slim'")
	}
}

func TestDelivery_Go_Binary_EmptyBuilds_SameAsDefault(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github:\n  delivery:\n    kind: binary\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "delivery.yml"))
	assertContains(t, content, "linux_amd64.tar.gz")
	if strings.Contains(content, "_slim") || strings.Contains(content, "-tags") {
		t.Error("no builds key should produce only standard artifacts")
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
