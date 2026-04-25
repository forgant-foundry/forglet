// Package cdk provides a forglet plugin that adds AWS CDK support to node-ts
// projects.
//
// When registered, CDKPlugin contributes CDK devDependencies (aws-cdk,
// aws-cdk-lib, constructs) and a cdk.json configuration file (via
// FormatJSON) to every node-ts project. It also implements [project.Scaffolder]
// to write bin/app.ts and lib/stack.ts exactly once during init.
//
// CDKPlugin is template-scoped: it checks meta.Template == "node-ts" and
// does nothing for other templates.
//
// Register in a custom binary:
//
//	commands.RegisterPlugin(cdk.New())
package cdk
