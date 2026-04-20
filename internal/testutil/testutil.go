package testutil

import (
	"os"
	"strings"
	"testing"
)

// TempDir returns a temp directory for the test. The path is always logged so
// it appears in output on failure or when running with -v.
//
// If FORGLET_KEEP_TEMP=1 the directory is created in the system temp dir with a
// readable name and is not cleaned up — useful for inspecting output during
// implementation or diagnosing failures.
func TempDir(t *testing.T) string {
	t.Helper()
	if os.Getenv("FORGLET_KEEP_TEMP") == "1" {
		safe := strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())
		dir, err := os.MkdirTemp("", "forglet-"+safe+"-*")
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("temp dir (kept): %s", dir)
		return dir
	}
	dir := t.TempDir()
	t.Logf("temp dir: %s", dir)
	return dir
}
