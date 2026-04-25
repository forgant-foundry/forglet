package npm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/forgant-foundry/forglet/internal/project"
)

// NpmScriptPolicyValidator enforces that @lavamoat/allow-scripts is configured
// in node-ts projects, preventing unauthorized lifecycle script execution.
// Supply chain attacks frequently abuse postinstall hooks; this validator confirms
// the synthesizer's policy enforcement is intact before synthesis completes.
type NpmScriptPolicyValidator struct{}

func New() *NpmScriptPolicyValidator { return &NpmScriptPolicyValidator{} }

func (v *NpmScriptPolicyValidator) Validate(meta project.Meta, _ map[string]any, dir string) ([]project.Violation, error) {
	if meta.Template != "node-ts" {
		return nil, nil
	}

	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil, fmt.Errorf("npm policy: read package.json: %w", err)
	}

	var pkg map[string]any
	if err := json.Unmarshal(b, &pkg); err != nil {
		return nil, fmt.Errorf("npm policy: parse package.json: %w", err)
	}

	var violations []project.Violation

	devDeps, _ := pkg["devDependencies"].(map[string]any)
	if devDeps["@lavamoat/allow-scripts"] == nil {
		violations = append(violations, project.Violation{
			File:     "package.json",
			Rule:     "npm-allow-scripts",
			Message:  "@lavamoat/allow-scripts must be in devDependencies",
			Severity: project.SeverityError,
		})
	}

	scripts, _ := pkg["scripts"].(map[string]any)
	if scripts["postinstall"] != "allow-scripts" {
		violations = append(violations, project.Violation{
			File:     "package.json",
			Rule:     "npm-allow-scripts",
			Message:  `scripts.postinstall must be "allow-scripts" to block unauthorized lifecycle scripts`,
			Severity: project.SeverityError,
		})
	}

	return violations, nil
}
