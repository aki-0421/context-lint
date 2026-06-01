---
name: context-lint-setup
description: Use this skill when a user wants to install context-lint, add a default context-lint config to a repository, optionally add GitHub Actions, or customize context-lint settings.
---

# Context Lint Setup

Use this skill to help a user install `context-lint` and set up a repository with a safe default configuration. Work interactively: inspect first, explain the command or file edit you plan to make, and get user approval before installing software, downloading release artifacts, editing shell startup files, adding CI, or writing repository files.

## Installation Support

Start by checking whether the command already exists:

```bash
command -v context-lint
context-lint --version
```

If `context-lint` is present, use the installed command. Offer to update it only when the user asks for installation, update, or repair.

If the command is missing, ask before installing. Prefer Go when available:

```bash
go install github.com/aki-0421/context-lint/cmd/context-lint@latest
```

After `go install`, verify `context-lint --version`. If the binary directory is not on `PATH`, report the likely directory from `go env GOBIN` or `go env GOPATH` and ask before editing any shell profile. Prefer giving a one-session `PATH` command over changing persistent shell configuration.

If Go is unavailable, ask before downloading from GitHub Releases. Use a temporary directory, detect OS and architecture, choose the latest asset matching `context-lint_<version>_<os>_<arch>.tar.gz` or the Windows `.zip`, download `checksums.txt` when available, verify the SHA256 checksum, and install the binary into a user-writable directory on `PATH` such as `~/.local/bin` on Unix-like systems or `%USERPROFILE%\bin` on Windows. Avoid `sudo` unless the user explicitly requests a system-wide install.

Never pipe remote scripts into a shell. Keep downloaded archives in a temp directory and remove temporary files after verification and install.

## Repository Setup Support

Use the user's current working directory as the target repository unless they provide another path. Before writing anything, inspect for existing config files:

```bash
for f in .context-lint.yaml .context-lint.yml .context-lint.json .context-lint.jsonc; do
  [ -f "$f" ] && printf '%s\n' "$f"
done
```

If a config already exists, read it and ask whether to keep it, minimally adjust it, or replace it. Do not overwrite existing configuration without explicit approval.

For a first setup, create a small default `.context-lint.yaml` and make the entry file an agent-facing Markdown instruction file, not a general project README. Select the entry file that is appropriate for the agent being used for setup:

- Use `CLAUDE.md` for Claude Code or Claude-centered workflows.
- Use `AGENTS.md` for Codex, OpenAI-style agents, or agents that follow the AGENTS.md convention.
- For another agent, use that agent's documented repository instruction file when it is Markdown. If the convention is unclear, ask the user which agent entry file to create.

If the appropriate entry file does not exist, ask before creating it. Keep the new file short: repository purpose, where important docs live, and basic setup or verification commands. Include only links and paths that already exist.

Default config template:

```yaml
linter:
  document:
    entry: <agent-entry-markdown-file>
    requiredReachable:
      - <required-docs-path>
    excludes:
      - <excluded-path>
```

Replace every placeholder with an actual path that fits the repository. If there is no required documentation path yet, omit `requiredReachable` at first. If there is no excluded path, omit `excludes`. Keep initial setup intentionally plain; do not add file-size limits, front matter exclusions, globs, or strict CI settings unless the user asks.

When the selected entry file is `CLAUDE.md` or another agent-specific Markdown file, use that path in `linter.document.entry` instead.

After writing the config, run:

```bash
context-lint
```

Summarize any findings in terms of the next safe documentation action. Ask before editing `AGENTS.md`, `CLAUDE.md`, another agent entry file, `README.md`, `docs/`, or other docs to fix reachability.

GitHub Actions setup is optional. Ask the user whether to add it, and make the default recommendation to skip CI until the local default setup is working. If the user chooses CI, create `.github/workflows/context-lint.yml` with a minimal non-strict workflow:

```yaml
name: context-lint

on:
  pull_request:
  push:
    branches:
      - main

permissions:
  contents: read

jobs:
  context-lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: aki-0421/context-lint@v1
```

Only add `strict: true` when the user explicitly wants documentation findings to fail CI.

## Configuration Customization Support

When the user asks to customize settings, translate their goal into the smallest config change:

- Use `entry` for the one Markdown file agents should read first.
- Use `requiredReachable` for documents or directories that must be reachable from the entry file.
- Use `excludes` for generated, vendored, archived, or intentionally ignored paths.
- Use `requiredReachableMaxFileSize` only when the user wants a different readability threshold.
- Use `frontMatter.excludeFileNames` for `context-lint list` warnings on route, generated, or conventional Markdown files that should not need front matter.

Customization workflow:

1. Read the existing config and inspect the relevant documentation tree.
2. Ask focused questions only for choices that cannot be inferred safely.
3. Edit the config minimally.
4. Run `context-lint` or `context-lint --format json` to verify behavior.
5. Present the resulting diagnostics, changed files, and any remaining choices.

For stricter enforcement, use `context-lint --strict` locally first. Add strict CI only after the user confirms the repository is ready for documentation findings to block pull requests.

## Troubleshooting

- `go install` succeeds but `context-lint` is not found: check `go env GOBIN` and `go env GOPATH`; the binary is usually in `$GOBIN` or `$(go env GOPATH)/bin`.
- No matching release artifact is found: list the latest release assets and report the OS/architecture you detected before trying another install method.
- The first lint run reports unreachable docs: add links from the entry file or an index document only after user approval.
- Existing CI already runs documentation checks: update the existing workflow instead of adding a duplicate workflow.
