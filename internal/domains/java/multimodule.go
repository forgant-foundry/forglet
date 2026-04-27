package java

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
)

// Multimodule synthesizes a Maven multi-module project.
// Managed files: pom.xml
// Scaffolded once: <module>/pom.xml, <module>/src/{main,test}/java/...
type Multimodule struct{}

func NewMultimodule() *Multimodule { return &Multimodule{} }

func (s *Multimodule) InitializeEvents(name string) (map[string][]eventing.Event, error) {
	events, err := makeEvents(1, []evtDef{
		{"init", map[string]any{
			"groupId":     defaultGroupID,
			"artifactId":  name,
			"version":     defaultVersion,
			"packaging":   "pom",
			"javaVersion": defaultJavaVersion,
		}},
		{"modules.added", map[string]any{
			"modules": map[string]any{
				"core": "",
			},
		}},
		{"managedTestDependency.added", map[string]any{
			"managedTestDependencies": map[string]any{
				"org.junit.jupiter:junit-jupiter": junitVersion,
			},
		}},
	})
	if err != nil {
		return nil, err
	}
	return map[string][]eventing.Event{"pom.xml": events}, nil
}

// OverlayEvents supports rc keys: groupId, version, javaVersion,
// dependencies, testDependencies (via commonOverlay), modules ([]string).
func (s *Multimodule) OverlayEvents(rc map[string]any) (map[string][]eventing.Event, error) {
	result, err := commonOverlay(rc)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = map[string][]eventing.Event{}
	}

	if mods, ok := asStringSlice(rc["modules"]); ok {
		modsMap := make(map[string]any, len(mods))
		for _, m := range mods {
			modsMap[m] = ""
		}
		events, err := makeEvents(overlaySeqBase+50, []evtDef{
			{"modules.added", map[string]any{"modules": modsMap}},
		})
		if err != nil {
			return nil, err
		}
		result["pom.xml"] = append(result["pom.xml"], events...)
	}

	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func (s *Multimodule) RCSchema() project.SchemaContribution {
	props := javaPomSchemaProps()
	props["modules"] = map[string]any{
		"type":        "array",
		"items":       map[string]any{"type": "string"},
		"description": "Maven module names to scaffold.",
	}
	return project.SchemaContribution{Properties: props}
}

func (s *Multimodule) Synthesize(dir string, aggregates map[string]*eventing.Aggregate) error {
	agg, ok := aggregates["pom.xml"]
	if !ok {
		return nil
	}
	b, err := renderMultimodulePom(agg)
	if err != nil {
		return fmt.Errorf("render pom.xml: %w", err)
	}
	if err := project.WriteManaged(filepath.Join(dir, "pom.xml"), b); err != nil {
		return err
	}

	groupID := nodeStr(agg, "groupId", defaultGroupID)
	artifactID := nodeStr(agg, "artifactId", "app")
	version := nodeStr(agg, "version", defaultVersion)

	for _, mod := range activeModules(agg) {
		if err := scaffoldModule(dir, mod, groupID, artifactID, version); err != nil {
			return err
		}
	}
	return nil
}

func renderMultimodulePom(agg *eventing.Aggregate) ([]byte, error) {
	var buf bytes.Buffer
	renderPomHeader(&buf)
	renderPomCoords(&buf, agg)

	if mods := activeModules(agg); len(mods) > 0 {
		fmt.Fprintf(&buf, "    <modules>\n")
		for _, m := range mods {
			writeln(&buf, 2, "module", m)
		}
		fmt.Fprintf(&buf, "    </modules>\n\n")
	}

	renderProperties(&buf, agg)
	renderManagedDependencies(&buf, agg)
	renderPluginManagement(&buf, []pomPlugin{surefirePlugin()})
	fmt.Fprintf(&buf, "</project>\n")
	return buf.Bytes(), nil
}

func renderManagedDependencies(buf *bytes.Buffer, agg *eventing.Aggregate) {
	compile := activeDeps(agg, "managedDependencies", "")
	test := activeDeps(agg, "managedTestDependencies", "test")
	all := append(compile, test...)
	if len(all) == 0 {
		return
	}
	fmt.Fprintf(buf, "    <dependencyManagement>\n")
	fmt.Fprintf(buf, "        <dependencies>\n")
	for _, d := range all {
		fmt.Fprintf(buf, "            <dependency>\n")
		writeln(buf, 4, "groupId", d.groupID)
		writeln(buf, 4, "artifactId", d.artifactID)
		if d.version != "" {
			writeln(buf, 4, "version", d.version)
		}
		if d.scope != "" {
			writeln(buf, 4, "scope", d.scope)
		}
		fmt.Fprintf(buf, "            </dependency>\n")
	}
	fmt.Fprintf(buf, "        </dependencies>\n")
	fmt.Fprintf(buf, "    </dependencyManagement>\n\n")
}

func renderPluginManagement(buf *bytes.Buffer, plugins []pomPlugin) {
	if len(plugins) == 0 {
		return
	}
	fmt.Fprintf(buf, "    <build>\n")
	fmt.Fprintf(buf, "        <pluginManagement>\n")
	fmt.Fprintf(buf, "            <plugins>\n")
	for _, p := range plugins {
		fmt.Fprintf(buf, "                <plugin>\n")
		writeln(buf, 5, "groupId", p.groupID)
		writeln(buf, 5, "artifactId", p.artifactID)
		writeln(buf, 5, "version", p.version)
		fmt.Fprintf(buf, "                </plugin>\n")
	}
	fmt.Fprintf(buf, "            </plugins>\n")
	fmt.Fprintf(buf, "        </pluginManagement>\n")
	fmt.Fprintf(buf, "    </build>\n")
}

func activeModules(agg *eventing.Aggregate) []string {
	node, ok := agg.Node("modules")
	if !ok {
		return nil
	}
	obj, ok := node.Value.(eventing.Object)
	if !ok {
		return nil
	}
	var mods []string
	for _, n := range obj {
		if n.Status == eventing.NodeActive {
			mods = append(mods, n.Name)
		}
	}
	sort.Strings(mods)
	return mods
}

func scaffoldModule(dir, mod, groupID, parentArtifactID, version string) error {
	modDir := filepath.Join(dir, mod)
	pomFile := filepath.Join(modDir, "pom.xml")
	if _, err := os.Stat(pomFile); os.IsNotExist(err) {
		if err := os.MkdirAll(modDir, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(pomFile, childPomXML(groupID, parentArtifactID, version, mod), 0644); err != nil {
			return err
		}
	}

	pkgDir := filepath.Join(modDir, "src", "main", "java", groupToPath(groupID))
	testPkgDir := filepath.Join(modDir, "src", "test", "java", groupToPath(groupID))
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		return err
	}
	return os.MkdirAll(testPkgDir, 0755)
}

func childPomXML(groupID, parentArtifactID, version, moduleName string) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	fmt.Fprintf(&buf, "<project xmlns=\"http://maven.apache.org/POM/4.0.0\"\n")
	fmt.Fprintf(&buf, "         xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\"\n")
	fmt.Fprintf(&buf, "         xsi:schemaLocation=\"http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd\">\n")
	fmt.Fprintf(&buf, "    <modelVersion>4.0.0</modelVersion>\n\n")
	fmt.Fprintf(&buf, "    <parent>\n")
	writeln(&buf, 2, "groupId", groupID)
	writeln(&buf, 2, "artifactId", parentArtifactID)
	writeln(&buf, 2, "version", version)
	fmt.Fprintf(&buf, "    </parent>\n\n")
	writeln(&buf, 1, "artifactId", moduleName)
	writeln(&buf, 1, "packaging", "jar")
	fmt.Fprintf(&buf, "\n    <dependencies>\n")
	fmt.Fprintf(&buf, "        <dependency>\n")
	writeln(&buf, 3, "groupId", "org.junit.jupiter")
	writeln(&buf, 3, "artifactId", "junit-jupiter")
	writeln(&buf, 3, "scope", "test")
	fmt.Fprintf(&buf, "        </dependency>\n")
	fmt.Fprintf(&buf, "    </dependencies>\n")
	fmt.Fprintf(&buf, "</project>\n")
	return buf.Bytes()
}
