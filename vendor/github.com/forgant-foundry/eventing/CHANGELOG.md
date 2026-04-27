# Changelog

All notable changes to this project will be documented in this file.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-04-26

### Added

- `Event` type — JSON transfer object with `ID`, `Type`, `Seq`, and `Payload` fields. `Seq` is a monotonic sequence number; `Event.Time()` recovers a `time.Time` when `Seq` was set from `time.Now().UnixNano()`.
- `Aggregate` type — built by calling `Apply(*Event)` in chronological order. Maintains a node tree where each node independently tracks the last value written to it and the event that wrote it (provenance).
- `Node` type with `Name`, `Value`, `EventID`, `EventType`, `Seq`, `Status`, and `MovedTo` fields.
- `Object` and `Array` value types for nested structures, equivalent to JSON objects and arrays.
- Null payload values mark a node `NodeDeleted`; the node remains in the tree and is excluded from all output.
- `{"__moveTo": [...path]}` payload values move a node to a target path (`NodeMoved`); intermediate `Object` nodes are created as needed.
- Eight flattening methods on `Aggregate`: `ToJSON`, `ToYAML`, `ToTOML`, `ToXML`, `ToDotenv`, `ToProperties`, `ToINI`, `ToHCL`. All formats sort keys alphabetically for deterministic output.
- JSON Schema for `Event` (`event.schema.json`), including the `null`-delete and `__moveTo`-move payload shapes.
- Golden file test suite under `testdata/` covering eight scenarios: `delete-node`, `move-node`, `nested-arrays`, `nested-config`, `project-config`, `special-chars`, `user-lifecycle`, `value-types`.
- MIT License.
- Forglet-managed project setup: `go.mod`, `.gitignore`, `LICENSE`, `.vergant.yml`, and `.github/workflows/delivery.yml` (vergant versioning + library-mode GitHub release on tags).

[1.0.0]: https://github.com/forgant-foundry/eventing/releases/tag/r1.0.0
