// Package project is the core of forglet's event-sourced synthesis engine.
// It defines the interfaces that all templates (Synthesizer) and extensions
// (Plugin, Scaffolder, Validator) implement, and provides the Project type
// that orchestrates the three-layer synthesis cycle.
//
// # Building a custom forglet binary
//
// Platform engineers build a custom binary by importing this package's
// interfaces via cmd/forglet/commands:
//
//	package main
//
//	import (
//	    "github.com/forgant-foundry/forglet/cmd/forglet/commands"
//	    "github.com/mycompany/forglet-plugins/standards"
//	)
//
//	func main() {
//	    commands.RegisterPlugin(standards.New())
//	    commands.Execute()
//	}
//
// # The three-layer synthesis cycle
//
// Every call to [Project.Synthesize] re-derives three ordered event layers
// and merges them into per-file aggregates:
//
//  1. Template layer — [Synthesizer.InitializeEvents] (base shape of managed files)
//  2. Plugin layer — each registered [Plugin.Weave] call (platform standards)
//  3. RC overlay layer — [Synthesizer.OverlayEvents] applied to .forglet.yml (project customisation)
//
// For scalar aggregate nodes (strings, numbers, booleans), later layers win.
// For Object nodes (e.g. devDependencies, dependencies), children accumulate
// from all layers — every layer contributes its entries without overwriting others.
//
// Because all layers are re-derived fresh on every synth, upgrading forglet or
// its plugins takes effect automatically on the next `forglet synth` — there is
// no persistent event log to migrate.
//
// # Key interfaces
//
// [Synthesizer] is the main extension point for defining a new template. It
// provides the base event shape, interprets .forglet.yml config into overlay
// events, and writes managed files from the merged aggregates. Implement all
// three methods; use [BuildAggregates] in tests.
//
// [Plugin] injects platform-wide standards between the template and RC overlay
// layers. Its [Plugin.Weave] method receives the [Meta], the full RC map, and
// an [EventStream] with full positional control: [EventStream.Append],
// [EventStream.InsertBefore], [EventStream.InsertAfter], [EventStream.Replace],
// [EventStream.Remove]. Plugins can also call [EventStream.SetFormat] to
// contribute to cross-cutting files (files not owned by the synthesizer, such
// as .gitignore or lerna.json) without touching project.go.
//
// [Scaffolder] is an optional interface plugins implement alongside [Plugin]
// to write one-time files (e.g. Dockerfile, bin/app.ts). [Scaffolder.Scaffold]
// is called once during [Project.Init], never during re-synth. Implementations
// must be idempotent — check for the file before writing.
//
// [Validator] is an optional post-synthesis policy gate. [Validator.Validate]
// receives [Meta], the RC map, and the project directory after files are written.
// Return [Violation] values to report warnings or errors; errors abort the build.
//
// # Plugin awareness hierarchy
//
// Contributors must respect the awareness hierarchy to avoid brittle coupling:
//
//   - Plugins may read meta.Template and meta.Name, inspect the RC map,
//     and call stream.Events() to see what other plugins contributed.
//   - Synthesizers may read meta and rc only — never inspect the stream.
//   - The project layer is format-agnostic — it never hardcodes filenames
//     or formats owned by a synthesizer or plugin.
//
// # Cross-cutting files
//
// A cross-cutting file is one no single synthesizer owns — any contributor may
// write events to it. A contributor registers a render format with
// [EventStream.SetFormat] alongside its [EventStream.Append] calls.
// [Project.Synthesize] then renders each registered file automatically using
// one of the [FileFormat] constants ([FormatPattern], [FormatJSON], [FormatYAML],
// [FormatText]). [FormatText] writes the aggregate's "text" node as raw bytes
// with no managed-comment marker — use it for LICENSE files and other content
// that must not be modified by forglet's comment injection.
// No changes to project.go are needed when a new cross-cutting file is introduced.
//
// # Aggregate provenance
//
// After synthesis, each managed file's aggregate is written to
// .forglet/<filename>.json. Every node in the tree carries EventID, EventType,
// and Seq identifying which event (template/plugin/rc) last set it — useful for
// debugging which layer is responsible for a generated value.
package project
