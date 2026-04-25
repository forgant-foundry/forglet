# Cutting releases

Releases are Git tags. The Go module system resolves versions from tags — no
registry, no build step, no CI required.

## Versioning rules

Follow semantic versioning: `vMAJOR.MINOR.PATCH`

| Change | Example | Bump |
|--------|---------|------|
| Bug fix, no API change | `v1.0.1` | patch |
| New exported symbol, backwards-compatible | `v1.1.0` | minor |
| Breaking change to any exported type or method | `v2.0.0` | major |

**Major version note:** Go modules require that v2+ releases change the module
import path. If you ever break the API, update `go.mod` and all import paths:
```
module github.com/forgant-foundry/eventing/v2
```
For v1 (where this library currently lives), no path change is needed.

## Checklist before tagging

- [ ] All tests pass: `go test ./...`
- [ ] Golden files are current: `go test -update` has not produced uncommitted diffs
- [ ] `go.mod` and `go.sum` are committed
- [ ] CLAUDE.md and README.md reflect any API changes
- [ ] Commit everything and push to `main`

## Creating a release on GitHub

1. Go to the repository on GitHub
2. Click **Releases** → **Draft a new release**
3. Click **Choose a tag** → type the new tag (e.g. `v1.0.0`) → **Create new tag on publish**
4. Set **Target** to `main`
5. Set the release title to the tag name: `v1.0.0`
6. Write release notes (see format below)
7. Click **Publish release**

GitHub creates the Git tag at publish time. The Go module proxy picks it up
within a few minutes.

## Release notes format

```
## What's changed

### Added
- ToTOML, ToDotenv, ToProperties, ToINI, ToHCL output formats

### Changed
- Event.Timestamp replaced by Event.Seq (int64 monotonic sequence number)

### Fixed
- JSON integers no longer rendered as floats in TOML output

## Upgrading

No import path changes. Run:

    go get github.com/forgant-foundry/eventing@v1.1.0
```

## Consuming a specific release

```bash
# Latest release
go get github.com/forgant-foundry/eventing@latest

# Specific version
go get github.com/forgant-foundry/eventing@v1.0.0

# Verify
go doc github.com/forgant-foundry/eventing
```

## Verifying the tag after publishing

```bash
go list -m github.com/forgant-foundry/eventing@latest
```

If the correct version is returned, the release is live and resolvable.
