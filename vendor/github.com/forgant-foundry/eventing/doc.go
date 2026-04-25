// Package eventing provides domain event sourcing primitives for building
// aggregates from ordered sequences of events.
//
// # Overview
//
// Two types form the core of the library:
//
//   - [Event] — a JSON transfer object representing something that happened.
//     Each event carries an ID, a type, a sequence number, and a JSON object
//     payload whose keys describe what changed.
//
//   - [Aggregate] — the root of a node tree built by applying events in sequence.
//     Each [Node] in the tree records its current value and the provenance of the
//     event that last set it (ID, type, seq). Provenance is tracked independently
//     per node: a later event only updates the nodes whose keys appear in its payload.
//
// # Node tree
//
// The aggregate is a tree of [Node] values. A node's Value is one of:
//
//   - A scalar: string, int64, float64, bool, or nil.
//   - An [Object]: an ordered collection of named child nodes (like a JSON object).
//     An empty Object{} represents {}.
//   - An [Array]: an ordered collection of unnamed child nodes (like a JSON array).
//     An empty Array{} represents [].
//
// Object children are distinguished from Array children by the presence of Name.
//
// # Building an aggregate
//
//	agg := eventing.NewAggregate()
//
//	for _, e := range events {
//	    if err := agg.Apply(e); err != nil {
//	        return err
//	    }
//	}
//
//	// Inspect provenance: which event last set "role"?
//	node, ok := agg.Node("role")
//	if ok {
//	    fmt.Println(node.Value)     // current value
//	    fmt.Println(node.EventID)   // event that set it
//	}
//
// # Deletions and moves
//
// A null payload value marks a node deleted. The node remains in the tree with
// Status == NodeDeleted and is excluded from all flattened output.
//
//	// payload: {"role": null}  →  role node: Status=NodeDeleted
//
// A payload value of {"__moveTo": ["path", "to", "target"]} moves a node.
// The source remains with Status == NodeMoved and MovedTo set to the full path;
// a node is created at the target path with the source value. Intermediate Object
// nodes are created as needed. Both source and target carry the event's provenance.
// Moved nodes are excluded from flattened output.
//
//	// payload: {"username": {"__moveTo": ["handle"]}}
//	// →  username node: Status=NodeMoved, MovedTo=["handle"]
//	// →  handle node:   Value=<old username value>
//
//	// payload: {"username": {"__moveTo": ["user", "handle"]}}
//	// →  username node: Status=NodeMoved, MovedTo=["user","handle"]
//	// →  user node:     Value=Object{ handle: <old username value> }
//
// # Sequence numbers
//
// [Event.Seq] is a monotonic int64. Two patterns are supported:
//
//   - Synthetic ordering (template synthesis, tests): use 1, 2, 3…
//   - Time-ordered domains: use time.Now().UnixNano(); call [Event.Time] to
//     recover the timestamp.
//
// # Flattening to config formats
//
// The aggregate can be serialised to eight config file formats. All formats
// emit values only — provenance is stripped. Deleted and moved nodes are
// excluded. Text-based formats sort keys alphabetically for deterministic output.
//
//	json, _       := agg.ToJSON()
//	yaml, _       := agg.ToYAML()
//	toml, _       := agg.ToTOML()
//	xml, _        := agg.ToXML("aggregate")
//	env, _        := agg.ToDotenv()
//	props, _      := agg.ToProperties() // Java .properties
//	ini, _        := agg.ToINI()        // [default] section
//	hcl, _        := agg.ToHCL()
//
// JSON, YAML, TOML, and XML support full nesting. dotenv, properties, INI,
// and HCL support nesting via JSON fallback encoding for complex values.
package eventing
