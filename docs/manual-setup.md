# Manual Setup

Prefer the [`context-lint-setup`](../skills/context-lint-setup/SKILL.md) Agent Skill when an AI agent can handle setup. Use this page for manual installation, configuration, CI, and CLI reference.

## Install

```bash
go install github.com/aki-0421/context-lint/cmd/context-lint@latest
```

## Basic Configuration

Create `.context-lint.yaml` at the project root:

```yaml
linter:
  document:
    entry: <agent-entry-markdown-file>
    requiredReachable:
      - <required-docs-path>
    excludes:
      - <excluded-path>
```

Replace the placeholder values with paths that fit the repository. Use an agent-facing entry file such as `AGENTS.md`, `CLAUDE.md`, or another Markdown instruction file supported by the agent workflow.

Omit `requiredReachable` or `excludes` when there is no path to list yet.

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

List Markdown front matter under a directory:

```bash
context-lint list docs
```

Only files with leading YAML front matter are printed. Markdown files without front matter are reported as warnings, or errors when `--strict` is enabled.

## GitHub Action

The simplest workflow only needs checkout and this action:

```yaml
name: context-lint

on:
  pull_request:
  push:
    branches:
      - <default-branch>

permissions:
  contents: read

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
    requiredReachableMaxFileSize: string | number
    excludes: string[]
    frontMatter:
      excludeFileNames: string[]
```

`entry` is required and must point to one Markdown file.

`requiredReachable` accepts files, directories, and glob patterns. Directories expand to Markdown files under that directory.

`requiredReachableMaxFileSize` is optional and can usually be omitted. It defaults to `32 KiB` and reports required Markdown files that reach or exceed the limit so agents and humans know to split large context files into smaller linked documents. Set it only when a project needs a different limit, using values such as `64 KiB`, `1 MiB`, or a byte count.

`excludes` accepts files, directories, and glob patterns. Excluded files are ignored by reachability and missing Markdown-reference checks.

`frontMatter.excludeFileNames` is optional. It excludes matching Markdown file names from missing-front-matter warnings and errors in `context-lint list`. `index.md` is always excluded by default as a routing document, so the field can usually be omitted.

Run `context-lint config-guide` to print a short configuration example for front matter exclusions.

## CLI Reference

```text
context-lint [flags]
context-lint list <path> [flags]
context-lint config-guide
```

Flags:

```text
--config <path>     Use a specific configuration file
--root <path>       Use a specific project root
--strict            Treat warnings as errors and exit non-zero when findings exist
--format <format>   Output format: human or json. Default: human
--no-color          Disable ANSI color
--version           Print the version
--help              Print help
```

The `list` command supports `--config`, `--root`, `--strict`, `--format`, `--no-color`, and recursively scans Markdown files under `<path>`.

The `config-guide` command prints configuration examples for common diagnostics, including excluding file names from missing-front-matter checks.

Exit codes:

| Condition | Default | `--strict` |
| --- | --- | --- |
| No diagnostics | `0` | `0` |
| Warnings only | `0` | `1` |
| Invalid configuration | `2` | `2` |
| Runtime failure | `2` | `2` |
