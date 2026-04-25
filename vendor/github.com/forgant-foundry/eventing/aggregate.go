package eventing

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// NodeStatus describes the lifecycle state of a node.
// The zero value (NodeActive, empty string) is omitted from JSON output.
type NodeStatus string

const (
	NodeActive  NodeStatus = ""
	NodeDeleted NodeStatus = "deleted"
	NodeMoved   NodeStatus = "moved"
)

// Object is an ordered collection of named child Nodes, equivalent to a JSON object.
// An empty Object{} represents an empty object.
type Object []*Node

// Array is an ordered collection of unnamed child Nodes, equivalent to a JSON array.
// An empty Array{} represents an empty array.
type Array []*Node

// Node is a tree element carrying a value and the provenance of the event that last set it.
// Name is empty for array items; the root Aggregate also has no name.
// Value is one of: string, int64, float64, bool, nil, Object, or Array.
type Node struct {
	Name      string     `json:"name,omitempty"`
	Value     any        `json:"value"`
	EventID   string     `json:"eventId"`
	EventType string     `json:"eventType"`
	Seq       int64      `json:"seq"`
	Status    NodeStatus `json:"status,omitempty"`
	MovedTo   []string   `json:"movedTo,omitempty"`
}

// Aggregate is the root of the node tree.
// It has no name and carries provenance for the last event applied.
type Aggregate struct {
	EventID   string `json:"eventId,omitempty"`
	EventType string `json:"eventType,omitempty"`
	Seq       int64  `json:"seq,omitempty"`
	Value     Object `json:"value"`
}

// NewAggregate returns an empty aggregate.
func NewAggregate() *Aggregate {
	return &Aggregate{}
}

// Apply merges an event's JSON payload into the aggregate.
// The payload must be a JSON object. For each key:
//   - A non-null value sets or replaces the node at that key.
//   - A null value marks the node deleted (NodeDeleted); the node remains in the tree.
//   - A value of {"__moveTo": ["path", "to", "target"]} moves the node to the given
//     path (NodeMoved). Intermediate Object nodes are created if they do not exist.
func (a *Aggregate) Apply(e *Event) error {
	dec := json.NewDecoder(bytes.NewReader(e.Payload))
	dec.UseNumber()
	var raw map[string]any
	if err := dec.Decode(&raw); err != nil {
		return fmt.Errorf("unmarshal event %s payload: %w", e.ID, err)
	}
	for k, v := range raw {
		if path, ok := moveToPath(v); ok {
			a.Value = applyMove(a.Value, k, path, e)
		} else if v == nil {
			a.Value = applyDelete(a.Value, k, e)
		} else {
			a.Value = applySet(a.Value, k, v, e)
		}
	}
	sort.Slice(a.Value, func(i, j int) bool { return a.Value[i].Name < a.Value[j].Name })
	a.EventID = e.ID
	a.EventType = e.Type
	a.Seq = e.Seq
	return nil
}

func moveToPath(v any) ([]string, bool) {
	m, ok := v.(map[string]any)
	if !ok || len(m) != 1 {
		return nil, false
	}
	raw, ok := m["__moveTo"]
	if !ok {
		return nil, false
	}
	arr, ok := raw.([]any)
	if !ok || len(arr) == 0 {
		return nil, false
	}
	path := make([]string, len(arr))
	for i, item := range arr {
		s, ok := item.(string)
		if !ok {
			return nil, false
		}
		path[i] = s
	}
	return path, true
}

// Node returns the top-level child node with the given name and whether it exists.
func (a *Aggregate) Node(name string) (*Node, bool) {
	for _, n := range a.Value {
		if n.Name == name {
			return n, true
		}
	}
	return nil, false
}

func applySet(obj Object, name string, v any, e *Event) Object {
	node := &Node{
		Name:      name,
		EventID:   e.ID,
		EventType: e.Type,
		Seq:       e.Seq,
		Value:     buildNodeValue(v, e),
	}
	for i, n := range obj {
		if n.Name == name {
			obj[i] = node
			return obj
		}
	}
	return append(obj, node)
}

func applyDelete(obj Object, name string, e *Event) Object {
	node := &Node{
		Name:      name,
		EventID:   e.ID,
		EventType: e.Type,
		Seq:       e.Seq,
		Status:    NodeDeleted,
	}
	for i, n := range obj {
		if n.Name == name {
			obj[i] = node
			return obj
		}
	}
	return append(obj, node)
}

func applyMove(obj Object, from string, path []string, e *Event) Object {
	var srcValue any
	for i, n := range obj {
		if n.Name == from {
			srcValue = n.Value
			obj[i] = &Node{
				Name:      from,
				EventID:   e.ID,
				EventType: e.Type,
				Seq:       e.Seq,
				Status:    NodeMoved,
				MovedTo:   path,
			}
			break
		}
	}
	return upsertAtPath(obj, path, srcValue, e)
}

// upsertAtPath inserts or replaces a node at the given path within obj,
// creating intermediate Object nodes as needed.
func upsertAtPath(obj Object, path []string, value any, e *Event) Object {
	name := path[0]
	if len(path) == 1 {
		node := &Node{
			Name:      name,
			EventID:   e.ID,
			EventType: e.Type,
			Seq:       e.Seq,
			Value:     value,
		}
		for i, n := range obj {
			if n.Name == name {
				obj[i] = node
				return obj
			}
		}
		return append(obj, node)
	}
	// Descend into the child Object at path[0], creating it if absent.
	for i, n := range obj {
		if n.Name == name {
			child, _ := n.Value.(Object)
			child = upsertAtPath(child, path[1:], value, e)
			sort.Slice(child, func(a, b int) bool { return child[a].Name < child[b].Name })
			obj[i] = &Node{
				Name:      name,
				EventID:   e.ID,
				EventType: e.Type,
				Seq:       e.Seq,
				Value:     child,
			}
			return obj
		}
	}
	child := upsertAtPath(Object{}, path[1:], value, e)
	sort.Slice(child, func(a, b int) bool { return child[a].Name < child[b].Name })
	return append(obj, &Node{
		Name:      name,
		EventID:   e.ID,
		EventType: e.Type,
		Seq:       e.Seq,
		Value:     child,
	})
}

func buildNodeValue(v any, e *Event) any {
	switch val := v.(type) {
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return i
		}
		f, _ := val.Float64()
		return f
	case map[string]any:
		obj := make(Object, 0, len(val))
		for k, child := range val {
			obj = append(obj, &Node{
				Name:      k,
				EventID:   e.ID,
				EventType: e.Type,
				Seq:       e.Seq,
				Value:     buildNodeValue(child, e),
			})
		}
		sort.Slice(obj, func(i, j int) bool { return obj[i].Name < obj[j].Name })
		return Object(obj)
	case []any:
		arr := make(Array, 0, len(val))
		for _, item := range val {
			arr = append(arr, &Node{
				EventID:   e.ID,
				EventType: e.Type,
				Seq:       e.Seq,
				Value:     buildNodeValue(item, e),
			})
		}
		return Array(arr)
	default:
		return v
	}
}

// flat converts the aggregate to a plain map for serialization.
// Deleted and moved nodes are excluded.
func (a *Aggregate) flat() map[string]any {
	return flatObject(a.Value)
}

func flatObject(obj Object) map[string]any {
	m := make(map[string]any, len(obj))
	for _, node := range obj {
		if node.Status == NodeDeleted || node.Status == NodeMoved {
			continue
		}
		m[node.Name] = flatValue(node.Value)
	}
	return m
}

func flatValue(v any) any {
	switch val := v.(type) {
	case Object:
		return flatObject(val)
	case Array:
		arr := make([]any, 0, len(val))
		for _, node := range val {
			if node.Status != NodeDeleted && node.Status != NodeMoved {
				arr = append(arr, flatValue(node.Value))
			}
		}
		return arr
	default:
		return v
	}
}

// sortedNodes returns active top-level nodes sorted alphabetically by name.
func (a *Aggregate) sortedNodes() []*Node {
	nodes := make([]*Node, 0, len(a.Value))
	for _, n := range a.Value {
		if n.Status == NodeActive {
			nodes = append(nodes, n)
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Name < nodes[j].Name })
	return nodes
}

// ToJSON returns the aggregate flattened to JSON (values only, no provenance).
func (a *Aggregate) ToJSON() ([]byte, error) {
	return json.Marshal(a.flat())
}

// ToYAML returns the aggregate flattened to YAML (values only, no provenance).
func (a *Aggregate) ToYAML() ([]byte, error) {
	return yaml.Marshal(a.flat())
}

// ToTOML returns the aggregate flattened to TOML (values only, no provenance).
func (a *Aggregate) ToTOML() ([]byte, error) {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(a.flat()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ToXML returns the aggregate flattened to XML (values only, no provenance).
// The root element is named by the root argument. Keys are sorted alphabetically.
// Nested objects produce nested elements; arrays produce repeated <item> elements.
func (a *Aggregate) ToXML(root string) ([]byte, error) {
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")

	el := xml.StartElement{Name: xml.Name{Local: root}}
	if err := enc.EncodeToken(el); err != nil {
		return nil, err
	}
	flat := a.flat()
	for _, k := range sortedMapKeys(flat) {
		if err := encodeXMLNode(enc, k, flat[k]); err != nil {
			return nil, err
		}
	}
	if err := enc.EncodeToken(el.End()); err != nil {
		return nil, err
	}
	if err := enc.Flush(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeXMLNode(enc *xml.Encoder, name string, v any) error {
	el := xml.StartElement{Name: xml.Name{Local: name}}
	switch val := v.(type) {
	case map[string]any:
		if err := enc.EncodeToken(el); err != nil {
			return err
		}
		for _, k := range sortedMapKeys(val) {
			if err := encodeXMLNode(enc, k, val[k]); err != nil {
				return err
			}
		}
		return enc.EncodeToken(el.End())
	case []any:
		if err := enc.EncodeToken(el); err != nil {
			return err
		}
		for _, item := range val {
			if err := encodeXMLNode(enc, "item", item); err != nil {
				return err
			}
		}
		return enc.EncodeToken(el.End())
	default:
		if err := enc.EncodeToken(el); err != nil {
			return err
		}
		if err := enc.EncodeToken(xml.CharData(scalarString(name, v))); err != nil {
			return err
		}
		return enc.EncodeToken(el.End())
	}
}

// ToDotenv returns the aggregate flattened to dotenv format (KEY=VALUE).
// String values are quoted if they contain whitespace or shell-significant characters.
// Nested objects and arrays fall back to JSON encoding.
func (a *Aggregate) ToDotenv() ([]byte, error) {
	var buf bytes.Buffer
	for _, node := range a.sortedNodes() {
		v := flatValue(node.Value)
		var val string
		if s, ok := v.(string); ok && strings.ContainsAny(s, " \t\n\r#\"'\\") {
			val = fmt.Sprintf("%q", s)
		} else {
			val = scalarString(node.Name, v)
		}
		fmt.Fprintf(&buf, "%s=%s\n", node.Name, val)
	}
	return buf.Bytes(), nil
}

// ToProperties returns the aggregate flattened to Java .properties format.
// Nested objects and arrays fall back to JSON encoding.
func (a *Aggregate) ToProperties() ([]byte, error) {
	var buf bytes.Buffer
	for _, node := range a.sortedNodes() {
		fmt.Fprintf(&buf, "%s=%s\n", node.Name, propertiesEscape(flatValue(node.Value)))
	}
	return buf.Bytes(), nil
}

// ToINI returns the aggregate flattened to INI format under a [default] section.
// Nested objects and arrays fall back to JSON encoding.
func (a *Aggregate) ToINI() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("[default]\n")
	for _, node := range a.sortedNodes() {
		fmt.Fprintf(&buf, "%s = %s\n", node.Name, scalarString(node.Name, flatValue(node.Value)))
	}
	return buf.Bytes(), nil
}

// ToHCL returns the aggregate flattened to HCL attribute assignment syntax.
// Nested objects produce HCL object expressions; arrays produce HCL tuple expressions.
func (a *Aggregate) ToHCL() ([]byte, error) {
	var buf bytes.Buffer
	for _, node := range a.sortedNodes() {
		val, err := formatHCLValue(flatValue(node.Value))
		if err != nil {
			return nil, fmt.Errorf("hcl value for key %s: %w", node.Name, err)
		}
		fmt.Fprintf(&buf, "%s = %s\n", node.Name, val)
	}
	return buf.Bytes(), nil
}

func formatHCLValue(v any) (string, error) {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val), nil
	case int64:
		return strconv.FormatInt(val, 10), nil
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(val), nil
	case map[string]any:
		var sb strings.Builder
		sb.WriteString("{\n")
		for _, k := range sortedMapKeys(val) {
			inner, err := formatHCLValue(val[k])
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&sb, "  %s = %s\n", k, inner)
		}
		sb.WriteString("}")
		return sb.String(), nil
	case []any:
		parts := make([]string, 0, len(val))
		for _, item := range val {
			inner, err := formatHCLValue(item)
			if err != nil {
				return "", err
			}
			parts = append(parts, inner)
		}
		return "[" + strings.Join(parts, ", ") + "]", nil
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}

func sortedMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// scalarString formats any value as a plain string for text-based formats.
// Complex values fall back to JSON encoding.
func scalarString(key string, v any) string {
	switch val := v.(type) {
	case string:
		return val
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("<%s: marshal error: %v>", key, err)
		}
		return string(b)
	}
}

// propertiesEscape applies Java .properties escaping rules to a value.
func propertiesEscape(v any) string {
	s := scalarString("", v)
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}
