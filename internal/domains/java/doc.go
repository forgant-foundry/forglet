// Package java provides Maven-based Java synthesizers for forglet.
//
// All four synthesizers manage pom.xml as a synthesized (always-overwritten)
// file and write scaffold files exactly once during init. The RC overlay
// (via .forglet.yml) can override groupId, version, javaVersion,
// dependencies, and testDependencies across all four templates.
//
// # Synthesizers
//
// [Flat] (template "java") — a plain single-module Maven project with
// JUnit Jupiter and maven-surefire-plugin. Scaffolds App.java and AppTest.java.
//
// [Multimodule] (template "java-multimodule") — a Maven multi-module parent POM
// (packaging=pom) with <dependencyManagement> for JUnit and <pluginManagement>
// for surefire. Each declared module gets a scaffolded child pom.xml (with parent
// back-reference) and src/{main,test}/java layout. Default module: "core". Add
// modules via the rc "modules" key ([]string).
//
// [Lambda] (template "java-lambda") — extends Flat with aws-lambda-java-core and
// aws-lambda-java-events compile dependencies and the maven-shade-plugin to produce
// a fat JAR suitable for Lambda deployment. Scaffolds Handler.java (implementing
// RequestHandler with record-based Request/Response inner types) and HandlerTest.java.
//
// [Spring] (template "java-spring") — a Spring Boot project using
// spring-boot-starter-parent as the parent POM. Uses <java.version> instead of
// maven.compiler.source/target. Includes spring-boot-starter-web and
// spring-boot-starter-test. Scaffolds Application.java, HelloController.java,
// src/main/resources/application.properties, and ApplicationTest.java.
//
// # RC overlay keys
//
// All four templates recognise the following .forglet.yml keys:
//
//	groupId: "com.mycompany"
//	version: "2.0.0-SNAPSHOT"
//	javaVersion: "17"
//	dependencies:
//	  "com.google.guava:guava": "33.0.0-jre"
//	testDependencies:
//	  "org.mockito:mockito-core": "5.11.0"
//
// Multimodule additionally accepts:
//
//	modules:
//	  - core
//	  - api
//	  - web
package java
