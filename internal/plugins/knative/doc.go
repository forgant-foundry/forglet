// Package knative provides forglet plugins that add Knative Serving support.
//
// # Node plugin
//
// [Plugin] (construct via [New]) targets node-ts projects. When registered, it
// contributes .knative/service.yaml (via FormatYAML) and .dockerignore (via
// FormatPattern). It also implements [project.Scaffolder] to write a
// Dockerfile exactly once during init.
//
// The plugin uses stream inspection to detect whether a CDK plugin has already
// contributed — if so, it adjusts the Knative service manifest accordingly.
//
// # Go plugin
//
// [GoPlugin] (construct via [NewGo]) targets go-knative projects. When registered,
// it contributes func.yaml (the Knative func CLI manifest, via FormatYAML). It
// also implements [project.Scaffolder] to write handle.go, main.go,
// handle_test.go, and a multi-stage Dockerfile exactly once during init.
//
// Always register both plugins together for go-knative projects:
//
//	commands.RegisterPlugin(knative.New(), knative.NewGo())
//
// # RC overlay keys (go-knative)
//
//	name: "my-handler"          # Knative func name in func.yaml
//	registry: "gcr.io/myproj"  # container registry in func.yaml
//
// # Register in a custom binary
//
//	commands.RegisterPlugin(knative.New())          // node-ts projects
//	commands.RegisterPlugin(knative.New(), knative.NewGo()) // also go-knative
package knative
