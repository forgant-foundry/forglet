package project_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
	"github.com/forgant-foundry/forglet/internal/testutil"
)

// stubSynth is a minimal Synthesizer for testing the project layer in isolation.
type stubSynth struct {
	template map[string][]eventing.Event
	overlay  map[string][]eventing.Event
}

func (s *stubSynth) InitializeEvents(_ string) (map[string][]eventing.Event, error) {
	return s.template, nil
}

func (s *stubSynth) OverlayEvents(_ map[string]any) (map[string][]eventing.Event, error) {
	return s.overlay, nil
}

func (s *stubSynth) Synthesize(dir string, aggregates map[string]*eventing.Aggregate) error {
	for filename, agg := range aggregates {
		b, err := agg.ToJSON()
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, filename), b, 0644); err != nil {
			return err
		}
	}
	return nil
}

func evt(t *testing.T, id, typ string, seq int64, payload any) eventing.Event {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return eventing.Event{ID: id, Type: typ, Seq: seq, Payload: json.RawMessage(b)}
}

func TestInit_CreatesManagedFiles(t *testing.T) {
	dir := testutil.TempDir(t)
	s := &stubSynth{
		template: map[string][]eventing.Event{
			"package.json": {evt(t, "e1", "init", 1, map[string]any{"name": "test-app", "version": "0.1.0"})},
		},
	}

	p := project.New(dir)
	if err := p.Init(project.Meta{Name: "test-app", Template: "node-ts"}, s); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatalf("package.json not written: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got["name"] != "test-app" {
		t.Errorf("name = %q, want %q", got["name"], "test-app")
	}

	if _, err := os.Stat(filepath.Join(dir, ".forglet", "package.json.json")); err != nil {
		t.Errorf(".forglet/package.json.json not created: %v", err)
	}

	meta, err := p.LoadMeta()
	if err != nil {
		t.Fatal(err)
	}
	if meta.Name != "test-app" || meta.Template != "node-ts" {
		t.Errorf("meta = %+v, want {Name:test-app Template:node-ts}", meta)
	}
}

func TestSynthesize_Idempotent(t *testing.T) {
	dir := testutil.TempDir(t)
	s := &stubSynth{
		template: map[string][]eventing.Event{
			"package.json": {evt(t, "e1", "init", 1, map[string]any{"name": "test-app"})},
		},
	}

	p := project.New(dir)
	if err := p.Init(project.Meta{Name: "test-app", Template: "node-ts"}, s); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(filepath.Join(dir, "package.json"))

	if err := p.Synthesize(s); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(filepath.Join(dir, "package.json"))

	if string(first) != string(second) {
		t.Error("synth is not idempotent: output changed between runs")
	}
}

func TestSynthesize_OverlayWinsOverTemplate(t *testing.T) {
	dir := testutil.TempDir(t)
	s := &stubSynth{
		template: map[string][]eventing.Event{
			"package.json": {evt(t, "e1", "init", 1, map[string]any{"version": "0.1.0"})},
		},
		overlay: map[string][]eventing.Event{
			"package.json": {evt(t, "e2", "version.override", 1000, map[string]any{"version": "1.2.3"})},
		},
	}

	p := project.New(dir)
	if err := p.Init(project.Meta{Name: "test-app", Template: "node-ts"}, s); err != nil {
		t.Fatal(err)
	}

	b, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	var got map[string]any
	json.Unmarshal(b, &got)

	if got["version"] != "1.2.3" {
		t.Errorf("version = %q, want overlay value %q", got["version"], "1.2.3")
	}
}

func TestBuildAggregates_Provenance(t *testing.T) {
	templateEvents := map[string][]eventing.Event{
		"package.json": {
			evt(t, "e1", "init", 1, map[string]any{"name": "test-app", "version": "0.1.0"}),
		},
	}
	overlayEvents := map[string][]eventing.Event{
		"package.json": {
			evt(t, "e2", "devDependency.added", 1000, map[string]any{"devDependencies": map[string]any{"prettier": "^3.0.0"}}),
		},
	}

	aggregates := project.BuildAggregates(templateEvents, overlayEvents)
	agg, ok := aggregates["package.json"]
	if !ok {
		t.Fatal("package.json aggregate not built")
	}

	cases := []struct {
		field     string
		wantEvent string
	}{
		{"name", "init"},
		{"version", "init"},
		{"devDependencies", "devDependency.added"},
	}
	for _, c := range cases {
		node, ok := agg.Node(c.field)
		if !ok {
			t.Errorf("node %q not found", c.field)
			continue
		}
		if node.EventType != c.wantEvent {
			t.Errorf("node %q: EventType = %q, want %q", c.field, node.EventType, c.wantEvent)
		}
	}
}

func TestSynthesize_ReadsRC(t *testing.T) {
	dir := testutil.TempDir(t)

	rc := []byte("devDependencies:\n  prettier: \"^3.0.0\"\n")
	if err := os.WriteFile(filepath.Join(dir, ".forglet.yml"), rc, 0644); err != nil {
		t.Fatal(err)
	}

	var gotRC map[string]any
	s := &stubSynth{
		template: map[string][]eventing.Event{
			"package.json": {evt(t, "e1", "init", 1, map[string]any{"name": "test-app"})},
		},
	}
	// wrap OverlayEvents to capture what the project layer passes
	capture := &rcCaptureSynth{Synthesizer: s, captured: &gotRC}

	p := project.New(dir)
	if err := p.Init(project.Meta{Name: "test-app", Template: "node-ts"}, capture); err != nil {
		t.Fatal(err)
	}

	if gotRC == nil {
		t.Fatal("rc was not passed to OverlayEvents")
	}
	devDeps, ok := gotRC["devDependencies"]
	if !ok {
		t.Error("devDependencies not present in rc passed to OverlayEvents")
	}
	m, _ := devDeps.(map[string]any)
	if m["prettier"] != "^3.0.0" {
		t.Errorf("devDependencies.prettier = %v, want %q", m["prettier"], "^3.0.0")
	}
}

// rcCaptureSynth wraps a Synthesizer and captures the rc map passed to OverlayEvents.
type rcCaptureSynth struct {
	project.Synthesizer
	captured *map[string]any
}

func (c *rcCaptureSynth) OverlayEvents(rc map[string]any) (map[string][]eventing.Event, error) {
	*c.captured = rc
	return c.Synthesizer.OverlayEvents(rc)
}
