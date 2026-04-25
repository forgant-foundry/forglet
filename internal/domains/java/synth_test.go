package java_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	java "github.com/forgant-foundry/forglet/internal/domains/java"
	"github.com/forgant-foundry/forglet/internal/project"
	"github.com/forgant-foundry/forglet/internal/testutil"
)

// ---- Flat (java) ------------------------------------------------------------------

func TestJavaFlat_InitializeEvents_ArtifactId(t *testing.T) {
	s := java.NewFlat()
	events, err := s.InitializeEvents("myapp")
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(events)
	n, ok := aggs["pom.xml"].Node("artifactId")
	if !ok {
		t.Fatal("no artifactId node")
	}
	if got := n.Value.(string); got != "myapp" {
		t.Errorf("artifactId = %q, want %q", got, "myapp")
	}
}

func TestJavaFlat_InitializeEvents_JUnitPresent(t *testing.T) {
	s := java.NewFlat()
	events, err := s.InitializeEvents("myapp")
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(events)
	dir := testutil.TempDir(t)
	if err := s.Synthesize(dir, aggs); err != nil {
		t.Fatal(err)
	}
	assertContains(t, readFile(t, filepath.Join(dir, "pom.xml")), "junit-jupiter")
}

func TestJavaFlat_OverlayEvents_Empty(t *testing.T) {
	s := java.NewFlat()
	events, err := s.OverlayEvents(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if events != nil {
		t.Error("expected nil for empty rc")
	}
}

func TestJavaFlat_OverlayEvents_GroupIdOverride(t *testing.T) {
	s := java.NewFlat()
	templateEvents, _ := s.InitializeEvents("myapp")
	overlayEvents, err := s.OverlayEvents(map[string]any{"groupId": "org.acme"})
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(templateEvents, overlayEvents)
	n, _ := aggs["pom.xml"].Node("groupId")
	if got := n.Value.(string); got != "org.acme" {
		t.Errorf("groupId = %q, want %q", got, "org.acme")
	}
}

func TestJavaFlat_OverlayEvents_DepsAccumulate(t *testing.T) {
	s := java.NewFlat()
	dir := testutil.TempDir(t)
	templateEvents, _ := s.InitializeEvents("myapp")
	overlayEvents, err := s.OverlayEvents(map[string]any{
		"dependencies": map[string]any{"com.google.guava:guava": "33.0.0-jre"},
	})
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(templateEvents, overlayEvents)
	if err := s.Synthesize(dir, aggs); err != nil {
		t.Fatal(err)
	}
	pom := readFile(t, filepath.Join(dir, "pom.xml"))
	assertContains(t, pom, "junit-jupiter")
	assertContains(t, pom, "guava")
}

func TestJavaFlat_Synthesize_PomXmlContent(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewFlat()
	if err := project.New(dir).Init(project.Meta{Name: "myapp", Template: "java"}, s); err != nil {
		t.Fatal(err)
	}
	pom := readFile(t, filepath.Join(dir, "pom.xml"))
	assertContains(t, pom, "<artifactId>myapp</artifactId>")
	assertContains(t, pom, "<groupId>com.example</groupId>")
	assertContains(t, pom, "maven-surefire-plugin")
}

func TestJavaFlat_Synthesize_AppJavaScaffolded(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewFlat()
	if err := project.New(dir).Init(project.Meta{Name: "myapp", Template: "java"}, s); err != nil {
		t.Fatal(err)
	}
	appFile := filepath.Join(dir, "src", "main", "java", "com", "example", "App.java")
	if _, err := os.Stat(appFile); err != nil {
		t.Errorf("App.java not created: %v", err)
	}
}

func TestJavaFlat_Synthesize_AppJavaNotOverwritten(t *testing.T) {
	dir := testutil.TempDir(t)
	appDir := filepath.Join(dir, "src", "main", "java", "com", "example")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	custom := "package com.example;\npublic class App { /* custom */ }\n"
	appFile := filepath.Join(appDir, "App.java")
	if err := os.WriteFile(appFile, []byte(custom), 0644); err != nil {
		t.Fatal(err)
	}
	s := java.NewFlat()
	if err := project.New(dir).Init(project.Meta{Name: "myapp", Template: "java"}, s); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, appFile); got != custom {
		t.Errorf("App.java was overwritten")
	}
}

func TestJavaFlat_Idempotent(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewFlat()
	p := project.New(dir)
	if err := p.Init(project.Meta{Name: "myapp", Template: "java"}, s); err != nil {
		t.Fatal(err)
	}
	first := readFile(t, filepath.Join(dir, "pom.xml"))
	if err := p.Synthesize(s); err != nil {
		t.Fatal(err)
	}
	second := readFile(t, filepath.Join(dir, "pom.xml"))
	if first != second {
		t.Errorf("pom.xml changed after second synth")
	}
}

func TestJavaFlat_Synthesize_PomXmlReadOnly(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewFlat()
	if err := project.New(dir).Init(project.Meta{Name: "myapp", Template: "java"}, s); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "pom.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0444 {
		t.Errorf("pom.xml mode = %04o, want 0444", info.Mode().Perm())
	}
}

func TestJavaFlat_Synthesize_PomXmlHasMarker(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewFlat()
	if err := project.New(dir).Init(project.Meta{Name: "myapp", Template: "java"}, s); err != nil {
		t.Fatal(err)
	}
	pom := readFile(t, filepath.Join(dir, "pom.xml"))
	assertContains(t, pom, "forglet")
}

// ---- Multimodule (java-multimodule) -----------------------------------------------

func TestJavaMultimodule_InitializeEvents_PackagingIsPom(t *testing.T) {
	s := java.NewMultimodule()
	events, err := s.InitializeEvents("myparent")
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(events)
	n, ok := aggs["pom.xml"].Node("packaging")
	if !ok {
		t.Fatal("no packaging node")
	}
	if got := n.Value.(string); got != "pom" {
		t.Errorf("packaging = %q, want %q", got, "pom")
	}
}

func TestJavaMultimodule_InitializeEvents_DefaultModuleIsCore(t *testing.T) {
	s := java.NewMultimodule()
	events, err := s.InitializeEvents("myparent")
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(events)
	dir := testutil.TempDir(t)
	if err := s.Synthesize(dir, aggs); err != nil {
		t.Fatal(err)
	}
	assertContains(t, readFile(t, filepath.Join(dir, "pom.xml")), "<module>core</module>")
}

func TestJavaMultimodule_InitializeEvents_JUnitManagedDep(t *testing.T) {
	s := java.NewMultimodule()
	events, err := s.InitializeEvents("myparent")
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(events)
	dir := testutil.TempDir(t)
	if err := s.Synthesize(dir, aggs); err != nil {
		t.Fatal(err)
	}
	pom := readFile(t, filepath.Join(dir, "pom.xml"))
	assertContains(t, pom, "<dependencyManagement>")
	assertContains(t, pom, "junit-jupiter")
}

func TestJavaMultimodule_OverlayEvents_Empty(t *testing.T) {
	s := java.NewMultimodule()
	events, err := s.OverlayEvents(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if events != nil {
		t.Error("expected nil for empty rc")
	}
}

func TestJavaMultimodule_OverlayEvents_ModulesAdded(t *testing.T) {
	s := java.NewMultimodule()
	templateEvents, _ := s.InitializeEvents("myparent")
	overlayEvents, err := s.OverlayEvents(map[string]any{
		"modules": []any{"core", "api", "web"},
	})
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(templateEvents, overlayEvents)
	dir := testutil.TempDir(t)
	if err := s.Synthesize(dir, aggs); err != nil {
		t.Fatal(err)
	}
	pom := readFile(t, filepath.Join(dir, "pom.xml"))
	assertContains(t, pom, "<module>api</module>")
	assertContains(t, pom, "<module>web</module>")
}

func TestJavaMultimodule_Synthesize_ParentPomHasPluginManagement(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewMultimodule()
	if err := project.New(dir).Init(project.Meta{Name: "myparent", Template: "java-multimodule"}, s); err != nil {
		t.Fatal(err)
	}
	assertContains(t, readFile(t, filepath.Join(dir, "pom.xml")), "<pluginManagement>")
}

func TestJavaMultimodule_Synthesize_ModuleDirScaffolded(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewMultimodule()
	if err := project.New(dir).Init(project.Meta{Name: "myparent", Template: "java-multimodule"}, s); err != nil {
		t.Fatal(err)
	}
	corePom := filepath.Join(dir, "core", "pom.xml")
	if _, err := os.Stat(corePom); err != nil {
		t.Errorf("core/pom.xml not created: %v", err)
	}
	coreSrc := filepath.Join(dir, "core", "src", "main", "java", "com", "example")
	if _, err := os.Stat(coreSrc); err != nil {
		t.Errorf("core src dir not created: %v", err)
	}
}

func TestJavaMultimodule_Synthesize_ChildPomHasParentRef(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewMultimodule()
	if err := project.New(dir).Init(project.Meta{Name: "myparent", Template: "java-multimodule"}, s); err != nil {
		t.Fatal(err)
	}
	childPom := readFile(t, filepath.Join(dir, "core", "pom.xml"))
	assertContains(t, childPom, "<parent>")
	assertContains(t, childPom, "<artifactId>myparent</artifactId>")
	assertContains(t, childPom, "<artifactId>core</artifactId>")
}

func TestJavaMultimodule_Synthesize_ModulePomNotOverwritten(t *testing.T) {
	dir := testutil.TempDir(t)
	modDir := filepath.Join(dir, "core")
	if err := os.MkdirAll(modDir, 0755); err != nil {
		t.Fatal(err)
	}
	custom := "<!-- custom -->\n<project/>\n"
	modPom := filepath.Join(modDir, "pom.xml")
	if err := os.WriteFile(modPom, []byte(custom), 0644); err != nil {
		t.Fatal(err)
	}
	s := java.NewMultimodule()
	if err := project.New(dir).Init(project.Meta{Name: "myparent", Template: "java-multimodule"}, s); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, modPom); got != custom {
		t.Errorf("core/pom.xml was overwritten")
	}
}

func TestJavaMultimodule_Idempotent(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewMultimodule()
	p := project.New(dir)
	if err := p.Init(project.Meta{Name: "myparent", Template: "java-multimodule"}, s); err != nil {
		t.Fatal(err)
	}
	first := readFile(t, filepath.Join(dir, "pom.xml"))
	if err := p.Synthesize(s); err != nil {
		t.Fatal(err)
	}
	second := readFile(t, filepath.Join(dir, "pom.xml"))
	if first != second {
		t.Errorf("pom.xml changed after second synth")
	}
}

func TestJavaMultimodule_Synthesize_PomXmlReadOnly(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewMultimodule()
	if err := project.New(dir).Init(project.Meta{Name: "myparent", Template: "java-multimodule"}, s); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "pom.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0444 {
		t.Errorf("pom.xml mode = %04o, want 0444", info.Mode().Perm())
	}
}

// ---- Lambda (java-lambda) ----------------------------------------------------------

func TestJavaLambda_InitializeEvents_ArtifactId(t *testing.T) {
	s := java.NewLambda()
	events, err := s.InitializeEvents("myhandler")
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(events)
	n, ok := aggs["pom.xml"].Node("artifactId")
	if !ok {
		t.Fatal("no artifactId node")
	}
	if got := n.Value.(string); got != "myhandler" {
		t.Errorf("artifactId = %q, want %q", got, "myhandler")
	}
}

func TestJavaLambda_InitializeEvents_LambdaCoreDep(t *testing.T) {
	s := java.NewLambda()
	events, err := s.InitializeEvents("myhandler")
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(events)
	dir := testutil.TempDir(t)
	if err := s.Synthesize(dir, aggs); err != nil {
		t.Fatal(err)
	}
	pom := readFile(t, filepath.Join(dir, "pom.xml"))
	assertContains(t, pom, "aws-lambda-java-core")
	assertContains(t, pom, "aws-lambda-java-events")
}

func TestJavaLambda_InitializeEvents_JUnitTestDep(t *testing.T) {
	s := java.NewLambda()
	events, err := s.InitializeEvents("myhandler")
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(events)
	dir := testutil.TempDir(t)
	if err := s.Synthesize(dir, aggs); err != nil {
		t.Fatal(err)
	}
	pom := readFile(t, filepath.Join(dir, "pom.xml"))
	assertContains(t, pom, "junit-jupiter")
	assertContains(t, pom, "<scope>test</scope>")
}

func TestJavaLambda_OverlayEvents_Empty(t *testing.T) {
	s := java.NewLambda()
	events, err := s.OverlayEvents(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if events != nil {
		t.Error("expected nil for empty rc")
	}
}

func TestJavaLambda_OverlayEvents_DepsAccumulate(t *testing.T) {
	s := java.NewLambda()
	dir := testutil.TempDir(t)
	templateEvents, _ := s.InitializeEvents("myhandler")
	overlayEvents, err := s.OverlayEvents(map[string]any{
		"dependencies": map[string]any{"com.fasterxml.jackson.core:jackson-databind": "2.17.0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(templateEvents, overlayEvents)
	if err := s.Synthesize(dir, aggs); err != nil {
		t.Fatal(err)
	}
	pom := readFile(t, filepath.Join(dir, "pom.xml"))
	assertContains(t, pom, "aws-lambda-java-core")
	assertContains(t, pom, "jackson-databind")
}

func TestJavaLambda_Synthesize_PomXmlHasShadePlugin(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewLambda()
	if err := project.New(dir).Init(project.Meta{Name: "myhandler", Template: "java-lambda"}, s); err != nil {
		t.Fatal(err)
	}
	assertContains(t, readFile(t, filepath.Join(dir, "pom.xml")), "maven-shade-plugin")
}

func TestJavaLambda_Synthesize_HandlerJavaScaffolded(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewLambda()
	if err := project.New(dir).Init(project.Meta{Name: "myhandler", Template: "java-lambda"}, s); err != nil {
		t.Fatal(err)
	}
	handlerFile := filepath.Join(dir, "src", "main", "java", "com", "example", "Handler.java")
	if _, err := os.Stat(handlerFile); err != nil {
		t.Errorf("Handler.java not created: %v", err)
	}
	assertContains(t, readFile(t, handlerFile), "RequestHandler")
}

func TestJavaLambda_Synthesize_HandlerTestJavaScaffolded(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewLambda()
	if err := project.New(dir).Init(project.Meta{Name: "myhandler", Template: "java-lambda"}, s); err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(dir, "src", "test", "java", "com", "example", "HandlerTest.java")
	if _, err := os.Stat(testFile); err != nil {
		t.Errorf("HandlerTest.java not created: %v", err)
	}
}

func TestJavaLambda_Synthesize_HandlerJavaNotOverwritten(t *testing.T) {
	dir := testutil.TempDir(t)
	handlerDir := filepath.Join(dir, "src", "main", "java", "com", "example")
	if err := os.MkdirAll(handlerDir, 0755); err != nil {
		t.Fatal(err)
	}
	custom := "package com.example;\npublic class Handler { /* custom */ }\n"
	handlerFile := filepath.Join(handlerDir, "Handler.java")
	if err := os.WriteFile(handlerFile, []byte(custom), 0644); err != nil {
		t.Fatal(err)
	}
	s := java.NewLambda()
	if err := project.New(dir).Init(project.Meta{Name: "myhandler", Template: "java-lambda"}, s); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, handlerFile); got != custom {
		t.Errorf("Handler.java was overwritten")
	}
}

func TestJavaLambda_Idempotent(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewLambda()
	p := project.New(dir)
	if err := p.Init(project.Meta{Name: "myhandler", Template: "java-lambda"}, s); err != nil {
		t.Fatal(err)
	}
	first := readFile(t, filepath.Join(dir, "pom.xml"))
	if err := p.Synthesize(s); err != nil {
		t.Fatal(err)
	}
	second := readFile(t, filepath.Join(dir, "pom.xml"))
	if first != second {
		t.Errorf("pom.xml changed after second synth")
	}
}

func TestJavaLambda_Synthesize_PomXmlReadOnly(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewLambda()
	if err := project.New(dir).Init(project.Meta{Name: "myhandler", Template: "java-lambda"}, s); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "pom.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0444 {
		t.Errorf("pom.xml mode = %04o, want 0444", info.Mode().Perm())
	}
}

// ---- Spring (java-spring) ----------------------------------------------------------

func TestJavaSpring_InitializeEvents_ArtifactId(t *testing.T) {
	s := java.NewSpring()
	events, err := s.InitializeEvents("myservice")
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(events)
	n, ok := aggs["pom.xml"].Node("artifactId")
	if !ok {
		t.Fatal("no artifactId node")
	}
	if got := n.Value.(string); got != "myservice" {
		t.Errorf("artifactId = %q, want %q", got, "myservice")
	}
}

func TestJavaSpring_InitializeEvents_SpringWebDep(t *testing.T) {
	s := java.NewSpring()
	events, err := s.InitializeEvents("myservice")
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(events)
	dir := testutil.TempDir(t)
	if err := s.Synthesize(dir, aggs); err != nil {
		t.Fatal(err)
	}
	pom := readFile(t, filepath.Join(dir, "pom.xml"))
	assertContains(t, pom, "spring-boot-starter-web")
	assertContains(t, pom, "spring-boot-starter-test")
}

func TestJavaSpring_InitializeEvents_DefaultVersionIsSnapshot(t *testing.T) {
	s := java.NewSpring()
	events, err := s.InitializeEvents("myservice")
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(events)
	n, _ := aggs["pom.xml"].Node("version")
	if got := n.Value.(string); got != "0.0.1-SNAPSHOT" {
		t.Errorf("version = %q, want 0.0.1-SNAPSHOT", got)
	}
}

func TestJavaSpring_OverlayEvents_Empty(t *testing.T) {
	s := java.NewSpring()
	events, err := s.OverlayEvents(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if events != nil {
		t.Error("expected nil for empty rc")
	}
}

func TestJavaSpring_OverlayEvents_DepsAccumulate(t *testing.T) {
	s := java.NewSpring()
	dir := testutil.TempDir(t)
	templateEvents, _ := s.InitializeEvents("myservice")
	overlayEvents, err := s.OverlayEvents(map[string]any{
		"dependencies": map[string]any{"org.springframework.boot:spring-boot-starter-data-jpa": ""},
	})
	if err != nil {
		t.Fatal(err)
	}
	aggs := project.BuildAggregates(templateEvents, overlayEvents)
	if err := s.Synthesize(dir, aggs); err != nil {
		t.Fatal(err)
	}
	pom := readFile(t, filepath.Join(dir, "pom.xml"))
	assertContains(t, pom, "spring-boot-starter-web")
	assertContains(t, pom, "spring-boot-starter-data-jpa")
}

func TestJavaSpring_Synthesize_PomXmlHasParent(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewSpring()
	if err := project.New(dir).Init(project.Meta{Name: "myservice", Template: "java-spring"}, s); err != nil {
		t.Fatal(err)
	}
	pom := readFile(t, filepath.Join(dir, "pom.xml"))
	assertContains(t, pom, "<parent>")
	assertContains(t, pom, "spring-boot-starter-parent")
	assertContains(t, pom, "<relativePath/>")
}

func TestJavaSpring_Synthesize_PomXmlUsesJavaVersionProperty(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewSpring()
	if err := project.New(dir).Init(project.Meta{Name: "myservice", Template: "java-spring"}, s); err != nil {
		t.Fatal(err)
	}
	pom := readFile(t, filepath.Join(dir, "pom.xml"))
	assertContains(t, pom, "<java.version>")
	if strings.Contains(pom, "maven.compiler.source") {
		t.Error("pom.xml should not contain maven.compiler.source for Spring projects")
	}
}

func TestJavaSpring_Synthesize_ApplicationJavaScaffolded(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewSpring()
	if err := project.New(dir).Init(project.Meta{Name: "myservice", Template: "java-spring"}, s); err != nil {
		t.Fatal(err)
	}
	appFile := filepath.Join(dir, "src", "main", "java", "com", "example", "Application.java")
	if _, err := os.Stat(appFile); err != nil {
		t.Errorf("Application.java not created: %v", err)
	}
	assertContains(t, readFile(t, appFile), "SpringApplication")
}

func TestJavaSpring_Synthesize_HelloControllerScaffolded(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewSpring()
	if err := project.New(dir).Init(project.Meta{Name: "myservice", Template: "java-spring"}, s); err != nil {
		t.Fatal(err)
	}
	ctrlFile := filepath.Join(dir, "src", "main", "java", "com", "example", "HelloController.java")
	if _, err := os.Stat(ctrlFile); err != nil {
		t.Errorf("HelloController.java not created: %v", err)
	}
}

func TestJavaSpring_Synthesize_ApplicationPropertiesScaffolded(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewSpring()
	if err := project.New(dir).Init(project.Meta{Name: "myservice", Template: "java-spring"}, s); err != nil {
		t.Fatal(err)
	}
	propsFile := filepath.Join(dir, "src", "main", "resources", "application.properties")
	if _, err := os.Stat(propsFile); err != nil {
		t.Errorf("application.properties not created: %v", err)
	}
	assertContains(t, readFile(t, propsFile), "server.port")
}

func TestJavaSpring_Synthesize_ApplicationJavaNotOverwritten(t *testing.T) {
	dir := testutil.TempDir(t)
	appDir := filepath.Join(dir, "src", "main", "java", "com", "example")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	custom := "package com.example;\npublic class Application { /* custom */ }\n"
	appFile := filepath.Join(appDir, "Application.java")
	if err := os.WriteFile(appFile, []byte(custom), 0644); err != nil {
		t.Fatal(err)
	}
	s := java.NewSpring()
	if err := project.New(dir).Init(project.Meta{Name: "myservice", Template: "java-spring"}, s); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, appFile); got != custom {
		t.Errorf("Application.java was overwritten")
	}
}

func TestJavaSpring_Idempotent(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewSpring()
	p := project.New(dir)
	if err := p.Init(project.Meta{Name: "myservice", Template: "java-spring"}, s); err != nil {
		t.Fatal(err)
	}
	first := readFile(t, filepath.Join(dir, "pom.xml"))
	if err := p.Synthesize(s); err != nil {
		t.Fatal(err)
	}
	second := readFile(t, filepath.Join(dir, "pom.xml"))
	if first != second {
		t.Errorf("pom.xml changed after second synth")
	}
}

func TestJavaSpring_Synthesize_PomXmlReadOnly(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewSpring()
	if err := project.New(dir).Init(project.Meta{Name: "myservice", Template: "java-spring"}, s); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "pom.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0444 {
		t.Errorf("pom.xml mode = %04o, want 0444", info.Mode().Perm())
	}
}

func TestJavaSpring_Synthesize_PomXmlHasSpringBootMavenPlugin(t *testing.T) {
	dir := testutil.TempDir(t)
	s := java.NewSpring()
	if err := project.New(dir).Init(project.Meta{Name: "myservice", Template: "java-spring"}, s); err != nil {
		t.Fatal(err)
	}
	assertContains(t, readFile(t, filepath.Join(dir, "pom.xml")), "spring-boot-maven-plugin")
}

// ---- helpers ----------------------------------------------------------------------

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func assertContains(t *testing.T, content, want string) {
	t.Helper()
	if !strings.Contains(content, want) {
		t.Errorf("expected %q in:\n%s", want, content)
	}
}
