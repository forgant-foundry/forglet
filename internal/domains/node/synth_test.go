package node_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/domains/node"
	"github.com/forgant-foundry/forglet/internal/project"
	"github.com/forgant-foundry/forglet/internal/testutil"
)

func buildAgg(t *testing.T, events []eventing.Event) *eventing.Aggregate {
	t.Helper()
	agg := eventing.NewAggregate()
	for i := range events {
		if err := agg.Apply(&events[i]); err != nil {
			t.Fatal(err)
		}
	}
	return agg
}

// TestTypeScript_InitializeEvents_TargetFiles checks the right files are produced.
func TestTypeScript_InitializeEvents_TargetFiles(t *testing.T) {
	s := node.NewTypeScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"package.json", "tsconfig.json"} {
		if _, ok := files[want]; !ok {
			t.Errorf("missing events for %s", want)
		}
	}
}

// TestTypeScript_InitializeEvents_PackageEventTypes checks all expected event types are present.
func TestTypeScript_InitializeEvents_PackageEventTypes(t *testing.T) {
	s := node.NewTypeScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}

	seen := make(map[string]bool)
	for _, e := range files["package.json"] {
		seen[e.Type] = true
	}
	for _, want := range []string{"init", "scripts.added", "devDependency.added"} {
		if !seen[want] {
			t.Errorf("package.json missing event type %q", want)
		}
	}
}

// TestTypeScript_InitializeEvents_NamePropagates checks the project name reaches the init event.
func TestTypeScript_InitializeEvents_NamePropagates(t *testing.T) {
	s := node.NewTypeScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range files["package.json"] {
		if e.Type != "init" {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(e.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload["name"] != "my-app" {
			t.Errorf("init payload name = %q, want %q", payload["name"], "my-app")
		}
	}
}

// TestTypeScript_InitializeEvents_SeqMonotonic checks seq numbers increase within each file.
func TestTypeScript_InitializeEvents_SeqMonotonic(t *testing.T) {
	s := node.NewTypeScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	for filename, events := range files {
		for i := 1; i < len(events); i++ {
			if events[i].Seq <= events[i-1].Seq {
				t.Errorf("%s: event[%d].Seq (%d) not greater than event[%d].Seq (%d)",
					filename, i, events[i].Seq, i-1, events[i-1].Seq)
			}
		}
	}
}

// TestTypeScript_OverlayEvents_Empty checks that an empty rc produces no events.
func TestTypeScript_OverlayEvents_Empty(t *testing.T) {
	s := node.NewTypeScript()
	events, err := s.OverlayEvents(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Errorf("expected no overlay events for empty rc, got %d files", len(events))
	}
}

// TestTypeScript_OverlayEvents_DevDependencies checks devDependencies are routed to package.json.
func TestTypeScript_OverlayEvents_DevDependencies(t *testing.T) {
	s := node.NewTypeScript()
	rc := map[string]any{
		"devDependencies": map[string]any{"prettier": "^3.0.0"},
	}
	files, err := s.OverlayEvents(rc)
	if err != nil {
		t.Fatal(err)
	}
	events, ok := files["package.json"]
	if !ok {
		t.Fatal("expected overlay events for package.json")
	}

	found := false
	for _, e := range events {
		if e.Type != "devDependency.added" {
			continue
		}
		var payload map[string]any
		json.Unmarshal(e.Payload, &payload)
		devDeps, _ := payload["devDependencies"].(map[string]any)
		if devDeps["prettier"] == "^3.0.0" {
			found = true
		}
	}
	if !found {
		t.Error("prettier not found in devDependency.added overlay event")
	}
}

// TestTypeScript_OverlayEvents_SeqHigherThanTemplate checks overlay seq numbers
// are always higher than any template event seq number.
func TestTypeScript_OverlayEvents_SeqHigherThanTemplate(t *testing.T) {
	s := node.NewTypeScript()

	templateFiles, _ := s.InitializeEvents("my-app")
	overlayFiles, _ := s.OverlayEvents(map[string]any{
		"devDependencies": map[string]any{"prettier": "^3.0.0"},
		"scripts":         map[string]any{"lint": "eslint src/"},
	})

	maxTemplateSeq := int64(0)
	for _, events := range templateFiles {
		for _, e := range events {
			if e.Seq > maxTemplateSeq {
				maxTemplateSeq = e.Seq
			}
		}
	}
	for filename, events := range overlayFiles {
		for _, e := range events {
			if e.Seq <= maxTemplateSeq {
				t.Errorf("%s overlay event %q: seq %d not greater than max template seq %d",
					filename, e.Type, e.Seq, maxTemplateSeq)
			}
		}
	}
}

// TestTypeScript_Synthesize_WritesFiles checks all expected files are created.
func TestTypeScript_Synthesize_WritesFiles(t *testing.T) {
	dir := testutil.TempDir(t)
	s := node.NewTypeScript()

	fileEvents, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	aggregates := project.BuildAggregates(fileEvents, nil)

	if err := s.Synthesize(dir, aggregates); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"package.json", "tsconfig.json", filepath.Join("src", "index.ts")} {
		if _, err := os.Stat(filepath.Join(dir, want)); err != nil {
			t.Errorf("%s not created: %v", want, err)
		}
	}
}

// TestTypeScript_Synthesize_PackageJSON checks package.json content is correct.
func TestTypeScript_Synthesize_PackageJSON(t *testing.T) {
	dir := testutil.TempDir(t)
	s := node.NewTypeScript()

	fileEvents, _ := s.InitializeEvents("my-app")
	aggregates := project.BuildAggregates(fileEvents, nil)

	if err := s.Synthesize(dir, aggregates); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pkg map[string]any
	if err := json.Unmarshal(b, &pkg); err != nil {
		t.Fatalf("package.json is not valid JSON: %v", err)
	}
	for key, want := range map[string]any{"name": "my-app", "version": "0.1.0"} {
		if pkg[key] != want {
			t.Errorf("package.json[%q] = %v, want %v", key, pkg[key], want)
		}
	}
	if pkg["scripts"] == nil {
		t.Error("package.json missing scripts")
	}
	if pkg["devDependencies"] == nil {
		t.Error("package.json missing devDependencies")
	}
}

// TestTypeScript_Synthesize_OverlayMergesIntoPackageJSON checks that overlay
// events (e.g. from .forglet.yml) are reflected in the output file.
func TestTypeScript_Synthesize_OverlayMergesIntoPackageJSON(t *testing.T) {
	dir := testutil.TempDir(t)
	s := node.NewTypeScript()

	templateEvents, _ := s.InitializeEvents("my-app")
	overlayEvents, _ := s.OverlayEvents(map[string]any{
		"devDependencies": map[string]any{"prettier": "^3.0.0"},
		"scripts":         map[string]any{"lint": "eslint src/"},
	})
	aggregates := project.BuildAggregates(templateEvents, overlayEvents)

	if err := s.Synthesize(dir, aggregates); err != nil {
		t.Fatal(err)
	}

	b, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	var pkg map[string]any
	json.Unmarshal(b, &pkg)

	devDeps, _ := pkg["devDependencies"].(map[string]any)
	if devDeps["prettier"] == nil {
		t.Error("prettier not in devDependencies")
	}
	if devDeps["typescript"] == nil {
		t.Error("typescript (from template) missing after overlay merge")
	}

	scripts, _ := pkg["scripts"].(map[string]any)
	if scripts["lint"] == nil {
		t.Error("lint script not in scripts")
	}
	if scripts["build"] == nil {
		t.Error("build script (from template) missing after overlay merge")
	}
}

// TestTypeScript_Synthesize_FilesReadOnly checks that managed files are written read-only.
func TestTypeScript_Synthesize_FilesReadOnly(t *testing.T) {
	dir := testutil.TempDir(t)
	s := node.NewTypeScript()
	fileEvents, _ := s.InitializeEvents("my-app")
	aggregates := project.BuildAggregates(fileEvents, nil)
	if err := s.Synthesize(dir, aggregates); err != nil {
		t.Fatal(err)
	}
	for _, filename := range []string{"package.json", "tsconfig.json"} {
		info, err := os.Stat(filepath.Join(dir, filename))
		if err != nil {
			t.Fatalf("%s not found: %v", filename, err)
		}
		if info.Mode().Perm() != 0444 {
			t.Errorf("%s mode = %04o, want 0444", filename, info.Mode().Perm())
		}
	}
}

// TestTypeScript_Synthesize_PackageJSONHasMarker checks the managed-file comment is present.
func TestTypeScript_Synthesize_PackageJSONHasMarker(t *testing.T) {
	dir := testutil.TempDir(t)
	s := node.NewTypeScript()
	fileEvents, _ := s.InitializeEvents("my-app")
	aggregates := project.BuildAggregates(fileEvents, nil)
	if err := s.Synthesize(dir, aggregates); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	var pkg map[string]any
	if err := json.Unmarshal(b, &pkg); err != nil {
		t.Fatalf("package.json is not valid JSON after marker: %v", err)
	}
	if _, ok := pkg["//"]; !ok {
		t.Error(`package.json missing "//" marker key`)
	}
}

// TestTypeScript_Synthesize_IndexTsNotOverwritten checks the scaffold file is preserved.
func TestTypeScript_Synthesize_IndexTsNotOverwritten(t *testing.T) {
	dir := testutil.TempDir(t)
	s := node.NewTypeScript()

	if err := os.MkdirAll(filepath.Join(dir, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	custom := []byte("console.log('custom');\n")
	if err := os.WriteFile(filepath.Join(dir, "src", "index.ts"), custom, 0644); err != nil {
		t.Fatal(err)
	}

	fileEvents, _ := s.InitializeEvents("my-app")
	aggregates := project.BuildAggregates(fileEvents, nil)
	if err := s.Synthesize(dir, aggregates); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "src", "index.ts"))
	if string(got) != string(custom) {
		t.Error("src/index.ts was overwritten by synth")
	}
}
