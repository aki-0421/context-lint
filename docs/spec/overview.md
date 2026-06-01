# context-lint Overview

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
- Detect configured `requiredReachable` Markdown files that exceed the readable file-size limit.
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
- `requiredReachableMaxFileSize`: Optional file-size limit for Markdown files expanded from `requiredReachable`.
- `excludes`: Files, directories, or globs excluded from linting.
- `frontMatter.excludeFileNames`: Markdown file names excluded from missing-front-matter diagnostics.
- Diagnostic: A warning or error emitted by the linter.
