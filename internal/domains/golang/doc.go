// Package golang provides Go synthesizers for forglet.
//
// All synthesizers manage go.mod (or go.work) as a synthesized file and write
// scaffold files exactly once during init. The RC overlay (via .forglet.yml)
// can override the module path, Go version, and required dependencies.
//
// # Synthesizers
//
// [Flat] (template "go") — a plain single-module Go project. Manages go.mod;
// scaffolds main.go with a minimal main function.
//
// [Workspace] (template "go-workspace") — a Go workspace (go.work). Manages
// go.work; scaffolds <name>/go.mod and <name>/main.go as the first workspace
// member. Use the rc "members" key ([]string) to declare additional members.
//
// [Lambda] (template "go-lambda") — extends Flat with the aws-lambda-go
// dependency and a Makefile that cross-compiles to a Lambda-ready "bootstrap"
// binary (GOOS=linux GOARCH=amd64). Scaffolds main.go with a typed
// lambda.Start handler.
//
// [GoKnative] (template "go-knative") — a Knative function project. Manages
// go.mod only (delegates to Flat); does not implement Scaffolder. The
// knative.GoPlugin writes func.yaml (via FormatYAML) and scaffolds handle.go,
// main.go, handle_test.go, and Dockerfile — register it alongside GoKnative.
//
// # RC overlay keys (all templates)
//
//	module: "github.com/myorg/myapp"
//	go: "1.23"
//	require:
//	  "github.com/spf13/cobra": "v1.8.0"
package golang
