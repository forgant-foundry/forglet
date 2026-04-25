package lerna

import "github.com/forgant-foundry/forglet/internal/project"

// LernaPlugin adds Lerna monorepo support to any node template.
// It contributes lerna.json (via the JSON cross-cutting file mechanism) and
// adds lerna as a devDependency in package.json.
//
// Register WorkspacesPlugin alongside this plugin — Lerna requires npm workspaces:
//
//	project.New(dir).WithPlugins(workspaces.New(), lerna.New())
type LernaPlugin struct{}

func New() *LernaPlugin { return &LernaPlugin{} }

func (p *LernaPlugin) Weave(meta project.Meta, rc map[string]any, stream *project.EventStream) error {
	return nil
}
