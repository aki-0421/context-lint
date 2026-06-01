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
- required documentation stays small enough to read and split into focused files;
- CI can enforce the documentation graph when a project is ready.

## Status

This project is early-stage. The core CLI, GitHub Action entry point, tests, CI, and release workflow are in place. Feedback and small, focused contributions are welcome.

## Agent-First Setup

This repository distributes installable Agent Skills for setting up and using `context-lint`:

- [`context-lint-setup`](skills/context-lint-setup/SKILL.md) helps AI agents install `context-lint`, add a default repository config, optionally add GitHub Actions, and customize settings.
- [`context-router`](skills/context-router/SKILL.md) helps AI agents maintain managed documentation front matter and `index.md` routes while creating, changing, or reorganizing docs.

Install the skills you need, then ask your agent to use them in the repository you have open:

```bash
npx skills add aki-0421/context-lint --list
npx skills add aki-0421/context-lint --skill context-lint-setup
npx skills add aki-0421/context-lint --skill context-router
```

Example prompt:

```text
Use $context-lint-setup to install context-lint and add a default config to this repository.
Use $context-router while creating or reorganizing managed documentation.
```

`context-lint-setup` is designed to inspect the repository first, choose or create the appropriate agent entry file such as `AGENTS.md` or `CLAUDE.md`, install the CLI safely, add a minimal default config, and ask before adding GitHub Actions. `context-router` is designed for ongoing documentation edits after a repository already has managed documentation.

For environments where an Agent Skill cannot be used, see [Manual Setup](docs/manual-setup.md).

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
