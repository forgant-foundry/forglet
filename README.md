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

A plugin is a Go package that implements `project.Plugin`:

```go
package myplugin

import (
    "encoding/json"

    "github.com/forgant-foundry/eventing"
    "github.com/forgant-foundry/forglet/internal/project"
)

type StandardsPlugin struct{}

func (p *StandardsPlugin) Weave(_ string, stream *project.EventStream) error {
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

## Build

```bash
go build -o forglet ./cmd/forglet
go test ./...
```

Set `FORGLET_KEEP_TEMP=1` when running tests to preserve temp directories for inspection.

## Templates

| Template  | Managed files                   |
|-----------|---------------------------------|
| `node-ts` | `package.json`, `tsconfig.json` |
