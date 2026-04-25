package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
	"github.com/forgant-foundry/forglet/internal/testutil"
)

type stubValidator struct {
	violations []project.Violation
	err        error
	captured   *capturedValidatorArgs
}

type capturedValidatorArgs struct {
	meta project.Meta
	rc   map[string]any
	dir  string
}

func (v *stubValidator) Validate(meta project.Meta, rc map[string]any, dir string) ([]project.Violation, error) {
	if v.captured != nil {
		*v.captured = capturedValidatorArgs{meta: meta, rc: rc, dir: dir}
	}
	return v.violations, v.err
}

type countingValidator struct{ count *int }

func (v *countingValidator) Validate(_ project.Meta, _ map[string]any, _ string) ([]project.Violation, error) {
	*v.count++
	return nil, nil
}

func TestWithValidators_NoViolations(t *testing.T) {
	dir := testutil.TempDir(t)
	s := &stubSynth{
		template: map[string][]eventing.Event{
			"package.json": {evt(t, "e1", "init", 1, map[string]any{"name": "app"})},
		},
	}
	p := project.New(dir).WithValidators(&stubValidator{})
	if err := p.Init(project.Meta{Name: "app", Template: "node-ts"}, s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWithValidators_ErrorViolation_FailsSynth(t *testing.T) {
	dir := testutil.TempDir(t)
	s := &stubSynth{
		template: map[string][]eventing.Event{
			"package.json": {evt(t, "e1", "init", 1, map[string]any{"name": "app"})},
		},
	}
	v := &stubValidator{
		violations: []project.Violation{
			{File: "package.json", Rule: "test-rule", Message: "oh no", Severity: project.SeverityError},
		},
	}
	p := project.New(dir).WithValidators(v)
	if err := p.Init(project.Meta{Name: "app", Template: "node-ts"}, s); err == nil {
		t.Fatal("expected error from SeverityError violation, got nil")
	}
}

func TestWithValidators_WarningViolation_DoesNotFail(t *testing.T) {
	dir := testutil.TempDir(t)
	s := &stubSynth{
		template: map[string][]eventing.Event{
			"package.json": {evt(t, "e1", "init", 1, map[string]any{"name": "app"})},
		},
	}
	v := &stubValidator{
		violations: []project.Violation{
			{File: "package.json", Rule: "test-rule", Message: "heads up", Severity: project.SeverityWarning},
		},
	}
	p := project.New(dir).WithValidators(v)
	if err := p.Init(project.Meta{Name: "app", Template: "node-ts"}, s); err != nil {
		t.Fatalf("unexpected error from SeverityWarning violation: %v", err)
	}
}

func TestWithValidators_ReceivedCorrectArgs(t *testing.T) {
	dir := testutil.TempDir(t)
	s := &stubSynth{
		template: map[string][]eventing.Event{
			"package.json": {evt(t, "e1", "init", 1, map[string]any{"name": "app"})},
		},
	}
	rcFile := []byte("someKey: someValue\n")
	if err := os.WriteFile(filepath.Join(dir, ".forglet.yml"), rcFile, 0644); err != nil {
		t.Fatal(err)
	}

	var captured capturedValidatorArgs
	v := &stubValidator{captured: &captured}
	meta := project.Meta{Name: "app", Template: "node-ts"}
	p := project.New(dir).WithValidators(v)
	if err := p.Init(meta, s); err != nil {
		t.Fatal(err)
	}

	if captured.meta.Name != "app" || captured.meta.Template != "node-ts" {
		t.Errorf("meta = %+v, want {Name:app Template:node-ts}", captured.meta)
	}
	if captured.dir != dir {
		t.Errorf("dir = %q, want %q", captured.dir, dir)
	}
	if captured.rc["someKey"] != "someValue" {
		t.Errorf("rc[someKey] = %v, want %q", captured.rc["someKey"], "someValue")
	}
}

func TestWithValidators_MultipleValidators_AllRun(t *testing.T) {
	dir := testutil.TempDir(t)
	s := &stubSynth{
		template: map[string][]eventing.Event{
			"package.json": {evt(t, "e1", "init", 1, map[string]any{"name": "app"})},
		},
	}
	count := 0
	p := project.New(dir).WithValidators(
		&countingValidator{count: &count},
		&countingValidator{count: &count},
		&countingValidator{count: &count},
	)
	if err := p.Init(project.Meta{Name: "app", Template: "node-ts"}, s); err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("validator run count = %d, want 3", count)
	}
}

func TestWithValidators_MultipleErrors_AllReported(t *testing.T) {
	dir := testutil.TempDir(t)
	s := &stubSynth{
		template: map[string][]eventing.Event{
			"package.json": {evt(t, "e1", "init", 1, map[string]any{"name": "app"})},
		},
	}
	v := &stubValidator{
		violations: []project.Violation{
			{File: "a.json", Rule: "rule-1", Message: "first", Severity: project.SeverityError},
			{File: "b.json", Rule: "rule-2", Message: "second", Severity: project.SeverityError},
		},
	}
	p := project.New(dir).WithValidators(v)
	err := p.Init(project.Meta{Name: "app", Template: "node-ts"}, s)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if !contains(msg, "rule-1") || !contains(msg, "rule-2") {
		t.Errorf("error message %q should mention both rules", msg)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := range s {
		if i+len(sub) <= len(s) && s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
