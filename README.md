# context-lint

`context-lint` is a CLI linter for AI-readable repository documentation.

It starts from one configured Markdown entry file, such as `AGENTS.md`, follows local references, and checks that required documents are reachable. It also detects local file references that point to missing files, including file paths written in prose or code blocks.

## Install

```bash
go install github.com/aki-0421/context-lint/cmd/context-lint@latest
```

## Configure

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

Supported configuration files:

- `.context-lint.yaml`
- `.context-lint.yml`
- `.context-lint.json`
- `.context-lint.jsonc`

## Run

```bash
context-lint
```

By default, document findings are warnings and exit with code `0`.

Use strict mode in CI when the documentation graph should be enforced:

```bash
context-lint --strict
```

Use JSON output for automation:

```bash
context-lint --format json
```

## GitHub Actions

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
      - uses: aki-0421/context-lint@v0.1.0
```

By default, the action uses the workflow working directory as the project root, discovers `.context-lint.{yaml,yml,json,jsonc}`, uses human output, and reports document findings as warnings.

Enable strict mode when the documentation graph should fail CI:

```yaml
- uses: aki-0421/context-lint@v0.1.0
  with:
    strict: true
```

Pass only the options you need to override:

```yaml
- uses: aki-0421/context-lint@v0.1.0
  with:
    config: .context-lint.yaml
    strict: true
```

## Release

Releases are built with GoReleaser. Tag a version and push it:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow builds archives and checksums for Linux, macOS, and Windows.
The same tag also publishes the GitHub Action entry point, so workflows can use `aki-0421/context-lint@v0.1.0`.

## Specification

See [docs/spec/context-lint.md](docs/spec/context-lint.md).
