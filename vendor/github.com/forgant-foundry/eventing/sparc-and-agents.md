This was a genuinely good session for the iterative, conversational style — design decisions like `Seq` vs `Timestamp` and the golden file test approach emerged from dialogue that's hard to front-load. But there are specific places where SPARC and agents would have improved both speed and outcomes.

---

## Where SPARC would have helped

SPARC (Specification → Pseudocode → Architecture → Refinement → Completion) front-loads design so you don't pivot mid-implementation.

**Specification phase** would have surfaced decisions we made late:
- The `Seq` vs `Timestamp` question — we changed the struct after the golden files were already generated
- The full list of output formats — we added TOML, dotenv, etc. as follow-ups rather than all at once
- The graph/nested support question — discovered after flat implementation was already in place

A good spec prompt at the start:
```
Before writing any code, produce a specification for a Go library that:
- Represents domain events as JSON transfer objects
- Builds aggregates that track per-attribute provenance
- Outputs to: JSON, YAML, TOML, XML, dotenv, .properties, INI, HCL
- Supports nested/graph values in appropriate formats
- Uses a monotonic sequence number, not a timestamp
Ask me to confirm before proceeding.
```

The `Plan` subagent (`/plan` or `Agent` with `subagent_type: "Plan"`) is well-suited for this — it designs before writing a line of code.

---

## Where parallel agents would have helped

The eight output formats are entirely independent. Instead of implementing them sequentially, you could spawn all eight in parallel:

```
Agent: implement ToJSON and ToYAML on Aggregate
Agent: implement ToTOML and ToXML on Aggregate  
Agent: implement ToDotenv and ToProperties on Aggregate
Agent: implement ToINI and ToHCL on Aggregate
```

Each agent works in an isolated worktree (`isolation: "worktree"`), then you merge. That's potentially an 8x speedup on the serialization work.

Similarly, the golden file scenarios (user-lifecycle, project-config, nested-config, etc.) could have been designed and generated in parallel.

---

## Where a review agent would have helped

After implementation, spawning a `general-purpose` agent to review the code independently caught the TOML integer issue (`5432.0`) before you saw it in the golden files:

```
Agent: review aggregate.go for correctness. Specifically check whether 
json.Unmarshal into `any` preserves integer types, and whether each 
To* method handles the full type set (string, int64, float64, bool, 
nil, map, slice) correctly. Report issues only.
```

That agent would have flagged the `float64` → TOML rendering problem in the spec phase, before any golden files existed.

---

## What the conversational approach did better

Honest answer: the design exploration — *why* Seq instead of Timestamp, *whether* to use golden files, *which* formats to support — benefited from dialogue. SPARC works best when requirements are reasonably clear upfront. For a library being designed from scratch with an evolving mental model, the back-and-forth was appropriate.

The practical hybrid: use conversation for design decisions, then hand off to SPARC + parallel agents once the shape is settled. In this session, that handoff point was roughly when you confirmed the golden file test approach — everything after that was mechanical enough to parallelize.
