package fly_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forgant-foundry/eventing"
	flyplugin "github.com/forgant-foundry/forglet/internal/plugins/fly"
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

// ---- fly.toml --------------------------------------------------------------------

func TestFly_FlyTomlCreated(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "fly: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "fly.toml")); err != nil {
		t.Errorf("fly.toml not created: %v", err)
	}
}

func TestFly_FlyTomlContainsAppName(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "fly: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "fly.toml"))
	assertContains(t, content, "myapp")
}

func TestFly_FlyTomlDefaults(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "fly: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "fly.toml"))
	assertContains(t, content, "iad")
	assertContains(t, content, "8080")
	assertContains(t, content, "256mb")
	assertContains(t, content, "/healthz")
}

func TestFly_FlyTomlRegionOverride(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "fly:\n  region: lax\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "fly.toml"))
	assertContains(t, content, "lax")
}

func TestFly_FlyTomlHealthPathOverride(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "fly:\n  healthPath: /health\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "fly.toml"))
	assertContains(t, content, "/health")
}

func TestFly_FlyTomlIdempotent(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "fly: true\n")
	s := &noopSynth{}

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, s); err != nil {
		t.Fatal(err)
	}
	first := readFile(t, filepath.Join(dir, "fly.toml"))

	if err := p.Synthesize(s); err != nil {
		t.Fatal(err)
	}
	second := readFile(t, filepath.Join(dir, "fly.toml"))

	if first != second {
		t.Errorf("fly.toml changed after second synth:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

// ---- deploy.yml ------------------------------------------------------------------

func TestFly_DeployWorkflowCreated(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "fly: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".github", "workflows", "deploy.yml")); err != nil {
		t.Errorf("deploy.yml not created: %v", err)
	}
}

func TestFly_DeployWorkflowHasFlyctlStep(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "fly: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "deploy.yml"))
	assertContains(t, content, "flyctl-actions")
	assertContains(t, content, "FLY_API_TOKEN")
	assertContains(t, content, "flyctl deploy --remote-only")
}

func TestFly_DeployWorkflowDefaultBranch(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "fly: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "deploy.yml"))
	assertContains(t, content, "main")
}

func TestFly_DeployWorkflowBranchOverride(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "fly:\n  branch: production\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, ".github", "workflows", "deploy.yml"))
	assertContains(t, content, "production")
}

// ---- Dockerfile ------------------------------------------------------------------

func TestFly_DockerfileScaffolded(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "fly: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "Dockerfile")); err != nil {
		t.Errorf("Dockerfile not scaffolded: %v", err)
	}
}

func TestFly_DockerfileNotOverwrittenOnResynth(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "fly: true\n")
	s := &noopSynth{}

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, s); err != nil {
		t.Fatal(err)
	}

	custom := "# custom dockerfile\n"
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(custom), 0644); err != nil {
		t.Fatal(err)
	}

	if err := p.Synthesize(s); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "Dockerfile"))
	if content != custom {
		t.Errorf("Dockerfile was overwritten on re-synth; got:\n%s", content)
	}
}

func TestFly_DockerfileMainPackage(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "fly:\n  main: ./cmd/server\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "Dockerfile"))
	assertContains(t, content, "./cmd/server")
}

func TestFly_DockerfileJava(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "fly: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "java"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "Dockerfile"))
	assertContains(t, content, "maven")
	assertContains(t, content, "eclipse-temurin")
	assertContains(t, content, "mvn")
}

func TestFly_DockerfileJavaSpring(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "fly: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "java-spring"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "Dockerfile"))
	assertContains(t, content, "maven")
}

func TestFly_DockerfileNodeTS(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "fly: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "node-ts"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "Dockerfile"))
	assertContains(t, content, "npm run build")
	assertContains(t, content, "dist/index.js")
}

func TestFly_DockerfileNodeJS(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "fly: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "node-js"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "Dockerfile"))
	assertNotContains(t, content, "npm run build")
	assertContains(t, content, "src/index.js")
}

func TestFly_DockerfileGoIsDefault(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "fly: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "Dockerfile"))
	assertContains(t, content, "golang:")
	assertContains(t, content, "go build")
}

// ---- disabled --------------------------------------------------------------------

func TestFly_NotCreatedWhenDisabled(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "github: true\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "fly.toml")); !os.IsNotExist(err) {
		t.Error("fly.toml should not be created when fly is not configured")
	}
	if _, err := os.Stat(filepath.Join(dir, ".github", "workflows", "deploy.yml")); !os.IsNotExist(err) {
		t.Error("deploy.yml should not be created when fly is not configured")
	}
	if _, err := os.Stat(filepath.Join(dir, "Dockerfile")); !os.IsNotExist(err) {
		t.Error("Dockerfile should not be scaffolded when fly is not configured")
	}
}

// ---- helpers --------------------------------------------------------------------

func setup(t *testing.T) (string, *project.Project) {
	t.Helper()
	dir := testutil.TempDir(t)
	p := project.New(dir).WithPlugins(flyplugin.New())
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

func assertNotContains(t *testing.T, content, want string) {
	t.Helper()
	if strings.Contains(content, want) {
		t.Errorf("expected %q NOT to appear in:\n%s", want, content)
	}
}
