# context-lint

[日本語](README.md)

`context-lint` is a linter and GitHub Action that checks whether AI-readable repository documentation forms a small, trusted graph reachable from one entry file.

Starting from bootstrap context that an AI agent always reads, such as `AGENTS.md` or `CLAUDE.md`, it follows path references and checks whether important documents are reachable, Markdown references exist, and documents are split into sizes that AI agents and humans can maintain.

## Philosophy

Repository documentation is not a temporary note. It is working infrastructure that should be designed, maintained, and checked in CI.

More context is not automatically better context. Good context is short, explicit, trusted, and linked deeply enough for the task at hand.

Documentation an AI cannot reach is effectively nonexistent for AI-assisted work. Even when an AI agent writes a document itself, that knowledge does not remain as repository knowledge unless the next session or another agent can reach it again from the entry file.

- Agents should be guided progressively to the documents they need instead of being handed one large explanation.
- Important design decisions, operating rules, specs, and commands that cannot be reached from the entry file are documentation debt.
- AI-readable documentation needs structure, ownership boundaries, and continuous verification just like human-readable documentation.

`context-lint` mechanically checks whether this structure is still intact.

## What It Checks

- Verify that an entry file such as `AGENTS.md` or `CLAUDE.md` exists at the project root.
- Extract file-path-like strings in Markdown prose and code blocks as references.
- Report local Markdown references that point to missing files.
- Check whether Markdown files configured in `requiredReachable` are reachable from the entry file.
- Recommend splitting Markdown files that are too large.
- Start in warning mode, then fail CI with `--strict` once the document graph has become part of the workflow.

## Setup

### Start With Agent Skills

If your AI agent can use Agent Skills, this is the recommended path. The philosophy of `context-lint` fits naturally with having agents help maintain the repository context they depend on.

```bash
npx skills add aki-0421/context-lint --list
npx skills add aki-0421/context-lint --skill context-lint-setup
npx skills add aki-0421/context-lint --skill context-router
```

Open the repository you want to configure and ask your agent:

```text
Use $context-lint-setup to install context-lint and add a default config to this repository.
Use $context-router while creating or reorganizing managed documentation.
```

`context-lint-setup` guides CLI installation, entry-file selection, minimal configuration, and the first lint run. `context-router` helps maintain front matter and `index.md` routes while creating, moving, or splitting managed Markdown documents.

### Manual Setup

Install the CLI with Go 1.23 or newer.

```bash
go install github.com/aki-0421/context-lint/cmd/context-lint@latest
```

Create `.context-lint.yaml` at the project root.

```yaml
linter:
  document:
    entry: AGENTS.md
    requiredReachable:
      - docs/**/*.md
```

Use `entry` for the short Markdown file an AI agent should read first. Use `requiredReachable` for documents, directories, or globs that must be reachable from that entry. If the repository does not have required documentation yet, omit `requiredReachable`.

Run the linter.

```bash
context-lint
```

Start with warnings, then switch to strict mode after the relationship between the entry file and required documents is stable.

```bash
context-lint --strict
context-lint --format json
```

For the full manual setup and CLI reference, see [docs/manual-setup.md](docs/manual-setup.md). GitHub Actions setup is documented separately in [docs/github-actions.en.md](docs/github-actions.en.md).

## Next Reads

- [Manual setup](docs/manual-setup.md)
- [GitHub Actions setup](docs/github-actions.en.md)
- [Specification](docs/spec/context-lint.md)
- [Contributing guide](CONTRIBUTING.md)

## License

MIT
