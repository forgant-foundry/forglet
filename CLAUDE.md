# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Project Is

Forglet is a Go CLI tool for template-based project management with event sourcing, inspired by Projen. Unlike one-shot generators, it maintains an ongoing relationship with projects it creates — files can be re-synthesized and updated from templates at any time, including when templates are upgraded.

## Commands

```bash
# Build
go build -o forglet ./cmd/forglet

# Test all packages
go test ./...

# Run a single test
go test -run TestName ./internal/project/
go test -run TestName ./internal/domains/node/
go test -run TestName ./internal/domains/golang/
go test -run TestName ./internal/plugins/git/
go test -run TestName ./internal/plugins/fly/

# Inspect output during test development
FORGLET_KEEP_TEMP=1 go test -v ./...
```

## Architecture

### Three-Layer Event Model

Every synthesis replays three ordered event layers into per-file aggregates. Later layers win for scalar fields; Object fields (like `devDependencies`) accumulate children from all layers.

```
1. Template layer   — synthesizer.InitializeEvents()   (base shape)
2. Plugin layer     — each registered plugin's Weave() (platform standards)
3. RC overlay layer — .forglet.yml in project root     (project customisation)
```

All three layers are re-derived at synth time — none are stored persistently. This means template upgrades, plugin updates, and `.forglet.yml` edits all take effect automatically on the next `forglet synth`.

### One Aggregate Per Managed File

Each output file has its own `eventing.Aggregate` from `github.com/forgant-foundry/eventing`. Event payloads (`json.RawMessage`) map 1:1 to the target file's structure — payload keys are the file's keys, with no translation layer. The aggregate is built by replaying all three layers' events in sequence; each node in the resulting tree carries per-field provenance (EventType, EventID, Seq) identifying which event last set it.

After synthesis, each file's aggregate is serialised to `.forglet/<filename>.json` so you can inspect provenance.

### Deep Merge Semantics

`BuildAggregates` in `internal/project/project.go` applies per-event mini-merges via `applyAll`:

- Each event is treated as its own mini-layer and deep-merged into the accumulating aggregate
- **Object fields** (e.g. `devDependencies`, `scripts`): children accumulate from all events — every event contributes its children
- **Scalar fields**: last event wins (last layer in the three-layer ordering)

This means multiple plugins (and the template) can all contribute to `devDependencies` without overwriting each other.

### Synthesizer Interface

```go
type Synthesizer interface {
    InitializeEvents(name string) (map[string][]eventing.Event, error)
    OverlayEvents(rc map[string]any) (map[string][]eventing.Event, error)
    Synthesize(dir string, aggregates map[string]*eventing.Aggregate) error
}
```

`Synthesize` receives the fully-merged aggregates and writes output files. Use `agg.ToJSON()` for JSON files, `agg.ToYAML()` for YAML files.

**`PostSynthesizer`** — an optional interface a synthesizer can implement to run post-synthesis operations that depend on the written files being present on disk:

```go
type PostSynthesizer interface {
    PostSynthesize(dir string) error
}
```

`Project.Synthesize` calls `PostSynthesize` immediately after `Synthesize` if the synthesizer implements it. Critically, it is NOT called when tests invoke `Synthesize` directly — keeping unit tests free of network and toolchain dependencies. The Go domain synthesizers (`Flat`, `Lambda`, `GoKnative`) implement it to run `go mod tidy`; `Workspace` implements it to run `go work sync`. Both handle the read-only permissions that `WriteManaged` applies before and after the command runs.

### Plugin Architecture

Plugins are Go packages compiled into a custom forglet binary. They implement `project.Plugin`:

```go
type Plugin interface {
    Weave(meta Meta, rc map[string]any, stream *EventStream) error
}
```

`meta` carries the template name and project name; `rc` is the full parsed `.forglet.yml` map. Together they let a plugin be both template-aware (select behaviour by `meta.Template`) and project-config-aware (activate conditionally via rc keys).

Platform engineers build a custom `main.go` that calls `commands.RegisterPlugin` before `commands.Execute`. The `cmd/forglet/commands/plugins.go` file holds the registered plugin slice; both `new` and `synth` commands pass it to `project.New(...).WithPlugins(registeredPlugins...)`.

#### Optional Plugin Interfaces

**`Scaffolder`** — plugins that need to write one-time files implement this alongside `Plugin`:

```go
type Scaffolder interface {
    Scaffold(dir string, meta Meta) error
}
```

`Scaffold` is called once during `Init`, after synthesis, and never during `Synthesize`. Implementations must be idempotent: check whether the file already exists before writing. Use this for files that belong to the developer (e.g. `bin/app.ts`, `Dockerfile`) — files that should never be overwritten once customised.

### Plugin Awareness Hierarchy

Contributors operate within a strict awareness hierarchy. Violating it creates coupling that makes the system brittle.

| Layer | Aware of |
|-------|----------|
| **Plugin** | Project meta (`meta.Name`, `meta.Template`), rc config, EventStream contents (can inspect other plugins' contributions), other registered plugins |
| **Domain (Synthesizer)** | Project meta and rc only |
| **Project layer** | Neither domains nor plugins by name |

Consequences:
- A plugin can check `meta.Template` to behave differently per domain
- A plugin can inspect `stream.Events()` to detect whether another plugin has already contributed to a file, and adjust accordingly (e.g. GitPlugin adding Knative-specific patterns when it detects Knative events)
- A synthesizer never needs to change when a new plugin is added
- The project layer never hardcodes plugin-owned filenames or formats

### Cross-Cutting Files

No contributor "owns" a cross-cutting file — any plugin, domain, or other contributor may write events to any file. The project layer is format-agnostic: it renders only what contributors declare on the stream.

**How it works:**

A contributor calls `stream.SetFormat(filename, format)` alongside its `stream.Append` calls. After all `Weave` calls, `Project.Synthesize` reads `stream.Formats()` and renders each aggregate that has a registered format.

```go
func (p *MyPlugin) Weave(meta project.Meta, rc map[string]any, stream *project.EventStream) error {
    stream.SetFormat(".gitignore", project.FormatPattern)
    stream.Append(".gitignore", eventing.Event{ /* patterns */ })

    stream.SetFormat("config.json", project.FormatJSON)
    stream.Append("config.json", eventing.Event{ /* config */ })
    return nil
}
```

**Supported formats:**

| Constant | Rendered as |
|----------|-------------|
| `project.FormatPattern` | One active key per line, `#` managed comment header |
| `project.FormatJSON` | Pretty-printed JSON object, `//` managed comment key |
| `project.FormatYAML` | YAML document, `#` managed comment header |
| `project.FormatText` | Raw string value of the aggregate's `text` node; **no managed comment** (use for files where comment syntax would corrupt content, e.g. `LICENSE`) |
| `project.FormatTOML` | TOML document, `#` managed comment header; uses `agg.ToTOML()` from the eventing library |

`SetFormat` is last-write-wins. Multiple contributors may call it for the same file; since they must agree on the format (a file is either JSON or YAML, not both), conflicts indicate a design error and will surface in tests.

Adding a new cross-cutting file requires **no changes to `project.go`** — just `SetFormat` + `Append` in any contributor.

The `internal/plugins/git` package is the reference for a conditional pattern file: `GitPlugin.Weave` checks `rc["git"].(bool)` before contributing, so no `.gitignore` aggregate is built when git is disabled and the file is simply not written.

### EventStream (plugin weaving)

`EventStream` in `internal/project/stream.go` wraps per-file event sequences with positional manipulation and format registration. Plugins have full control over event order, not just append rights. After all mutations, `Events()` re-normalises seq numbers (position 0 → seq 1).

| Method | Behaviour |
|--------|-----------|
| `Append(file, events...)` | Add after existing events |
| `InsertAfter(file, type, events...)` | After last match; append if no match |
| `InsertBefore(file, type, events...)` | Before first match; prepend if no match |
| `Replace(file, type, events...)` | Remove all of type, insert at first match position |
| `Remove(file, type)` | Remove all events of that type |
| `SetFormat(file, format)` | Register render format for a cross-cutting file |

### `.forglet.yml` — Template-Aware Configuration

The rc file format is **template-aware**: the synthesizer defines what keys it recognises and what those keys do. The key in the rc file might generate events across multiple files or enforce constraints — it is not a generic key-value system.

```yaml
# .forglet.yml

# Template-aware keys — interpreted by the synthesizer's OverlayEvents:
devDependencies:        # node-ts: adds to package.json devDependencies
  prettier: "^3.0.0"
scripts:                # node-ts: adds to package.json scripts
  lint: "eslint src/"

# Cross-cutting keys — interpreted by registered plugins, not the synthesizer:
git: true               # GitPlugin: creates .gitignore with template-appropriate patterns
```

Template-aware keys are synthesizer-specific — `OverlayEvents(rc)` translates them into events for the right files. Cross-cutting keys are read by plugins; adding a new cross-cutting behaviour never requires touching a synthesizer. The RC overlay always wins over both template and plugin layers.

### Module Structure

Single `go.mod` at the root (`github.com/forgant-foundry/forglet`):
- `cmd/forglet/` — CLI entry point (Cobra); `commands/new.go` maps template names to synthesizers; `commands/plugins.go` holds `RegisterPlugin`
- `internal/project/` — `Project` type, `Synthesizer`/`Plugin`/`Scaffolder` interfaces, `EventStream` (with `SetFormat`), aggregate merge logic, cross-cutting file rendering
- `internal/domains/node/` — `node-ts` and `node-js` synthesizers (`package.json`, `tsconfig.json`, `src/index.ts` / `index.js` scaffolds)
- `internal/domains/golang/` — `go` and `go-workspace` synthesizers (`go.mod` / `go.work`, module scaffolds)
- `internal/plugins/git/` — cross-cutting `.gitignore` support, driven by `meta.Template` + `rc["git"]`
- `internal/plugins/github/` — GitHub Actions workflows (`ci.yml`, `release.yml`, `delivery.yml` via `FormatYAML`) and `.vergant.yml` (via `FormatYAML`); delivery uses vergant for branch-driven CD; supports `kind: library` for no-artifact releases; all vergant config fields (`majorVersion`, `defaultBranch`, branch regexes, `mode`) are driven from the `delivery` section of `.forglet.yml`
- `internal/plugins/license/` — managed `LICENSE` file via `FormatText`; built-in SPDX texts (MIT, Apache-2.0, GPL-3.0, AGPL-3.0, ISC); extended by platform teams via `WithCustom`
- `internal/plugins/workspaces/` — npm workspaces (`private: true`, `workspaces: ["packages/*"]`)
- `internal/plugins/lerna/` — Lerna monorepo (`lerna.json` via `FormatJSON`, `lerna` devDependency)
- `internal/plugins/cdk/` — AWS CDK (`cdk.json` via `FormatJSON`, CDK devDependencies, scaffolds `bin/app.ts` + `lib/stack.ts`)
- `internal/plugins/knative/` — Knative Serving; `Plugin` (node-ts): `.knative/service.yaml` via `FormatYAML`, `.dockerignore` via `FormatPattern`, scaffolds `Dockerfile`; `GoPlugin` (go-knative): `func.yaml` via `FormatYAML`, scaffolds `Dockerfile`, `handle.go`, `main.go`, `handle_test.go`
- `internal/plugins/fly/` — Fly.io deployment; `fly.toml` via `FormatTOML`, `.github/workflows/deploy.yml` via `FormatYAML`, scaffolds a language-appropriate `Dockerfile` once (Go multi-stage, Java Maven, Node-TS with build step, Node-JS production-only); activated by `fly: true` or `fly: {region, port, memory, cpus, healthPath, main, branch}` in `.forglet.yml`; `main` is Go-only
- `internal/testutil/` — shared `TempDir` helper

### Testing Conventions

Always write tests in the same pass as implementation. Use `testutil.TempDir(t)` for all filesystem-touching tests. Set `FORGLET_KEEP_TEMP=1` to preserve temp dirs for manual inspection.

Test layers:
- **Unit tests** — test `InitializeEvents`, `OverlayEvents`, `BuildAggregates`, and `EventStream` operations in isolation
- **Integration tests** — `project.Init` → assert output files → `project.Synthesize` again → assert idempotent
- **Provenance tests** — after `BuildAggregates`, assert fields carry the expected `EventType`

### Adding a New Domain

1. Add a package under `internal/domains/<name>/`
2. Implement `project.Synthesizer` (all three methods); use `makeEvents`-style pattern with `json.RawMessage` payloads
3. Optionally implement `project.PostSynthesizer` (`PostSynthesize` method) if the domain needs post-synthesis toolchain operations (e.g. dependency resolution)
4. Register the template name in `cmd/forglet/commands/new.go`

### Adding a New Plugin

1. Add a package under `internal/plugins/<name>/`
2. Implement `project.Plugin` (`Weave` method)
3. For each cross-cutting file the plugin contributes to, call `stream.SetFormat(filename, format)` alongside `stream.Append` — do not add filenames to `project.go`
4. Optionally implement `project.Scaffolder` (`Scaffold` method) for one-time files
5. For scaffold file content, embed files from a `scaffold/` subdirectory using `//go:embed` rather than inlining content as Go string literals — this keeps scaffolds readable and diffable:
   ```go
   //go:embed scaffold/handler.go
   var handlerScaffold []byte
   ```
6. Register via `commands.RegisterPlugin` in the custom `main.go`

The plugin must not assume it is the only contributor to any file. Check `meta.Template` for domain-awareness and inspect `stream.Events()` for plugin-awareness.

## Evolution

### Mutual Client Relationship with vergant

Forglet and vergant are mutual clients of each other. This relationship is not incidental — it is the primary integration test for both projects.

**Vergant is a client of forglet.** Vergant is a Go project managed by forglet: it carries a `.forglet.yml` at its root and its scaffold is synthesized by forglet's `go` domain. When the Go synthesizer or any plugin used by vergant changes, re-synthesizing vergant is the live integration test. Vergant's `.forglet.yml` is the sharpest feedback on whether the Go domain serves real projects well.

**Forglet is a client of vergant.** Forglet uses vergant for its own release versioning. Forglet's branch-driven release lifecycle — dev builds on feature branches, candidates on integration, releases on main — is managed by vergant. Every forglet release is tagged by vergant. When vergant's branching strategy or tagging behaviour changes, forglet's release pipeline is the live target.

### Implications

- Changes to the `go` synthesizer should be validated by running `forglet synth` in the vergant repo before shipping.
- Changes to vergant's core versioning logic should be validated by running vergant against the forglet repo before shipping.
- Features each project needs from the other are the highest-signal driver of evolution: if managing vergant exposes a gap in forglet's Go support, that gap is real. If versioning forglet exposes a gap in vergant's strategy logic, that gap is real.
- The `internal/plugins/github` delivery workflow (`delivery: true` in `.forglet.yml`) is the integration point: it wires vergant into a managed project's CI pipeline so that `forglet new go myproject` + enabling delivery bootstraps both the project scaffold and its release pipeline in one operation.

## Design Notes

### The Managed-File Marker and Split Ownership in Go Projects

When forglet synthesizes a Go project's `go.mod`, it writes a comment at the top of the file:

```
// ~~ Generated by forglet. To modify, edit .forglet.yml and run 'forglet synth' ~~
```

The instruction is correct — but only for the direct dependencies. The `require` block forglet writes contains only the dependencies declared in `.forglet.yml`. The Go toolchain adds a second `require` block for indirect (transitive) dependencies via `go mod tidy`.

Forglet now runs `go mod tidy` automatically via `PostSynthesize` on every synth. The indirect block is populated without any manual step. The marker's guidance ("edit `.forglet.yml` and run `forglet synth`") is now accurate for the full file — direct deps are managed via `.forglet.yml`, indirect deps are resolved automatically on each synth.

The remaining nuance: a developer who manually edits the indirect block will have those edits overwritten on the next synth, since `go mod tidy` re-derives the full transitive graph from scratch. This is correct behaviour — indirect deps are not yours to manage.

*Discovered during the first live `forglet synth` run on the vergant project. Resolved by introducing `PostSynthesizer`.*

## Documentation Maintenance

### Keep docs current when you change code

Whenever you make an implementation change, consider whether documentation needs to follow. Three places to check:

1. **`doc.go` files** — every package under `internal/` and `internal/plugins/` has one. If you add, rename, or remove a type, interface, format constant, or RC key, update the corresponding doc.go. These are the primary reference for developers implementing the same interface in other packages.

2. **`README.md`** — targets both tool users and tool developers. Update it when you change the public RC key surface (`.forglet.yml` key reference), add or remove plugins from the default binary, change a template's managed or scaffolded files, or change any workflow a user would follow.

3. **`CLAUDE.md`** — this file. Update it when you discover a design constraint, learn *why* a decision was made, or clarify a convention that isn't obvious from reading the code. The goal is that the next conversation can start from a complete picture.

### Capture architectural decisions as they emerge

As we work together, watch for moments when:
- A decision involves a trade-off that would surprise a future reader (record the why, not just the what)
- A convention is established that applies beyond the current task (e.g. a new interface pattern, a new plugin capability)
- An error or wrong turn reveals a constraint worth remembering (like the KnativeFunc → GoKnative rename, or the FormatText requirement for LICENSE)

When you notice one, update CLAUDE.md in the same commit as the code — don't defer it.
