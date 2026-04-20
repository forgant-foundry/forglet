package commands

import "github.com/forgant-foundry/forglet/internal/project"

var registeredPlugins []project.Plugin

// RegisterPlugin registers one or more plugins to be applied during every
// synthesis. Call this from a custom main.go before commands.Execute() to
// build a platform-specific forglet binary that includes your plugins.
func RegisterPlugin(plugins ...project.Plugin) {
	registeredPlugins = append(registeredPlugins, plugins...)
}
