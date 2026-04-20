package project_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forgant-foundry/forglet/internal/project"
	"github.com/forgant-foundry/forglet/internal/testutil"
)

func TestWriteManaged_CreatesReadOnly(t *testing.T) {
	dir := testutil.TempDir(t)
	path := filepath.Join(dir, "file.txt")
	if err := project.WriteManaged(path, []byte("hello\n")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0444 {
		t.Errorf("file mode = %04o, want 0444", info.Mode().Perm())
	}
}

func TestWriteManaged_OverwritesExistingReadOnly(t *testing.T) {
	dir := testutil.TempDir(t)
	path := filepath.Join(dir, "file.txt")
	if err := project.WriteManaged(path, []byte("first\n")); err != nil {
		t.Fatal(err)
	}
	if err := project.WriteManaged(path, []byte("second\n")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "second\n" {
		t.Errorf("content = %q, want %q", string(b), "second\n")
	}
}

func TestAddJSONMarker_HasCommentKey(t *testing.T) {
	input := []byte("{\n  \"name\": \"my-app\"\n}\n")
	out := project.AddJSONMarker(input)
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if _, ok := m["//"]; !ok {
		t.Error(`"//" key not present in output`)
	}
}

func TestAddJSONMarker_CommentKeyIsFirst(t *testing.T) {
	input := []byte("{\n  \"name\": \"my-app\"\n}\n")
	out := project.AddJSONMarker(input)
	s := string(out)
	commentIdx := strings.Index(s, `"//"`)
	nameIdx := strings.Index(s, `"name"`)
	if commentIdx == -1 {
		t.Fatal(`"//" key not found in output`)
	}
	if nameIdx == -1 {
		t.Fatal(`"name" key not found in output`)
	}
	if commentIdx > nameIdx {
		t.Errorf(`"//" appears after "name"; want it first`)
	}
}

func TestAddTextMarker_PrependsPoundComment(t *testing.T) {
	input := []byte("node_modules/\n")
	out := project.AddTextMarker(input, "#")
	s := string(out)
	if !strings.HasPrefix(s, "# ") {
		t.Errorf("output does not start with '# ', got: %q", s)
	}
	if !strings.Contains(s, "node_modules/") {
		t.Error("original content missing from output")
	}
}

func TestAddTextMarker_PrependsSlashComment(t *testing.T) {
	input := []byte("module myapp\n")
	out := project.AddTextMarker(input, "//")
	s := string(out)
	if !strings.HasPrefix(s, "// ") {
		t.Errorf("output does not start with '// ', got: %q", s)
	}
	if !strings.Contains(s, "module myapp") {
		t.Error("original content missing from output")
	}
}

func TestAddTextMarker_ContainsManagedText(t *testing.T) {
	out := project.AddTextMarker([]byte("content\n"), "#")
	if !strings.Contains(string(out), "forglet") {
		t.Error("marker does not mention forglet")
	}
}
