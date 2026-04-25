Your remote uses the custom SSH host alias `forgant.github.com` — that tells me you have a multi-account SSH config. There are two things to wire up in the consuming project's environment.

**1. Tell Go not to use the public proxy for your private org**

```bash
go env -w GOPRIVATE=github.com/forgant-foundry/*
```

This stops Go from trying the public module proxy and checksum database for anything under your org.

**2. Tell git to use your SSH alias instead of HTTPS**

Go fetches modules via HTTPS by default, but your repo requires the `forgant.github.com` SSH alias. Add this to `~/.gitconfig`:

```ini
[url "git@forgant.github.com:forgant-foundry/"]
    insteadOf = https://github.com/forgant-foundry/
```

This rewrites any HTTPS fetch for your org to go through the right SSH identity.

**3. Add the dependency in the consuming project**

```bash
go get github.com/forgant-foundry/eventing@latest
```

**4. Import and use it**

```go
import "github.com/forgant-foundry/eventing"

agg := eventing.NewAggregate()
agg.Apply(&eventing.Event{
    ID:      "evt-1",
    Type:    "project.initialized",
    Seq:     1,
    Payload: json.RawMessage(`{"name":"my-project"}`),
})
out, _ := agg.ToYAML()
```

**Verify the setup works**

```bash
go doc github.com/forgant-foundry/eventing
```

If that prints the package overview, everything is wired correctly.
