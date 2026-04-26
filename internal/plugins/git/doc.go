// Package git provides a forglet plugin that synthesizes a .gitignore file
// appropriate for the active template.
//
// Activate in .forglet.yml:
//
//	git: true
//
// When active, five system categories are always included and each can be
// suppressed individually:
//
//	forglet   — .forglet/ (aggregate snapshots; always re-derived, never commit)
//	macos     — .DS_Store, .AppleDouble, Thumbs.db, etc.
//	jetbrains — .idea/, *.iml, *.ipr, *.iws
//	vscode    — .vscode/
//	eclipse   — .classpath, .project, .settings/
//
// Plus stack-specific patterns chosen by meta.Template (Go build artifacts,
// Java target/, Node node_modules/, etc.).
//
// .gitignore is a cross-cutting file: GitPlugin calls stream.SetFormat and
// stream.Append rather than managing the file inside a synthesizer, so no
// synthesizer needs to change when new templates are added.
//
// Customise per-project in .forglet.yml:
//
//	gitignore:
//	  add:
//	    - ".env.local"    # extra patterns to include
//	  exclude:
//	    - "eclipse"       # suppress a system category by name
//	    - "vscode"
//
// Flat-list shorthand (backward compatible, equivalent to add:):
//
//	gitignore:
//	  - ".env.local"
//
// Register in a custom binary:
//
//	commands.RegisterPlugin(git.New())
package git
