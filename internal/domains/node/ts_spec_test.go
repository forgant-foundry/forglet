package node_test

// Spec tests for additional node-ts requirements not yet implemented.
// All tests here are intentionally failing until the synthesizer is updated.
//
// Certified against: node v22.14.0 / npm 11.7.0 / typescript 6.0.3
// Reference project: .scratch/my-node-ts-app (npm init + npm install -D typescript ts-node @types/node)

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

const (
	tsSpecNodeVersion        = ">=22.0.0" // node v22.14.0
	tsSpecNPMVersion         = ">=11.0.0" // npm 11.7.0
	tsSpecTypescriptVersion  = "^6.0.0"   // installed: 6.0.3
	tsSpecTSNodeVersion      = "^10.9.2"  // installed: 10.9.2
	tsSpecAtTypesNodeVersion = "^25.0.0"  // installed: 25.6.0
)

// --- engines field ---

func TestNodeTS_PackageJSONHasEnginesNode(t *testing.T) {
	s := node.NewTypeScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	agg := buildAgg(t, files["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	engines, _ := pkg["engines"].(map[string]any)
	if engines["node"] != tsSpecNodeVersion {
		t.Errorf("engines.node = %v, want %q", engines["node"], tsSpecNodeVersion)
	}
}

func TestNodeTS_PackageJSONHasEnginesNPM(t *testing.T) {
	s := node.NewTypeScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	agg := buildAgg(t, files["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	engines, _ := pkg["engines"].(map[string]any)
	if engines["npm"] != tsSpecNPMVersion {
		t.Errorf("engines.npm = %v, want %q", engines["npm"], tsSpecNPMVersion)
	}
}

// --- devDependencies ---

func TestNodeTS_PackageJSONHasTypescriptV6(t *testing.T) {
	s := node.NewTypeScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	agg := buildAgg(t, files["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	devDeps, _ := pkg["devDependencies"].(map[string]any)
	if devDeps["typescript"] != tsSpecTypescriptVersion {
		t.Errorf("devDependencies.typescript = %v, want %q", devDeps["typescript"], tsSpecTypescriptVersion)
	}
}

func TestNodeTS_PackageJSONHasTSNode(t *testing.T) {
	s := node.NewTypeScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	agg := buildAgg(t, files["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	devDeps, _ := pkg["devDependencies"].(map[string]any)
	if devDeps["ts-node"] != tsSpecTSNodeVersion {
		t.Errorf("devDependencies.ts-node = %v, want %q", devDeps["ts-node"], tsSpecTSNodeVersion)
	}
}

func TestNodeTS_PackageJSONHasAtTypesNode(t *testing.T) {
	s := node.NewTypeScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	agg := buildAgg(t, files["package.json"])
	b, _ := agg.ToJSON()
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	devDeps, _ := pkg["devDependencies"].(map[string]any)
	if devDeps["@types/node"] != tsSpecAtTypesNodeVersion {
		t.Errorf("devDependencies.@types/node = %v, want %q", devDeps["@types/node"], tsSpecAtTypesNodeVersion)
	}
}

// --- tsconfig compiler options ---

func TestNodeTS_TsconfigHasSourceMap(t *testing.T) {
	s := node.NewTypeScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	opts := tsconfigOpts(t, files)
	if opts["sourceMap"] != true {
		t.Errorf("compilerOptions.sourceMap = %v, want true", opts["sourceMap"])
	}
}

func TestNodeTS_TsconfigHasDeclaration(t *testing.T) {
	s := node.NewTypeScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	opts := tsconfigOpts(t, files)
	if opts["declaration"] != true {
		t.Errorf("compilerOptions.declaration = %v, want true", opts["declaration"])
	}
}

func TestNodeTS_TsconfigHasDeclarationMap(t *testing.T) {
	s := node.NewTypeScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	opts := tsconfigOpts(t, files)
	if opts["declarationMap"] != true {
		t.Errorf("compilerOptions.declarationMap = %v, want true", opts["declarationMap"])
	}
}

func TestNodeTS_TsconfigHasNoUncheckedIndexedAccess(t *testing.T) {
	s := node.NewTypeScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	opts := tsconfigOpts(t, files)
	if opts["noUncheckedIndexedAccess"] != true {
		t.Errorf("compilerOptions.noUncheckedIndexedAccess = %v, want true", opts["noUncheckedIndexedAccess"])
	}
}

func TestNodeTS_TsconfigHasExactOptionalPropertyTypes(t *testing.T) {
	s := node.NewTypeScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	opts := tsconfigOpts(t, files)
	if opts["exactOptionalPropertyTypes"] != true {
		t.Errorf("compilerOptions.exactOptionalPropertyTypes = %v, want true", opts["exactOptionalPropertyTypes"])
	}
}

func TestNodeTS_TsconfigHasSkipLibCheck(t *testing.T) {
	s := node.NewTypeScript()
	files, err := s.InitializeEvents("my-app")
	if err != nil {
		t.Fatal(err)
	}
	opts := tsconfigOpts(t, files)
	if opts["skipLibCheck"] != true {
		t.Errorf("compilerOptions.skipLibCheck = %v, want true", opts["skipLibCheck"])
	}
}

// Synthesize-level: verify the engines field reaches the written file.
func TestNodeTS_Synthesize_PackageJSONHasEnginesNode(t *testing.T) {
	dir := testutil.TempDir(t)
	s := node.NewTypeScript()
	files, _ := s.InitializeEvents("my-app")
	aggregates := project.BuildAggregates(files, nil)
	if err := s.Synthesize(dir, aggregates); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pkg map[string]any
	json.Unmarshal(b, &pkg)
	engines, _ := pkg["engines"].(map[string]any)
	if engines["node"] != tsSpecNodeVersion {
		t.Errorf("engines.node = %v, want %q", engines["node"], tsSpecNodeVersion)
	}
}

// tsconfigOpts is a local helper that extracts compilerOptions from tsconfig events.
func tsconfigOpts(t *testing.T, files map[string][]eventing.Event) map[string]any {
	t.Helper()
	agg := buildAgg(t, files["tsconfig.json"])
	b, _ := agg.ToJSON()
	var tsconfig map[string]any
	json.Unmarshal(b, &tsconfig)
	opts, _ := tsconfig["compilerOptions"].(map[string]any)
	return opts
}
