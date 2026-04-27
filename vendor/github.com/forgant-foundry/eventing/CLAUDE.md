# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

A Go library for domain event sourcing, intended for use in project templating tooling (similar to Projen). A series of `Event` values (JSON transfer objects) are applied to an `Aggregate`, which builds a node tree where each node tracks its current value and the event that last set it (provenance). The aggregate can be flattened to various config file formats for output.

## Commands

```bash
go test ./...                        # run all tests
go test ./... -run TestName          # run a single test
go test ./... -v                     # verbose output
go test ./... -update                # regenerate golden files
go build ./...                       # build
```

## Architecture

Two types, one package (`eventing`):

**`Event`** (`event.go`) — a JSON transfer object with `ID`, `Type` (required), `Seq int64`, and `Payload` (`json.RawMessage`). The payload must be a JSON object (key/value pairs). `Seq` is a monotonic sequence number: use `1, 2, 3...` for synthetic ordering or `time.Now().UnixNano()` for time-ordered domains. `Event.Time()` recovers the time from a nanosecond-based `Seq`.

**`Aggregate`** (`aggregate.go`) — built by calling `Apply(*Event)` one or more times in chronological order. The root of a tree of `Node` values; it carries event provenance for the last event applied but has no name. Each `Node` holds:

- `Name string` — empty for array items and the aggregate root
- `Value any` — one of: scalar (`string`, `int64`, `float64`, `bool`, `nil`), `Object`, or `Array`
- `EventID`, `EventType`, `Seq` — provenance of the event that last set this node
- `Status NodeStatus` — `""` (active), `"deleted"`, or `"moved"`
- `MovedTo []string` — full target path when `Status == "moved"`

**`Object`** (`[]*Node` with named children) — equivalent to a JSON object. An empty `Object{}` represents `{}`. Object vs Array is determined by whether children have `Name` set.

**`Array`** (`[]*Node` with unnamed children) — equivalent to a JSON array. An empty `Array{}` represents `[]`.

Top-level and nested Object children are always sorted alphabetically by name for deterministic output.

### Special payload values

| Payload value | Effect |
|---|---|
| `null` | Node is marked `NodeDeleted`; remains in tree with `Value: nil` |
| `{"__moveTo": ["target"]}` | Same-level move: source marked `NodeMoved`, value copied to `"target"` |
| `{"__moveTo": ["a", "b", "c"]}` | Cross-level move: source marked `NodeMoved`, value placed at `a.b.c`; intermediate Object nodes created as needed |

### Flattening

All flattening methods strip provenance and return only values. Deleted and moved nodes are excluded. All text-based formats sort keys alphabetically for deterministic output. Complex values (nested objects, arrays) fall back to JSON encoding within flat formats (dotenv, properties, INI).

| Method | Format |
|--------|--------|
| `ToJSON()` | JSON |
| `ToYAML()` | YAML |
| `ToTOML()` | TOML |
| `ToXML(root string)` | XML — caller names the root element |
| `ToDotenv()` | dotenv — strings with spaces/special chars are quoted |
| `ToProperties()` | Java `.properties` — escapes `\n`, `\r`, `\t`, backslash |
| `ToINI()` | INI — emits a `[default]` section header |
| `ToHCL()` | HCL — strings double-quoted, numbers/bools unquoted |

## Testing

Unit tests (`aggregate_test.go`) cover provenance behavior, deletions, moves, and error cases. Format output is tested via golden files in `testdata/`.

Each scenario directory under `testdata/` contains:
- `events.json` — ordered array of `Event` objects to apply
- `aggregate.json` — expected node tree with full provenance
- `flat.*` — expected output for each format

To add a scenario: create a new directory under `testdata/`, add `events.json`, then run `go test -update` to generate the golden files.

When output format changes intentionally, run `go test -update`, review the diff, and commit.

## Releases

Releases are automated via `.github/workflows/delivery.yml`. Pushing to `main` triggers vergant, which computes the next semantic version and cuts a tag; the workflow then creates a GitHub release with generated notes.

Follow semantic versioning (`vMAJOR.MINOR.PATCH`):

| Change | Bump |
|--------|------|
| Bug fix, no API change | patch |
| New exported symbol, backwards-compatible | minor |
| Breaking change to any exported type or method | major |

Before merging to `main`:

- All tests pass: `go test ./...`
- Golden files are current (no uncommitted diffs after `go test -update`)
- `go.mod` and `go.sum` are committed
- CLAUDE.md and README.md reflect any API changes

**Major version note:** v2+ releases require updating the module path in `go.mod` and all import paths to `github.com/forgant-foundry/eventing/v2`. The library is currently at v1, so no path change is needed.

## Dependencies

- `gopkg.in/yaml.v3` — YAML serialization
- `github.com/BurntSushi/toml` — TOML serialization
- `github.com/stretchr/testify` — test assertions (test only)
