# context-lint Specification

## Purpose

`context-lint` is a CLI linter that keeps repository documentation readable and reachable for AI agents.

The OpenAI Harness Engineering article describes a repository knowledge model where `AGENTS.md` is not a giant manual, but a short table of contents that points agents to deeper, trusted documents. `context-lint` turns that idea into a mechanical check: starting from one configured Markdown entry file, it verifies that important documents are reachable through a tree of local references.

Reference:

- https://openai.com/ja-JP/index/harness-engineering/

## Scope

### Problems It Solves

- Important documents are not reachable from the entry context file.
- Markdown document references point to files that do not exist.
- Markdown file paths written in prose or code blocks act as practical references, even when they are not strict Markdown links.
- AI-readable documentation structure is not continuously validated in CI.

### Initial Scope

- Implement the tool as a Go CLI.
- Make it installable with `go install`.
- Provide exit codes and output formats suitable for GitHub Actions.
- Load configuration from `.context-lint.{ext}` at the project root.
- Support YAML, JSON, and JSONC configuration files.
- Extract references from Markdown links and file-path-like text.
- Detect local Markdown document references that do not exist.
- Detect configured `requiredReachable` files, directories, or globs that are not reachable from the entry file.
- Report AI-readable warnings by default.
- Treat the same findings as errors when `--strict` is enabled.

### Out Of Scope For The Initial Version

- Checking external URL availability.
- Detecting stale documentation from Git history.
- Semantic consistency checks between code behavior and documentation content.
- Multiple entry files in one run.
- Recursive parsing of document formats other than Markdown.

## Terms

- Entry file: The single Markdown file an AI agent should read first, such as `AGENTS.md`.
- Reachable: A file can be reached by following local references starting from the entry file.
- Reference: A Markdown link, image link, local HTML link, or file-path-like string in Markdown content.
- `requiredReachable`: Files, directories, or globs that must be reachable from the entry file.
- `excludes`: Files, directories, or globs excluded from linting.
- `frontMatter.excludeFileNames`: Markdown file names excluded from missing-front-matter diagnostics.
- Diagnostic: A warning or error emitted by the linter.

## Configuration

### Discovery

`context-lint` treats the current working directory, or the directory passed with `--root`, as the project root. It searches for configuration files in this order:

1. `.context-lint.yaml`
2. `.context-lint.yml`
3. `.context-lint.json`
4. `.context-lint.jsonc`

If multiple configuration files exist, the first one in the list is used and the rest are ignored. If `--config` is provided, only that file is used.

### Example

```yaml
linter:
  document:
    entry: AGENTS.md
    requiredReachable:
      - docs
    excludes:
      - README.md
    frontMatter:
      excludeFileNames:
        - CHANGELOG.md
```

### Schema

```yaml
linter:
  document:
    entry: string
    requiredReachable: string[]
    excludes: string[]
    frontMatter:
      excludeFileNames: string[]
```

### Fields

`linter.document.entry` is required. It must be a project-root-relative path to a single Markdown file.

`linter.document.requiredReachable` is optional. Each item may be a file, directory, or glob. When a directory is specified, Markdown files under that directory are treated as required targets.

`linter.document.excludes` is optional. Each item may be a file, directory, or glob. Excluded files are ignored by reachability checks, missing Markdown-reference checks, and `requiredReachable` expansion.

`linter.document.frontMatter.excludeFileNames` is optional. Each item is interpreted as a file name, not a project-root-relative path. Matching Markdown files are skipped only for missing-front-matter diagnostics from `context-lint list`. `index.md` is always skipped by default because it is treated as a routing document.

`context-lint config-guide` prints a short configuration example that shows this setting.

## CLI

### Command

```bash
context-lint [flags]
context-lint list <path> [flags]
context-lint config-guide
```

### Flags

```text
--config <path>     Use a specific configuration file
--root <path>       Use a specific project root
--strict            Treat warnings as errors and exit non-zero when findings exist
--format <format>   Output format: human or json. Default: human
--no-color          Disable ANSI color
--version           Print the version
--help              Print help
```

The `list` command prints only the leading YAML front matter blocks for Markdown files under `<path>`. Files without front matter are not printed as front matter entries; they emit `CL007` diagnostics. With `--strict`, those diagnostics are errors and the command exits non-zero. Missing-front-matter diagnostics are not emitted for `index.md` files, nor for file names listed in `linter.document.frontMatter.excludeFileNames`.

The `config-guide` command prints configuration examples for common diagnostics. `CL007` fix messages should mention this command so users can discover `linter.document.frontMatter.excludeFileNames` when they want to exclude routing or generated files.

For `--format json`, `list` returns:

```json
{
  "ok": true,
  "strict": false,
  "root": "/repo",
  "path": "docs",
  "frontMatter": [
    {
      "file": "docs/guide.md",
      "content": "---\ntitle: Guide\n---\n"
    }
  ],
  "diagnostics": []
}
```

### Exit Codes

| Condition | Default | `--strict` |
| --- | --- | --- |
| No diagnostics | `0` | `0` |
| Warnings only | `0` | `1` |
| Invalid configuration | `2` | `2` |
| Runtime failure | `2` | `2` |

In the initial version, all document findings are warnings by default. With `--strict`, the same findings become errors.

## Validation Model

### Flow

1. Resolve the project root.
2. Load the configuration file.
3. Validate that `entry` exists and is a Markdown file.
4. Expand `requiredReachable` and `excludes`.
5. Build a reference graph starting from `entry`.
6. Check whether referenced local Markdown files exist.
7. Check whether all required targets are reachable.
8. Print diagnostics and decide the exit code.

### Reference Graph

Each Markdown file is a node. Each local file reference inside that Markdown file is an edge.

If the referenced target is a Markdown file, the target is parsed recursively. If an explicit Markdown or HTML link points to a directory, Markdown files under that directory are treated as candidate targets. Non-Markdown targets, such as code files, images, PDFs, and text files, may be recorded when they exist but do not produce missing-file diagnostics when absent.

### Path Resolution

Relative paths are resolved from the directory of the Markdown file that contains the reference.

Root-relative paths are resolved from the project root. For example, `/docs/spec.md` resolves to `<root>/docs/spec.md`.

Anchor links use only their file portion for Markdown existence checks. For example, `docs/api.md#usage` resolves to `docs/api.md`. The initial version does not validate whether the anchor exists.

External URLs, email addresses, and URI-scheme references are ignored. Examples include `https://example.com`, `mailto:foo@example.com`, and `vscode://file/...`.

## Markdown Parsing

Markdown parsing should use a CommonMark-compatible parser. The initial implementation should use an existing Go library to extract inline links, reference links, and image links.

### Extracted References

- Inline links: `[text](docs/foo.md)`
- Image links: `![alt](docs/image.png)`
- Reference links: `[text][foo]` and `[foo]: docs/foo.md`
- Autolinks that can be interpreted as local files
- Local references in HTML `a[href]` and `img[src]` attributes
- File-path-like strings in prose or code blocks

### File-Path-Like Strings

A non-link string is treated as a reference candidate when it matches these rules:

- It contains `/` or `\`.
- Or it has a known extension such as `.md`, `.mdx`, `.txt`, `.json`, `.yaml`, `.yml`, `.png`, `.jpg`, `.jpeg`, `.svg`, or `.pdf`.
- It can be split at boundaries such as whitespace, quotes, parentheses, or punctuation.
- It is not an external URL or URI-scheme reference.
- It does not resolve outside the project root.
- It is not a known command/package pattern that is not a file reference, such as Go's `./...`.

Examples:

```md
See docs/architecture/index.md for details.
The configuration lives in `.context-lint.yaml`.
docs/spec/*.md should be reachable from AGENTS.md.
```

### Reference Kinds

File-path-like strings may produce false positives, so diagnostics must include the extraction source.

- `kind: markdown-link`
- `kind: html-attr`
- `kind: path-text`
- `kind: path-code`

Missing-target diagnostics are limited to references whose file portion explicitly targets Markdown (`.md`, `.mdx`, or `.markdown`). Non-Markdown code paths, assets, path aliases, and globs are ignored when they do not exist.

## Diagnostics

### Diagnostic Codes

| Code | Meaning |
| --- | --- |
| `CL001` | Entry file does not exist |
| `CL002` | Entry file is not a Markdown file |
| `CL003` | Local Markdown reference target does not exist |
| `CL004` | `requiredReachable` target is not reachable from the entry file |
| `CL005` | Configuration format or value is invalid |
| `CL006` | Reference attempts to escape the project root |
| `CL007` | Markdown file does not have front matter |

### Human Output

The default human output should explain what an AI agent or developer should fix next.

Example:

```text
warning CL003 docs/index.md:12
  docs/index.md references docs/missing.md, but that file does not exist.
  Fix: create docs/missing.md, correct the path, or remove the reference.

warning CL004 AGENTS.md
  requiredReachable target docs/security.md is not reachable from AGENTS.md.
  Fix: add a path from AGENTS.md to docs/security.md, directly or through an index document.
```

With `--strict`:

```text
error CL003 docs/index.md:12
  docs/index.md references docs/missing.md, but that file does not exist.
  Fix: create docs/missing.md, correct the path, or remove the reference.
```

### JSON Output

`--format json` should provide a stable machine-readable shape for GitHub Actions, reviewdog, and custom CI tooling.

```json
{
  "ok": false,
  "strict": true,
  "root": "/repo",
  "entry": "AGENTS.md",
  "diagnostics": [
    {
      "code": "CL003",
      "severity": "error",
      "file": "docs/index.md",
      "line": 12,
      "column": 5,
      "message": "docs/index.md references docs/missing.md, but that file does not exist.",
      "fix": "Create docs/missing.md, correct the path, or remove the reference.",
      "reference": "docs/missing.md",
      "kind": "markdown-link"
    }
  ]
}
```

## `requiredReachable` Expansion

Each `requiredReachable` item is interpreted in this order:

1. Existing file path
2. Existing directory path
3. Glob

If the item is a file, that file is the required target.

If the item is a directory, Markdown files under that directory are recursively treated as required targets.

If the item is a glob, matched Markdown files are treated as required targets. Non-Markdown matches are ignored for reachability in the initial version.

If an item expands to zero targets, the tool should emit `CL005` because the configuration is probably wrong.

## `excludes`

Files matching `excludes` are removed from:

- Reference graph nodes
- Missing-reference checks
- `requiredReachable` expansion
- Reachability diagnostics

References to excluded files are not reported. This allows generated files, externally synchronized files, or intentionally hidden context to remain outside the AI-readable document graph.

## GitHub Actions Usage

### Warning Mode

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
      - uses: <owner>/context-lint@v1
```

### Strict Mode

```yaml
- uses: <owner>/context-lint@v1
  with:
    strict: true
```

Warning mode is recommended for initial adoption. Teams can switch to strict mode after the document graph is stable.

## Distribution

- Publish the project as a Go module.
- Put the CLI entry point under `cmd/context-lint`.
- Support installation with `go install github.com/<owner>/context-lint/cmd/context-lint@latest`.
- Prepare GitHub Releases for major OS binaries.
- Provide a root `action.yml` so workflows can run `uses: <owner>/context-lint@vX.Y.Z`.
- Build the CLI inside the composite action from the checked-out action source.
- Keep GitHub Action inputs optional when a CLI default or automatic discovery is available.

## Implementation Direction

### Package Layout

```text
action.yml               Composite GitHub Action entry point
cmd/context-lint/        CLI entry point
internal/config/         Configuration discovery, loading, and normalization
internal/document/       Markdown parsing, reference extraction, graph building
internal/glob/           requiredReachable and excludes expansion
internal/diagnostic/     Diagnostic model and human/json output
internal/runner/         Execution flow called by the CLI
```

### Recommended Libraries

- CLI: standard `flag` or a lightweight CLI library
- Markdown: CommonMark-compatible Go library
- YAML: `gopkg.in/yaml.v3`
- JSONC: comment stripping or a JSONC-aware library
- Glob: a `**`-aware library such as `doublestar`

Library choices should prioritize small dependency footprint, maintainability, and CI stability.

## Testing Strategy

### Unit Tests

- Configuration discovery order.
- YAML, JSON, and JSONC loading.
- Markdown link extraction.
- File-path-like text extraction.
- Relative paths, root-relative paths, and anchor paths.
- `requiredReachable` expansion.
- `excludes` application.
- Strict and non-strict exit codes.

### Integration Tests

Create small temporary document trees and verify CLI behavior.

Representative cases:

- All required documents are reachable and no diagnostics are emitted.
- A Markdown link points to a missing Markdown file.
- Prose contains a Markdown file path to a missing file.
- Missing non-Markdown paths are ignored.
- Only part of `requiredReachable` is reachable.
- `excludes` suppresses an otherwise unreachable file.
- `--format json` returns stable JSON.

## Future Extensions

- Markdown anchor validation.
- Stale-document detection.
- Ownership validation through `CODEOWNERS` or front matter.
- Additional rule categories beyond `requiredReachable`.
- SARIF output.
- GitHub annotations output.
- `context-lint init` for configuration generation.
- Advanced multi-entry configuration.
- Configurable file-path extraction rules and extension lists.

## Non-Functional Requirements

- Run fast enough for CI in large repositories.
- Keep output readable for both humans and AI agents.
- Include an actionable fix suggestion in each diagnostic.
- Avoid reading files outside the project root.
- Keep `go install` lightweight by limiting dependencies.
