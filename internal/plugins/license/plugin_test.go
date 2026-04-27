package license_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/forgant-foundry/eventing"
	licenseplugin "github.com/forgant-foundry/forglet/internal/plugins/license"
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

// ---- MIT -------------------------------------------------------------------------

func TestLicense_MIT_FileCreated(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "license: MIT\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "LICENSE")); err != nil {
		t.Errorf("LICENSE not created: %v", err)
	}
}

func TestLicense_MIT_ContainsMITText(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "license: MIT\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "LICENSE"))
	assertContains(t, content, "MIT License")
	assertContains(t, content, "Permission is hereby granted")
}

func TestLicense_MIT_DefaultYearIsCurrentYear(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "license: MIT\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "LICENSE"))
	assertContains(t, content, fmt.Sprintf("%d", time.Now().Year()))
}

func TestLicense_MIT_YearFromRC(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "license:\n  spdx: MIT\n  year: 2021\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "LICENSE"))
	assertContains(t, content, "2021")
}

func TestLicense_MIT_AuthorSubstituted(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "license:\n  spdx: MIT\n  year: 2024\n  author: \"Acme Corp\"\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "LICENSE"))
	assertContains(t, content, "2024 Acme Corp")
}

func TestLicense_MIT_NoAuthor_NoTrailingSpace(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "license:\n  spdx: MIT\n  year: 2024\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "LICENSE"))
	// copyright line should end with the year, no trailing space
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "Copyright") {
			if strings.HasSuffix(line, " ") {
				t.Errorf("copyright line has trailing space: %q", line)
			}
		}
	}
}

// ---- Apache-2.0 -----------------------------------------------------------------

func TestLicense_Apache2_FileCreated(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "license: Apache-2.0\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "LICENSE"))
	assertContains(t, content, "Apache License")
	assertContains(t, content, "Version 2.0")
}

func TestLicense_Apache2_Alias(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "license: apache-2.0\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	assertContains(t, readFile(t, filepath.Join(dir, "LICENSE")), "Apache License")
}

// ---- ISC -------------------------------------------------------------------------

func TestLicense_ISC_FileCreated(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "license: ISC\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "LICENSE"))
	assertContains(t, content, "ISC License")
	assertContains(t, content, "Permission to use")
}

func TestLicense_ISC_AuthorSubstituted(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "license:\n  spdx: ISC\n  year: 2024\n  author: \"Forgant Foundry\"\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	assertContains(t, readFile(t, filepath.Join(dir, "LICENSE")), "2024 Forgant Foundry")
}

// ---- GPL-3.0 / AGPL-3.0 ---------------------------------------------------------

func TestLicense_GPL3_FileCreated(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "license: GPL-3.0\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "LICENSE"))
	assertContains(t, content, "GNU GENERAL PUBLIC LICENSE")
	assertContains(t, content, "Version 3")
}

func TestLicense_AGPL3_FileCreated(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "license: AGPL-3.0\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "LICENSE"))
	assertContains(t, content, "GNU AFFERO GENERAL PUBLIC LICENSE")
}

func TestLicense_GPL3_Aliases(t *testing.T) {
	for _, alias := range []string{"gpl-3.0", "GPL3", "GPLv3", "gpl-3.0-only"} {
		t.Run(alias, func(t *testing.T) {
			dir, p := setup(t)
			writeRC(t, dir, "license: "+alias+"\n")

			if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
				t.Fatal(err)
			}
			assertContains(t, readFile(t, filepath.Join(dir, "LICENSE")), "GNU GENERAL PUBLIC LICENSE")
		})
	}
}

// ---- Custom license -------------------------------------------------------------

func TestLicense_Custom_FileCreated(t *testing.T) {
	dir, p := setupWith(t, licenseplugin.WithCustom("acme-proprietary", "Proprietary License\n\nCopyright (c) {{COPYRIGHT}}\nAll rights reserved.\n"))
	writeRC(t, dir, "license: acme-proprietary\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "LICENSE"))
	assertContains(t, content, "Proprietary License")
	assertContains(t, content, "All rights reserved")
}

func TestLicense_Custom_CopyrightSubstituted(t *testing.T) {
	dir, p := setupWith(t, licenseplugin.WithCustom("corp-v1", "Copyright (c) {{COPYRIGHT}}\n"))
	writeRC(t, dir, "license:\n  spdx: corp-v1\n  year: 2023\n  author: \"Corp Inc\"\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	assertContains(t, readFile(t, filepath.Join(dir, "LICENSE")), "2023 Corp Inc")
}

func TestLicense_Custom_OverridesBuiltIn(t *testing.T) {
	dir, p := setupWith(t, licenseplugin.WithCustom("MIT", "Custom MIT variant\n"))
	writeRC(t, dir, "license: MIT\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "LICENSE"))
	assertContains(t, content, "Custom MIT variant")
	if strings.Contains(content, "Permission is hereby granted") {
		t.Error("custom MIT should override the built-in text")
	}
}

func TestLicense_Custom_LookupCaseInsensitive(t *testing.T) {
	dir, p := setupWith(t, licenseplugin.WithCustom("MyLicense", "My License Text\n"))
	writeRC(t, dir, "license: mylicense\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	assertContains(t, readFile(t, filepath.Join(dir, "LICENSE")), "My License Text")
}

// ---- Error cases ----------------------------------------------------------------

func TestLicense_UnknownSPDX_ReturnsError(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "license: UNKNOWN-99\n")

	err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{})
	if err == nil {
		t.Error("expected error for unknown license identifier")
	}
}

// ---- File properties ------------------------------------------------------------

func TestLicense_FileIsReadOnly(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "license: MIT\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(dir, "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0444 {
		t.Errorf("LICENSE mode = %04o, want 0444", info.Mode().Perm())
	}
}

func TestLicense_NoManagedMarker(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "license: MIT\n")

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "LICENSE"))
	if strings.Contains(content, "forglet") {
		t.Error("LICENSE file must not contain the forglet managed-comment marker")
	}
}

func TestLicense_Absent_NoFile(t *testing.T) {
	dir, p := setup(t)
	// no license key in rc

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, &noopSynth{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "LICENSE")); !os.IsNotExist(err) {
		t.Error("expected no LICENSE file when license key is absent")
	}
}

func TestLicense_Idempotent(t *testing.T) {
	dir, p := setup(t)
	writeRC(t, dir, "license:\n  spdx: MIT\n  year: 2024\n  author: \"Acme Corp\"\n")
	s := &noopSynth{}

	if err := p.Init(project.Meta{Name: "myapp", Template: "go"}, s); err != nil {
		t.Fatal(err)
	}
	first := readFile(t, filepath.Join(dir, "LICENSE"))

	if err := p.Synthesize(s); err != nil {
		t.Fatal(err)
	}
	second := readFile(t, filepath.Join(dir, "LICENSE"))

	if first != second {
		t.Errorf("LICENSE changed after second synth:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

// ---- helpers --------------------------------------------------------------------

func setup(t *testing.T) (string, *project.Project) {
	t.Helper()
	return setupWith(t)
}

func setupWith(t *testing.T, opts ...licenseplugin.Option) (string, *project.Project) {
	t.Helper()
	dir := testutil.TempDir(t)
	p := project.New(dir).WithPlugins(licenseplugin.New(opts...))
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
		t.Errorf("expected %q in content (first 200 chars):\n%s", want, content[:min(200, len(content))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
