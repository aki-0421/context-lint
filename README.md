# context-lint

`context-lint` is a CLI and GitHub Action for keeping AI-readable repository documentation reachable from one Markdown entry point.

It starts from a configured entry file, such as `AGENTS.md`, follows local Markdown references and Markdown file-path-like text, and reports documents that are missing or required but unreachable. The goal is simple: if an AI agent is expected to understand a repository from its docs, the docs must form a navigable graph.

## Why

AI agents work best when repository context is short, explicit, and linked. A single entry file should act like a table of contents, pointing to deeper documents only when they are relevant.

`context-lint` helps maintain that shape by checking:

- the entry Markdown file exists;
- local Markdown document references point to existing files;
- Markdown file paths written in prose or code blocks point to existing files;
- required documentation is reachable from the entry file;
- CI can enforce the documentation graph when a project is ready.

## Status

This project is early-stage. The core CLI, GitHub Action entry point, tests, CI, and release workflow are in place. Feedback and small, focused contributions are welcome.

## Install

```bash
go install github.com/aki-0421/context-lint/cmd/context-lint@latest
```

## Quick Start

Create `.context-lint.yaml` at the project root:

```yaml
linter:
  document:
    entry: AGENTS.md
    requiredReachable:
      - docs
    excludes:
      - README.md
```

Run the linter:

```bash
context-lint
```

By default, document findings are warnings and exit with code `0`.

Use strict mode when findings should fail the command:

```bash
context-lint --strict
```

Use JSON output for automation:

```bash
context-lint --format json
```

## GitHub Action

The simplest workflow only needs checkout and this action:

```yaml
name: context-lint

on:
  pull_request:
  push:
    branches: [main]

jobs:
  context-lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: aki-0421/context-lint@v1
```

The action uses the workflow working directory as the project root, discovers `.context-lint.{yaml,yml,json,jsonc}`, uses human output, and reports document findings as warnings.

Enable strict mode when the documentation graph should fail CI:

```yaml
- uses: aki-0421/context-lint@v1
  with:
    strict: true
```

Pass only the options you need to override:

```yaml
- uses: aki-0421/context-lint@v1
  with:
    config: .context-lint.yaml
    strict: true
```

## Configuration

Supported configuration files, in discovery order:

1. `.context-lint.yaml`
2. `.context-lint.yml`
3. `.context-lint.json`
4. `.context-lint.jsonc`

Schema:

```yaml
linter:
  document:
    entry: string
    requiredReachable: string[]
    excludes: string[]
```

`entry` is required and must point to one Markdown file.

`requiredReachable` accepts files, directories, and glob patterns. Directories expand to Markdown files under that directory.

`excludes` accepts files, directories, and glob patterns. Excluded files are ignored by reachability and missing Markdown-reference checks.

## CLI Reference

```text
--config <path>     Use a specific configuration file
--root <path>       Use a specific project root
--strict            Treat warnings as errors and exit non-zero when findings exist
--format <format>   Output format: human or json. Default: human
--no-color          Disable ANSI color
--version           Print the version
--help              Print help
```

Exit codes:

| Condition | Default | `--strict` |
| --- | --- | --- |
| No diagnostics | `0` | `0` |
| Warnings only | `0` | `1` |
| Invalid configuration | `2` | `2` |
| Runtime failure | `2` | `2` |

## Development

Requirements:

- Go 1.23 or newer

Common commands:

```bash
go test ./...
go vet ./...
go run ./cmd/context-lint --strict
```

Build locally:

```bash
go build ./cmd/context-lint
```

Check the release configuration:

```bash
go run github.com/goreleaser/goreleaser/v2@latest check
```

## Contributing

Contributions are welcome, especially focused fixes, tests, documentation improvements, and small rule improvements.

Please follow these guidelines:

- Keep changes small and reviewable.
- Open an issue first for large behavior changes or new rule categories.
- Add or update tests for user-visible behavior.
- Keep documentation in English.
- Run `go test ./...` before opening a pull request.
- Do not commit generated release artifacts from `dist`.
- Be respectful and assume good intent in issues, reviews, and discussions.

By submitting a contribution, you agree that your contribution will be licensed under the MIT License.

## Security

Please do not report security issues in public issues. Use GitHub private vulnerability reporting if it is available for this repository, or contact the maintainer privately through GitHub.

## Specification

See [docs/spec/context-lint.md](docs/spec/context-lint.md).

## License

MIT
