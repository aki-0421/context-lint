# context-lint Diagnostics

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
| `CL008` | `requiredReachable` target exceeds the file-size limit |

### Human Output

The default human output should explain what an AI agent or developer should fix next.

Example:

```text
warning CL003 <source-file>:12
  <source-file> references <missing-file>, but that file does not exist.
  Fix: create <missing-file>, correct the path, or remove the reference.

warning CL004 AGENTS.md
  requiredReachable target <required-file> is not reachable from AGENTS.md.
  Fix: add a path from AGENTS.md to <required-file>, directly or through an index document.

warning CL008 <large-file>
  requiredReachable target <large-file> is 33 KiB, at or above the 32 KiB limit.
  Fix: split <large-file> into smaller focused Markdown files, then link them from AGENTS.md or an index document so agents and humans can read the context progressively.
```

With `--strict`:

```text
error CL003 <source-file>:12
  <source-file> references <missing-file>, but that file does not exist.
  Fix: create <missing-file>, correct the path, or remove the reference.
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
      "file": "<source-file>",
      "line": 12,
      "column": 5,
      "message": "<source-file> references <missing-file>, but that file does not exist.",
      "fix": "Create <missing-file>, correct the path, or remove the reference.",
      "reference": "<missing-file>",
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

Each expanded Markdown target is also checked against `linter.document.requiredReachableMaxFileSize`. The default is `32 KiB` even when the field is absent from the configuration. Projects can raise or lower the limit by adding the optional field:

```yaml
linter:
  document:
    entry: AGENTS.md
    requiredReachable:
      - docs
    requiredReachableMaxFileSize: 64 KiB
```

## `excludes`

Files matching `excludes` are removed from:

- Reference graph nodes
- Missing-reference checks
- `requiredReachable` expansion
- Reachability diagnostics

References to excluded files are not reported. This allows generated files, externally synchronized files, or intentionally hidden context to remain outside the AI-readable document graph.
