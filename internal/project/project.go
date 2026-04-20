package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/forgant-foundry/eventing"
	"gopkg.in/yaml.v3"
)

const forgletDir = ".forglet"

// Meta holds project-level metadata stored in .forglet/project.json.
type Meta struct {
	Name     string `json:"name"`
	Template string `json:"template"`
}

// Synthesizer builds and writes managed project files from three event layers.
type Synthesizer interface {
	// InitializeEvents returns the base template events per managed file.
	InitializeEvents(name string) (map[string][]eventing.Event, error)

	// OverlayEvents translates .forglet.yml config into overlay events per file.
	// The synthesizer interprets the rc keys — a single key may produce events
	// for multiple files or trigger additional behaviour beyond writing values.
	OverlayEvents(rc map[string]any) (map[string][]eventing.Event, error)

	// Synthesize writes managed files to dir given the fully-merged aggregates.
	Synthesize(dir string, aggregates map[string]*eventing.Aggregate) error
}

// Plugin contributes to a project by weaving events into the event stream
// between the template layer and the RC overlay layer. Plugins receive the
// full stream and can insert, replace, or remove events at any position —
// not just append — giving full control over the synthesized output.
type Plugin interface {
	Weave(projectName string, stream *EventStream) error
}

// Project manages a forglet project rooted at a directory.
type Project struct {
	root    string
	plugins []Plugin
}

func New(root string) *Project {
	return &Project{root: root}
}

// WithPlugins registers plugins to be applied during synthesis.
func (p *Project) WithPlugins(plugins ...Plugin) *Project {
	p.plugins = append(p.plugins, plugins...)
	return p
}

// Init creates .forglet/, writes metadata, and synthesizes all managed files.
func (p *Project) Init(meta Meta, s Synthesizer) error {
	if err := os.MkdirAll(filepath.Join(p.root, forgletDir), 0755); err != nil {
		return err
	}
	if err := p.SaveMeta(meta); err != nil {
		return err
	}
	return p.Synthesize(s)
}

// Synthesize re-derives all event layers and writes managed files.
// Layers are applied in order: template → plugins → RC overlay.
// All layers are re-derived fresh each call — this is how template upgrades
// and plugin updates take effect automatically.
func (p *Project) Synthesize(s Synthesizer) error {
	meta, err := p.LoadMeta()
	if err != nil {
		return err
	}
	rc, err := p.loadRC()
	if err != nil {
		return err
	}

	templateEvents, err := s.InitializeEvents(meta.Name)
	if err != nil {
		return err
	}

	stream := NewEventStream(templateEvents)
	for _, plugin := range p.plugins {
		if err := plugin.Weave(meta.Name, stream); err != nil {
			return err
		}
	}

	overlayEvents, err := s.OverlayEvents(rc)
	if err != nil {
		return err
	}

	aggregates := BuildAggregates(stream.Events(), overlayEvents)
	for filename, agg := range aggregates {
		if err := p.saveAggregate(filename, agg); err != nil {
			return err
		}
	}
	return s.Synthesize(p.root, aggregates)
}

// BuildAggregates builds per-file aggregates by sequentially deep-merging N
// event layers. Each layer is applied after the previous — later layers win
// for scalar nodes, and Object nodes are merged so all layers contribute children.
// Exported for use in tests.
func BuildAggregates(layers ...map[string][]eventing.Event) map[string]*eventing.Aggregate {
	if len(layers) == 0 {
		return map[string]*eventing.Aggregate{}
	}
	result := applyAll(layers[0])
	for _, layer := range layers[1:] {
		result = mergeLayers(result, applyAll(layer))
	}
	return result
}

func mergeLayers(base, overlay map[string]*eventing.Aggregate) map[string]*eventing.Aggregate {
	merged := make(map[string]*eventing.Aggregate, len(base))
	for file, bAgg := range base {
		if oAgg, ok := overlay[file]; ok {
			merged[file] = mergeAggregates(bAgg, oAgg)
		} else {
			merged[file] = bAgg
		}
	}
	for file, oAgg := range overlay {
		if _, ok := merged[file]; !ok {
			merged[file] = oAgg
		}
	}
	return merged
}

// applyAll builds an aggregate per file by treating each event as its own
// mini-layer and deep-merging it into the accumulating aggregate. This means
// Object fields (e.g. devDependencies, scripts) accumulate children across
// events rather than replacing each other, while scalars use last-write-wins.
func applyAll(fileEvents map[string][]eventing.Event) map[string]*eventing.Aggregate {
	aggs := make(map[string]*eventing.Aggregate, len(fileEvents))
	for file, events := range fileEvents {
		agg := eventing.NewAggregate()
		for i := range events {
			mini := eventing.NewAggregate()
			mini.Apply(&events[i])
			agg = mergeAggregates(agg, mini)
		}
		aggs[file] = agg
	}
	return aggs
}

// mergeAggregates deep-merges overlay into base. Object nodes are merged
// recursively so both layers contribute children; scalars use overlay.
func mergeAggregates(base, overlay *eventing.Aggregate) *eventing.Aggregate {
	result := eventing.NewAggregate()
	result.Value = mergeObjects(base.Value, overlay.Value)
	result.EventID = overlay.EventID
	result.EventType = overlay.EventType
	result.Seq = overlay.Seq
	return result
}

func mergeObjects(base, overlay eventing.Object) eventing.Object {
	overlayByName := make(map[string]*eventing.Node, len(overlay))
	for _, n := range overlay {
		overlayByName[n.Name] = n
	}

	result := make(eventing.Object, 0, len(base)+len(overlay))
	for _, baseNode := range base {
		oNode, inOverlay := overlayByName[baseNode.Name]
		if !inOverlay {
			result = append(result, baseNode)
			continue
		}
		delete(overlayByName, baseNode.Name)

		baseObj, baseIsObj := baseNode.Value.(eventing.Object)
		oObj, oIsObj := oNode.Value.(eventing.Object)
		if baseIsObj && oIsObj {
			result = append(result, &eventing.Node{
				Name:      baseNode.Name,
				Value:     mergeObjects(baseObj, oObj),
				EventID:   oNode.EventID,
				EventType: oNode.EventType,
				Seq:       oNode.Seq,
			})
		} else {
			result = append(result, oNode)
		}
	}
	for _, n := range overlay {
		if _, remaining := overlayByName[n.Name]; remaining {
			result = append(result, n)
		}
	}
	return result
}

func (p *Project) SaveMeta(meta Meta) error {
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(p.root, forgletDir, "project.json"), b, 0644)
}

func (p *Project) LoadMeta() (Meta, error) {
	b, err := os.ReadFile(filepath.Join(p.root, forgletDir, "project.json"))
	if err != nil {
		return Meta{}, fmt.Errorf("not a forglet project: %w", err)
	}
	var meta Meta
	return meta, json.Unmarshal(b, &meta)
}

// loadRC reads .forglet.yml from the project root.
// Returns an empty map if the file does not exist.
func (p *Project) loadRC() (map[string]any, error) {
	b, err := os.ReadFile(filepath.Join(p.root, ".forglet.yml"))
	if os.IsNotExist(err) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	var rc map[string]any
	return rc, yaml.Unmarshal(b, &rc)
}

// saveAggregate writes the aggregate snapshot to .forglet/<filename>.json.
func (p *Project) saveAggregate(filename string, agg *eventing.Aggregate) error {
	b, err := json.MarshalIndent(agg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(p.root, forgletDir, filename+".json"), b, 0644)
}
