# context-lint Actions And Distribution

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
