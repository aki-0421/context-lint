# AGENTS.md

This repository contains `context-lint`, a Go CLI that validates AI-readable documentation graphs.

Start here:

- [Specification](docs/spec/context-lint.md)
- [README](README.md)

Implementation map:

- `cmd/context-lint` contains the CLI entry point.
- `internal/config` loads `.context-lint.{yaml,yml,json,jsonc}`.
- `internal/document` extracts Markdown references and builds the reachability graph.
- `internal/pattern` expands required and excluded paths.
- `internal/diagnostic` renders human and JSON diagnostics.
- `internal/runner` wires configuration, linting, output, and exit codes together.
- `action.yml` exposes the CLI as a composite GitHub Action.

Development commands:

```bash
go test ./...
go run ./cmd/context-lint --strict
```
