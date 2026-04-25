package npm_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/forgant-foundry/forglet/internal/baseunit/npm"
	"github.com/forgant-foundry/forglet/internal/domains/node"
	"github.com/forgant-foundry/forglet/internal/project"
	"github.com/forgant-foundry/forglet/internal/testutil"
)

func writePackageJSON(t *testing.T, dir string, content map[string]any) {
	t.Helper()
	b, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), b, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestNpmScriptPolicyValidator_Pass(t *testing.T) {
	dir := testutil.TempDir(t)
	writePackageJSON(t, dir, map[string]any{
		"devDependencies": map[string]any{
			"@lavamoat/allow-scripts": "^3.3.1",
		},
		"scripts": map[string]any{
			"postinstall": "allow-scripts",
		},
	})

	violations, err := npm.New().Validate(project.Meta{Name: "app", Template: "node-ts"}, nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Errorf("expected no violations, got %v", violations)
	}
}

func TestNpmScriptPolicyValidator_MissingDevDep(t *testing.T) {
	dir := testutil.TempDir(t)
	writePackageJSON(t, dir, map[string]any{
		"devDependencies": map[string]any{"typescript": "^5.0.0"},
		"scripts":         map[string]any{"postinstall": "allow-scripts"},
	})

	violations, err := npm.New().Validate(project.Meta{Name: "app", Template: "node-ts"}, nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !hasRule(violations, "npm-allow-scripts") {
		t.Error("expected npm-allow-scripts violation for missing @lavamoat/allow-scripts devDependency")
	}
	if !allSeverity(violations, project.SeverityError) {
		t.Error("expected all violations to be SeverityError")
	}
}

func TestNpmScriptPolicyValidator_WrongPostinstall(t *testing.T) {
	dir := testutil.TempDir(t)
	writePackageJSON(t, dir, map[string]any{
		"devDependencies": map[string]any{"@lavamoat/allow-scripts": "^3.3.1"},
		"scripts":         map[string]any{"postinstall": "echo dangerous"},
	})

	violations, err := npm.New().Validate(project.Meta{Name: "app", Template: "node-ts"}, nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !hasRule(violations, "npm-allow-scripts") {
		t.Error("expected violation for wrong postinstall value")
	}
}

func TestNpmScriptPolicyValidator_MissingPostinstall(t *testing.T) {
	dir := testutil.TempDir(t)
	writePackageJSON(t, dir, map[string]any{
		"devDependencies": map[string]any{"@lavamoat/allow-scripts": "^3.3.1"},
		"scripts":         map[string]any{"build": "tsc"},
	})

	violations, err := npm.New().Validate(project.Meta{Name: "app", Template: "node-ts"}, nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !hasRule(violations, "npm-allow-scripts") {
		t.Error("expected violation for missing postinstall script")
	}
}

func TestNpmScriptPolicyValidator_BothMissing_TwoViolations(t *testing.T) {
	dir := testutil.TempDir(t)
	writePackageJSON(t, dir, map[string]any{
		"name": "bare-app",
	})

	violations, err := npm.New().Validate(project.Meta{Name: "app", Template: "node-ts"}, nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 2 {
		t.Errorf("expected 2 violations (devDep + postinstall), got %d", len(violations))
	}
}

func TestNpmScriptPolicyValidator_NonNodeTemplate_Skipped(t *testing.T) {
	dir := testutil.TempDir(t)
	// No package.json — if the validator didn't skip, os.ReadFile would error.
	for _, tmpl := range []string{"go", "go-workspace"} {
		violations, err := npm.New().Validate(project.Meta{Name: "mymod", Template: tmpl}, nil, dir)
		if err != nil {
			t.Errorf("template %q: unexpected error: %v", tmpl, err)
		}
		if len(violations) != 0 {
			t.Errorf("template %q: expected no violations, got %d", tmpl, len(violations))
		}
	}
}

func TestNpmScriptPolicyValidator_Integration_DefaultNodeTsSynth(t *testing.T) {
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

	violations, err := npm.New().Validate(project.Meta{Name: "my-app", Template: "node-ts"}, nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Errorf("default node-ts synth should pass npm policy, got: %v", violations)
	}
}

func TestNpmScriptPolicyValidator_Integration_AllowScriptsInPackageJSON(t *testing.T) {
	dir := testutil.TempDir(t)
	s := node.NewTypeScript()

	templateEvents, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	rc := map[string]any{
		"allowScripts": map[string]any{"esbuild": true},
	}
	overlayEvents, err := s.OverlayEvents(rc)
	if err != nil {
		t.Fatal(err)
	}
	aggregates := project.BuildAggregates(templateEvents, overlayEvents)
	if err := s.Synthesize(dir, aggregates); err != nil {
		t.Fatal(err)
	}

	violations, err := npm.New().Validate(project.Meta{Name: "my-app", Template: "node-ts"}, rc, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Errorf("expected no violations after allowScripts overlay, got: %v", violations)
	}

	b, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	lavamoat, _ := pkg["lavamoat"].(map[string]any)
	if lavamoat == nil {
		t.Fatal("package.json missing lavamoat key")
	}
	allowScripts, _ := lavamoat["allowScripts"].(map[string]any)
	if allowScripts["esbuild"] != true {
		t.Errorf("lavamoat.allowScripts.esbuild = %v, want true", allowScripts["esbuild"])
	}
}

func hasRule(violations []project.Violation, rule string) bool {
	for _, v := range violations {
		if v.Rule == rule {
			return true
		}
	}
	return false
}

func allSeverity(violations []project.Violation, s project.Severity) bool {
	for _, v := range violations {
		if v.Severity != s {
			return false
		}
	}
	return true
}
