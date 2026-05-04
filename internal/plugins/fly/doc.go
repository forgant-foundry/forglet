// Package fly synthesizes Fly.io deployment files for Go, Java, and Node.js projects.
// It manages fly.toml (FormatTOML) and .github/workflows/deploy.yml (FormatYAML),
// and scaffolds a language-appropriate Dockerfile once during Init.
//
// Activated via .forglet.yml:
//
//	fly: true                        # defaults only
//	fly:
//	  region: iad                    # Fly.io primary region (default: iad)
//	  port: 8080                     # internal HTTP port (default: 8080)
//	  memory: 256mb                  # VM memory (default: 256mb)
//	  cpus: 1                        # VM CPUs (default: 1)
//	  healthPath: /healthz           # health check HTTP path (default: /healthz)
//	  main: ./cmd/myapp              # Go main package for Dockerfile CMD (default: .; Go only)
//	  branch: main                   # deploy trigger branch (default: main)
//
// Dockerfile variants (selected by template prefix):
//
//	go*        multi-stage Go build (golang:1.24-alpine → alpine:3.21); main controls the build target
//	java*      multi-stage Maven build (maven:3.9-eclipse-temurin-21 → eclipse-temurin:21-jre-alpine)
//	node-ts*   multi-stage TypeScript build (node:22-alpine); runs npm run build, serves dist/index.js
//	node-js    single-stage Node.js (node:22-alpine); production deps only, serves src/index.js
package fly
