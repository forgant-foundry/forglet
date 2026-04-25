# Changelog

## [1.0.0] - 2026-04-25

First release. Everything below was built together as a cohesive v1.

---

### Core synthesis engine

**Three-layer event model.** Every synthesis replays three ordered layers into per-file aggregates: template events (base shape), plugin events (platform standards), and RC overlay events (.forglet.yml). Layers are re-derived fresh on every `forglet synth` — there is no stored event log to migrate. This is the central design choice that separates forglet from one-shot generators: upgrading forglet and running synth is enough to pick up template improvements across all existing projects.

**Deep-merge semantics.** Scalar fields use last-layer-wins; Object fields (dependencies, scripts, properties) accumulate children from all layers. This lets multiple plugins and the template itself contribute to the same file without overwriting each other's entries.

**Per-file aggregates with provenance.** Each managed file has its own `eventing.Aggregate` written to `.forglet/<filename>.json` after synthesis. Every node in the tree carries `EventID`, `EventType`, and `Seq` identifying which layer last set it. This makes it possible to audit exactly which template, plugin, or RC key is responsible for any generated value.

**Managed vs scaffold-once files.** Managed files (e.g. `pom.xml`, `package.json`, `go.mod`) are always overwritten on synth and marked read-only between runs. Scaffold files (e.g. `src/index.ts`, `Handler.java`) are written exactly once during `forglet new` and never touched again. The distinction is intentional: managed files are the template's responsibility, scaffold files belong to the developer.

**RC overlay via `.forglet.yml`.** Per-project customisation lives in a YAML file at the project root. The synthesizer defines which keys it recognises and what they produce — there is no generic key-value pass-through. This keeps the config surface explicit and template-aware rather than open-ended.

---

### Templates

**Go** (`go`, `go-workspace`, `go-lambda`, `go-knative`)

- `go` — single-module project. Manages `go.mod`; scaffolds `main.go`.
- `go-workspace` — Go workspace (`go.work`) with a first member module. Scaffolds `<name>/go.mod` and `<name>/main.go`.
- `go-lambda` — extends `go` with `github.com/aws/aws-lambda-go`, a cross-compilation `Makefile`, and a typed `lambda.Start` handler scaffold. The Makefile targets `GOOS=linux GOARCH=amd64 CGO_ENABLED=0` to produce the `bootstrap` binary Lambda expects.
- `go-knative` — manages both `go.mod` and `func.yaml` (the Knative func CLI manifest). Scaffolds an HTTP handler, a main server, and a handler test. `func.yaml` is rendered via `agg.ToYAML()` so its shape is event-sourced like all other managed files.

**Java** (`java`, `java-multimodule`, `java-lambda`, `java-spring`)

Maven only. Gradle is intentionally excluded to keep the implementation surface small and to reflect Maven's dominance in enterprise Java environments.

- `java` — single-module Maven project. Manages `pom.xml` with JUnit Jupiter and maven-surefire-plugin; scaffolds `App.java` and `AppTest.java`.
- `java-multimodule` — parent POM (`packaging=pom`) with `<dependencyManagement>` for JUnit and `<pluginManagement>` for surefire. Each module in the `modules` rc key gets a scaffolded child `pom.xml` (with parent back-reference) and `src/{main,test}/java` layout. Defaults to a single `core` module.
- `java-lambda` — extends `java` with `aws-lambda-java-core` and `aws-lambda-java-events` compile dependencies and the maven-shade-plugin to produce a fat JAR. Scaffolds `Handler.java` implementing `RequestHandler` with record-based `Request` and `Response` inner types, and `HandlerTest.java`.
- `java-spring` — Spring Boot project using `spring-boot-starter-parent` as the parent POM. Uses `<java.version>` instead of `maven.compiler.source/target` (Spring convention). The `spring-boot-maven-plugin` has no `<version>` element — it is inherited from the parent. Scaffolds `Application.java`, `HelloController.java`, `application.properties`, and `ApplicationTest.java`.

**Node.js** (`node-ts`, `node-js`, `node-lambda`)

- `node-ts` — TypeScript project. Manages `package.json` and `tsconfig.json`; scaffolds `src/index.ts`.
- `node-js` — plain JavaScript project. Manages `package.json`; scaffolds `src/index.js`.
- `node-lambda` — TypeScript Lambda project using esbuild for bundling. The package is `private: true`; the build script bundles `packages/handler/index.ts` to `dist/handler.js`. Scaffolds a typed async handler and a per-package `package.json` under `packages/handler/`.

---

### Plugin architecture

**`Plugin` interface.** Plugins implement a single `Weave(meta Meta, rc map[string]any, stream *EventStream) error` method. They run between the template layer and the RC overlay, so the RC always wins over plugin-contributed values. Plugins receive both the project `Meta` (including template name) and the full RC map, making them template-aware and project-config-aware without needing to inspect the filesystem.

**`EventStream` with positional control.** Plugins have full control over event order — not just append rights. `Append`, `InsertBefore`, `InsertAfter`, `Replace`, and `Remove` all operate by position within each file's event slice. Seq numbers are re-normalised on `Events()` so plugins never need to manage them manually. This matters when a plugin needs to slot its events between specific template events rather than pile on at the end.

**`Scaffolder` interface.** Plugins that need to write one-time developer-owned files implement `Scaffold(dir string, meta Meta) error` alongside `Plugin`. Scaffold runs once during `forglet new`, never during `forglet synth`. Implementations are expected to be idempotent — check before writing.

**`Validator` interface.** Validators run after synthesis to assert policy. They receive `Meta`, the RC map, and the project directory. Returning a `Violation` with `SeverityError` aborts the build; `SeverityWarning` logs and continues. Validators are the right place for supply-chain policy, naming conventions, or any constraint that should be enforced globally across an organisation.

**Cross-cutting files via `SetFormat`.** No plugin or synthesizer "owns" a cross-cutting file. Any contributor calls `stream.SetFormat(filename, format)` to register a render format alongside its `stream.Append` calls. After all `Weave` calls, the project layer renders each registered file using `FormatPattern` (one key per line), `FormatJSON` (pretty-printed JSON), or `FormatYAML`. Adding a new cross-cutting file requires no changes to `project.go`.

**Plugin awareness hierarchy.** Plugins may read `meta.Template`, inspect the RC map, and call `stream.Events()` to see what other plugins have already contributed. Synthesizers may read meta and rc only — they never inspect the stream. The project layer is format-agnostic and never hardcodes filenames or formats owned by a plugin. This hierarchy prevents coupling and means a synthesizer never needs to change when a new plugin is added.

**Compiled plugins, not loaded at runtime.** Platform engineers build a custom binary by importing forglet as a Go module, calling `commands.RegisterPlugin` and `commands.RegisterValidator` before `commands.Execute`, then distributing that binary in place of the vanilla `forglet` binary. This trades runtime flexibility for type safety, compile-time validation, and a single self-contained binary — the right trade-off for an internal platform tool.

---

### Built-in plugins (default binary)

**`GitPlugin`** (`internal/plugins/git`). Synthesizes a `.gitignore` appropriate for the active template. Activated by `git: true` in `.forglet.yml`. Template-aware: selects pattern sets for node, go, and java templates; falls back to a minimal common set for unrecognised templates. Demonstrates the cross-cutting file pattern: the synthesizer has no knowledge of `.gitignore`.

**`NpmScriptPolicyValidator`** (`internal/baseunit/npm`). Enforces that `@lavamoat/allow-scripts` is configured in node-ts projects, preventing unauthorized npm lifecycle script execution. Runs after every synthesis of a node-ts project.

---

### Available plugins for custom binaries

The following plugins are included in the repository and available for platform engineers to register in custom forglet binaries. They are not wired into the default binary because they represent optional platform choices rather than universal defaults.

**`CDKPlugin`** (`internal/plugins/cdk`). Adds AWS CDK devDependencies (`aws-cdk`, `aws-cdk-lib`, `constructs`) and a `cdk.json` to node-ts projects. Implements `Scaffolder` to write `bin/app.ts` and `lib/stack.ts` once on init.

**`KnativePlugin`** (`internal/plugins/knative`). Adds `.knative/service.yaml` and `.dockerignore` to node-ts projects. Implements `Scaffolder` to write a `Dockerfile` once on init.

**`LernaPlugin`** (`internal/plugins/lerna`). Adds `lerna.json` and the `lerna` devDependency to node-ts projects. Designed to be used alongside `WorkspacesPlugin`.

**`WorkspacesPlugin`** (`internal/plugins/workspaces`). Sets `private: true` and `workspaces: ["packages/*"]` on node-ts projects, configuring the project root as an npm workspace host.

---

### Release infrastructure

**goreleaser.** Cross-compiles binaries for linux/darwin × amd64/arm64. Version is injected at build time via `-X github.com/forgant-foundry/forglet/cmd/forglet/commands.Version={{.Version}}` so `forglet --version` always reflects the release tag.

**GitHub Actions release workflow.** Triggers on `v*` tags. Builds from the vendored module tree (`vendor/`) so the workflow requires only `GITHUB_TOKEN` — no additional secrets or private module access configuration needed. Produces release archives and a SHA-256 checksum file.
