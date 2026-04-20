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

`Synthesize` receives the fully-merged aggregates and writes output files. Use `agg.ToJSON()` for JSON files.

### Plugin Architecture

Plugins are Go packages compiled into a custom forglet binary. They implement `project.Plugin`:

```go
type Plugin interface {
    Weave(projectName string, stream *EventStream) error
}
```

Platform engineers build a custom `main.go` that calls `commands.RegisterPlugin` before `commands.Execute`. The `cmd/forglet/commands/plugins.go` file holds the registered plugin slice; both `new` and `synth` commands pass it to `project.New(...).WithPlugins(registeredPlugins...)`.

### EventStream (plugin weaving)

`EventStream` in `internal/project/stream.go` wraps `map[string][]eventing.Event` with positional manipulation. This gives plugins full control over event order, not just append rights. After all mutations, `Events()` re-normalises seq numbers (position 0 → seq 1) to avoid conflicts.

| Method | Behaviour |
|--------|-----------|
| `Append(file, events...)` | Add after existing events |
| `InsertAfter(file, type, events...)` | After last match; append if no match |
| `InsertBefore(file, type, events...)` | Before first match; prepend if no match |
| `Replace(file, type, events...)` | Remove all of type, insert at first match position |
| `Remove(file, type)` | Remove all events of that type |

### `.forglet.yml` — Template-Aware Configuration

The rc file format is **template-aware**: the synthesizer defines what keys it recognises and what those keys do. The key in the rc file might generate events across multiple files or enforce constraints — it is not a generic key-value system.

```yaml
# .forglet.yml (node-ts supported keys)
devDependencies:
  prettier: "^3.0.0"
scripts:
  lint: "eslint src/"
```

Each synthesizer implements `OverlayEvents(rc)` to translate these keys into events. The RC overlay always wins over template and plugin layers.

### Module Structure

Single `go.mod` at the root (`github.com/forgant-foundry/forglet`):
- `cmd/forglet/` — CLI entry point (Cobra); `commands/new.go` maps template names to synthesizers; `commands/plugins.go` holds `RegisterPlugin`
- `internal/project/` — `Project` type, `Synthesizer` and `Plugin` interfaces, `EventStream`, aggregate merge logic
- `internal/domains/node/` — `node-ts` synthesizer (`package.json`, `tsconfig.json`, `src/index.ts` scaffold)
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
3. Register the template name in `cmd/forglet/commands/new.go`
