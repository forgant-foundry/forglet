// Package workspaces provides a forglet plugin that configures npm workspaces
// in node-ts projects.
//
// When registered, WorkspacesPlugin sets private=true and adds
// "workspaces": ["packages/*"] to package.json, making the project root a
// npm workspace host. It is designed to be used alongside the lerna plugin
// for Lerna-managed monorepos, but works independently for plain npm workspaces.
//
// WorkspacesPlugin is template-scoped to node-ts and does nothing for other templates.
//
// Register in a custom binary:
//
//	commands.RegisterPlugin(workspaces.New())
package workspaces
