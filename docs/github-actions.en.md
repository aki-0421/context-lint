# GitHub Actions Setup

`context-lint` can run as a composite action.

For initial adoption, run it in non-strict warning mode. Enable strict mode after the document graph is stable and reachability problems should block pull requests.

## Basic Workflow

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

This uses the workflow working directory as the project root, discovers `.context-lint.{yaml,yml,json,jsonc}` automatically, prints human output, and reports documentation findings as warnings.

## Strict Mode

Enable `strict: true` when the documentation graph should be enforced.

```yaml
- uses: aki-0421/context-lint@v1
  with:
    strict: true
```

## Options

Pass only the values you need to override.

```yaml
- uses: aki-0421/context-lint@v1
  with:
    config: .context-lint.yaml
    root: .
    format: human
    strict: true
```

`config` is the configuration file path, `root` is the project root to lint, `format` is `human` or `json`, and `strict` treats warnings as errors.
