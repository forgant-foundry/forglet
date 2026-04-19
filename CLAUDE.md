# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Project Is

Forglet is a Go CLI tool for template-based project management with event sourcing, inspired by Projen. Unlike one-shot generators, it maintains an ongoing template relationship — projects can be re-synthesized and upgraded from updated templates at any time.

## Commands

```bash
# Build
go build ./...
go build -o forglet ./cmd/forglet

# Test all packages
go test ./...

# Run a single test
go test -run TestName ./internal/core/
go test -run TestName ./internal/domains/node/

# Lint (if golangci-lint is available)
golangci-lint run
```

## Architecture

### Core Concepts

**Event Sourcing:** Every project change is an immutable event appended to `.forglet/events.jsonl`. Project state is always derived by replaying events — never edited directly. This provides full audit trail and reproducibility.

**Synthesizer Pattern:** Each language/framework domain implements a `Synthesizer` interface with two operations:
- `InitializeEvents()` — emits the initial events for a new project (dependencies, scripts, config)
- `Synthesize()` — reads current state and generates all managed files

### Module Structure

The project uses a Go workspace with separate modules:
- `cmd/forglet/` — CLI entry point using Cobra; `commands/register.go` maps template names to synthesizer implementations
- `internal/core/` — event types, project state management, filesystem abstraction, and the `Synthesizer` interface
- `internal/domains/{node,java,python,golang}/` — language-specific synthesizer implementations

### Key Data Flows

**New project (`forglet new <template> <name>`):**
1. `Project.Initialize()` creates `.forglet/` and appends a `project.created` event
2. The matching `Synthesizer.InitializeEvents()` appends dependency/script/config events
3. `Project.Synthesize()` replays all events to build state, then calls `Synthesizer.Synthesize()` to write files

**Re-synthesis (`forglet synth`):**
1. `Project.LoadProject()` reads `.forglet/events.jsonl` and replays events to restore state
2. `Project.Synthesize()` regenerates all managed files from current state

### Event Types

`project.created`, `dependency.added`, `script.added`, `configuration.updated`, `file.generated`

### Testing Conventions

`internal/core/testing.go` provides `NewMemoryFileSystem()` — use this for synthesizer tests to avoid touching the real filesystem.

### Adding a New Domain

Implement the `Synthesizer` interface from `internal/core/`, add a new package under `internal/domains/`, and register the template name in `cmd/forglet/commands/register.go`.
