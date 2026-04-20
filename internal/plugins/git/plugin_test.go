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

func TestGitPlugin_Go_CreatesGitignore(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "git: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".gitignore"))
	for _, want := range []string{"*.exe", "*.out", "*.test"} {
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
	for _, want := range []string{"*.exe", "*.out", "*.test"} {
		if !strings.Contains(content, want) {
			t.Errorf("expected %q in .gitignore:\n%s", want, content)
		}
	}
}

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
