package project_test

import (
	"encoding/json"
	"testing"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
)

func TestBuildAggregates_TemplateOnly(t *testing.T) {
	template := map[string][]eventing.Event{
		"package.json": {evt(t, "e1", "init", 1, map[string]any{"name": "app", "version": "0.1.0"})},
	}
	aggs := project.BuildAggregates(template, nil)

	agg, ok := aggs["package.json"]
	if !ok {
		t.Fatal("package.json aggregate not built")
	}
	node, ok := agg.Node("name")
	if !ok || node.Value != "app" {
		t.Errorf("name = %v, want %q", node.Value, "app")
	}
}

func TestBuildAggregates_OverlayOnly(t *testing.T) {
	overlay := map[string][]eventing.Event{
		"extra.json": {evt(t, "e1", "init", 1000, map[string]any{"key": "val"})},
	}
	aggs := project.BuildAggregates(nil, overlay)

	if _, ok := aggs["extra.json"]; !ok {
		t.Error("overlay-only file not present in aggregates")
	}
}

func TestBuildAggregates_ScalarOverlayWins(t *testing.T) {
	template := map[string][]eventing.Event{
		"package.json": {evt(t, "e1", "init", 1, map[string]any{"version": "0.1.0"})},
	}
	overlay := map[string][]eventing.Event{
		"package.json": {evt(t, "e2", "version.override", 1000, map[string]any{"version": "2.0.0"})},
	}
	aggs := project.BuildAggregates(template, overlay)

	node, ok := aggs["package.json"].Node("version")
	if !ok {
		t.Fatal("version node not found")
	}
	if node.Value != "2.0.0" {
		t.Errorf("version = %v, want %q (overlay should win)", node.Value, "2.0.0")
	}
	if node.EventType != "version.override" {
		t.Errorf("version EventType = %q, want %q", node.EventType, "version.override")
	}
}

func TestBuildAggregates_ObjectChildrenMerged(t *testing.T) {
	template := map[string][]eventing.Event{
		"package.json": {evt(t, "e1", "devDependency.added", 1,
			map[string]any{"devDependencies": map[string]any{"typescript": "^5.0.0"}})},
	}
	overlay := map[string][]eventing.Event{
		"package.json": {evt(t, "e2", "devDependency.added", 1000,
			map[string]any{"devDependencies": map[string]any{"prettier": "^3.0.0"}})},
	}
	aggs := project.BuildAggregates(template, overlay)

	b, _ := aggs["package.json"].ToJSON()
	var got map[string]any
	json.Unmarshal(b, &got)

	devDeps, _ := got["devDependencies"].(map[string]any)
	if devDeps["typescript"] == nil {
		t.Error("typescript (template) missing after merge")
	}
	if devDeps["prettier"] == nil {
		t.Error("prettier (overlay) missing after merge")
	}
}

func TestBuildAggregates_ObjectMergeChildProvenance(t *testing.T) {
	template := map[string][]eventing.Event{
		"package.json": {evt(t, "e1", "devDependency.added", 1,
			map[string]any{"devDependencies": map[string]any{"typescript": "^5.0.0"}})},
	}
	overlay := map[string][]eventing.Event{
		"package.json": {evt(t, "e2", "rc.devDependency.added", 1000,
			map[string]any{"devDependencies": map[string]any{"prettier": "^3.0.0"}})},
	}
	aggs := project.BuildAggregates(template, overlay)

	devDepsNode, ok := aggs["package.json"].Node("devDependencies")
	if !ok {
		t.Fatal("devDependencies node not found")
	}
	children, ok := devDepsNode.Value.(eventing.Object)
	if !ok {
		t.Fatal("devDependencies value is not an Object")
	}

	byName := make(map[string]*eventing.Node)
	for _, c := range children {
		byName[c.Name] = c
	}

	if byName["typescript"] == nil || byName["typescript"].EventType != "devDependency.added" {
		t.Errorf("typescript child EventType = %q, want %q", byName["typescript"].EventType, "devDependency.added")
	}
	if byName["prettier"] == nil || byName["prettier"].EventType != "rc.devDependency.added" {
		t.Errorf("prettier child EventType = %q, want %q", byName["prettier"].EventType, "rc.devDependency.added")
	}
}

func TestBuildAggregates_OverlayChildWinsForSameKey(t *testing.T) {
	template := map[string][]eventing.Event{
		"package.json": {evt(t, "e1", "scripts.added", 1,
			map[string]any{"scripts": map[string]any{"build": "tsc", "start": "node dist/index.js"}})},
	}
	overlay := map[string][]eventing.Event{
		"package.json": {evt(t, "e2", "scripts.override", 1000,
			map[string]any{"scripts": map[string]any{"build": "tsc --watch"}})},
	}
	aggs := project.BuildAggregates(template, overlay)

	b, _ := aggs["package.json"].ToJSON()
	var got map[string]any
	json.Unmarshal(b, &got)

	scripts, _ := got["scripts"].(map[string]any)
	if scripts["build"] != "tsc --watch" {
		t.Errorf("build = %v, want %q (overlay child should win)", scripts["build"], "tsc --watch")
	}
	if scripts["start"] == nil {
		t.Error("start (template-only child) missing after merge")
	}
}
