[English](../llm-providers.md) | [한국어](../ko/llm-providers.md) | [简体中文](../zh-Hans/llm-providers.md) | [繁體中文](../zh-Hant/llm-providers.md) | [日本語](llm-providers.md) | [Español](../es/llm-providers.md)

<a id="llm-provider-templates"></a>
# LLM プロバイダーのテンプレート



ScopeWeaver は **Anthropic**、**OpenAI Chat Completions**、**OpenAI Responses**、ネイティブ **Ollama** と任意の **OpenCode SDK ブリッジ**に対応します。元の 4 形式は独自ベース URL、プロキシ、セッションヘッダー、再試行、レート、同時実行設定を保持し、クラウド、ゲートウェイ、セルフホスト OpenAI 互換サーバーの動作は変わりません。SDK は独立した認証情報と下記の制限を使い、API／OpenCode Go を置き換えません。

SDK テンプレートも要求を送信しません。モデル探索には明示的な**モデル読み込み**操作が必要です。

テンプレートは、認証情報を保存したり、プロファイルを有効化したり、既存のものを変更したりせず、**新規プロファイルのフォーム**を設定します。Ollama のテンプレートは選択したサーバーのインストール済みモデルも読みますが、クラウドのテンプレートは要求を送りません。適用後も形式、ベース URL、モデルを編集できます。

<a id="zai-glm-53"></a>
## Z.ai GLM-5.3

![GLM-5.3 の新規プロファイルのテンプレート](../../screenshots/en/llm-templates.png)

API キー未入力のテンプレートのプレビューです。実際のプロバイダー接続ではありません。

**LLM → 新規 → 設定テンプレート**を開きます。API 課金には **General API（推奨）**、**Coding Plan（参考）**は ScopeWeaver に対する Z.ai の許可を得た後にのみ選択してください。

| フィールド | General API | Coding Plan の参考設定 |
| --- | --- | --- |
| 形式 | OpenAI Chat Completions | OpenAI Chat Completions |
| ベース URL | `https://api.z.ai/api/paas/v4` | `https://api.z.ai/api/coding/paas/v4` |
| モデル | `glm-5.3` | `glm-5.3` |
| 思考機能 | `enabled` | `enabled` |
| 推論強度 | `low` | `low` |
| コンテキストウィンドウ（K）| `0`（既定 200K）| `0`（既定 200K）|
| 出力上限 | `16384` | `16384` |

エンドポイントと課金は別です。General API の使用量は Coding Plan のサブスクリプション枠とは分離されています。[Z.ai の接続設定](https://zcode.z.ai/en/docs/configuration)を参照し、テスト前にアカウントのモデル利用権限と課金を確認してください。

GLM-5.3 はテキストを受け付け、コンテキストは 1M トークンで、推論は有効のままにする必要があります。強度は `low`、`high`、`max` に対応します。このテンプレートは `low` を使い、同じ検出事項に到達する小規模タスクの[測定](#measured-effect-glm-proxy)では、トークン数は `max` の半分を大きく下回りました。難しいタスクでは上げてください。モデル欄は編集できるため、別モデルを選ぶ場合は推論とコンテキストも見直してください。[公式 GLM-5.3 ガイド](https://docs.z.ai/guides/llm/glm-5.3)を参照。

**Coding Plan の対応境界:** Z.ai は Coding Plan を公式対応ツールに制限しています。ScopeWeaver は一覧に含まれず、エンドポイントのテンプレートは許可や対応を証明しません。使用前に Z.ai の許可を得てください。[ツール連携](https://docs.z.ai/devpack/tool/others)と[利用方針](https://docs.z.ai/devpack/usage-policy)を参照。ScopeWeaver は対応ツールを偽装したり、課金エンドポイントを自動で切り替えたりしません。

プロファイル名と自分の API キーを入力します。どちらのテンプレートも、入力済みの名前、キー、プロキシ、セッションヘッダー、その他の無関係な設定を保持します。準備できたら保存し、新規プロファイルは別途有効化してください。既存の**接続をテスト**と**モデルを読み込む**は、クリック時にのみ実際のプロバイダーへ要求を送ります。この変更で実際のプロバイダーテストを行うわけではありません。

公式資料の確認日: **2026-10-03**。プロバイダーの可用性、アカウントの権利、方針は変わる場合があります。

<a id="opencode-go-opencode-subscription-gateway"></a>
<a id="opencode-sdk-bridge"></a>

## OpenCode SDK ブリッジ

`opencode-sdk` は独立した任意のプロファイル形式であり、OpenCode 内蔵 HTTP API ではありません。ScopeWeaver の Norma が対話、ツール、承認、スコープ、再試行、圧縮を管理します。ブリッジはモデルの 1 ステップとホストツール要求の収集のみを処理します。

| `SCOPEWEAVER_OPENCODE_RUNTIME` | 固定ランタイム | 引数・履歴の境界 |
| --- | --- | --- |
| `native`（既定） | 公式 CLI／クライアント／SDK **2.1.4** | 応答とホスト履歴の生ツール引数 JSON を 64 ビット整数も含め保持。正本の生引数以外で危険な数値は拒否。 |
| `legacy` | SDK／CLI **1.19.4** | 正しく安全に表現できる JSON、絶対値 **9,007,199,254,740,991** 以下の整数のみ。空白や同値 HTML／Unicode エスケープは正規化され、生のバイトは保持しません。危険な整数と不透明／署名付き思考履歴は拒否。生引数の忠実性が必要なら `native` を使用。 |

ネイティブは固定した本物の CLI `serve --stdio`、`run --server --session` と公式クライアントでセッション、イベント、中断を処理します。呼び出しごとに私有の一時 home／設定／データと DB を所有します。実際の組み込みツール定義・実装は残しますが、権限は ASK、`--auto` は使わず、公開 `execute.before` が非ホストツールをすべて実行前に拒否します。ホスト収集要求だけを Norma に戻して実行・承認します。OpenCode のモデル／環境／ツール文脈をホスト system と履歴に加えて保持します。管理できないシステム／MDM 設定がある場合の起動拒否は、読み取り・変更を行わずに適用します。詳細はアダプターガイドにあります。

旧版のカスタムプロバイダーは明示的な OpenAI 互換マッピングを使います。必要な POSIX プロセスツリー終了処理が未実装のため、**両ランタイムとも Windows 非対応**です。ローカルのランタイム検証範囲は **macOS arm64** です。CI は **Linux x64** で実際のランタイムを使うテストを実行します。他のプラットフォームを検証済みとはしません。未テストの SDK バージョンには別途検証が必要です。

[アダプターの起動手順](../../adapters/opencode-sdk/README.ja.md)に従い **Bun 1.3.14** を使います。ブリッジは独自設定を持ち、保存済み OpenCode／クラウド認証情報を取り込みません。空白以外の文字 **32 文字以上**の独立した `SCOPEWEAVER_OPENCODE_TOKEN` を設定し、**LLM → 新規 → 設定テンプレート → OpenCode SDK ブリッジ**に入力します。OpenCode Go／Console API キーを貼り付けないでください。

```sh
cd adapters/opencode-sdk
bun install --frozen-lockfile --ignore-scripts
export SCOPEWEAVER_OPENCODE_TOKEN="$(openssl rand -hex 32)"
bun run start
```

トークンをプロファイルの**ブリッジトークン**欄用に保持します。既定は `native`、起動前の `SCOPEWEAVER_OPENCODE_RUNTIME=legacy` で旧版を選択します。待受は `127.0.0.1:4097`。明示的プロバイダーには `SCOPEWEAVER_OPENCODE_PROVIDER_CONFIG` の専用 JSON を使います。ユーザーの認証ファイルではなく、アダプター記載の形式を使用してください。オプションのプラットフォーム依存を省略しません。

`SCOPEWEAVER_OPENCODE_TIMEOUT_MS` は **0–86,400,000 ミリ秒**、既定 **0** はホストによるキャンセルに委ねます。要求本文は **16 MiB** が上限で、超過は明示的に失敗します。他の出力・終了処理制限はアダプターガイドを参照してください。

| 項目 | 値 |
| --- | --- |
| 形式 | `opencode-sdk` |
| ブリッジ URL | `http://127.0.0.1:4097` |
| モデル | **モデル読み込み**で選択、または正確な `providerID/modelID` |
| ブリッジトークン | ブリッジの `SCOPEWEAVER_OPENCODE_TOKEN`。クラウド API キーではありません |
| ストリーミング | 有効。選択した固定ランタイムを使用 |
| 思考／推論強度 | テンプレートでは省略。モデル対応値のみ指定 |
| 出力上限 | `0` は従来のホスト経路の既定値。モデルの宣言する小さい上限で制限 |
| プロキシ／独自セッションヘッダー | 非対応。テンプレートで両方を消去 |

テンプレートは未保存のクラウドキーを消し、モデルを空にして、**探索・推論・保存・有効化を行いません**。トークン入力後、明示的に**モデル読み込み**を押すと推論せずに一覧を取得します。保存と有効化は別操作です。**接続テスト**はクリック時だけモデル要求を実行し、メタデータの取得成功は推論可能という証明ではありません。

**過去の無料モデル検証 — 2026-10-25 KST：** 一覧にある場合、`opencode/deepseek-v4.1-flash-free` を選べます。当時の未ログイン macOS arm64 実行では本物の CLI 2.1.4 を使い、UI 接続テスト、手動 ASK 承認後のホスト Bash 書き込み、正確な `FREE_OK` と改行のファイル内容、最終回答が成功しました。推論 POST **3 回**、再試行無効、有料代替なしで、ネイティブツール実行は観測していません。これは当時の小さな隔離ワークフローだけの証拠であり、今回の再実行、永続提供、無制限利用を示しません。現在の提供状況、資格、制限を確認してください。旧版のホステッド無料アクセスは未検証です。

編集中に空トークンで保存値を保持できるのは同じ SDK URL のみです。URL 変更で入力トークンを消し、宛先・形式変更後の空欄では保存認証情報を流用しません。`localhost`、`127.0.0.1`、`::1` は別スコープです。モデル ID／出力上限は現在のメタデータを使います。推論設定はモデル・プロバイダー次第で、未対応なら明示的に失敗します。

**ネットワークとアカウント：** ブリッジと ScopeWeaver は同じネットワーク名前空間で動かします。コンテナー内 localhost はコンテナー自身です。下流はクラウドの場合があるため、**ローカル専用 LLM** モードは SDK を拒否します。ブリッジプロキシは非対応で、下流接続はブリッジ側で設定します。無料モデル ID は購読資格、継続提供、無制限利用を保証しません。規約と現在の制限を確認してください。ブリッジは有料代替やアカウント取り込みを行いません。ホストの設定済みフェイルオーバーは別管理です。既存 API／OpenCode Go プロファイルは独立しています。

**ダウングレード準備：** DB のプロファイル形式 `CHECK` 制約に `opencode-sdk` を追加します。SDK プロファイルが残ると旧バイナリは起動できません。DB をバックアップし、現バージョンで旧対応形式に変換するか SDK プロファイルを削除してから戻します。過去の検証は使い捨て DB のみで、稼働 DB は移行していません。

## OpenCode Go（opencode のサブスクリプションゲートウェイ）

**LLM → 新規 → 設定テンプレート**で **OpenCode Go · glm-5.3-flash** を選択します。

| フィールド | 値 |
| --- | --- |
| 形式 | OpenAI Chat Completions |
| ベース URL | `https://opencode.ai/zen/go/v1` |
| モデル | `glm-5.3-flash` |
| 思考機能 | 省略（モデルの既定値）|
| 推論強度 | `max` |
| コンテキストウィンドウ（K）| `0`（既定 200K）|
| 出力上限 | `16384`、`max_tokens` として送信 |
| セッションヘッダー | `x-opencode-session`（ゲートウェイで必須。テンプレートで設定）|

このゲートウェイは OpenAI 互換です。opencode 1.18.5 のプロバイダー定義（`opencode-go`）は、`https://opencode.ai/zen/go/v1` で `@ai-sdk/openai-compatible` と環境変数 `OPENCODE_API_KEY` を使います（2026-10-20 確認）。GLM 系の強度は `low`、`high`、`max` に対応し、このテンプレートはオペレーターの要求で意図的に `max` を設定するため、推論トークンは最も多くなります。**モデルを読み込む**はゲートウェイの一覧を取得し、モデル欄は `qwen3.7-plus` など他の ID に編集できます。全リクエストで `x-opencode-session` が必須です（ない場合は `400 MissingSessionID`）。テンプレートはこの名前を**セッションヘッダー**に設定し、ScopeWeaver は独自のセッション ID を値として送信します。

**認証情報と規約:** キーは opencode のサブスクリプションに属し、opencode が独自の認証ファイルに保存します。ScopeWeaver はそのファイルを読みません。キーをプロファイルに貼り付けるか `OPENCODE_API_KEY` を設定してください。キーは opencode のサブスクリプション規約に従います。別のツールからの使用が許されることを保存前に確認してください。Z.ai Coding Plan と同様、テンプレートは許可を証明しません。

環境変数による設定:

```bash
SCOPEWEAVER_LLM_PROVIDER=opencode-go
SCOPEWEAVER_LLM_MODEL=glm-5.3-flash
# Key: OPENCODE_API_KEY is used when no generic key is set; SCOPEWEAVER_LLM_API_KEY overrides.
```

`SCOPEWEAVER_LLM_PROVIDER=opencode-go` は OpenAI Chat Completions の通信形式に対応し、ベース URL とモデルが未設定なら上記表の既定値を使います。

<a id="controlling-token-use"></a>
## トークン使用量の管理

エージェントはターンごとに、システムプロンプト、ツールのスキーマ、ここまでの会話を再送します。そのため圧縮で履歴が短くなるまで入力が増え続け、繰り返した入力が安いキャッシュ料金になる量はプロバイダーのキャッシュが決めます。この節はクラウドのプロファイルについて [Issue #24](https://github.com/cskwork/scopeweaver/issues/24) に回答します。ネイティブ Ollama は独自の保護を持ち、[後述](#runtime-safeguards)します。

**証拠の状態:** 測定は DeepSeek の代理として、Coding Plan 上の Z.ai GLM-5.3 を使っています。GLM はトークナイザー、キャッシュの集計、価格が異なり、**DeepSeek の数値は未検証**です。DeepSeek、Claude、OpenAI、Qwen、Kimi、OpenRouter、OpenCode Go の対応は、実際の呼び出しではなく、各社の文書化された応答に基づくテスト用データで検証しました。下記の Claude 5.5 の全要求・再生動作も含め、Claude API キーは使用していません。

<a id="economy-templates"></a>
### 節約用テンプレート

**LLM → 新規 → 設定テンプレート**で *（節約）* の項目を選択します。6 つすべてが出力上限 `16384`、コンテキストウィンドウ `0`（圧縮の既定値 200K）を設定します。GLM と同様、入力済みキーを保持し、保存や有効化は行いません。

| テンプレート | 形式 | ベース URL | モデル | 思考機能 | 推論強度 | 出力フィールド |
| --- | --- | --- | --- | --- | --- | --- |
| DeepSeek | OpenAI Chat Completions | `https://api.deepseek.com` | `deepseek-flash` | 省略（DeepSeek の既定: 有効）| `low` | `max_tokens` |
| Anthropic Claude | Anthropic | `https://api.anthropic.com` | `claude-haiku-5-5` | `disabled` | `low` | `max_tokens` |
| OpenAI | OpenAI Responses | `https://api.openai.com/v1` | `gpt-6-luna` | 省略 | `low` | `max_output_tokens` |
| Alibaba Qwen（国際版）| OpenAI Chat Completions | `https://dashscope-intl.aliyuncs.com/compatible-mode/v1` | `qwen3.8-flash` | 省略 | `low` | `max_completion_tokens` |
| Moonshot Kimi | OpenAI Chat Completions | `https://api.moonshot.ai/v1` | `kimi-k3` | 省略 | `low` | `max_completion_tokens` |
| OpenRouter | OpenAI Chat Completions | `https://openrouter.ai/api/v1` | `deepseek/deepseek-v4.1-flash` | 省略 | `low` | `max_tokens` |

- 上記すべてで、**強度 `low` は未検証の節約用既定値**です。結果が途中で切れる・浅すぎる場合、まず各社の既定値（DeepSeek `high`、GPT-6 Luna `medium`、Qwen `xhigh`、Kimi K3 `max`）へ上げ、次に出力上限を上げてください。
- Claude: テンプレートは出力トークン節約のため Haiku の思考を無効にします。Haiku で無効にできる強度は `low`、`medium`、`high` のみです。Sonnet と Opus 5.5 は無効な思考設定を拒否するため、節約テンプレートはありません。思考を**省略**または**適応型**で使ってください（[Claude 5.5 の思考](#claude-55-thinking)参照）。
- OpenAI: GPT-6 Luna の Chat Completions は強度 `none` の場合のみツールを受け付けるため、テンプレートは Responses 形式を使います。
- OpenRouter: モデル ID と `reasoning_effort` の対応を、2026-10-16 に公開モデル一覧で確認しました。OpenRouter 経由の Claude は Chat Completions から送れないキャッシュマーカーが必要なため、テンプレートはプレフィックスを自動キャッシュする DeepSeek Flash を使います。
- Z.ai GLM-5.3: General と Coding Plan のテンプレートは、下記で測定した節約設定の強度 `low`、出力 `16384`、コンテキスト `0` を設定します。以前は `max`、`1000`、`0` でした。GLM-5.3 は思考が必須のため有効のままです。

<a id="claude-55-thinking"></a>
### Claude 5.5 の思考

この規則は、公式ホスト（`https://api.anthropic.com`、または空のベース URL）上の `claude-…-5-5`（日付付きは `claude-…-5-5-YYYYMMDD`）という形式のモデル ID のみに適用します。

- 思考を空欄（**省略**）にすると、`thinking: {type: adaptive, display: summarized}` を送ります。新しい**適応型**も同じです。**オン**（`enabled`）にも `display: summarized` を付けます。
- ツールのループでは、空テキストの思考や `redacted_thinking` も、対応するツール呼び出しとともに再生します。プロバイダーごとに最新 128 回のツール呼び出し分（最大 4 MiB）をメモリーに保持します。要約テキスト付きのブロックはセッション履歴にも保存しますが、空・秘匿ブロックはループ途中の再起動では残りません。
- 要約と承認判定の呼び出しは `temperature` を送りません。また、Sonnet、Opus、および `xhigh`・`max` の Haiku は `disabled` を拒否するため、その思考無効化を外し、プロファイルの思考設定を代わりに適用します。
- **オフ**（`disabled`）のプロファイルは、エージェントのターンでは引き続き `disabled` を送るため、Sonnet と Opus 5.5 では**省略**か**適応型**が必要です。
- 他社の Anthropic 互換エンドポイント（DeepSeek、Kimi、Z.ai）と他のモデル ID は変更せず、空の思考は引き続き省略します。
- **要求・ストリームのテスト用データでのみ検証**しています。実際の Claude の動作と課金は未検証です。

<a id="measured-effect-glm-proxy"></a>
### 測定した効果（GLM を代理に使用）

小規模なタスク 1 件（ローカル対象の 3 ルート、認証なしで読めるシークレットを 1 つ設置）を、プロファイルごとに 3 回、新しい DB で実行し、レポート担当の会話が完了するまで待ちました。クレジットは Z.ai 公開の GLM-5.3 係数、(非キャッシュ入力 × 6.9 + キャッシュ入力 × 1.7 + 出力 × 24) / 10,000 で計算し、請求書から取得した値ではありません。

| プロファイル | 入力 + 出力トークンの中央値 | クレジットの中央値 | 設置した問題の検出 |
| --- | --- | --- | --- |
| 基準: 以前の Z.ai Coding テンプレート（強度 `max`、コンテキスト `1000`、出力 `0`）| 209,233 (172,576–216,452) | 75.1 (64.3–88.6) | 3 回中 3 回 |
| 節約: 強度 `low`、出力 `16384`、コンテキスト `0` | 96,023 (95,060–105,023) | 26.6 (25.0–27.4) | 3 回中 3 回。毎回レポート担当が報告 |

- 節約設定は基準より中央値で入力 + 出力トークンが 54%、クレジットが 65% 少なく、すべての節約実行が最も安い基準実行を下回りました。最大の単一呼び出しは入力 13,287、出力 3,829 で、200K ウィンドウにも出力上限にも達していません。ここでの削減は強度 `low` によるものです。出力上限とウィンドウは大きなタスクを制限しますが、効果は測定していません。実行間の最大/最小の差は、基準 1.25×、節約 1.10× でした。
- 処理区分別クレジット: 基準はワーカー 29.6%、プランナー 29.3%、レポート担当 29.1%、目標設定 12.0%。節約はレポート担当 38.8%、プランナー 34.1%、ワーカー 20.2%、目標設定 6.9%。圧縮は起きず、セッション要約の呼び出しはありません。判定とウェブ検索は無効でした。
- 入力のキャッシュ割合は基準 85.2%、節約 92.3%。推論を含む出力は、基準のクレジットの 39.3%、節約の 23.4% でした。当時はまだ推論トークンを別に記録していませんでした。

GLM-5.3 は常に思考するため、この実行は思考無効化の効果を測定していません。

実際のブリッジ確認（Claude は `claude -p`、OpenAI は `codex exec`）は、ローカルの橋渡し経由のモデル動作をテストし、API の通信形式や課金は検証しません。

`codex exec` のブリッジで OpenAI テンプレート（gpt-6-luna、Responses）のタスクを 1 件、最後まで実行しました。ツールのループが完了し、設置した問題を検出してレポート担当が記録し、その後の補足の質問にも回答しました（中断扱いではありません）。`llm_usage` は `btw` の行も含め、推論とキャッシュ読み取りのトークンを記録しました。追加の 2 回はブリッジが遅いためハーネスの制限時間内に完了せず、コードの結果ではなくブリッジの速度制限です。`claude -p` はツールのプロトコルを動かせず、取りやめました。Claude と OpenAI の要求・使用量処理はテスト用データで実証しています。

<a id="recommended-settings"></a>
### 推奨設定

1. 各社の節約テンプレートから始め、追加履歴が必要でなければコンテキストウィンドウ `0` を保ちます。
2. 出力上限は正の値を保ちます。`0` はフィールドの省略ではありません（下記参照）。
3. 強度は低く保ち、結果が悪化する場合のみ上げます。
4. 承認判定、ウェブ検索、noa のコンテキスト圧縮は、必要な場合以外は無効にします（すべて既定で無効）。
5. 小さなタスクでは、ワークエージェントを既定の 3 より減らすことも検討できます。効果は未測定です。
6. 検出事項ごとの詳細レポートが不要なら、**レポート作成担当**のトリガーを無効にします（**システム → エージェント**）。
7. 最初のタスク後に、処理区分ごとの使用量を確認してください（[トークンの使用先の確認](#checking-where-tokens-go)）。

<a id="context-window-output-limit-thinking-and-effort"></a>
### コンテキストウィンドウ、出力上限、思考、強度

| 設定 | クラウドプロファイルの動作 | 指針 |
| --- | --- | --- |
| コンテキストウィンドウ（K）| ローカルの圧縮予算のみで、プロバイダーには送らない。`0` = 200K。値は 32～1000 に制限 | 大きいウィンドウは圧縮を遅らせ、各呼び出しの履歴が増える。200K では約 115K を超えると古いツール結果を消し始め、約 179K で履歴を要約 |
| 出力上限 | 応答ごとの上限。**`0` は省略ではない**。エージェントは 32768、Anthropic の直接呼び出しは 8192、ネイティブ Ollama は独自の既定値 | 上限は短い応答を促すのではなく、長い応答を停止する。低すぎると切り詰める。テンプレートは 16384 |
| 思考機能 | `thinking.type`。空なら省略。ただし公式 Claude 5.5 では `adaptive` を送信。推論は出力として課金 | DeepSeek は既定で思考し、`disabled` を文書化。ツール使用時は以前の推論も入力として再送。GLM-5.3 は常時思考。Claude テンプレートは無効化 |
| 推論強度 | `low`～`max`。空なら省略 | テンプレートは `low`。測定済みは GLM-5.3 のみで、他社では未検証の節約用既定値。対応値は会社ごとに異なる |

<a id="context-optimization"></a>
### コンテキストの最適化

- **会話の圧縮**（組み込み、有効）: ウィンドウの上限付近で古いメッセージを要約し、最新 8 件を保持します。要約は追加の 1 呼び出しで、出力上限まで、最大 20000（上限 `0` でも 20000）に制限します。
- **古いツール出力の消去:** ウィンドウの 60% を超えると、最新 5 件より古い Read、Bash、Grep、Glob、WebFetch、WebSearch、Edit、Write の結果を置き換えます。グラフ、証拠、カスタムツールの結果は次の要約まで残します。
- **ツール出力上限:** 各テキストブロックを 30,000 バイトに制限し、全出力は参照先付きでディスクに残します。
- **グラフ圧縮:** 非稼働の探索グラフ部分をダイジェストにまとめます。対象グループごとに 1 呼び出し（出力上限 1500）。
- **noa コンテキスト圧縮**（システム設定、実験的、無効）は、組み込み圧縮を置き換えます。
- 72% の開始閾値と 16/32 KiB の出力上限は、ネイティブ Ollama のみに適用します。

**プロンプトのキャッシュ:**

- Claude: プランナー、ワーカー、メインエージェント、チャット（レポート担当を含む）で、システムプロンプトとその前のツールスキーマを常にキャッシュ対象にします。他の Anthropic 形式の要求でも、システムプロンプトがあり境界がなければ全体にマークします。対象は目標設定、会話・グラフの要約、補足の質問の準備、承認判定です。モデルの最小キャッシュ長より短いプロンプトは、マーカーがあってもキャッシュされません。異なるシステムプロンプトの初回は入力料金の 1.25× で書き込み、有効期間内の後続はキャッシュ料金で読みます。ワーカーのプロンプトは探索方針ごとに異なるため、各方針で 1 回書き込みます。マーカーの配置はテスト済みですが、**実際の削減効果は未検証**です（Claude キーなし）。
- DeepSeek、Kimi、Z.ai はプレフィックスを自動キャッシュします。DeepSeek と Kimi は、Anthropic 互換エンドポイントでブロックのマーカーを無視すると文書化しています。OpenAI、Qwen、OpenRouter（DeepSeek Flash）も一致するプレフィックスを自動キャッシュします。
- 3 種のクラウド形式すべてで、同じセッション内の連続するツールのターンが、同じシステムプロンプト・ツールスキーマを保持し、履歴だけを追記するとテストで確認しています。圧縮、終了処理、新しく解放した遅延ツールはプレフィックスを変えます。編集したテンプレートに `{{.Now}}` があると、実行ごとにシステムプロンプトが変わります。

<a id="workers-retries-and-task-settings"></a>
### ワーカー、再試行、タスクの設定

- **ワーカーの同時実行数・エージェント数**（システム設定、既定はタスクごとに 3）: 各ワーカーは別の探索方針を取得し、システムプロンプトとツールスキーマ全体を持つ独自のセッションを実行します。
- **LLM リクエストの最大同時実行数**（システム設定、既定 `0` = 無制限）と、プロファイルの**最大同時リクエスト数**は呼び出しのペースを制御し、呼び出しごとのトークン数は減らしません。
- 再試行は要求全体を再送します。既定は**接続** 3 回、**空の応答** 2 回（OpenAI 形式のみ）、**同じプロバイダーの安全な期間** 2 回（出力前）、モデルエラー後の探索方針の再実行 2 回、推論だけの応答後の続行が最大 2 回です。LLM ページのプロファイル別または全体方針で回数を `-1` にすると、その層を無効化します。失敗した試行の課金はプロバイダーに依存します。
- エージェントがターン・時間の予算を使い切るかタスクが終了処理に入ると、既定で最大 10 ターンの終了処理を追加します。
- プロバイダープールのフェイルオーバー（既定で無効）は、別プロファイルで要求全体を再実行します。

<a id="known-causes-of-extra-calls"></a>
### 呼び出しが増える既知の原因

- 長いセッション: 毎ターン全履歴を再送するため、後半のターンほど高くなります。
- プランナー: 起動ごと（開始、探索方針完了、検出事項追加、待機ハートビート）に新しいセッションで、複数ターンを使います。
- 目標の分解は計画前に実行し、最大 8 ターンです。
- **レポート担当:** `report_finding` ごとの有効なトリガーを持つ初期設定済みのカスタムエージェントです。検出事項ごとに会話を作り、最大 5 並列で実行します。
- **DeepSeek のウェブ検索:** 検索ごとに有効プロファイルのキーで別の DeepSeek 呼び出しを行います（出力上限 1024、1 呼び出し最大 3 検索）。エージェントの検索と設定画面の検索テストは、`llm_usage` に `web_search` 区分と有効プロファイルの名前・モデルで記録し、LLM の記録には含めません。DeepSeek は検索応答の使用量形式を文書化していないため、Anthropic 形式の使用量を読みます。**使用量フレームのない応答は 0 トークンで記録します。呼び出し回数は分かりますがトークン数は不明です。** 請求分は DeepSeek コンソールで確認してください。
- 承認判定（既定で無効）: 一致するルールのないツール操作ごとに 1 呼び出し。補足の質問は要求時のみで、回答前に最大 12 回の要約を行います。
- **接続をテスト**は、記録されない実際の呼び出しを行います（出力上限 8192）。
- 集計の誤りを修正: Anthropic 形式のストリーム（Claude と互換エンドポイント）が `llm_usage` の出力を過大計上していました。累積出力 10、20 のテストストリームを、20 でなく 30 と記録していました。過大だったのは ScopeWeaver の数値だけで、プロバイダーの課金ではありません。
- 修正: 内部のセッション・会話記録の合計も、同じ 10→20 のストリームを 30 と過大計上していましたが、現在は 20 です。これは `llm_usage` とは別で、過大計上による影響は圧縮を早めることだけでした。
- 修正: OpenAI Responses の補足の質問は、停止イベントがないため中断と報告されていました。現在は `response.completed` で正常に終了し、`response.incomplete` は引き続き中断とします。

<a id="checking-where-tokens-go"></a>
### トークンの使用先の確認

- **タスク → 対象タスク → 概要 → LLM トークン使用量（モデル別）**: 常時有効の `llm_usage` から、呼び出し、入力、出力、**推論（出力に含む）**、キャッシュ読み取りをモデル別に表示します。タスクのタグがある行のみで、レポート担当の行にはタスクがありません。DeepSeek の検索は有効プロファイルのモデルの下に表示します。
- **ダッシュボード**と**インターセプトルール → 承認のトークン使用量**も、**推論（出力に含む）**を表示します。
- **LLM の記録**: 呼び出しごとに処理区分、トークン、待ち時間、生の要求・応答を保存します。既定では無効なので、タスク前に**記録**を有効にしてください。完全なプロンプトと応答を保存します。
- 特定の期間の `llm_usage` を SQL で照会できます。区分は `worker` 列で、`planner`、`worker`、`goals`、`compactor`、`judge`、補足の質問の `btw`、`mainagent`、DeepSeek 検索の `web_search` です。レポート担当、再テスト、カスタムエージェントの会話は空です。会話の要約と終了処理は元のエージェントの区分を保持します。

```sql
SELECT COALESCE(worker, '(none)') AS lane, model, COUNT(*) AS calls,
       SUM(input_tokens) AS input, SUM(cache_read) AS cache_read,
       SUM(cache_write) AS cache_write, SUM(output_tokens) AS output,
       SUM(reasoning_tokens) AS reasoning
FROM llm_usage
WHERE ts >= '2026-10-16 09:00' AND ts < '2026-10-16 10:00'  -- your task's time range
GROUP BY 1, 2
ORDER BY SUM(input_tokens) + SUM(output_tokens) DESC;
```

`cache_read` と `cache_write` はすでに `input`、`reasoning` はすでに `output` に含まれます。重複して加算しないでください。

キャッシュ読み取りは `prompt_tokens_details.cached_tokens`、DeepSeek 最上位の `prompt_cache_hit_tokens`、Kimi の旧最上位 `cached_tokens` から集計します。書き込みは Claude に加え、OpenAI Chat、OpenAI Responses、Kimi K3、OpenRouter の `cache_write_tokens`、Qwen の `cache_creation_input_tokens` から記録します。これらの対応はテスト用データのみで検証しています。

<a id="accounting-gaps-and-follow-ups"></a>
### 集計の不足と今後の対応

- 別の推論数値を返さないプロバイダーでは 0 です。0 は、モデルが推論しなかった証明ではありません。
- DeepSeek 検索の使用量フィールド名は未文書化です（[追加呼び出しの原因](#known-causes-of-extra-calls)参照）。
- Gemini のテンプレートは未対応で、Norma SDK v0.4.4 への更新が必要です（今後の対応）。
- Claude の履歴のキャッシュ（最後のメッセージのマーカー）は SDK の対応が必要です（今後の対応）。
- 統一された `reasoning.effort` のみを認識する OpenRouter モデルは、SDK が送れる唯一の形式 `reasoning_effort` を無視する場合があります（今後の対応）。

<a id="native-ollama-local-inference"></a>
## ネイティブ Ollama（ローカル推論）

ScopeWeaver は Ollama のネイティブ `/api/chat` を使います。OpenAI 互換エンドポイントではコンテキストサイズを設定できないため（[Ollama の OpenAI 互換性](https://docs.ollama.com/api/openai-compatibility)）、`/v1` に向けたプロファイルでは `num_ctx` を保証できません。ネイティブ形式は毎回送信します。ベース URL の末尾の `/v1`、`/api`、`/api/chat` はサーバーのルートに正規化します。

Ollama の既定コンテキストは資料で異なり、[FAQ](https://docs.ollama.com/faq) は 4,096、[コンテキスト長](https://docs.ollama.com/context-length)は VRAM ベース（24 GiB 未満は約 4K、24～48 GiB は 32K、それ以上は 256K）としています。そのため、ScopeWeaver はサーバーの既定値に依存しません。

<a id="templates"></a>
### テンプレート

**LLM → 新規 → 設定テンプレート → ローカル推論（Ollama）**を開きます。

| フィールド | 同じマシン | Docker からホスト | 同梱の Compose サービス |
| --- | --- | --- | --- |
| 形式 | Ollama（ネイティブ /api/chat）| Ollama（ネイティブ /api/chat）| Ollama（ネイティブ /api/chat）|
| ベース URL | `http://127.0.0.1:11434` | `http://host.docker.internal:11434` | `http://ollama:11434` |
| モデル | このサーバーのインストール済みモデル | このサーバーのインストール済みモデル | このサーバーのインストール済みモデル |
| API キー | 空 | 空 | 空 |
| コンテキストウィンドウ（K）| `32`（`num_ctx` 32,768）| `32` | `32` |
| 最大同時リクエスト数 | `1` | `1` | `1` |
| キープアライブ | `30m` | `30m` | `30m` |
| 出力上限 | `4096` | `4096` | `4096` |
| 思考 / 強度 | 省略（モデルの既定値）| 省略 | 省略 |

Ollama テンプレートの選択は、選択した接続先の `/api/tags` とモデルのメタデータを読み、`tools` を宣言するものを優先してインストール済みモデルを選びます。新規で Ollama 形式を選ぶ場合も同じです。接続先が空、または未変更のクラウドテンプレートのままなら、ホストの既定値を使い、独自の接続先は変更しません。検出中はモデル欄を空にし、失敗・空の一覧ではエラーとともに空欄を保ちます。接続先、任意のトークン、プロキシを確認し、**モデルを読み込む**かタグを手入力してください。モデルや接続欄を編集すると保留中の選択を中止し、遅い応答は編集を上書きしません。既存プロファイルを開くと保存済みモデルを保持します。選択でダウンロード・実行・保存・有効化は行いません。**接続をテスト**はクリック時のみ実行します。

**API キーの動作:** Ollama テンプレートと形式への切り替えは、検出前に未保存のキーを消去します。名前、プロキシ、セッションヘッダー、その他の入力設定は保持しますが、クラウドの認証情報がローカルへ送られないよう、**未保存の API キーは消去**します。GLM テンプレートは引き続き入力済みキーを保持します。ローカル Ollama は空欄、前段のリバースプロキシで必要な場合のみトークンを入力してください（`Authorization: Bearer` で送信）。キーなしでもテスト、保存、有効化、フェイルオーバーが可能で、カードはキー不足の警告でなく**キー不要**と表示します。保存済みネイティブ Ollama の編集で空欄なら保存済みトークンを保持し、クラウドから Ollama への切り替えで空欄なら以前のクラウドキーを削除します。

**他の形式のキー:** Anthropic と OpenAI 形式は、ベース URL が明示的にローカル（名前解決先がループバックかプライベートアドレスのみ）でない限りキーが必要です。そのため vLLM や LM Studio などのキー不要なローカル互換サーバーでは空欄を使えます。ScopeWeaver は非機密の内部プレースホルダーを送信し、SDK が環境の `OPENAI_API_KEY`・`ANTHROPIC_API_KEY` を継承しないようにします。ローカル以外でキーがなければ拒否します（*llm: API key required for non-local compatible endpoint*）。

<a id="context-output-and-residency-controls"></a>
### コンテキスト、出力、常駐の制御

| 設定 | Ollama の動作 | クラウド形式 |
| --- | --- | --- |
| コンテキストウィンドウ（K）| 2 進の K 単位を `options.num_ctx` で送信（32 = 32,768）。`0` は 32、16 未満は 16。圧縮予算にも使用 | 変更なし。圧縮予算のみ。`0` = 200K |
| 出力上限 | `options.num_predict` として送信。`0` は 4,096 | `0` はエージェントのターンで 32768（Anthropic の直接呼び出しは 8192）を送信 |
| キープアライブ | `keep_alive` として送信。空欄は `30m`（Ollama 自体の既定は 5 分）| 送信しない |
| 最大同時リクエスト数 | `0` は 1。Ollama サーバーごとに受け入れゲートを共有し、複数プロファイルでも最小の設定上限を適用 | `0` は**システム設定 → LLM リクエストの最大同時実行数**を継承 |

待機中の承認判定、メインエージェント、補足の質問は、バックグラウンド作業より先に次の空き枠を取得します。各グループ内は FIFO で、実行中の推論は中断しません。診断とウォームアップはバックグラウンドのままです。

同時実行数は、サーバーの `OLLAMA_NUM_PARALLEL` が 1 より大きい場合のみ上げてください。Ollama のメモリー使用量は `OLLAMA_NUM_PARALLEL × context length` に従って増えます（[FAQ](https://docs.ollama.com/faq)）。

<a id="runtime-safeguards"></a>
### 実行時の保護

- **ウォームアップ:** ネイティブプロファイルの最初の推論前に `/api/show` を呼び、`tools` を宣言しないモデルを拒否し、プロファイルの `num_ctx` と `keep_alive` を使う空の `/api/chat` でモデルを読み込みます。同じ同時実行枠の中で行います。
- **圧縮:** ネイティブ Ollama の 64K 以下のウィンドウでは、72% で履歴を圧縮し、完全なツール呼び出し・結果の対を保持します。実験的な noa が有効でも、この保護を優先して noa を省略し、警告を出します。32K 以下では各ツール出力のモデルに渡す部分を 16,384 **バイト**（16 KiB、Unicode の文字数でなく SDK の `len(s)`）に制限し、32K 超～64K は 32,768 バイト（32 KiB）です。全出力はツール結果から参照する退避ファイルに保持します。クラウド形式の圧縮と noa の動作は変えません。
- **承認判定:** 明示的な判定プロファイルを優先します。それ以外は呼び出し元の実行で実際に使ったネイティブプロファイル（紐付け、プール、フェイルオーバーを含む）を優先し、クラウド・起源不明は有効なプロファイルを使います。ネイティブ判定の既定は**90 秒**、失敗時は **ask** です。保存済みのタイムアウト・失敗時設定が優先し、人間の承認は引き続き既定で 300 秒後に deny となります。ネイティブ出力を decision/comment の JSON スキーマに制約し、さらに検証します。設定フォームは、サーバーの設定を正常に読み込むまで保存できません。

<a id="generic-openai-compatible-servers-vllm-lm-studio"></a>
### 一般的な OpenAI 互換サーバー（vLLM、LM Studio）

既存の形式を選択できます。**OpenAI（Chat Completions）**を選び、ベース URL をサーバーの `/v1` に向けてください。これらにネイティブ Ollama 形式は使わないでください。

- ネイティブ形式の**モデルを読み込む**は `/api/tags` を読み、HTTP 404・405 なら `/v1/models` に戻るため、一般的なサーバーのモデル一覧が表示される場合があります。
- チャットは**黙って切り替えません**。ネイティブは常に `/api/chat` を呼び、一般的なサーバーにはありません。発生する「モデル・エンドポイントなし」や `ollama pull` の案内は該当せず、モデルを取得しても vLLM・LM Studio の修正にはなりません。OpenAI 形式に変更してください。

<a id="load-models-and-test-connection"></a>
### モデルの読み込みと接続テスト

**モデルを読み込む**は `/api/tags` と `/api/show` を読み、パラメーター数、量子化、`tools` / `thinking` / `vision` のバッジを表示します。`tools` のないメタデータには「このモデルはネイティブのツール呼び出し対応を宣言していません。ScopeWeaver のエージェントは、資産グラフと探索グラフの更新にツール呼び出しが必要です」と警告します。メタデータの取得失敗は、非対応でなく不明と表示します。

**接続をテスト**は `/api/show` を確認し、プロファイルの `num_ctx`、`keep_alive`、`num_predict` で 1 ターンのツール呼び出しスモークテストを行い、`/api/ps` を読みます。要求以上のコンテキスト割り当て、ツール呼び出しの返却、コールドロードのミリ秒、毎秒の生成トークン数を表示します。モデルがなければ、`ollama pull qwen2.5-coder:32b` など、サーバーで実行する正確なコマンドを示します。ScopeWeaver はモデルを取得せず、Modelfile も作成しません。

<a id="host-and-container-routing"></a>
### ホストとコンテナーの経路

| ScopeWeaver の実行場所 | Ollama の実行場所 | ベース URL |
| --- | --- | --- |
| ホスト | 同じホスト | `http://127.0.0.1:11434` |
| Docker Compose 内 | ホスト | `http://host.docker.internal:11434` |
| Docker Compose 内 | 同梱の `ollama` サービス | `http://ollama:11434` |

コンテナー内の `127.0.0.1` はコンテナー自身です。Compose は `host.docker.internal` をホストゲートウェイ（`host-gateway`）に割り当てます。Ollama は既定で `127.0.0.1:11434` にバインドします。Linux ではホストの Ollama をコンテナーから到達するアドレス（例: `OLLAMA_HOST=0.0.0.0:11434`）で待ち受ける必要があり、ネットワークに公開されます。ファイアウォールでポート 11434 を制限してください。

**任意の Compose サービス:** `docker compose --profile local-llm up -d` は、`OLLAMA_NO_CLOUD=1`、`OLLAMA_KEEP_ALIVE=30m`、`OLLAMA_NUM_PARALLEL=1`、`OLLAMA_CONTEXT_LENGTH=32768`、ボリューム `ollama_data` で `ollama` を起動します。`docker compose exec ollama ollama pull qwen2.5-coder:14b` でモデルを取得できます。プロファイルなしなら従来どおりです。同梱サービスは GPU を要求しません。**macOS の Docker コンテナーは Apple GPU を使えず、このサービスは CPU のみで動作します**。実用的な速度には Mac 上で Ollama をネイティブ実行し、ホストのテンプレートを使ってください。

<a id="network-policy-local_llm_only"></a>
### ネットワーク方針（`local_llm_only`）

既定では無効でクラウドも利用可能です。**システム設定 → LLM のローカル限定ネットワーク**または `SCOPEWEAVER_LOCAL_LLM_ONLY=true` で有効にすると:

- LLM の HTTP（推論、モデル一覧、接続テスト、ウォームアップ、承認判定）は、**ループバックと明示的に信頼した CIDR**（`local_llm_allowed_cidrs`）のみを許可します。プライベート IPv4、IPv6 ULA、LAN、Docker のアドレスを暗黙には信頼しません。管理する推論サーバーだけを、できれば IPv4 は `/32`、IPv6 は `/128` で追加してください。
- リンクローカル、マルチキャスト、未指定のアドレスは、信頼する CIDR 内でも常に拒否します。接続を開くたびに DNS の全解決アドレスを確認し、すべてが許可される必要があります。リダイレクトと LLM プロファイルのプロキシはすべて拒否します。
- **すべてのバックエンドのエージェント検索、検索接続の検査、更新の確認・ダウンロードをブロックします。** 更新画面は、この意図した方針をネットワーク障害と区別して示します。ローカルのバージョン、更新進捗、ロールバックは維持し、迂回のためにキャッシュしたリモートリリース情報を返しません。
- 実行時の変更は、既存セッションも含め再起動なしで適用します。有効化前に許可したダウンロードも、次の保護された通信操作・本文読み取りで停止し、一部の準備ファイルを削除します。送信済みの通信は取り消せません。
- `SCOPEWEAVER_LOCAL_LLM_ALLOWED_CIDRS`（または保持された `ARTEX_LOCAL_LLM_ALLOWED_CIDRS`）も適用します。保存済み一覧にはコピーせず、UI では編集できません。環境の `true` は UI で無効にできない最低条件です。
- 対象への通信、対象用ツール、全体のプロキシは接続を維持します。これはアプリケーションの方針であり、暗号学的な保証やホスト全体のオフライン保証ではありません。

**既存の厳密な LAN/Docker プロファイルの移行:** 同じマシンの `http://127.0.0.1:11434` に CIDR は不要です。LAN の `http://192.168.1.20:11434` は `192.168.1.20/32`、IPv6 の `fd00::20` は `fd00::20/128` が必要です。Docker は、ホストとコンテナーで結果が異なる場合があるため、**ScopeWeaver のコンテナー内から**名前解決してください:

```bash
# Existing Compose service; read its actual name resolution and network addresses.
docker compose exec scopeweaver getent ahosts host.docker.internal
docker compose exec scopeweaver getent ahosts ollama
docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}} {{.GlobalIPv6Address}}{{end}}' "$(docker compose ps -q ollama)"
```

`getent` がなければ、`docker compose exec scopeweaver cat /etc/hosts` でホストゲートウェイの既存割り当てを、`docker inspect` で Ollama コンテナーのアドレスを確認します。例のコピーや Docker サブネット全体でなく、ScopeWeaver が実際に到達する推論 IP を使ってください。**追加で信頼する CIDR**または環境の許可リスト（例: `SCOPEWEAVER_LOCAL_LLM_ALLOWED_CIDRS=192.168.1.20/32`）に追加します。`host.docker.internal` や `ollama` という名前自体は暗黙の信頼を与えません。再作成やアドレス変更後に再確認し、**モデルを読み込む**・**接続をテスト**の再試行前に限定した許可リストを更新してください。

ScopeWeaver は、Ollama サーバーのその先の動作は確認できません。**Ollama はクラウドモデルへ要求を転送できます**。同梱 Compose は `OLLAMA_NO_CLOUD=1` を設定しますが、ホストやリモートでは別途無効にする必要があります（[FAQ](https://docs.ollama.com/faq) に従い、`OLLAMA_NO_CLOUD=1` または `~/.ollama/server.json` の `disable_ollama_cloud`）。

<a id="environment-and-configuration"></a>
### 環境変数と設定

`SCOPEWEAVER_LLM_*` は `ARTEX_LLM_*` より優先し、明示的に空でも旧変数を上書きします。両者は `config.json` の `llm`（パスは `SCOPEWEAVER_CONFIG` または `ARTEX_CONFIG`）より優先します。

| 接尾辞 | 意味 |
| --- | --- |
| `PROVIDER` | `anthropic`、`openai`、`openai-responses`、`opencode-go`、`ollama` |
| `MODEL`, `BASE_URL`, `PROXY` | モデル、接続先、任意の LLM プロキシ |
| `API_KEY` | ベース URL が明示的にローカルでない Anthropic/OpenAI 形式では必須。Ollama では任意のトークン |
| `CONTEXT_WINDOW_K`, `MAX_TOKENS`, `KEEP_ALIVE`, `MAX_CONCURRENT` | 上記の制御項目 |
| `THINKING_TYPE`, `REASONING_EFFORT`, `STREAM` | 既存の推論・ストリーミング設定 |

方針の変数: `SCOPEWEAVER_LOCAL_LLM_ONLY` / `ARTEX_LOCAL_LLM_ONLY`、カンマ区切りの `SCOPEWEAVER_LOCAL_LLM_ALLOWED_CIDRS` / `ARTEX_LOCAL_LLM_ALLOWED_CIDRS`。

起動時、共通キーが未設定なら、Anthropic/OpenAI は `ANTHROPIC_API_KEY` / `OPENAI_API_KEY` を認証付きローカルゲートウェイも含め使用します。`opencode-go` は `OPENAI_API_KEY` ではなく `OPENCODE_API_KEY` のみを使い、`OPENCODE_API_KEY` だけがあればプロバイダーを推定します。これらのキーがある環境で意図的にキー不要のローカルプロファイルを起動するには、`SCOPEWEAVER_LLM_API_KEY=`（または `ARTEX_LLM_API_KEY=`）を明示します。ファイルの認証情報を上書きし、各社キーへのフォールバックを抑止します。ネイティブ Ollama は `OLLAMA_API_KEY` のみに戻り、Anthropic、OpenAI、OpenCode のキーは使いません。

UI の新規プロファイルを空のキーで保存しても、環境の認証情報は取り込みません。空のキーでの**接続をテスト**は、既存プロファイル ID があれば保存済みキーを再利用できます。ID がなければ、ローカル Anthropic/OpenAI のテストは、新しい下書きも含め、形式・正規化したベース URL・プロキシが有効サービスとすべて一致する場合のみ有効キーを再利用します。別のローカル接続先、形式、プロキシでは継承しません。ネイティブ Ollama は有効なクラウドキーを決して継承しません。既存のローカル以外のテストは、有効キーへのフォールバックを保持します。

```bash
# .env for ScopeWeaver in Docker with Ollama on the host
# Replace your-installed-tool-model with a tool-capable tag shown by Load models.
SCOPEWEAVER_LLM_PROVIDER=ollama
SCOPEWEAVER_LLM_BASE_URL=http://host.docker.internal:11434
SCOPEWEAVER_LLM_MODEL=your-installed-tool-model
SCOPEWEAVER_LLM_CONTEXT_WINDOW_K=32
SCOPEWEAVER_LLM_MAX_CONCURRENT=1
SCOPEWEAVER_LLM_KEEP_ALIVE=30m
SCOPEWEAVER_LLM_MAX_TOKENS=4096
SCOPEWEAVER_LOCAL_LLM_ONLY=false   # add the actual host /32 or /128 before enabling strict mode
# SCOPEWEAVER_LOCAL_LLM_ALLOWED_CIDRS=192.168.1.20/32  # replace with the resolved inference IP
```

```json
{
  "llm": { "provider": "ollama", "base_url": "http://127.0.0.1:11434", "model": "your-installed-tool-model", "context_window_k": 32, "llm_max_concurrent": 1, "keep_alive": "30m", "max_tokens": 4096 },
  "local_llm_only": false,
  "local_llm_allowed_cidrs": []
}
```

<a id="bounded-diagnostics-and-telemetry"></a>
### 範囲を限定した診断と測定

**接続をテスト**は意図的に推論を行い、テンプレートの選択はインストール済みメタデータの検出のみです。ツール・コンテキスト、読み込み時間・速度に加え、ネイティブ診断は `/api/ps` の選択したランナーの `size` と `size_vram` を GPU メモリー割合として示します。総量がない・0、VRAM 量がない場合は、GPU 0% ではなく**不明**です。正の総量に明示的な VRAM 0 なら 0% です。メモリーの推定であり、正確な重みの分割、GPU 容量の検査、モデル常駐の保証ではありません。

80% 以下では CPU 処理で生成が遅くなる可能性を警告します。他の読み込み済みモデル、同じ接続先でモデル・コンテキストが異なる明示的な判定設定は、別のメモリー・再読み込み警告になります。同じモデル・コンテキストは再読み込みを減らす場合がありますが、常駐は保証しません。診断ではモデルのダウンロードやプロファイルの保存・有効化は行いません。

上限付近のコンテキスト警告は、報告トークン数に基づくサーバーログの経験則で、切り詰めの検出器ではありません。Ollama の完全な生の時間フィールドは記録有効時のみ保存し、トークン・キャッシュ使用量は独立して計測します。小さいモデルは引き続きタスク別の検証が必要で、下記の信頼性の制限も残ります。

<a id="hardware-tiers"></a>
### ハードウェアの目安

GPU VRAM または Apple の統合メモリーに対する、おおよその開始点です。下記のツール対応モデルを想定します。**ベンチマークではなく推定です。** コンテキストは設定する**コンテキストウィンドウ（K）**で、モデル自身の上限とメモリー（重み + コンテキストと `OLLAMA_NUM_PARALLEL` で増える KV キャッシュ）の両方に収まる必要があります。要求した割り当てを報告する**接続をテスト**で確認してください。

| メモリー | モデルの例 | コンテキストウィンドウ（K）| 役割 |
| --- | --- | --- | --- |
| 8–12 GB | `qwen2.5-coder:7b`, `qwen3:8b` | `16`–`32` | 承認判定・補助用。メインエージェントとしては弱い |
| 16–24 GB | `qwen2.5-coder:14b`, `qwen3:14b` | `32` | ワークステーションのメインエージェント |
| 32–48 GB | `qwen2.5-coder:32b`, `qwen3:32b` | `32` | メインエージェント。`64` はモデルの上限が対応する場合のみ。これらの公称値は 32K と 40K |
| 64–128 GB+ | `llama3.3:70b` | `64`（モデルの公称値は 128K）| 大容量ホストのメインエージェント。64K にはこの範囲の上側の容量が必要 |

テンプレートは選択サーバーのインストール済みモデルから選び、32K のコンテキストを使います。ハードウェア例は自動のモデル選択ではありません。

<a id="model-candidates"></a>
### モデル候補

[Ollama ライブラリー](https://ollama.com/library)で `tools` を宣言するモデルのみを掲載しています（2026-10-15 確認）。ダウンロードサイズは既定の量子化です。**メモリーの指針は推定**で、重みと、コンテキスト・並列要求で増える KV キャッシュを含みます。ScopeWeaver はモデル・ハードウェアのベンチマークを行っておらず、ツール呼び出しの品質はモデルに依存します。

| モデル | ダウンロード | 注記 |
| --- | --- | --- |
| `qwen2.5-coder:7b` | 4.7 GB | 最小の実用候補。判定用の候補でもある |
| `qwen2.5-coder:14b` | 9.0 GB | ワークステーション向けの候補 |
| `qwen2.5-coder:32b` | 20 GB | より大きなホスト向けの候補 |
| `qwen3:8b`, `qwen3:14b`, `qwen3:32b` | 5.2 / 9.3 / 20 GB | `thinking` も宣言 |
| `llama3.3:70b` | 43 GB | 大容量ホスト向け |

`qwen3:72b` と `llama3.2:8b` はライブラリーに存在せず、`llama3.2` は 1B・3B です。使用前に自分のサーバーの**モデルを読み込む**で `tools` を確認してください。

**観測した制限（2026-10-15）:** `tools` の宣言は通信上のツール呼び出しを保証しません。ブランチのテストで `qwen2.5-coder:7b` はツール風の JSON を通常テキストとして書き、ツールは実行されませんでした。`qwen3:4b` は診断に合格しましたが長い自律タスクは完了しませんでした。小さいモデルはメインエージェント用として未検証と扱ってください。[検証](VERIFICATION.md#autonomous-task-reliability-is-not-validated)を参照。

どのプロバイダーでも、[README](../../README.ja.md) の使用制限が適用され、ローカルの隔離環境でのみ実行してください。

[韓国語の案内](../ko/llm-providers.md)
