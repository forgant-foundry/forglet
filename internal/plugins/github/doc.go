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
//	    kind: binary                # cross-compile + upload artifacts
//	    main: ./cmd/myapp           # Go main package path (default: ".")
//	    builds:                     # additional Go build variants (binary kind only)
//	      - tags: no_embeddings     #   build tags for this variant (space-separated)
//	        suffix: slim            #   artifact suffix (e.g. myapp_${SEM}_linux_amd64_slim.tar.gz)
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
//   - Go: cross-compile to linux/darwin/windows, archive as tar.gz/zip, upload via gh.
//     The optional builds array adds extra variants: each entry recompiles with the given
//     build tags and produces a parallel set of platform artifacts with the suffix
//     appended before the extension (e.g. suffix "slim" → myapp_${SEM}_linux_amd64_slim.tar.gz).
//     The binary inside every archive — standard and variant — is always named after the
//     project (e.g. myapp); the suffix appears only in the archive filename.
//     All variant artifacts are included in the shared checksums file. suffix is required
//     on each entry (entries without one are silently skipped to avoid name collisions).
//   - Node: npm version + npm publish to the npm registry (requires NPM_TOKEN secret)
//   - Java: mvn versions:set + mvn package, upload target/*.jar via gh
//
// kind: library — vergant versioning only, no artifact build or upload:
//   - All groups: gh release create with --generate-notes, no files attached
//
// When ci is also enabled alongside delivery, the delivery workflow embeds a
// test job (identical steps to ci.yml) and gates the delivery job on it via
// needs: [test]. The standalone ci.yml is still emitted for PR feedback;
// delivery.yml handles the combined test+release path for branch pushes.
//
// Non-Go templates install vergant via go install; Go is pre-installed on
// GitHub Actions runners. The boolean shorthand (delivery: true) is equivalent
// to delivery: { kind: binary }.
//
// Register in a custom binary:
//
//	commands.RegisterPlugin(github.New())
package github
