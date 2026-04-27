package commands

import (
	"encoding/json"
	"testing"
)

func TestBuildSchema_HasJSONSchemaFields(t *testing.T) {
	schema := buildSchema()

	if schema["$schema"] != "http://json-schema.org/draft-07/schema#" {
		t.Error("missing $schema")
	}
	if schema["type"] != "object" {
		t.Error("expected type: object")
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties is not a map")
	}
	for _, key := range []string{"name", "template"} {
		if _, ok := props[key]; !ok {
			t.Errorf("missing top-level property %q", key)
		}
	}
}

func TestBuildSchema_TemplateEnumContainsAllTemplates(t *testing.T) {
	schema := buildSchema()
	props := schema["properties"].(map[string]any)
	tmplProp := props["template"].(map[string]any)
	enum, ok := tmplProp["enum"].([]any)
	if !ok || len(enum) == 0 {
		t.Fatal("template enum is missing or empty")
	}
	enumSet := make(map[string]bool, len(enum))
	for _, v := range enum {
		enumSet[v.(string)] = true
	}
	for want := range synthesizers {
		if !enumSet[want] {
			t.Errorf("template %q missing from schema enum", want)
		}
	}
}

func TestBuildSchema_HasAllOfForSchemerSynthesizers(t *testing.T) {
	schema := buildSchema()
	allOf, ok := schema["allOf"].([]any)
	if !ok || len(allOf) == 0 {
		t.Fatal("expected allOf with at least one synthesizer entry")
	}
	// Every allOf entry must have if.properties.template.const
	for i, entry := range allOf {
		m := entry.(map[string]any)
		ifBlock, ok := m["if"].(map[string]any)
		if !ok {
			t.Errorf("allOf[%d] missing if block", i)
			continue
		}
		ifProps := ifBlock["properties"].(map[string]any)
		tmpl := ifProps["template"].(map[string]any)
		if tmpl["const"] == nil {
			t.Errorf("allOf[%d] if.properties.template.const is nil", i)
		}
		thenBlock, ok := m["then"].(map[string]any)
		if !ok {
			t.Errorf("allOf[%d] missing then block", i)
			continue
		}
		if _, ok := thenBlock["properties"]; !ok {
			t.Errorf("allOf[%d] then has no properties", i)
		}
	}
}

func TestBuildSchema_IsJSONSerializable(t *testing.T) {
	schema := buildSchema()
	if _, err := json.Marshal(schema); err != nil {
		t.Errorf("schema is not JSON-serializable: %v", err)
	}
}

func TestBuildSchema_AllOfIsDeterministic(t *testing.T) {
	first, _ := json.Marshal(buildSchema())
	second, _ := json.Marshal(buildSchema())
	if string(first) != string(second) {
		t.Error("buildSchema is not deterministic across calls")
	}
}
