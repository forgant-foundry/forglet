package project

// Schemer is an optional interface that plugins and synthesizers may implement
// to contribute their rc key definitions to 'forglet schema'. Implementations
// return a SchemaContribution describing the .forglet.yml keys they handle.
//
// For plugins, returned Properties are cross-cutting — valid for any template
// and merged into the top-level JSON Schema properties object.
//
// For synthesizers, returned Properties are template-specific. The schema
// assembler wraps them in an if/then block keyed on the registered template
// name, so they appear as suggestions only when that template is active.
type Schemer interface {
	RCSchema() SchemaContribution
}

// SchemaContribution holds JSON Schema property definitions for one rc contributor.
// Each entry in Properties maps an rc key name to its JSON Schema definition
// (a map[string]any conforming to JSON Schema draft-07).
type SchemaContribution struct {
	Properties map[string]any
}
