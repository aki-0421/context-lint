# context-lint

[English](README.en.md)

`context-lint` は、AI エージェントが参照するリポジトリ文書を「1つの入口からたどれる、小さく信頼できる文書グラフ」として検査するリンターです。

`AGENTS.md` や `CLAUDE.md` のようなAIエージェントが常に参照するブートストラップコンテキストからパス参照をたどり、重要な文書が到達可能か、Markdown の参照先が存在するか、AI と人間が維持しやすい大きさに分割されているかを確認します。

## 基本思想

リポジトリの文書は一時的なメモではありません。
適切に設計され、保守され、CI で壊れていないことを確認されるべき作業基盤です。

また、AI エージェントに渡すコンテキストは、多ければよいわけではありません。
よいコンテキストは、短く、明示的で、信頼でき、必要な深さまでリンクでたどれます。

AIが参照できないドキュメントは、AI との協働において存在していないに等しいものです。
AI自身が書いたドキュメントであっても、次のセッションや別のエージェントが入口ファイルから再び参照できなければ、リポジトリの知識としては残りません。

- エージェントには、長大な説明を丸ごと読ませるのではなく、必要な文書へ段階的に案内します。
- 重要な設計判断、運用ルール、仕様、コマンドが入口ファイルからたどれない状態は、文書の負債です。
- AI が読む文書にも、人間が読む文書と同じように構造、責任範囲、継続的な検証が必要です。

`context-lint` は、この構造が保たれているかを機械的に確認します。

## 何を検査するか

- プロジェクトルートに `AGENTS.md` や `CLAUDE.md` が存在するかを確認する。
- Markdownの文中やコードブロック内のファイルパスらしい文字列を参照として抽出する。
- 存在しないローカル Markdown 参照を報告する。
- `requiredReachable` に指定したMarkdownファイル群が入口から到達可能か確認する。
- Markdown が大きすぎる場合、分割を促す。
- 通常は警告として導入し、文書グラフがチームに根づいたら `--strict` で CI を失敗させる。

## セットアップ

### Agent Skill で始める

AI エージェントが Agent Skill を使える環境では、この方法が推奨です。`context-lint` の思想そのものが、エージェントと一緒にリポジトリの文脈を整えることに向いています。

```bash
npx skills add aki-0421/context-lint --list
npx skills add aki-0421/context-lint --skill context-lint-setup
npx skills add aki-0421/context-lint --skill context-router
```

セットアップしたいリポジトリを開き、エージェントに次のように依頼します。

```text
Use $context-lint-setup to install context-lint and add a default config to this repository.
Use $context-router while creating or reorganizing managed documentation.
```

`context-lint-setup` は CLI の導入、入口ファイルの選択、最小構成の作成、初回 lint を案内します。`context-router` は、管理対象の Markdown を作成・移動・分割するときに front matter と `index.md` の経路を保つためのスキルです。

### 手動で始める

Go 1.23 以上を用意して CLI をインストールします。

```bash
go install github.com/aki-0421/context-lint/cmd/context-lint@latest
```

プロジェクトルートに `.context-lint.yaml` を作成します。

```yaml
linter:
  document:
    entry: AGENTS.md
    requiredReachable:
      - docs/**/*.md
```

`entry` には、AI エージェントが最初に読む短い Markdown ファイルを指定します。`requiredReachable` には、入口から到達可能であるべき文書、ディレクトリ、glob を指定します。まだ必須文書がない場合は `requiredReachable` を省略できます。

実行します。

```bash
context-lint
```

最初は警告として運用し、入口ファイルと必須文書の関係が安定してから strict mode に切り替えるのがおすすめです。

```bash
context-lint --strict
context-lint --format json
```

詳しい手動セットアップと CLI リファレンスは [docs/manual-setup.md](docs/manual-setup.md) を参照してください。GitHub Actions での導入は [docs/github-actions.md](docs/github-actions.md) に分けています。

## 次に読むもの

- [手動セットアップ](docs/manual-setup.md)
- [GitHub Actions で使う](docs/github-actions.md)
- [仕様](docs/spec/context-lint.md)
- [コントリビューター向けガイド](CONTRIBUTING.md)

## ライセンス

MIT
