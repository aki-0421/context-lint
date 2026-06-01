# context-lint Implementation And Testing

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
- Required target file-size diagnostics.
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
