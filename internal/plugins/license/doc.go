// Package license provides a forglet plugin that synthesizes a managed LICENSE
// file from a library of SPDX license texts.
//
// Activate in .forglet.yml:
//
//	license: MIT                   # shorthand — current year, no author line
//	license:
//	  spdx: MIT
//	  year: 2024
//	  author: "Acme Corp"
//
// Built-in SPDX identifiers (case-insensitive, common aliases accepted):
//
//	MIT          — MIT License (has {{COPYRIGHT}} placeholder)
//	Apache-2.0   — Apache License 2.0 (no copyright placeholder; put copyright in NOTICE)
//	GPL-3.0      — GNU General Public License v3 (no copyright placeholder)
//	AGPL-3.0     — GNU Affero General Public License v3 (no copyright placeholder)
//	ISC          — ISC License (has {{COPYRIGHT}} placeholder)
//
// For licenses with a {{COPYRIGHT}} placeholder, the substituted value is
// "YEAR AUTHOR" when both are provided, or just "YEAR" when author is omitted.
//
// Platform teams can add proprietary or custom license texts at construction time:
//
//	commands.RegisterPlugin(license.New(
//	    license.WithCustom("acme-proprietary-v2", proprietaryLicenseText),
//	))
//
// Custom names are matched case-insensitively and take precedence over built-ins,
// allowing a built-in text to be overridden when needed. The license text may
// contain the {{COPYRIGHT}} placeholder, which is substituted the same way.
//
// LICENSE is rendered via project.FormatText — the file contains only the
// license text, with no forglet managed-comment marker, so it passes GitHub's
// license detection and standard license tooling without modification.
//
// Register in a custom binary:
//
//	commands.RegisterPlugin(license.New())
package license
