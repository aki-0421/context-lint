# context-lint Validation And Parsing

## Validation Model

### Flow

1. Resolve the project root.
2. Load the configuration file.
3. Validate that `entry` exists and is a Markdown file.
4. Expand `requiredReachable` and `excludes`.
5. Build a reference graph starting from `entry`.
6. Check whether referenced local Markdown files exist.
7. Check whether all required targets are reachable.
8. Check whether all required Markdown targets are within the configured file-size limit.
9. Print diagnostics and decide the exit code.

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

- Inline links: `[text](<markdown-file>)`
- Image links: `![alt](docs/image.png)`
- Reference links: `[text][foo]` and `[foo]: <markdown-file>`
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
See <nested-document-path> for details.
The configuration lives in `.context-lint.yaml`.
<documentation-glob> should be reachable from AGENTS.md.
```

### Reference Kinds

File-path-like strings may produce false positives, so diagnostics must include the extraction source.

- `kind: markdown-link`
- `kind: html-attr`
- `kind: path-text`
- `kind: path-code`

Missing-target diagnostics are limited to references whose file portion explicitly targets Markdown (`.md`, `.mdx`, or `.markdown`). Non-Markdown code paths, assets, path aliases, and globs are ignored when they do not exist.
