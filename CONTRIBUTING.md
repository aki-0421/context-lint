# Contributing

Thanks for helping improve `context-lint`. Contributions are welcome, especially focused fixes, tests, documentation improvements, and small rule improvements.

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

## Guidelines

- Keep changes small and reviewable.
- Open an issue first for large behavior changes or new rule categories.
- Add or update tests for user-visible behavior.
- Keep specifications and developer-facing docs in English unless a document is intentionally localized.
- Keep `README.md` and `README.en.md` aligned when changing user-facing setup or product positioning.
- Run `go test ./...` before opening a pull request.
- Do not commit generated release artifacts from `dist`.
- Be respectful and assume good intent in issues, reviews, and discussions.

By submitting a contribution, you agree that your contribution will be licensed under the MIT License.

## Security

Please do not report security issues in public issues. Use GitHub private vulnerability reporting if it is available for this repository, or contact the maintainer privately through GitHub.
