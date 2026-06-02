# GitHub Actions で使う

`context-lint` は composite action として実行できます。

初期導入では non-strict のまま、警告として走らせるのがおすすめです。文書グラフが安定し、到達性の問題を pull request で止めたい段階になったら strict mode を有効にします。

## 基本ワークフロー

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

この設定では、ワークフローの作業ディレクトリをプロジェクトルートとして扱い、`.context-lint.{yaml,yml,json,jsonc}` を自動検出します。出力は human format で、ドキュメント上の検出結果は警告として扱われます。

## Strict Mode

文書グラフを必ず守りたい段階になったら `strict: true` を指定します。

```yaml
- uses: aki-0421/context-lint@v1
  with:
    strict: true
```

## オプション

必要な値だけを指定します。

```yaml
- uses: aki-0421/context-lint@v1
  with:
    config: .context-lint.yaml
    root: .
    format: human
    strict: true
```

`config` は設定ファイルのパス、`root` は検査対象のプロジェクトルート、`format` は `human` または `json`、`strict` は警告をエラーとして扱うかどうかを指定します。
