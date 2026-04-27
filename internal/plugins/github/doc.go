// Package github provides a forglet plugin that synthesizes GitHub Actions
// workflow files appropriate for the active template.
//
// Activate in .forglet.yml:
//
//	github: true                    # CI workflow only (push + PR on main)
//	github:
//	  ci: true                      # .github/workflows/ci.yml
//	  release: true                 # .github/workflows/release.yml (goreleaser / v*-tags)
//	  delivery: true                # .github/workflows/delivery.yml (vergant / binary, default)
//	  delivery:
//	    kind: library               # vergant versioning + plain GitHub release, no artifacts
//	  defaultBranch: "develop"      # override the CI trigger branch (default: main)
//
// All workflow files are cross-cutting managed files rendered via FormatYAML.
// Content is template-aware:
//
//   - go/*:        go test ./..., goreleaser release, cross-compile binaries
//   - java/*:      mvn test, mvn package + action-gh-release / mvn versions:set + GitHub release
//   - node-ts/js:  npm ci + npm test, npm run build + action-gh-release / npm version + npm publish
//   - node-lambda: npm ci + npm run build (CI only; delivery not applicable)
//
// Unknown templates are ignored.
//
// # Delivery workflow (delivery: true)
//
// Emits .github/workflows/delivery.yml — a branch-driven CD pipeline using
// vergant for versioning. A single workflow fires on every branch push:
// vergant tags the commit, then conditional release steps run only when the
// tag carries the r* release prefix.
//
// kind: binary (default) — build and upload release artifacts:
//   - Go: cross-compile to linux/darwin/windows, archive as tar.gz/zip, upload via gh
//   - Node: npm version + npm publish to the npm registry (requires NPM_TOKEN secret)
//   - Java: mvn versions:set + mvn package, upload target/*.jar via gh
//
// kind: library — vergant versioning only, no artifact build or upload:
//   - All groups: gh release create with --generate-notes, no files attached
//
// Non-Go templates install vergant via go install; Go is pre-installed on
// GitHub Actions runners. The boolean shorthand (delivery: true) is equivalent
// to delivery: { kind: binary }.
//
// Register in a custom binary:
//
//	commands.RegisterPlugin(github.New())
package github
