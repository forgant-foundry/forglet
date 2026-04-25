# eventing

A Go library for domain event sourcing. Apply a sequence of events to build an aggregate node tree where each node tracks both its current value and the event that last set it. Flatten the aggregate to any of eight config file formats.

## Install

This library is hosted under a private GitHub org. One-time setup for any consuming project:

**1. Exclude from the public module proxy**

```bash
go env -w GOPRIVATE=github.com/forgant-foundry/*
```

**2. Rewrite HTTPS fetches to use your SSH identity**

Add to `~/.gitconfig` (adjust the SSH host alias to match your `~/.ssh/config`):

```ini
[url "git@forgant.github.com:forgant-foundry/"]
    insteadOf = https://github.com/forgant-foundry/
```

**3. Add the dependency**

```bash
go get github.com/forgant-foundry/eventing@latest
```

Verify with `go doc github.com/forgant-foundry/eventing`.

## Usage

```go
import "github.com/forgant-foundry/eventing"

// Define events as JSON transfer objects.
// Seq is a monotonic sequence number. Use 1, 2, 3... for synthetic ordering,
// or time.Now().UnixNano() for time-ordered domains (recover via event.Time()).
events := []*eventing.Event{
    {
        ID:      "evt-1",
        Type:    "project.initialized",
        Seq:     1,
        Payload: json.RawMessage(`{"name":"my-app","port":8080,"debug":false}`),
    },
    {
        ID:      "evt-2",
        Type:    "project.configured",
        Seq:     2,
        Payload: json.RawMessage(`{"port":9090,"debug":true}`),
    },
}

// Apply events to build an aggregate
agg := eventing.NewAggregate()
for _, e := range events {
    if err := agg.Apply(e); err != nil {
        log.Fatal(err)
    }
}

// Each node records its value and the event that last set it
node, _ := agg.Node("port")
fmt.Println(node.Value)     // 9090
fmt.Println(node.EventID)   // "evt-2"

// Flatten to a config format (values only, no provenance)
yaml, _ := agg.ToYAML()
toml, _ := agg.ToTOML()
env,  _ := agg.ToDotenv()
```

## Node tree

The aggregate is a tree of `Node` values. A node's value is one of:

- A **scalar**: `string`, `int64`, `float64`, `bool`, or `nil`
- An **`Object`**: named child nodes — equivalent to a JSON object
- An **`Array`**: unnamed child nodes — equivalent to a JSON array

The aggregate root has no name but carries provenance for the last event applied.

## Deletions and moves

A `null` payload value marks a node deleted. The node remains in the tree with `Status == "deleted"` and is excluded from all flattened output.

```json
{ "role": null }
```

A `{"__moveTo": [...path]}` payload value moves a node. The source remains in the tree with `Status == "moved"` and `MovedTo` set to the full target path. The value is placed at the target; intermediate Object nodes are created as needed.

```json
{ "username": {"__moveTo": ["handle"]} }
```

```json
{ "username": {"__moveTo": ["user", "handle"]} }
```

## Supported output formats

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

All formats sort keys alphabetically for deterministic output. Deleted and moved nodes are excluded. Complex values (nested objects, arrays) fall back to JSON encoding within flat formats (dotenv, properties, INI).

## Concepts

**Event** — a JSON transfer object. The `Payload` field must be a JSON object; its keys become nodes in the aggregate tree. `Seq` orders events: use `1, 2, 3...` for synthetic sequencing or `time.Now().UnixNano()` for real-time domains — call `event.Time()` to recover the timestamp.

**Aggregate** — built by applying events in chronological order. Each node records the last value written to it and the event that wrote it, independently of other nodes. This lets you trace any value in a flattened config file back to its origin event.

When two events set the same node, the later event wins for that node — other nodes from the earlier event are unaffected.
