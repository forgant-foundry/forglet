package commands

import "fmt"

// RunHelp prints implementation reference to stdout.
// With no args it prints an overview of all commands and topics.
// Pass "plugin" or "domain" to print topic-specific implementer reference.
func RunHelp(args []string) {
	topic := ""
	if len(args) > 0 {
		topic = args[0]
	}
	switch topic {
	case "plugin":
		fmt.Print(helpPlugin)
	case "domain":
		fmt.Print(helpDomain)
	default:
		fmt.Print(helpOverview)
	}
}

const helpOverview = `forglet — template-based project management with event sourcing

COMMANDS
  forglet new <template> <name>   create a new project from a template
  forglet synth                   re-synthesize managed files in the current directory
  forglet schema                  print JSON Schema for .forglet.yml (pipe to a file for IDE autocomplete)
  forglet help [topic]            show reference documentation

IMPLEMENTATION TOPICS
  forglet help plugin   implementing plugins, scaffolders, and schema contributions
  forglet help domain   implementing domain synthesizers

TEMPLATES
  go, go-workspace, go-lambda, go-knative
  java, java-multimodule, java-lambda, java-spring
  node-ts, node-js, node-lambda
`

const helpPlugin = `IMPLEMENTING A PLUGIN

Plugins are compiled into a custom forglet binary. They receive project Meta
(name, template) and the full RC map (.forglet.yml contents) so they can be
both template-aware and project-config-aware.

INTERFACE

  type Plugin interface {
      Weave(meta Meta, rc map[string]any, stream *EventStream) error
  }

  meta.Template  template name ("go", "node-ts", etc.)
  meta.Name      project name

OPTIONAL INTERFACES

  type Scaffolder interface {
      Scaffold(dir string, meta Meta) error
  }
  Called once during 'forglet new', never on re-synth.
  Must be idempotent: check whether the file exists before writing.

  type Schemer interface {
      RCSchema() SchemaContribution
  }
  Contributes this plugin's rc key definitions to 'forglet schema'.
  Return cross-cutting keys (valid regardless of template).

EVENTSTREAM METHODS

  Append(file, events...)                   add after existing events
  InsertAfter(file, eventType, events...)   after last match; append if no match
  InsertBefore(file, eventType, events...)  before first match; prepend if no match
  Replace(file, eventType, events...)       remove all of type, insert at first match
  Remove(file, eventType)                   remove all events of that type
  SetFormat(file, format)                   register render format (call alongside Append)

FILE FORMATS (pass to SetFormat)

  project.FormatJSON     pretty-printed JSON object, // managed comment key
  project.FormatYAML     YAML document, # managed comment header
  project.FormatPattern  one active key per line, # managed comment header
  project.FormatText     raw string value of "text" node; no managed comment

MINIMAL EXAMPLE

  package myplugin

  import (
      "encoding/json"
      "github.com/forgant-foundry/eventing"
      "github.com/forgant-foundry/forglet/internal/project"
  )

  type Plugin struct{}

  func (p *Plugin) Weave(meta project.Meta, rc map[string]any, stream *project.EventStream) error {
      if meta.Template != "node-ts" {
          return nil
      }
      payload, _ := json.Marshal(map[string]any{
          "devDependencies": map[string]any{"eslint": "^9.0.0"},
      })
      stream.SetFormat("package.json", project.FormatJSON)
      stream.Append("package.json", eventing.Event{
          ID: "standards.eslint", Type: "devDependency.added", Seq: 1, Payload: payload,
      })
      return nil
  }

  func (p *Plugin) RCSchema() project.SchemaContribution {
      return project.SchemaContribution{
          Properties: map[string]any{
              "lint": map[string]any{
                  "type":        "boolean",
                  "description": "Add ESLint to node-ts projects.",
              },
          },
      }
  }

REGISTRATION

  // cmd/myforglet/main.go
  commands.RegisterPlugin(&myplugin.Plugin{})
  if err := commands.Execute(); err != nil { ... }

AWARENESS HIERARCHY — do not violate

  Plugin     aware of: meta.Template, rc, stream.Events() (other plugins' contributions)
  Domain     aware of: meta and rc only — never inspects the stream
  Project    aware of: neither domains nor plugins by name
`

const helpDomain = `IMPLEMENTING A DOMAIN

A domain synthesizer is registered under a template name and drives the three
event layers for that project type.

INTERFACE

  type Synthesizer interface {
      InitializeEvents(name string) (map[string][]eventing.Event, error)
      OverlayEvents(rc map[string]any) (map[string][]eventing.Event, error)
      Synthesize(dir string, aggregates map[string]*eventing.Aggregate) error
  }

  InitializeEvents  return the base template events (layer 1).
                    Key is the managed filename (e.g. "go.mod", "package.json").
  OverlayEvents     translate .forglet.yml rc map into overlay events (layer 3).
                    Same key structure. Return nil, nil when rc has nothing relevant.
  Synthesize        write output files from fully-merged aggregates.
                    Use agg.ToJSON() for JSON, agg.ToYAML() for YAML.
                    Use project.WriteManaged(path, b) to write a 0444 managed file.
                    Use project.AddTextMarker(b, "//") to prepend the managed comment.

OPTIONAL INTERFACES

  type PostSynthesizer interface {
      PostSynthesize(dir string) error
  }
  Called after Synthesize by Project.Synthesize — not called in tests.
  Use for toolchain steps that require written files to be present on disk
  (e.g. 'go mod tidy', 'go work sync'). Manage file permissions around the
  call if the synthesizer writes 0444 files that the toolchain needs to modify.

  type Schemer interface {
      RCSchema() SchemaContribution
  }
  Contributes this synthesizer's template-specific rc key definitions to
  'forglet schema'. The schema assembler wraps them in an if/then block
  for the registered template name.

MINIMAL EXAMPLE

  package mytemplate

  import (
      "encoding/json"
      "path/filepath"
      "github.com/forgant-foundry/eventing"
      "github.com/forgant-foundry/forglet/internal/project"
  )

  type Synth struct{}

  func New() *Synth { return &Synth{} }

  func (s *Synth) InitializeEvents(name string) (map[string][]eventing.Event, error) {
      b, _ := json.Marshal(map[string]any{"name": name, "version": "0.1.0"})
      return map[string][]eventing.Event{
          "config.json": {{ID: "init", Type: "init", Seq: 1, Payload: b}},
      }, nil
  }

  func (s *Synth) OverlayEvents(rc map[string]any) (map[string][]eventing.Event, error) {
      v, ok := rc["version"].(string)
      if !ok || v == "" {
          return nil, nil
      }
      b, _ := json.Marshal(map[string]any{"version": v})
      return map[string][]eventing.Event{
          "config.json": {{ID: "ver", Type: "version.set", Seq: 1000, Payload: b}},
      }, nil
  }

  func (s *Synth) Synthesize(dir string, aggregates map[string]*eventing.Aggregate) error {
      agg, ok := aggregates["config.json"]
      if !ok {
          return nil
      }
      b, _ := agg.ToJSON()
      return project.WriteManaged(filepath.Join(dir, "config.json"), b)
  }

  func (s *Synth) RCSchema() project.SchemaContribution {
      return project.SchemaContribution{
          Properties: map[string]any{
              "version": map[string]any{
                  "type":        "string",
                  "description": "Project version.",
              },
          },
      }
  }

REGISTRATION

  // cmd/forglet/commands/new.go (or your custom binary's equivalent)
  var synthesizers = map[string]project.Synthesizer{
      // existing entries ...
      "mytemplate": mytemplate.New(),
  }
`
