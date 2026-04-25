// Package git provides a forglet plugin that synthesizes a .gitignore file
// appropriate for the active template.
//
// Activate by setting "git: true" in .forglet.yml. The plugin is template-aware:
// it selects a pattern set matched to meta.Template (e.g. node_modules for
// node-* templates, binaries and vendor for go-* templates). Unrecognised
// templates receive a minimal common set.
//
// .gitignore is a cross-cutting file — GitPlugin calls stream.SetFormat and
// stream.Append rather than managing the file inside a synthesizer, so no
// synthesizer needs to change when a new template is added.
//
// Register in a custom binary:
//
//	commands.RegisterPlugin(git.New())
package git
