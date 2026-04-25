// Package knative provides a forglet plugin that adds Knative Serving support
// to node-ts projects.
//
// When registered, the plugin contributes .knative/service.yaml (via
// FormatYAML) and .dockerignore (via FormatPattern). It also implements
// [project.Scaffolder] to write a Dockerfile exactly once during init.
//
// The plugin is template-scoped to node-ts and uses stream inspection to
// detect whether a CDK plugin has already contributed — if so, it adjusts
// the Knative service manifest accordingly.
//
// Register in a custom binary:
//
//	commands.RegisterPlugin(knative.New())
package knative
