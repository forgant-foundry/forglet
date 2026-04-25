// Package lerna provides a forglet plugin that adds Lerna monorepo support
// to node-ts projects.
//
// When registered, LernaPlugin contributes a lerna.json configuration file
// (via FormatJSON) and adds lerna as a devDependency to package.json. It is
// designed to be used alongside the workspaces plugin — register both to get
// a fully configured npm workspaces + Lerna project.
//
// LernaPlugin is template-scoped to node-ts and does nothing for other templates.
//
// Register in a custom binary:
//
//	commands.RegisterPlugin(workspaces.New(), lerna.New())
package lerna
