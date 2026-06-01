# context-lint Configuration And CLI

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
        - <front-matter-excluded-file>
```

### Schema

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

### Fields

`linter.document.entry` is required. It must be a project-root-relative path to a single Markdown file.

`linter.document.requiredReachable` is optional. Each item may be a file, directory, or glob. When a directory is specified, Markdown files under that directory are treated as required targets.

`linter.document.requiredReachableMaxFileSize` is optional and can usually be omitted. When omitted, the default limit is `32 KiB`. The value may be a byte count or a human-readable binary size such as `64 KiB` or `1 MiB`. Required Markdown targets at or above the limit produce a diagnostic that recommends splitting the file into smaller linked documents for both AI agents and humans.

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
      "file": "<front-matter-file>",
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
