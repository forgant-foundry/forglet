package cdk

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
)

// CDKPlugin adds AWS CDK support to any node template using the per-function
// Lambda pattern (NodejsFunction per workspace package). It manages package.json
// (CDK deps + scripts) and cdk.json via events, and scaffolds bin/app.ts and
// lib/stack.ts once during Init.
//
// CDKPlugin is monorepo-tool-agnostic. Register it with either:
//
//	project.New(dir).WithPlugins(workspaces.New(), cdk.New())              // workspaces only
//	project.New(dir).WithPlugins(workspaces.New(), lerna.New(), cdk.New()) // workspaces + lerna
type CDKPlugin struct{}

func New() *CDKPlugin { return &CDKPlugin{} }

func (p *CDKPlugin) Weave(meta project.Meta, rc map[string]any, stream *project.EventStream) error {
	depPayload, err := json.Marshal(map[string]any{
		"devDependencies": map[string]any{
			"aws-cdk-lib": "^2.0.0",
			"constructs":  "^10.0.0",
			"aws-cdk":     "^2.0.0",
			"esbuild":     "^0.25.0",
		},
	})
	if err != nil {
		return err
	}
	stream.Append("package.json", eventing.Event{
		ID:      newID(),
		Type:    "devDependency.added",
		Seq:     1,
		Payload: json.RawMessage(depPayload),
	})

	scriptPayload, err := json.Marshal(map[string]any{
		"scripts": map[string]any{
			"deploy":  "cdk deploy",
			"destroy": "cdk destroy",
			"synth":   "cdk synth",
		},
	})
	if err != nil {
		return err
	}
	stream.Append("package.json", eventing.Event{
		ID:      newID(),
		Type:    "scripts.added",
		Seq:     1,
		Payload: json.RawMessage(scriptPayload),
	})

	cdkPayload, err := json.Marshal(map[string]any{
		"app": "npx ts-node --prefer-ts-exts bin/app.ts",
	})
	if err != nil {
		return err
	}
	stream.SetFormat("cdk.json", project.FormatJSON)
	stream.Append("cdk.json", eventing.Event{
		ID:      newID(),
		Type:    "cdk.configured",
		Seq:     1,
		Payload: json.RawMessage(cdkPayload),
	})

	if enabled, _ := rc["git"].(bool); enabled {
		ignorePayload, err := json.Marshal(map[string]any{
			"cdk.out/":      "",
			".cdk.staging/": "",
		})
		if err != nil {
			return err
		}
		stream.SetFormat(".gitignore", project.FormatPattern)
		stream.Append(".gitignore", eventing.Event{
			ID:      newID(),
			Type:    "git.ignore",
			Seq:     1,
			Payload: json.RawMessage(ignorePayload),
		})
	}

	return nil
}

func (p *CDKPlugin) Scaffold(dir string, meta project.Meta) error {
	if err := scaffoldOnce(filepath.Join(dir, "bin", "app.ts"), binAppTS()); err != nil {
		return err
	}
	return scaffoldOnce(filepath.Join(dir, "lib", "stack.ts"), libStackTS())
}

func scaffoldOnce(path string, content []byte) error {
	if _, err := os.Stat(path); err == nil {
		return nil // already exists
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}

func binAppTS() []byte {
	return []byte(`#!/usr/bin/env node
import * as cdk from 'aws-cdk-lib';
import { AppStack } from '../lib/stack';

const app = new cdk.App();
new AppStack(app, 'AppStack');
`)
}

func libStackTS() []byte {
	return []byte(`import * as cdk from 'aws-cdk-lib';
import { NodejsFunction } from 'aws-cdk-lib/aws-lambda-nodejs';
import { Construct } from 'constructs';

export class AppStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    // Add Lambda functions here using NodejsFunction, e.g.:
    // new NodejsFunction(this, 'GetUser', {
    //   entry: 'packages/get-user/index.ts',
    //   handler: 'handler',
    // });
  }
}
`)
}

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
