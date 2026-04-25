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
  .forglet/
    project.json        # records template name
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

## Private module access

Forglet and its dependencies (`eventing`) live in private GitHub repositories. Go's toolchain needs explicit configuration to fetch them without falling back to the public module proxy.

### Local development

Add the following to your shell profile (or `.envrc` if you use direnv):

```bash
export GOPRIVATE=github.com/forgant-foundry/*
```

This tells Go not to route `forgant-foundry/*` imports through the public proxy or checksum database. Authentication is handled by git — configure it once with a [GitHub personal access token](https://github.com/settings/tokens) that has `Contents: Read` on the relevant repos:

```bash
git config --global \
  url."https://<YOUR_GITHUB_TOKEN>@github.com/forgant-foundry/".insteadOf \
  "https://github.com/forgant-foundry/"
```

After that, `go get`, `go mod tidy`, and `go build` all work without any further prompts.

### CI

In GitHub Actions, store a PAT with `repo` read scope as a repository secret named `FORGANT_TOKEN` and configure the same git URL rewrite in your workflow:

```yaml
- name: Configure private module access
  env:
    FORGANT_TOKEN: ${{ secrets.FORGANT_TOKEN }}
  run: |
    git config --global \
      url."https://${FORGANT_TOKEN}@github.com/forgant-foundry/".insteadOf \
      "https://github.com/forgant-foundry/"
```

Set `GOPRIVATE=github.com/forgant-foundry/*` and `GONOSUMDB=github.com/forgant-foundry/*` as environment variables on your build step. The release workflow in `.github/workflows/release.yml` already does this.

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
go test ./internal/plugins/git/

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
cat .forglet/project.json           # records template + project name
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

| Template       | Managed files                        | Scaffolded (once)              |
|----------------|--------------------------------------|--------------------------------|
| `node-ts`      | `package.json`, `tsconfig.json`      | `src/index.ts`                 |
| `go`           | `go.mod`                             | `main.go`                      |
| `go-workspace` | `go.work`                            | `<name>/go.mod`, `<name>/main.go` |

Cross-cutting files (e.g. `.gitignore`) are managed by plugins, not by synthesizers, and work across all templates.
