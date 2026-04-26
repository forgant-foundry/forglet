// Package github provides a forglet plugin that synthesizes GitHub Actions
// workflow files appropriate for the active template.
//
// Activate in .forglet.yml:
//
//	github: true                    # CI workflow only (push + PR on main)
//	github:
//	  ci: true                      # .github/workflows/ci.yml
//	  release: true                 # .github/workflows/release.yml
//	  defaultBranch: "develop"      # override the CI trigger branch (default: main)
//
// Both workflow files are cross-cutting managed files rendered via FormatYAML.
// Content is template-aware:
//
//   - go/*:        go test ./..., goreleaser release
//   - java/*:      mvn test, mvn package + action-gh-release
//   - node-ts/js:  npm ci + npm test, npm run build + action-gh-release
//   - node-lambda: npm ci + npm run build, npm run build + action-gh-release
//
// Unknown templates are ignored.
//
// Register in a custom binary:
//
//	commands.RegisterPlugin(github.New())
package github
