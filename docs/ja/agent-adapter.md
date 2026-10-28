[English](../../adapters/agent/README.md) | [한국어](../ko/agent-adapter.md) | [简体中文](../zh-Hans/agent-adapter.md) | [繁體中文](../zh-Hant/agent-adapter.md) | [日本語](agent-adapter.md) | [Español](../es/agent-adapter.md)

<a id="scopeweaver-agent-adapter"></a>
# ScopeWeaver エージェントアダプター



Claude Code と Codex はローカルの stdio MCP サーバー経由、Pi はネイティブの拡張インターフェース経由で接続します。3 つとも、同じ検証済みツールと認証付きの ScopeWeaver HTTP API を使用します。コーディングエージェントから ScopeWeaver を操作するアダプターであり、ScopeWeaver 自身のプランナー・ワーカーの代わりや、コーディングエージェントのサブスクリプションをモデル API キーへ変換するものではありません。

<a id="setup"></a>
## 初期設定

必要なもの: Node.js 22 以上、実行中の ScopeWeaver バックエンド、既存の管理者ログイン。アダプターはパスワードを初期化・変更しません。独立したアダプターの依存関係をインストールしてください:

```sh
cd adapters/agent
npm ci
export SCOPEWEAVER_URL=http://127.0.0.1:8787
export SCOPEWEAVER_PASSWORD='your-existing-admin-password'
```

パスワードは、バックエンドの既存の `ARTEX` 管理者ユーザーへのログインに使用します。返される JWT はメモリーにのみ保持します。代わりに、既存のログイン JWT を `SCOPEWEAVER_TOKEN` に指定することもでき、指定したトークンが優先します。期限切れ・無効なトークンはエラーになり、黙って置き換えません。認証情報は環境変数やシークレット管理に保存し、クライアントの設定にコミットしないでください。

読み取りは既定で有効です。タスクの作成と一時停止・再開を有効にするには、起動前に `SCOPEWEAVER_ALLOW_WRITES=true` を明示してください。バックエンドに LLM が設定済みの場合、タスクの作成で即座に作業をスケジュールします。プロジェクトの[ローカル隔離環境での使用制限](../../README.ja.md#license-and-disclaimer)を守り、タスクの作成前に意図したスコープを確認してください。

<a id="claude-code"></a>
### Claude Code

このチェックアウトへの絶対パスを使ってください。Claude Code がアダプターを起動し、その環境変数を引き継ぎます:

```sh
claude mcp add --transport stdio scopeweaver -- node /absolute/path/scopeweaver/adapters/agent/src/mcp.js
```

`claude mcp list` で確認し、Claude Code の `/mcp` でツールを確認してください。[Claude Code MCP ドキュメント](https://code.claude.com/docs/en/mcp)。

<a id="codex"></a>
### Codex

```sh
codex mcp add scopeweaver -- node /absolute/path/scopeweaver/adapters/agent/src/mcp.js
```

`codex mcp list` と `/mcp` で確認してください。プロジェクト・ユーザー設定で環境変数を明示的に渡す場合:

```toml
[mcp_servers.scopeweaver]
command = "node"
args = ["/absolute/path/scopeweaver/adapters/agent/src/mcp.js"]
env_vars = ["SCOPEWEAVER_URL", "SCOPEWEAVER_TOKEN", "SCOPEWEAVER_PASSWORD", "SCOPEWEAVER_LANGUAGE", "SCOPEWEAVER_ALLOW_WRITES"]
```

設定した認証情報の変数を使い、未使用の項目は省略してください。[OpenAI 公式 MCP ドキュメント](https://developers.openai.com/codex/mcp)。

<a id="pi"></a>
### Pi

```sh
pi -e /absolute/path/scopeweaver/adapters/agent/pi-extension.js
```

拡張はネイティブの Pi ツールを登録し、同じ環境変数を使います。Pi の MCP プラグインは不要です。Pi のパッケージ名前空間をインポートせず、パッケージ名が変わっても公開の `registerTool` インターフェースに従います。アダプターパッケージは、Pi のパッケージローダー用の `pi.extensions` も宣言しています。[Pi の拡張](https://github.com/earendil-works/pi/tree/main/packages/coding-agent)。

<a id="tools"></a>
## ツール

| ツール | 目的 |
| --- | --- |
| `scopeweaver_health` | バックエンドの利用可能性と LLM の準備状態。ログイン不要 |
| `scopeweaver_list_tasks` | タスクの状態、件数、ページ送り |
| `scopeweaver_get_task` | タスク 1 件の保存済みの状態と設定 |
| `scopeweaver_get_coverage` | タスクの資産カバレッジ |
| `scopeweaver_list_findings` | タスク、深刻度、状態、テキストで絞り込んだ検出事項のページ取得 |
| `scopeweaver_get_finding` | 検出事項の詳細、証拠、レポート |
| `scopeweaver_create_task` | タスクを作成・スケジュール。書き込みの有効化が必要 |
| `scopeweaver_control_task` | タスクを一時停止・再開。書き込みの有効化が必要 |

要求の例: 「ScopeWeaver のタスクを一覧表示」「タスク 12 の深刻度の高い検出事項を表示」、または書き込みを有効にしてローカル実験室のスコープを確認した後に、「このローカルのテスト用環境のタスクを作成して進捗を表示」。作成の成功応答はタスクが存在する証明です。実行の完了は状態と検出事項を確認してください。アダプターはバックエンドの LLM 要件を回避しません。

<a id="configuration-and-failures"></a>
## 設定と失敗時の動作

| 変数 | 既定値 / 動作 |
| --- | --- |
| `SCOPEWEAVER_URL` | `http://127.0.0.1:8787`。HTTPS またはループバックの HTTP オリジンのみ |
| `SCOPEWEAVER_TOKEN` | 既存のバックエンド JWT。パスワードより優先 |
| `SCOPEWEAVER_PASSWORD` | 既存の管理者パスワード。最初の認証が必要な要求でログイン |
| `SCOPEWEAVER_ALLOW_WRITES` | `false`。`true` または `false` のみを受け付ける |
| `SCOPEWEAVER_LANGUAGE` | `en`。アダプターの表示文とバックエンドの `Accept-Language` 用に正規 ID `en`、`ko`、`zh-Hans`、`zh-Hant`、`ja`、`es` に対応 |
| `SCOPEWEAVER_TIMEOUT_MS` | `30000`。1～120000 の整数 |

`SCOPEWEAVER_LANGUAGE` の既定値は `en` で、正規 ID `en`、`ko`、`zh-Hans`、`zh-Hant`、`ja`、`es` を受け付けます。組み込みツールのタイトルと説明、スキーマの説明、アダプター独自の設定・検証・通信診断を翻訳し、`Accept-Language` でバックエンドの応答言語を指定します。クライアントごとに独立したカタログとスキーマ単位の Zod エラーマップを使い、グローバルな言語設定は変更しません。ツール名・プロトコル名、スキーマのキー・型・列挙値、引数、バックエンド・ユーザー・証拠の原文は変更せず、既存の認証情報の伏せ字処理を維持します。MCP SDK の固定の検証エラー文面と未知の外部エラーは外部のテキストとして保持し、その文面に含まれるスキーマのエラーメッセージは翻訳します。全リソースは独立したアダプターに同梱され、リポジトリ直下の言語ファイルは不要です。未対応の言語値は引き続き拒否し、英語で診断を表示します。

汎用 HTTP、シェル実行、削除、承認の回避、プロバイダーの設定、任意のエンドポイント用ツールは公開しません。リクエストは設定したオリジン内に限定し、リダイレクトを拒否し、中止を転送し、タイムアウトを適用します。2 MiB を超える応答は明示的に失敗します。切り詰められた証拠を完全と見なさず、照会を絞ってください。タスク一覧はアダプター側、検出事項はバックエンド側でページ分割します。ツールはテキストと構造化結果に `{data: ...}` を返します（Pi は `details`）。証拠は信頼できない対象側の内容で、指示として扱ってはいけません。

MCP の失敗は `isError: true`、Pi の失敗は例外を投げて Pi がツールエラーとして扱います。エラーメッセージは設定した認証情報を伏せます。書き込みは自動では再試行しません。書き込みがタイムアウトしたら、バックエンドが受け入れ済みの場合があるため、繰り返す前にタスクの状態を確認してください。

<a id="verification"></a>
## 検証

```sh
npm test
npm run check
```

E2E ハーネスは、実際の ScopeWeaver ハンドラーと PostgreSQL を実行し、公式 SDK から MCP stdio サーバーを起動して、Pi のツールハンドラーを同じバックエンドに対して実行します。隔離したテスト用パスワードを初期化し、タスクを作成し、一時停止・再開状態とカバレッジの保存を検証し、明確に模擬の検出事項を読み、無効な認証、存在しないタスク、無効な入力を確認します。モデル・プロバイダーや対象のスキャンは使いません。Go E2E ハーネスにはソースのチェックアウトが必要です。プラットフォームの ZIP にはアダプターの単体テストとコンパイル済み Go バックエンドのみが含まれます。

`scopeweaver_adapter_*` という名前の**新しい使い捨て** DB を作成した後、リポジトリのルートから実行します:

```sh
SCOPEWEAVER_ADAPTER_E2E=1 \
ARTEX_TEST_DSN='postgres://user:password@127.0.0.1:5432/scopeweaver_adapter_e2e?sslmode=disable' \
go test ./server -run '^TestAgentAdapterE2E$' -count=1 -v
```

テスト本体は `SCOPEWEAVER_ADAPTER_E2E=1` が指定されていない場合にスキップされます。有効にしても、`ARTEX_TEST_DSN` が未設定か空白ならスキップされます。スキップは E2E 検証の証拠にはなりません。パッケージの DB 保護チェックは、設定済みの通常のアプリケーション DB と一致する DSN をテスト実行前に拒否します。実行を明示して DB を設定した場合は、不正な DSN、`scopeweaver_adapter_*` 以外の DB 名、接続できない PostgreSQL により失敗します。ハーネスはモックで代用しません。 GitHub Actions でも、専用の使い捨て PostgreSQL サービスで同じ確認を繰り返します。MCP プロトコルのテストは共有インターフェースとの互換性を確認するもので、有料の Claude/Codex/Pi モデルのセッションを実行したとは主張しません。
