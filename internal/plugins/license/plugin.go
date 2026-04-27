package license

import (
	"crypto/rand"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
)

//go:embed licenses
var licensesFS embed.FS

// Plugin writes a managed LICENSE file chosen by rc["license"].
// Built-in SPDX identifiers: MIT, Apache-2.0, GPL-3.0, AGPL-3.0, ISC.
// Platform teams supply additional license texts via WithCustom.
type Plugin struct {
	custom map[string]string
}

// Option configures Plugin at construction time.
type Option func(*Plugin)

// WithCustom registers a license text under the given name, available as
// rc["license"] = name. The name lookup is case-insensitive. Custom entries
// take precedence over built-in SPDX texts, allowing built-ins to be overridden.
// The text may contain a {{COPYRIGHT}} placeholder (replaced with "YEAR AUTHOR"
// or just "YEAR" when no author is configured).
func WithCustom(name, text string) Option {
	return func(p *Plugin) {
		p.custom[name] = text
	}
}

func New(opts ...Option) *Plugin {
	p := &Plugin{custom: map[string]string{}}
	for _, o := range opts {
		o(p)
	}
	return p
}

func (p *Plugin) Weave(_ project.Meta, rc map[string]any, stream *project.EventStream) error {
	spdx, year, author := parseRC(rc)
	if spdx == "" {
		return nil
	}

	tmpl, ok := p.resolve(spdx)
	if !ok {
		return fmt.Errorf("license plugin: unknown license %q (built-in: MIT, Apache-2.0, GPL-3.0, AGPL-3.0, ISC)", spdx)
	}

	copyright := year
	if author != "" {
		copyright = year + " " + author
	}
	text := strings.ReplaceAll(tmpl, "{{COPYRIGHT}}", copyright)

	payload, err := json.Marshal(map[string]any{"text": text})
	if err != nil {
		return err
	}
	stream.SetFormat("LICENSE", project.FormatText)
	stream.Append("LICENSE", eventing.Event{
		ID:      newID(),
		Type:    "license.configured",
		Seq:     1,
		Payload: json.RawMessage(payload),
	})
	return nil
}

// resolve finds the license text for spdx. Custom entries are checked first
// (case-insensitive), then built-in SPDX texts via canonicalSPDX normalization.
func (p *Plugin) resolve(spdx string) (string, bool) {
	for k, v := range p.custom {
		if strings.EqualFold(k, spdx) {
			return v, true
		}
	}
	canonical := canonicalSPDX(spdx)
	if canonical == "" {
		return "", false
	}
	data, err := licensesFS.ReadFile("licenses/" + canonical + ".txt")
	if err != nil {
		return "", false
	}
	return string(data), true
}

// canonicalSPDX maps common aliases to the filename used in the licenses/ dir.
func canonicalSPDX(spdx string) string {
	switch strings.ToUpper(spdx) {
	case "MIT":
		return "MIT"
	case "APACHE-2.0", "APACHE2", "APACHE 2.0":
		return "Apache-2.0"
	case "GPL-3.0", "GPL-3.0-ONLY", "GPL3", "GPLV3":
		return "GPL-3.0"
	case "AGPL-3.0", "AGPL-3.0-ONLY", "AGPL3", "AGPLV3":
		return "AGPL-3.0"
	case "ISC":
		return "ISC"
	default:
		return ""
	}
}

// parseRC extracts license config from the rc map.
// Shorthand: license: MIT
// Map form:  license: { spdx: MIT, year: 2024, author: "Acme Corp" }
func parseRC(rc map[string]any) (spdx, year, author string) {
	year = fmt.Sprintf("%d", time.Now().Year())
	switch v := rc["license"].(type) {
	case string:
		spdx = v
	case map[string]any:
		spdx, _ = v["spdx"].(string)
		switch y := v["year"].(type) {
		case string:
			if y != "" {
				year = y
			}
		case int:
			year = fmt.Sprintf("%d", y)
		}
		author, _ = v["author"].(string)
	}
	return
}

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
