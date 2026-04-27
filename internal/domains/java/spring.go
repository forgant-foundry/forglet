package java

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
)

const (
	springBootVersion    = "3.3.0"
	defaultSpringVersion = "0.0.1-SNAPSHOT"
)

// Spring synthesizes a Spring Boot Maven project.
// Managed files: pom.xml
// Scaffolded once: src/main/java/.../Application.java, HelloController.java,
//
//	src/main/resources/application.properties,
//	src/test/java/.../ApplicationTest.java
type Spring struct{}

func NewSpring() *Spring { return &Spring{} }

func (s *Spring) InitializeEvents(name string) (map[string][]eventing.Event, error) {
	events, err := makeEvents(1, []evtDef{
		{"init", map[string]any{
			"groupId":           defaultGroupID,
			"artifactId":        name,
			"version":           defaultSpringVersion,
			"packaging":         "jar",
			"javaVersion":       defaultJavaVersion,
			"springBootVersion": springBootVersion,
		}},
		{"dependency.added", map[string]any{
			"dependencies": map[string]any{
				"org.springframework.boot:spring-boot-starter-web": "",
			},
		}},
		{"testDependency.added", map[string]any{
			"testDependencies": map[string]any{
				"org.springframework.boot:spring-boot-starter-test": "",
			},
		}},
	})
	if err != nil {
		return nil, err
	}
	return map[string][]eventing.Event{"pom.xml": events}, nil
}

// OverlayEvents supports rc keys: groupId, version, javaVersion,
// dependencies (map[string]string), testDependencies (map[string]string).
func (s *Spring) OverlayEvents(rc map[string]any) (map[string][]eventing.Event, error) {
	return commonOverlay(rc)
}

func (s *Spring) RCSchema() project.SchemaContribution {
	return project.SchemaContribution{Properties: javaPomSchemaProps()}
}

func (s *Spring) Synthesize(dir string, aggregates map[string]*eventing.Aggregate) error {
	agg, ok := aggregates["pom.xml"]
	if !ok {
		return nil
	}
	b, err := renderSpringPom(agg)
	if err != nil {
		return fmt.Errorf("render pom.xml: %w", err)
	}
	if err := project.WriteManaged(filepath.Join(dir, "pom.xml"), b); err != nil {
		return err
	}

	groupID := nodeStr(agg, "groupId", defaultGroupID)
	pkgDir := filepath.Join(dir, "src", "main", "java", groupToPath(groupID))
	resDir := filepath.Join(dir, "src", "main", "resources")
	testPkgDir := filepath.Join(dir, "src", "test", "java", groupToPath(groupID))

	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(resDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(testPkgDir, 0755); err != nil {
		return err
	}

	scaffolds := []struct {
		path    string
		content func() []byte
	}{
		{filepath.Join(pkgDir, "Application.java"), func() []byte { return applicationJava(groupID) }},
		{filepath.Join(pkgDir, "HelloController.java"), func() []byte { return helloControllerJava(groupID) }},
		{filepath.Join(resDir, "application.properties"), func() []byte { return []byte("server.port=8080\n") }},
		{filepath.Join(testPkgDir, "ApplicationTest.java"), func() []byte { return applicationTestJava(groupID) }},
	}
	for _, sc := range scaffolds {
		if _, err := os.Stat(sc.path); os.IsNotExist(err) {
			if err := os.WriteFile(sc.path, sc.content(), 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

func renderSpringPom(agg *eventing.Aggregate) ([]byte, error) {
	var buf bytes.Buffer
	renderPomHeader(&buf)

	sbv := nodeStr(agg, "springBootVersion", springBootVersion)
	fmt.Fprintf(&buf, "    <parent>\n")
	writeln(&buf, 2, "groupId", "org.springframework.boot")
	writeln(&buf, 2, "artifactId", "spring-boot-starter-parent")
	writeln(&buf, 2, "version", sbv)
	fmt.Fprintf(&buf, "        <relativePath/>\n")
	fmt.Fprintf(&buf, "    </parent>\n\n")

	renderPomCoords(&buf, agg)

	jv := nodeStr(agg, "javaVersion", defaultJavaVersion)
	fmt.Fprintf(&buf, "    <properties>\n")
	writeln(&buf, 2, "java.version", jv)
	fmt.Fprintf(&buf, "    </properties>\n\n")

	renderDependencies(&buf, agg)
	renderBuild(&buf, []pomPlugin{springMavenPlugin()})
	fmt.Fprintf(&buf, "</project>\n")
	return buf.Bytes(), nil
}

func springMavenPlugin() pomPlugin {
	return pomPlugin{
		groupID:    "org.springframework.boot",
		artifactID: "spring-boot-maven-plugin",
		// version intentionally empty — inherited from spring-boot-starter-parent
	}
}

func applicationJava(pkg string) []byte {
	return []byte("package " + pkg + ";\n\n" +
		"import org.springframework.boot.SpringApplication;\n" +
		"import org.springframework.boot.autoconfigure.SpringBootApplication;\n\n" +
		"@SpringBootApplication\n" +
		"public class Application {\n" +
		"    public static void main(String[] args) {\n" +
		"        SpringApplication.run(Application.class, args);\n" +
		"    }\n" +
		"}\n")
}

func helloControllerJava(pkg string) []byte {
	return []byte("package " + pkg + ";\n\n" +
		"import org.springframework.web.bind.annotation.GetMapping;\n" +
		"import org.springframework.web.bind.annotation.RestController;\n\n" +
		"@RestController\n" +
		"public class HelloController {\n\n" +
		"    @GetMapping(\"/\")\n" +
		"    public String hello() {\n" +
		"        return \"Hello, World!\";\n" +
		"    }\n" +
		"}\n")
}

func applicationTestJava(pkg string) []byte {
	return []byte("package " + pkg + ";\n\n" +
		"import org.junit.jupiter.api.Test;\n" +
		"import org.springframework.boot.test.context.SpringBootTest;\n\n" +
		"@SpringBootTest\n" +
		"class ApplicationTest {\n" +
		"    @Test\n" +
		"    void contextLoads() {}\n" +
		"}\n")
}
