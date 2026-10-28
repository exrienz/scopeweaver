[English](../../../../../skills/playwright-cli/references/tracing.md) | [한국어](../../../../ko/skills/playwright-cli/references/tracing.md) | [简体中文](../../../../zh-Hans/skills/playwright-cli/references/tracing.md) | [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/tracing.md) | [日本語](tracing.md) | [Español](../../../../es/skills/playwright-cli/references/tracing.md)

<a id="tracing"></a>
# トレース



デバッグや分析用に、詳細な実行トレースを記録します。DOM スナップショット、スクリーンショット、ネットワークの動作、コンソールログを含みます。

<a id="basic-usage"></a>
## 基本的な使い方

```bash
# Start trace recording
playwright-cli tracing-start

# Perform actions
playwright-cli open https://example.com
playwright-cli click e1
playwright-cli fill e2 "test"

# Stop trace recording
playwright-cli tracing-stop
```

<a id="trace-output-files"></a>
## トレースの出力ファイル

記録を開始すると、Playwright は複数のファイルを持つ `traces/` ディレクトリを作成します:

<a id="trace-timestamptrace"></a>
### `trace-{timestamp}.trace`

**操作ログ** — メインのトレースファイルで、次を含みます:
- すべての操作（クリック、入力、ページ移動）
- 各操作前後の DOM スナップショット
- 各ステップのスクリーンショット
- 時間の情報
- コンソールのメッセージ
- ソースの位置

<a id="trace-timestampnetwork"></a>
### `trace-{timestamp}.network`

**ネットワークログ** — 完全なネットワークの動作:
- すべての HTTP リクエストとレスポンス
- リクエストのヘッダーと本文
- レスポンスのヘッダーと本文
- 時間（DNS、接続、TLS、TTFB、ダウンロード）
- リソースのサイズ
- 失敗したリクエストとエラー

<a id="resources"></a>
### `resources/`

**リソースディレクトリ** — キャッシュしたリソース:
- 画像、フォント、スタイルシート、スクリプト
- 再生用のレスポンス本文
- ページの状態の再構成に必要な資産

<a id="what-traces-capture"></a>
## トレースが記録する内容

| 分類 | 詳細 |
|----------|---------|
| **操作** | クリック、入力、ホバー、キーボード入力、ページ移動 |
| **DOM** | 各操作前後の完全な DOM スナップショット |
| **スクリーンショット** | 各ステップの表示状態 |
| **ネットワーク** | すべてのリクエスト、レスポンス、ヘッダー、本文、時間 |
| **コンソール** | すべての console.log、warn、error メッセージ |
| **時間** | 各操作の正確な時間 |

<a id="use-cases"></a>
## 使用例

<a id="debugging-failed-actions"></a>
### 失敗した操作のデバッグ

```bash
playwright-cli tracing-start
playwright-cli open https://app.example.com

# This click fails - why?
playwright-cli click e5

playwright-cli tracing-stop
# Open trace to see DOM state when click was attempted
```

<a id="analyzing-performance"></a>
### パフォーマンスの分析

```bash
playwright-cli tracing-start
playwright-cli open https://slow-site.com
playwright-cli tracing-stop

# View network waterfall to identify slow resources
```

<a id="capturing-evidence"></a>
### 証拠の記録

```bash
# Record a complete user flow for documentation
playwright-cli tracing-start

playwright-cli open https://app.example.com/checkout
playwright-cli fill e1 "4111111111111111"
playwright-cli fill e2 "12/25"
playwright-cli fill e3 "123"
playwright-cli click e4

playwright-cli tracing-stop
# Trace shows exact sequence of events
```

<a id="trace-vs-video-vs-screenshot"></a>
## トレース・動画・スクリーンショットの比較

| 機能 | トレース | 動画 | スクリーンショット |
|---------|-------|-------|------------|
| **形式** | .trace ファイル | .webm 動画 | .png/.jpeg 画像 |
| **DOM の確認** | 可 | 不可 | 不可 |
| **ネットワークの詳細** | 可 | 不可 | 不可 |
| **ステップごとの再生** | 可 | 連続 | 1 フレーム |
| **ファイルサイズ** | 中 | 大 | 小 |
| **適した用途** | デバッグ | デモ | 素早い記録 |

<a id="best-practices"></a>
## 推奨する使い方

<a id="1-start-tracing-before-the-problem"></a>
### 1. 問題の発生前にトレースを開始する

```bash
# Trace the entire flow, not just the failing step
playwright-cli tracing-start
playwright-cli open https://example.com
# ... all steps leading to the issue ...
playwright-cli tracing-stop
```

<a id="2-clean-up-old-traces"></a>
### 2. 古いトレースを削除する

トレースは大量のディスク領域を使う場合があります:

```bash
# Remove traces older than 7 days
find .playwright-cli/traces -mtime +7 -delete
```

<a id="limitations"></a>
## 制限

- トレースは自動操作の負荷を増やします。
- 大きなトレースは大量のディスク領域を使う場合があります。
- 動的な内容の一部は、完全には再生できない場合があります。
