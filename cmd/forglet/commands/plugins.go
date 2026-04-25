package commands

import "github.com/forgant-foundry/forglet/internal/project"

var registeredPlugins []project.Plugin

// RegisterPlugin registers one or more plugins to be applied during every
// synthesis. Call this from a custom main.go before commands.Execute() to
// build a platform-specific forglet binary that includes your plugins.
func RegisterPlugin(plugins ...project.Plugin) {
	registeredPlugins = append(registeredPlugins, plugins...)
}

var registeredValidators []project.Validator

// RegisterValidator registers one or more BaseUnit validators to run after every
// synthesis. Call this from a custom main.go before commands.Execute() to enforce
// project policy in a platform-specific forglet binary.
func RegisterValidator(validators ...project.Validator) {
	registeredValidators = append(registeredValidators, validators...)
}
