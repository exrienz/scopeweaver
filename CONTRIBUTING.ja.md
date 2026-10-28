[English](CONTRIBUTING.md) | [한국어](CONTRIBUTING.ko.md) | [简体中文](CONTRIBUTING.zh-Hans.md) | [繁體中文](CONTRIBUTING.zh-Hant.md) | [日本語](CONTRIBUTING.ja.md) | [Español](CONTRIBUTING.es.md)

<a id="contributing-to-scopeweaver"></a>
# ScopeWeaver への貢献


バグ修正、上流 ARTEX からのバックポート、依存関係の更新、ドキュメント、テストの貢献を歓迎します。

<a id="before-you-start"></a>
## 作業を始める前に

- [使用上の制限](README.ja.md#license-and-disclaimer)を読んでください。実際のオンラインシステムを対象にする機能の追加や、
  制限を弱める変更はできません。
- 大きな変更では、まず Issue を作成して進め方を合意してください。
- プルリクエストは 1 つの主題に絞ってください。小さな PR ほど早くレビューできます。

<a id="requirements-for-a-pull-request"></a>
## プルリクエストの要件

- **出荷するすべての言語と対応するドキュメントを完成させること。** ユーザー向けのテキストと、保守対象の利用者・貢献者向けガイドには、英語、韓国語、簡体字中国語（`zh-Hans`）、繁体字中国語（`zh-Hant`）、日本語（`ja`）、スペイン語（`es`）が必要です:
  - サーバーのメッセージ: `locale.Register(key, en, ko)` との互換性を保ち、`locale.RegisterAll` と `locale/` の言語別リソースで追加の出荷言語を含めます。
  - ウェブ UI: EN/KO テーブルと `messages/{zh-hans,zh-hant,ja,es}.ts` に、現在の英語のすべてのキーを用意します。
  - ドキュメント: `README.md`、`CONTRIBUTING.md`、保守対象のガイドに、対応する `ko`、`zh-Hans`、`zh-Hant`、`ja`、`es` 版を用意します。実行用の英語スキルのエントリーは保持します。
- **変更した動作のテスト。** その変更がなければ失敗するテストを追加・更新してください。
  合格させるために既存の検査を無効化・スキップ・緩和してはいけません。
- **使い捨てデータベースのみ。** PostgreSQL に書き込むテストでは、
  `scopeweaver_test_*`（エージェントアダプターの E2E テストは `scopeweaver_adapter_*`）という名前の新しいデータベースを使用してください。
- **上流の作業を明記する。** ARTEX のバックポートでは、`Upstream-commit: mhtsec/ARTEX@<sha>` と、
  元の作者の `Co-authored-by:` トレーラーを追加し、`docs/PROVENANCE.md` と `docs/ko/PROVENANCE.md` を更新してください。
- **ライセンス。** 貢献はリポジトリの [AGPL-3.0](LICENSE) ライセンスの下で受け入れます。

<a id="adding-or-changing-a-locale"></a>
## ロケールの追加・変更

[ロケールガイド](docs/ja/i18n.md)に従い、共有の `locale/definitions.json`（正規 ID、言語の自称、Intl の対応、別名）、対応するベクトル、完全なフロントエンド・バックエンドのカタログ、date-fns の対応、対象を絞ったテスト、ブラウザーのスモークテストを更新してください。`zh` 単独、`zh-CN`、`zh-SG` は `zh-Hans` に、`zh-TW`、`zh-HK`、`zh-MO` は `zh-Hant` に対応します。明示された文字体系は、矛盾する地域指定より優先します。キー、意味、プレースホルダー、リッチテキストのタグは英語を基準とします。バックエンドの実行時の英語・キーへのフォールバックは互換性の保護であり、言語を出荷できる証明ではありません。ユーザーの内容、下書き、生の証拠、識別子、過去のデータは翻訳しないでください。翻訳をレビューして懸念を記録し、証拠なしに母語話者の承認を主張しないでください。英語・韓国語の組も含め、保守する機能ガイドを 6 言語すべてで完全に維持してください。 単独で配布するアダプターの正規 ID 一覧と、ルートのシェル・バッチ用メッセージヘルパーも同期してください。`node e2e/script-locales.mjs` と、Windows 上の実動作を確認する `node e2e/windows-launcher.mjs`／Windows CI を実行します。配布パッケージのアダプターは、実行時にリポジトリルートの JSON に依存してはいけません。

<a id="checks-to-run"></a>
## 実行する検査

[.github/workflows/ci.yml](.github/workflows/ci.yml) の CI ジョブに対応します:

```bash
# patched Norma SDK (a separate Go module)
(cd third_party/norma && go test ./... -count=1)


# backend database-backed suites (db, server, agent): against a throwaway database
# that is created and dropped for you. They require ARTEX_TEST_DSN and skip (never
# fall back to the live DSN) without it — see docs/VERIFICATION.md.
make test-db

# backend: every Go package (set ARTEX_TEST_DSN to a disposable scopeweaver_test_* database)
ARTEX_TEST_DSN=postgres://sw_user_test@/scopeweaver_test_1?host=/tmp&sslmode=disable go test ./... -count=1

# frontend
cd web && npm ci && npm test && npm run build:static

# agent adapter (and its real-backend E2E test against a scopeweaver_adapter_* database)
cd adapters/agent && npm ci && npm test && npm run check
SCOPEWEAVER_ADAPTER_E2E=1 go test ./server -run '^TestAgentAdapterE2E$' -count=1
```

`e2e` ワークフローは、すべての PR とマージ後に、実際のバイナリに対する必須のブラウザースモークテストも実行します。実行中のサーバーと新しい `scopeweaver_test_*` データベースに対し、ローカルで実行する方法は次のとおりです（`psql` と、同じチェックアウトからビルドした UI 埋め込みバイナリが必要）:

```bash
cd e2e && npm ci && npx playwright install chromium
SCOPEWEAVER_URL=http://127.0.0.1:8787 ARTEX_PG_DSN=postgres://.../scopeweaver_test_e2e \
  SCOPEWEAVER_BINARY=/absolute/path/to/scopeweaver npm run smoke
```

既存のスモークテストがパスワードを初期化した後、ダッシュボードの回帰テストは、同じテスト DB に対して、そのバイナリの隔離したコピーを空いているループバックポートで開始します。英語・韓国語の従来方式・新方式のトークン積み上げ表示とプロファイルの絞り込みを確認し、サーバーを停止してテストデータを削除します。ダッシュボードの証拠と子サーバーのログは `e2e/e2e-results/dashboard-token-chart/` に保存します。

必須のスモークテストは、6 言語すべてで設定の復元、初期設定・ログイン・タスク・設定・チャットの下書き、最初の API の言語、SSE 再接続・絞り込みの保持、数値の表示形式を確認します。

PR に、実行したことと実行できなかったことを記載してください。

<a id="review-and-merging"></a>
## レビューとマージ

各 `feat/*` または `fix/*` ブランチは最新の `dev` から作成し、`dev` 宛てに PR を開いてください。統合後、その正確な `dev` コミットのすべての検査と E2E を待ち、このリポジトリの `dev` から `main` への昇格 PR を開きます。`dev` が `main` の祖先であり続けるよう、昇格履歴はマージコミットで保持してください。昇格後も検査を確認し、次の作業は `dev` から開始します。`branch-policy` は他の PR 経路を拒否します。

1. 現在の候補で、必須のすべての検査に合格する必要があります: `branch-policy`、`frontend`、
   `agent-adapter`、`backend`、`windows`、`chat-browser`、`smoke`。`dev` と `main` はどちらもマージ後に CI と E2E を実行します。初回の貢献者については、メンテナーが CI 実行を承認します。
2. PR の差分について、正しさ、安全性、出荷する翻訳の完全性、6 言語のドキュメントの対応をレビューします。
3. **すべての PR は、マージ前にメンテナーが最終の人間による確認を行います。**
   この確認なしではマージしません。

リリースは、マージの前後に完全な E2E テストを行った後、メンテナーが作成します。
