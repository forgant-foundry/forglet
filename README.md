# forglet

Template-based project management with event sourcing. Unlike one-shot generators, forglet maintains an ongoing relationship with projects it creates — files can be re-synthesized and updated from the template at any time.

## How it works

Synthesis is driven by three ordered event layers that are re-derived fresh every time:

```
1. Template layer   — built-in template events (base shape)
2. Plugin layer     — platform engineering plugins (company standards)
3. RC overlay layer — .forglet.yml in your project (project customisation)
```

Later layers win for scalar fields; Object fields (like `devDependencies`) accumulate children from all layers. Because layers are re-derived at synth time, updating forglet and running `forglet synth` automatically picks up template improvements — there's no stored event log to migrate.

Each managed file has its own aggregate snapshot in `.forglet/<filename>.json` that shows which event (template/plugin/rc) last set each field.

## Create a node-ts project

```bash
forglet new node-ts my-app
```

This creates:

```
my-app/
  package.json          # managed — overwritten on synth
  tsconfig.json         # managed — overwritten on synth
  src/index.ts          # scaffold — written once, never overwritten
  .forglet.yml          # records template name + project name (add your config here)
  .forglet/
    package.json.json   # aggregate snapshot with per-field provenance
    tsconfig.json.json  # aggregate snapshot
```

## Apply an update to an existing project

When you upgrade forglet or its templates, re-run synthesis inside the project directory:

```bash
cd my-app
forglet synth
```

The template layer is re-derived from the current version of forglet, the plugin layer from whatever plugins are compiled into your binary, and `.forglet.yml` is re-read. All three layers are merged and `package.json` / `tsconfig.json` are rewritten. `src/index.ts` is never touched after initial creation.

## Customise with .forglet.yml

Create a `.forglet.yml` in your project root to override or extend the template's defaults:

```yaml
# .forglet.yml
devDependencies:
  prettier: "^3.0.0"
  eslint: "^8.0.0"

scripts:
  lint: "eslint src/"
  format: "prettier --write ."
```

The RC overlay always wins over both the template and plugins — it is the last layer applied. Run `forglet synth` after editing `.forglet.yml`.

### .forglet.yml key reference

Each template defines its own recognised keys. Unrecognised keys are silently ignored.

**Go templates** (`go`, `go-lambda`)

```yaml
module: "github.com/myorg/myapp"   # module path in go.mod
go: "1.23"                         # Go version
require:
  github.com/spf13/cobra: "v1.8.0" # additional dependencies
```

**Go workspace** (`go-workspace`)

```yaml
go: "1.23"
use:                               # additional workspace members (beyond the default)
  - services/api
  - services/worker
```

**Go Knative** (`go-knative`)

```yaml
module: "github.com/myorg/myhandler"
go: "1.23"
require:
  github.com/some/lib: "v1.0.0"
name: "my-handler"                 # Knative func name in func.yaml
registry: "gcr.io/myproject"       # container registry in func.yaml
```

**Java templates** (`java`, `java-lambda`, `java-spring`)

```yaml
groupId: "com.mycompany"
version: "2.0.0-SNAPSHOT"
javaVersion: "17"
dependencies:
  "com.google.guava:guava": "33.0.0-jre"
testDependencies:
  "org.mockito:mockito-core": "5.11.0"
```

**Java multimodule** (`java-multimodule`)

Same keys as above, plus:

```yaml
modules:
  - core
  - api
  - web
```

**Node templates** (`node-ts`, `node-js`, `node-lambda`)

```yaml
dependencies:
  express: "^4.18.0"
devDependencies:
  prettier: "^3.0.0"
scripts:
  lint: "eslint src/"
  format: "prettier --write ."
allowScripts:                      # @lavamoat/allow-scripts configuration
  "esbuild": true
```

**Cross-cutting keys** (all templates, interpreted by plugins)

```yaml
git: true                              # GitPlugin: .gitignore with template-appropriate patterns
gitignore:                             # GitPlugin: additional patterns appended after template defaults
  - ".env.local"
  - "secrets.json"

github: true                           # GitHubPlugin: CI workflow on push/PR to main
github:
  ci: true                             # .github/workflows/ci.yml
  release: true                        # .github/workflows/release.yml (goreleaser / v*-tags)
  delivery: true                       # .github/workflows/delivery.yml + .vergant.yml
  delivery:
    kind: library                      # vergant versioning + plain GitHub release, no artifacts (default)
    kind: binary                       # cross-compile Go binaries + upload artifacts
    main: ./cmd/myapp                  # Go main package path for binary builds (default: ".")
    majorVersion: 2                    # .vergant.yml: current major version (default: 1)
    defaultBranch: develop             # .vergant.yml: trunk branch (default: main)
    supportBranchRegEx: "^release/.*" # .vergant.yml: support branch pattern
    devBranchRegEx: "^feature/(.+)$"  # .vergant.yml: dev/pre-release branch pattern
    patchBranchRegEx: "^hotfix/(.+)$" # .vergant.yml: patch/pre-release branch pattern
    mode: candidate                    # .vergant.yml: "release" or "candidate" (default: release)
  defaultBranch: "develop"             # CI trigger branch (default: main)

license: MIT                           # LicensePlugin: managed LICENSE file (shorthand)
license:                               # LicensePlugin: with year and copyright holder
  id: Apache-2.0                       # built-in: MIT, Apache-2.0, GPL-3.0, AGPL-3.0, ISC; or any custom name
  year: 2024
  author: "Acme Corp"

fly: true                              # FlyPlugin: fly.toml, deploy.yml workflow, and Dockerfile scaffold
fly:
  region: iad                          # Fly.io primary region (default: iad)
  port: 8080                           # internal HTTP port (default: 8080)
  memory: 256mb                        # VM memory (default: 256mb)
  cpus: 1                              # VM CPUs (default: 1)
  healthPath: /healthz                 # health check HTTP path (default: /healthz)
  main: ./cmd/myapp                    # Go main package path for Dockerfile CMD (default: .; Go only)
  branch: main                         # deploy trigger branch (default: main)
```

## Plugins

### Default binary

The stock `forglet` binary ships with these plugins registered:

**GitPlugin** — synthesizes a `.gitignore` appropriate for the active template. Activate per-project with `git: true` in `.forglet.yml`. Template-aware: selects pattern sets for node, go, and java templates.

**GitHubActionsPlugin** — synthesizes GitHub Actions workflow files. Activate per-project with `github:` in `.forglet.yml`. Supports CI, goreleaser-based release, and vergant-based delivery workflows.

**LicensePlugin** — synthesizes a managed `LICENSE` file from built-in SPDX texts. Activate per-project with `license:` in `.forglet.yml`.

**NpmScriptPolicyValidator** — enforces that `@lavamoat/allow-scripts` is configured in node-ts projects. Runs automatically after every synthesis; no activation required.

### Available for custom binaries

The following plugins are included in the repository and can be registered in a custom binary.

| Package | Type | What it does |
|---|---|---|
| `internal/plugins/cdk` | Plugin + Scaffolder | AWS CDK devDependencies, `cdk.json`, and `bin/app.ts` + `lib/stack.ts` scaffolds for `node-ts` |
| `internal/plugins/fly` | Plugin + Scaffolder | Fly.io deployment: `fly.toml`, `.github/workflows/deploy.yml`, and a language-appropriate `Dockerfile` scaffold (Go multi-stage, Java Maven, Node-TS with build step, Node-JS production-only) |
| `internal/plugins/knative` | Plugin + Scaffolder | `node-ts`: `.knative/service.yaml`, `.dockerignore`, `Dockerfile` scaffold; `go-knative` (via `knative.NewGo()`): `func.yaml`, `Dockerfile`, handler scaffolds |
| `internal/plugins/lerna` | Plugin | `lerna.json` and `lerna` devDependency for `node-ts`; pair with WorkspacesPlugin |
| `internal/plugins/workspaces` | Plugin | `private: true` and `workspaces: ["packages/*"]` for `node-ts` |

Register any combination in a custom `main.go`:

```go
package main

import (
    "fmt"
    "os"

    "github.com/forgant-foundry/forglet/cmd/forglet/commands"
    "github.com/forgant-foundry/forglet/internal/plugins/cdk"
    "github.com/forgant-foundry/forglet/internal/plugins/git"
    "github.com/forgant-foundry/forglet/internal/plugins/lerna"
    "github.com/forgant-foundry/forglet/internal/plugins/workspaces"
)

func main() {
    commands.RegisterPlugin(git.New(), workspaces.New(), lerna.New(), cdk.New())
    if err := commands.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

## Implement a plugin

A plugin is a Go package that implements `project.Plugin`. Plugins receive both the project `Meta` (including the template name) and the full RC map, so they can be both template-aware and conditionally activated by per-project config:

```go
package myplugin

import (
    "encoding/json"

    "github.com/forgant-foundry/eventing"
    "github.com/forgant-foundry/forglet/internal/project"
)

type StandardsPlugin struct{}

func (p *StandardsPlugin) Weave(meta project.Meta, rc map[string]any, stream *project.EventStream) error {
    // Template-aware: only add ESLint config for node-ts projects.
    if meta.Template != "node-ts" {
        return nil
    }
    payload, _ := json.Marshal(map[string]any{
        "devDependencies": map[string]any{"eslint": "^8.0.0"},
    })
    stream.Append("package.json", eventing.Event{
        ID:      "standards.lint",
        Type:    "devDependency.added",
        Seq:     1,
        Payload: payload,
    })
    return nil
}
```

`EventStream` gives plugins full control over the event order — not just append rights:

| Method | Behaviour |
|--------|-----------|
| `Append(file, events...)` | Add after all existing events |
| `InsertAfter(file, eventType, events...)` | Insert after the last event of that type; append if no match |
| `InsertBefore(file, eventType, events...)` | Insert before the first event of that type; prepend if no match |
| `Replace(file, eventType, events...)` | Remove all events of that type, insert replacement at first match position |
| `Remove(file, eventType)` | Remove all events of that type |

Plugins run after the template layer and before the `.forglet.yml` overlay, so the overlay can still override anything a plugin sets.

### Cross-cutting files

Some files — like `.gitignore` — are not owned by any one synthesizer. Instead, any plugin can contribute events to them by appending to the appropriate filename in the stream. The project layer recognises these **cross-cutting files** and renders them automatically (one pattern per line, sorted alphabetically), without the synthesizer needing to know they exist.

The built-in `internal/plugins/git` plugin demonstrates this: when `git: true` appears in `.forglet.yml`, it selects the right patterns for the project's template and appends them to `.gitignore`. No synthesizer changes are needed when a new template is added — only the plugin's pattern map needs a new entry.

### One-time scaffold files

Plugins that need to write files the developer owns (e.g. `Dockerfile`, `bin/app.ts`) implement `project.Scaffolder` alongside `project.Plugin`. `Scaffold` is called once during `forglet new` and never again during `forglet synth`, so the developer's edits are preserved.

Scaffold file content should be embedded with `//go:embed` rather than inlined as string literals. Place the files in a `scaffold/` subdirectory within your plugin package:

```go
import _ "embed"

//go:embed scaffold/app.ts
var appTSScaffold []byte

func (p *MyPlugin) Scaffold(dir string, meta project.Meta) error {
    dest := filepath.Join(dir, "bin", "app.ts")
    if _, err := os.Stat(dest); err == nil {
        return nil // already exists — never overwrite
    }
    return os.WriteFile(dest, appTSScaffold, 0644)
}
```

## Wire a plugin into a custom binary

Plugins are compiled into the forglet binary rather than loaded at runtime. Create a custom `main.go` that calls `commands.RegisterPlugin` before `commands.Execute`:

```go
// cmd/my-forglet/main.go
package main

import (
    "github.com/forgant-foundry/forglet/cmd/forglet/commands"
    "github.com/mycompany/forglet-standards/myplugin"
)

func main() {
    commands.RegisterPlugin(&myplugin.StandardsPlugin{})
    commands.Execute()
}
```

Build and distribute this binary to your engineers in place of the vanilla `forglet` binary. All `forglet new` and `forglet synth` invocations will then include the plugin's events automatically.

## Add a proprietary license

Platform teams that need a proprietary or non-SPDX license do not write a new plugin — they extend the built-in `LicensePlugin` at construction time using `WithCustom` and register the extended plugin in their custom `main.go`:

```go
// cmd/myforglet/main.go
package main

import (
    "fmt"
    "os"

    "github.com/forgant-foundry/forglet/cmd/forglet/commands"
    licenseplugin "github.com/forgant-foundry/forglet/internal/plugins/license"
)

// Embed or load the license text however suits the platform — literal string,
// //go:embed, read from a secure store at build time, etc.
const proprietaryLicense = `Proprietary Software License

Copyright (c) {{COPYRIGHT}} Acme Corp. All rights reserved.

This software and its source code are confidential and proprietary to
Acme Corp. Unauthorised copying, distribution, or use is strictly
prohibited without prior written consent from Acme Corp.
`

func main() {
    commands.RegisterPlugin(licenseplugin.New(
        licenseplugin.WithCustom("acme-proprietary", proprietaryLicense),
    ))
    // register other platform plugins...
    if err := commands.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

Projects managed by this binary then select the license in `.forglet.yml`:

```yaml
license:
  id: acme-proprietary
  year: 2024
  author: "Acme Corp"
```

The `{{COPYRIGHT}}` placeholder is replaced with `"YEAR AUTHOR"` (or just `"YEAR"` when author is omitted), identically to the built-in SPDX licenses. Custom names are matched case-insensitively. If a custom name matches a built-in SPDX identifier, the custom text wins — this lets a platform team substitute a modified MIT or Apache text when required.

The `LICENSE` file is written verbatim with no forglet managed-comment marker, so it passes GitHub's license detection and standard license tooling unchanged.


## Try it yourself

### Automated tests

The test suite covers every layer — synthesizer unit tests, `EventStream` operations, aggregate merge provenance, and full `Init` → `Synthesize` integration runs.

```bash
# Run everything
go test ./...

# Run a single package
go test ./internal/project/
go test ./internal/domains/node/
go test ./internal/domains/golang/
go test ./internal/domains/java/
go test ./internal/plugins/git/
go test ./internal/plugins/fly/

# Run a single test by name
go test -run TestSynthesizeIsIdempotent ./internal/project/

# Keep the temp directories that filesystem tests create so you can inspect
# the generated files and .forglet/*.json aggregate snapshots
FORGLET_KEEP_TEMP=1 go test -v ./internal/project/
```

Integration tests use `testutil.TempDir(t)`, which prints the preserved path when `FORGLET_KEEP_TEMP=1` is set — follow that path to see exactly what the synthesizer wrote.

### Driving the CLI by hand

The fastest way to feel the three-layer model is to build the binary and run it against a throwaway directory inside the repo:

```bash
# 1. Build the binary at the repo root
go build -o forglet ./cmd/forglet

# 2. Create a scratch workspace that git will ignore
mkdir -p .scratch && cd .scratch

# 3. Create a new project from a template
../forglet new node-ts hello

# 4. Inspect what was generated
cd hello
ls -la
cat package.json
cat .forglet.yml                    # records template + project name
cat .forglet/package.json.json      # aggregate snapshot with per-field provenance
```

Every field in `.forglet/package.json.json` carries an `EventType`, `EventID`, and `Seq` identifying which layer last set it — this is the clearest way to see the three-layer model in action.

Now exercise the RC overlay and re-synth:

```bash
# Still in .scratch/hello
cat > .forglet.yml <<'YAML'
devDependencies:
  prettier: "^3.0.0"
scripts:
  format: "prettier --write ."
git: true
YAML

../../forglet synth

cat package.json                    # prettier + format script now present
cat .gitignore                      # GitPlugin wrote node-ts patterns
cat .forglet/package.json.json      # prettier's provenance is the rc overlay
```

Because all three layers re-derive on every synth, editing `.forglet.yml` and re-running `forglet synth` is the full update loop — there is no event log to migrate.

Try the other templates the same way:

```bash
cd ../..              # back to .scratch
../forglet new go my-go-app
../forglet new go-workspace my-workspace
```

When you're done, `rm -rf .scratch` wipes every scratch project in one go. `.scratch/` is just a convention — it is not special-cased by forglet, so add it to your local `.git/info/exclude` if you want git to ignore it.

## Templates

| Template            | Managed files                     | Scaffolded (once)                                           |
|---------------------|-----------------------------------|-------------------------------------------------------------|
| `go`                | `go.mod`                          | `main.go`                                                   |
| `go-workspace`      | `go.work`                         | `<name>/go.mod`, `<name>/main.go`                           |
| `go-lambda`         | `go.mod`, `Makefile`              | `main.go`                                                   |
| `go-knative`        | `go.mod`, `func.yaml`             | `handle.go`, `main.go`, `handle_test.go`                    |
| `java`              | `pom.xml`                         | `App.java`, `AppTest.java`                                  |
| `java-multimodule`  | `pom.xml`                         | `<module>/pom.xml`, `<module>/src/{main,test}/java/…`       |
| `java-lambda`       | `pom.xml`                         | `Handler.java`, `HandlerTest.java`                          |
| `java-spring`       | `pom.xml`                         | `Application.java`, `HelloController.java`, `application.properties`, `ApplicationTest.java` |
| `node-ts`           | `package.json`, `tsconfig.json`   | `src/index.ts`                                              |
| `node-js`           | `package.json`                    | `src/index.js`                                              |
| `node-lambda`       | `package.json`, `tsconfig.json`   | `packages/handler/index.ts`, `packages/handler/package.json` |

Cross-cutting files (e.g. `.gitignore`) are managed by plugins, not by synthesizers, and work across all templates.
