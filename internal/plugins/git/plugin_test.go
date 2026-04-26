package git_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forgant-foundry/eventing"
	gitplugin "github.com/forgant-foundry/forglet/internal/plugins/git"
	"github.com/forgant-foundry/forglet/internal/project"
	"github.com/forgant-foundry/forglet/internal/testutil"
)

// noopSynth is a synthesizer that produces no events and writes no files.
// It lets plugin tests focus solely on what the plugin layer contributes.
type noopSynth struct{}

func (s *noopSynth) InitializeEvents(_ string) (map[string][]eventing.Event, error) {
	return nil, nil
}
func (s *noopSynth) OverlayEvents(_ map[string]any) (map[string][]eventing.Event, error) {
	return nil, nil
}
func (s *noopSynth) Synthesize(_ string, _ map[string]*eventing.Aggregate) error { return nil }

// ---- System categories -----------------------------------------------------------

func TestGitPlugin_SystemCategories_AlwaysPresent(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: true\n")

	if err := p.Init(project.Meta{Name: "my-app", Template: "node-ts"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".gitignore"))
	for _, want := range []string{".forglet/", ".DS_Store", ".idea/", ".vscode/", ".classpath"} {
		if !strings.Contains(content, want) {
			t.Errorf("expected system pattern %q in .gitignore:\n%s", want, content)
		}
	}
}

func TestGitPlugin_ExcludeCategory(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: true\ngitignore:\n  exclude:\n    - eclipse\n    - vscode\n")

	if err := p.Init(project.Meta{Name: "my-app", Template: "node-ts"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".gitignore"))
	if strings.Contains(content, ".classpath") {
		t.Error("eclipse category should be excluded")
	}
	if strings.Contains(content, ".vscode/") {
		t.Error("vscode category should be excluded")
	}
	// other system categories still present
	if !strings.Contains(content, ".DS_Store") {
		t.Error("macos category should still be present")
	}
	if !strings.Contains(content, ".idea/") {
		t.Error("jetbrains category should still be present")
	}
}

func TestGitPlugin_ForgletCategoryAlwaysPresent(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".gitignore"))
	if !strings.Contains(content, ".forglet/") {
		t.Errorf(".forglet/ not found in .gitignore:\n%s", content)
	}
}

// ---- Template-specific patterns -------------------------------------------------

func TestGitPlugin_NodeTs_CreatesGitignore(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: true\n")

	if err := p.Init(project.Meta{Name: "my-app", Template: "node-ts"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".gitignore"))
	for _, want := range []string{"node_modules/", "dist/", ".env"} {
		if !strings.Contains(content, want) {
			t.Errorf("expected %q in .gitignore:\n%s", want, content)
		}
	}
}

func TestGitPlugin_NodeJs_CreatesGitignore(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: true\n")

	if err := p.Init(project.Meta{Name: "my-app", Template: "node-js"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".gitignore"))
	for _, want := range []string{"node_modules/", ".env"} {
		if !strings.Contains(content, want) {
			t.Errorf("expected %q in .gitignore:\n%s", want, content)
		}
	}
	if strings.Contains(content, "dist/") {
		t.Error("node-js .gitignore should not contain dist/")
	}
}

func TestGitPlugin_NodeLambda_CreatesGitignore(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: true\n")

	if err := p.Init(project.Meta{Name: "my-handler", Template: "node-lambda"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".gitignore"))
	for _, want := range []string{"node_modules/", "dist/", ".env"} {
		if !strings.Contains(content, want) {
			t.Errorf("expected %q in .gitignore:\n%s", want, content)
		}
	}
}

func TestGitPlugin_Go_CreatesGitignore(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".gitignore"))
	for _, want := range []string{"*.exe", "*.out", "*.test", "coverage.out", ".forglet/"} {
		if !strings.Contains(content, want) {
			t.Errorf("expected %q in .gitignore:\n%s", want, content)
		}
	}
}

func TestGitPlugin_GoWorkspace_CreatesGitignore(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: true\n")

	if err := p.Init(project.Meta{Name: "myws", Template: "go-workspace"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".gitignore"))
	for _, want := range []string{"*.exe", "*.out", "*.test", "coverage.out"} {
		if !strings.Contains(content, want) {
			t.Errorf("expected %q in .gitignore:\n%s", want, content)
		}
	}
}

func TestGitPlugin_GoLambda_HasBootstrap(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: true\n")

	if err := p.Init(project.Meta{Name: "myhandler", Template: "go-lambda"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".gitignore"))
	for _, want := range []string{"*.exe", "*.test", "bootstrap"} {
		if !strings.Contains(content, want) {
			t.Errorf("expected %q in .gitignore:\n%s", want, content)
		}
	}
}

func TestGitPlugin_GoKnative_CreatesGitignore(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: true\n")

	if err := p.Init(project.Meta{Name: "myfunc", Template: "go-knative"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".gitignore"))
	for _, want := range []string{"*.exe", "*.test", "coverage.out"} {
		if !strings.Contains(content, want) {
			t.Errorf("expected %q in .gitignore:\n%s", want, content)
		}
	}
}

func TestGitPlugin_Java_CreatesGitignore(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "java"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".gitignore"))
	for _, want := range []string{"target/", "*.class", ".idea/"} {
		if !strings.Contains(content, want) {
			t.Errorf("expected %q in .gitignore:\n%s", want, content)
		}
	}
}

func TestGitPlugin_JavaMultimodule_CreatesGitignore(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: true\n")

	if err := p.Init(project.Meta{Name: "myparent", Template: "java-multimodule"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".gitignore"))
	for _, want := range []string{"target/", "*.class", ".idea/"} {
		if !strings.Contains(content, want) {
			t.Errorf("expected %q in .gitignore:\n%s", want, content)
		}
	}
}

func TestGitPlugin_JavaLambda_CreatesGitignore(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: true\n")

	if err := p.Init(project.Meta{Name: "myhandler", Template: "java-lambda"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".gitignore"))
	for _, want := range []string{"target/", "*.class"} {
		if !strings.Contains(content, want) {
			t.Errorf("expected %q in .gitignore:\n%s", want, content)
		}
	}
}

func TestGitPlugin_JavaSpring_CreatesGitignore(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: true\n")

	if err := p.Init(project.Meta{Name: "myservice", Template: "java-spring"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".gitignore"))
	for _, want := range []string{"target/", "*.class", ".idea/"} {
		if !strings.Contains(content, want) {
			t.Errorf("expected %q in .gitignore:\n%s", want, content)
		}
	}
}

// ---- Custom patterns -----------------------------------------------------------

func TestGitPlugin_AddPatterns_MapForm(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: true\ngitignore:\n  add:\n    - \".env.local\"\n    - \"secrets.json\"\n")

	if err := p.Init(project.Meta{Name: "my-app", Template: "node-ts"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".gitignore"))
	for _, want := range []string{"node_modules/", ".env.local", "secrets.json"} {
		if !strings.Contains(content, want) {
			t.Errorf("expected %q in .gitignore:\n%s", want, content)
		}
	}
}

func TestGitPlugin_CustomPatterns_FlatListBackwardCompat(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: true\ngitignore:\n  - \".env.local\"\n  - \"secrets.json\"\n")

	if err := p.Init(project.Meta{Name: "my-app", Template: "node-ts"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".gitignore"))
	for _, want := range []string{"node_modules/", ".env.local", "secrets.json"} {
		if !strings.Contains(content, want) {
			t.Errorf("expected %q in .gitignore:\n%s", want, content)
		}
	}
}

func TestGitPlugin_CustomPatterns_OnlyWhenGitEnabled(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "gitignore:\n  - \".env.local\"\n")

	if err := p.Init(project.Meta{Name: "my-app", Template: "node-ts"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".gitignore")); !os.IsNotExist(err) {
		t.Error("expected no .gitignore when git key absent")
	}
}

// ---- Disabled / absent ---------------------------------------------------------

func TestGitPlugin_NoGitKey_NoGitignore(t *testing.T) {
	dir, p := setup(t)

	if err := p.Init(project.Meta{Name: "my-app", Template: "node-ts"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".gitignore")); !os.IsNotExist(err) {
		t.Error("expected no .gitignore when git key absent")
	}
}

func TestGitPlugin_GitFalse_NoGitignore(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: false\n")

	if err := p.Init(project.Meta{Name: "my-app", Template: "node-ts"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".gitignore")); !os.IsNotExist(err) {
		t.Error("expected no .gitignore when git: false")
	}
}

func TestGitPlugin_UnknownTemplate_Noop(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: true\n")

	if err := p.Init(project.Meta{Name: "x", Template: "unknown"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".gitignore")); !os.IsNotExist(err) {
		t.Error("expected no .gitignore for unknown template")
	}
}

// ---- Managed file properties ---------------------------------------------------

func TestGitPlugin_GitignoreReadOnly(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: true\n")

	if err := p.Init(project.Meta{Name: "my-app", Template: "node-ts"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0444 {
		t.Errorf(".gitignore mode = %04o, want 0444", info.Mode().Perm())
	}
}

func TestGitPlugin_GitignoreHasMarker(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: true\n")

	if err := p.Init(project.Meta{Name: "my-app", Template: "node-ts"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".gitignore"))
	if !strings.HasPrefix(content, "# ") {
		t.Errorf(".gitignore does not start with '#' comment marker, got: %q", content)
	}
	if !strings.Contains(content, "forglet") {
		t.Error(".gitignore marker does not mention forglet")
	}
}

func TestGitPlugin_Idempotent(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: true\n")
	s := &noopSynth{}

	if err := p.Init(project.Meta{Name: "my-app", Template: "node-ts"}, s); err != nil {
		t.Fatal(err)
	}
	first := readFile(t, filepath.Join(dir, ".gitignore"))

	if err := p.Synthesize(s); err != nil {
		t.Fatal(err)
	}
	second := readFile(t, filepath.Join(dir, ".gitignore"))

	if first != second {
		t.Errorf(".gitignore changed after second synth:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

// ---- helpers ----------------------------------------------------------------------

func setup(t *testing.T) (string, *project.Project) {
	t.Helper()
	dir := testutil.TempDir(t)
	p := project.New(dir).WithPlugins(gitplugin.New())
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
