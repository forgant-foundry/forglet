package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

const manifestFile = "manifest.json"

// loadManifest reads the managed-file manifest from .forglet/manifest.json.
// Returns an empty set when the file does not exist (first synth).
func (p *Project) loadManifest() (map[string]struct{}, error) {
	b, err := os.ReadFile(filepath.Join(p.root, forgletDir, manifestFile))
	if os.IsNotExist(err) {
		return map[string]struct{}{}, nil
	}
	if err != nil {
		return nil, err
	}
	var files []string
	if err := json.Unmarshal(b, &files); err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(files))
	for _, f := range files {
		set[f] = struct{}{}
	}
	return set, nil
}

// saveManifest writes the set of currently managed files to .forglet/manifest.json.
func (p *Project) saveManifest(files map[string]struct{}) error {
	sorted := make([]string, 0, len(files))
	for f := range files {
		sorted = append(sorted, f)
	}
	sort.Strings(sorted)
	b, err := json.MarshalIndent(sorted, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(p.root, forgletDir, manifestFile), b, 0644)
}

// removeOrphans deletes files that were managed in the previous synth but are
// no longer in the current managed set. Files the user has made writable (mode
// != 0444) are skipped — they have been taken out of forglet's management.
func (p *Project) removeOrphans(old, current map[string]struct{}) error {
	for f := range old {
		if _, stillManaged := current[f]; stillManaged {
			continue
		}
		path := filepath.Join(p.root, f)
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode().Perm() != 0444 {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
