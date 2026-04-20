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

// stubPlugin is a Plugin that appends a fixed set of events via Weave.
type stubPlugin struct {
	events map[string][]eventing.Event
}

func (p *stubPlugin) Weave(_ string, stream *project.EventStream) error {
	for file, events := range p.events {
		stream.Append(file, events...)
	}
	return nil
}

// insertingPlugin is a Plugin that uses InsertAfter to weave at a specific position.
type insertingPlugin struct {
	file      string
	after     string
	newEvents []eventing.Event
}

func (p *insertingPlugin) Weave(_ string, stream *project.EventStream) error {
	stream.InsertAfter(p.file, p.after, p.newEvents...)
	return nil
}

// replacingPlugin is a Plugin that replaces events of a given type.
type replacingPlugin struct {
	file        string
	replaceType string
	newEvents   []eventing.Event
}

func (p *replacingPlugin) Weave(_ string, stream *project.EventStream) error {
	stream.Replace(p.file, p.replaceType, p.newEvents...)
	return nil
}

func TestWithPlugins_Append_EventsAppliedDuringSynth(t *testing.T) {
	dir := testutil.TempDir(t)
	s := &stubSynth{
		template: map[string][]eventing.Event{
			"package.json": {evt(t, "e1", "init", 1, map[string]any{"name": "app"})},
		},
	}
	plugin := &stubPlugin{events: map[string][]eventing.Event{
		"package.json": {evt(t, "e2", "plugin.added", 1, map[string]any{
			"devDependencies": map[string]any{"eslint": "^8.0.0"},
		})},
	}}

	p := project.New(dir).WithPlugins(plugin)
	if err := p.Init(project.Meta{Name: "app", Template: "node-ts"}, s); err != nil {
		t.Fatal(err)
	}

	b, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	var got map[string]any
	json.Unmarshal(b, &got)

	devDeps, _ := got["devDependencies"].(map[string]any)
	if devDeps["eslint"] == nil {
		t.Error("eslint (from plugin) not in output package.json")
	}
}

func TestWithPlugins_InsertAfter_PositionReflectedInAggregate(t *testing.T) {
	dir := testutil.TempDir(t)
	s := &stubSynth{
		template: map[string][]eventing.Event{
			"package.json": {
				evt(t, "e1", "init", 1, map[string]any{"name": "app"}),
				evt(t, "e2", "dep.added", 2, map[string]any{"version": "0.1.0"}),
			},
		},
	}
	plugin := &insertingPlugin{
		file:      "package.json",
		after:     "init",
		newEvents: []eventing.Event{evt(t, "eX", "plugin.added", 99, map[string]any{"pluginKey": "val"})},
	}

	p := project.New(dir).WithPlugins(plugin)
	if err := p.Init(project.Meta{Name: "app", Template: "node-ts"}, s); err != nil {
		t.Fatal(err)
	}

	b, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	var got map[string]any
	json.Unmarshal(b, &got)
	if got["pluginKey"] != "val" {
		t.Error("plugin-inserted event not reflected in output")
	}
}

func TestWithPlugins_Replace_OverridesTemplateEvent(t *testing.T) {
	dir := testutil.TempDir(t)
	s := &stubSynth{
		template: map[string][]eventing.Event{
			"package.json": {
				evt(t, "e1", "init", 1, map[string]any{"name": "app", "version": "0.1.0"}),
			},
		},
	}
	plugin := &replacingPlugin{
		file:        "package.json",
		replaceType: "init",
		newEvents:   []eventing.Event{evt(t, "eX", "plugin.init", 99, map[string]any{"name": "app", "version": "9.9.9"})},
	}

	p := project.New(dir).WithPlugins(plugin)
	if err := p.Init(project.Meta{Name: "app", Template: "node-ts"}, s); err != nil {
		t.Fatal(err)
	}

	b, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	var got map[string]any
	json.Unmarshal(b, &got)
	if got["version"] != "9.9.9" {
		t.Errorf("version = %v, want %q (plugin Replace should have overridden template)", got["version"], "9.9.9")
	}
}

func TestWithPlugins_OverlayStillWinsOverPlugin(t *testing.T) {
	dir := testutil.TempDir(t)
	s := &stubSynth{
		template: map[string][]eventing.Event{
			"package.json": {evt(t, "e1", "init", 1, map[string]any{"version": "0.1.0"})},
		},
		overlay: map[string][]eventing.Event{
			"package.json": {evt(t, "e3", "rc.override", 1000, map[string]any{"version": "2.0.0"})},
		},
	}
	plugin := &stubPlugin{events: map[string][]eventing.Event{
		"package.json": {evt(t, "e2", "plugin.version", 1, map[string]any{"version": "1.0.0"})},
	}}

	p := project.New(dir).WithPlugins(plugin)
	if err := p.Init(project.Meta{Name: "app", Template: "node-ts"}, s); err != nil {
		t.Fatal(err)
	}

	b, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	var got map[string]any
	json.Unmarshal(b, &got)
	if got["version"] != "2.0.0" {
		t.Errorf("version = %v, want %q (overlay must beat plugin)", got["version"], "2.0.0")
	}
}

func TestWithPlugins_MultiplePlugins_BothApplied(t *testing.T) {
	dir := testutil.TempDir(t)
	s := &stubSynth{
		template: map[string][]eventing.Event{
			"package.json": {evt(t, "e1", "init", 1, map[string]any{"name": "app"})},
		},
	}
	p := project.New(dir).WithPlugins(
		&stubPlugin{events: map[string][]eventing.Event{
			"package.json": {evt(t, "e2", "plugin1.added", 1, map[string]any{
				"devDependencies": map[string]any{"eslint": "^8.0.0"},
			})},
		}},
		&stubPlugin{events: map[string][]eventing.Event{
			"package.json": {evt(t, "e3", "plugin2.added", 1, map[string]any{
				"devDependencies": map[string]any{"jest": "^29.0.0"},
			})},
		}},
	)

	if err := p.Init(project.Meta{Name: "app", Template: "node-ts"}, s); err != nil {
		t.Fatal(err)
	}

	b, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	var got map[string]any
	json.Unmarshal(b, &got)
	devDeps, _ := got["devDependencies"].(map[string]any)
	if devDeps["eslint"] == nil {
		t.Error("eslint (plugin1) missing")
	}
	if devDeps["jest"] == nil {
		t.Error("jest (plugin2) missing")
	}
}
