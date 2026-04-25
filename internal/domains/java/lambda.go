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
	lambdaCoreVersion   = "1.2.3"
	lambdaEventsVersion = "3.11.4"
)

// Lambda synthesizes a Maven project for AWS Lambda (fat JAR via shade plugin).
// Managed files: pom.xml
// Scaffolded once: src/main/java/.../Handler.java, src/test/java/.../HandlerTest.java
type Lambda struct{}

func NewLambda() *Lambda { return &Lambda{} }

func (s *Lambda) InitializeEvents(name string) (map[string][]eventing.Event, error) {
	events, err := makeEvents(1, []evtDef{
		{"init", map[string]any{
			"groupId":     defaultGroupID,
			"artifactId":  name,
			"version":     defaultVersion,
			"packaging":   "jar",
			"javaVersion": defaultJavaVersion,
		}},
		{"dependency.added", map[string]any{
			"dependencies": map[string]any{
				"com.amazonaws:aws-lambda-java-core":   lambdaCoreVersion,
				"com.amazonaws:aws-lambda-java-events": lambdaEventsVersion,
			},
		}},
		{"testDependency.added", map[string]any{
			"testDependencies": map[string]any{
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
// dependencies (map[string]string), testDependencies (map[string]string).
func (s *Lambda) OverlayEvents(rc map[string]any) (map[string][]eventing.Event, error) {
	return commonOverlay(rc)
}

func (s *Lambda) Synthesize(dir string, aggregates map[string]*eventing.Aggregate) error {
	agg, ok := aggregates["pom.xml"]
	if !ok {
		return nil
	}
	b, err := renderLambdaPom(agg)
	if err != nil {
		return fmt.Errorf("render pom.xml: %w", err)
	}
	if err := project.WriteManaged(filepath.Join(dir, "pom.xml"), b); err != nil {
		return err
	}

	groupID := nodeStr(agg, "groupId", defaultGroupID)
	pkgDir := filepath.Join(dir, "src", "main", "java", groupToPath(groupID))
	testPkgDir := filepath.Join(dir, "src", "test", "java", groupToPath(groupID))

	handlerFile := filepath.Join(pkgDir, "Handler.java")
	if _, err := os.Stat(handlerFile); os.IsNotExist(err) {
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(handlerFile, handlerJava(groupID), 0644); err != nil {
			return err
		}
	}

	testFile := filepath.Join(testPkgDir, "HandlerTest.java")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		if err := os.MkdirAll(testPkgDir, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(testFile, handlerTestJava(groupID), 0644); err != nil {
			return err
		}
	}

	return nil
}

func renderLambdaPom(agg *eventing.Aggregate) ([]byte, error) {
	var buf bytes.Buffer
	renderPomHeader(&buf)
	renderPomCoords(&buf, agg)
	renderProperties(&buf, agg)
	renderDependencies(&buf, agg)
	renderBuild(&buf, []pomPlugin{surefirePlugin(), shadePlugin()})
	fmt.Fprintf(&buf, "</project>\n")
	return buf.Bytes(), nil
}

func handlerJava(pkg string) []byte {
	return []byte("package " + pkg + ";\n\n" +
		"import com.amazonaws.services.lambda.runtime.Context;\n" +
		"import com.amazonaws.services.lambda.runtime.RequestHandler;\n\n" +
		"public class Handler implements RequestHandler<Handler.Request, Handler.Response> {\n\n" +
		"    @Override\n" +
		"    public Response handleRequest(Request input, Context context) {\n" +
		"        return new Response(\"Hello, \" + input.name() + \"!\");\n" +
		"    }\n\n" +
		"    public record Request(String name) {}\n" +
		"    public record Response(String message) {}\n" +
		"}\n")
}

func handlerTestJava(pkg string) []byte {
	return []byte("package " + pkg + ";\n\n" +
		"import org.junit.jupiter.api.Test;\n" +
		"import static org.junit.jupiter.api.Assertions.*;\n\n" +
		"class HandlerTest {\n" +
		"    @Test\n" +
		"    void handlerReturnsGreeting() {\n" +
		"        Handler handler = new Handler();\n" +
		"        Handler.Response response = handler.handleRequest(new Handler.Request(\"World\"), null);\n" +
		"        assertEquals(\"Hello, World!\", response.message());\n" +
		"    }\n" +
		"}\n")
}
