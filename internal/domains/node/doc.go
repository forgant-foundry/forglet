// Package node provides Node.js synthesizers for forglet.
//
// All synthesizers manage package.json (and tsconfig.json for TypeScript
// variants) as synthesized files and write scaffold files exactly once during
// init. The RC overlay (via .forglet.yml) can add devDependencies, scripts,
// and other package.json fields.
//
// # Synthesizers
//
// [TypeScript] (template "node-ts") — a TypeScript project with ts-node,
// typescript, and @types/node as devDependencies, plus a tsconfig.json.
// Scaffolds src/index.ts.
//
// [JavaScript] (template "node-js") — a plain JavaScript project. Manages
// package.json only; scaffolds src/index.js.
//
// [Lambda] (template "node-lambda") — a TypeScript Lambda project using
// esbuild to bundle each function. The package is marked private=true with
// a build script that produces dist/handler.js from packages/handler/index.ts.
// Scaffolds packages/handler/index.ts (a typed async Lambda handler) and
// packages/handler/package.json.
//
// # RC overlay keys (all templates)
//
//	devDependencies:
//	  prettier: "^3.0.0"
//	scripts:
//	  lint: "eslint src/"
package node
