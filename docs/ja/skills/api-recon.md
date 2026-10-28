[English](../../../skills/api-recon/SKILL.md) | [한국어](../../ko/skills/api-recon.md) | [简体中文](../../zh-Hans/skills/api-recon.md) | [繁體中文](../../zh-Hant/skills/api-recon.md) | [日本語](api-recon.md) | [Español](../../es/skills/api-recon.md)

---
name: api-recon
description: 許可されたスコープ内で、ウェブサイトの API インターフェースを一覧化する場合に使用します。
---

<a id="api-recon-frontend-interface-discovery"></a>
# API Recon（フロントエンドのインターフェース検出）



許可を受け、バックエンド API（パス、メソッド、パラメーター、レスポンス本文）、フロントエンドのルート、UI のトリガー（タブ、ダイアログ、テーブル操作）を、可能な限り完全に検出します。

---

<a id="boundaries-and-prohibitions-required-reading"></a>
## 境界と禁止事項（必読）

このスキルは API とパラメーターを一覧化するだけです。脆弱性の発見や悪用は行いません。

<a id="task-boundaries"></a>
### タスクの境界

| 範囲 | 許可されること | 禁止 |
|---|---|---|
| 目的 | パス、メソッド、パラメーター、ルート、UI のトリガーの列挙 | SQLi/XSS、アクセス制御への攻撃、総当たり、脆弱性のファジング、改ざんしたリクエストによる攻撃、破壊的操作 |
| 認証 | クライアント側のログインガードをフックし、スタブ・モックで置換 | 認証情報の要求・推測、実際のログインフォームの送信 |
| 実行時 | 認証情報なしでフックし、モックでログイン後の SPA シェルを描画 | 実際のバックエンドセッションを必要とする処理 |

<a id="dynamic-analysis-without-credentials-phase-3-default"></a>
### 認証情報なしの動的解析（フェーズ 3 の既定）

1. `preload.js` / `runtime_harvest.js` で、ログイン、権限、メニュー、その他の初期化 API をインターセプトしてスタブ化します。
2. 正しい形と成功コードのモック応答本文を返します。データは空でも構いません。
3. バックエンドがない、または 401 が返る状態でもログイン後のページを描画し、追加の XHR/fetch/WebSocket リクエストを起こします。
4. 空のデータ、空のテーブル、仮の UI は想定内です。実際のログインや脆弱性テストに切り替えないでください。

モックでフロントエンドのルート・コンポーネントをマウントし、送信リクエストだけを記録します。目的はフロントエンドが他に何を送るかの検出であり、実際のバックエンドデータの取得ではありません。

<a id="workflow-prohibitions"></a>
### 作業手順の禁止事項

| 禁止 | 必須の代替手段 |
|---|---|
| フェーズ 1 の完了前に、メインの `index-*.js` から grep/curl/Read で API パスを抽出 | `OUTDIR/harvest_static.py` を実行 |
| `extract_apis.py` などの手書きの代替 | `OUTDIR/harvest_static.py` を調整して再実行 |
| 2 回以上の失敗後も同じ grep・コマンドを反復 | 戦略を変更: tool_logs を読み、harvest を調整し、参照資料を確認 |
| ゲート A/B を飛ばして元の `scripts/` テンプレートを実行 | OUTDIR にコピーして対象に合わせて調整 |
| 実際の認証情報、OTP、OAuth による認証 | 上記のスタブ・モックを使用 |
| スタブを省略し、アクセス制御・インジェクションのテストで実データを取得 | 偵察の範囲で送信内容のみを記録 |
| 削除、機密データの取り出し、一括書き込み、その他の不可逆な操作 | coverage のクリックにも適用 |
| 実行時の解析と動的列挙なしに、全ページ・API の確認を主張 | 完了条件を満たすか、制限を明示 |
| トリガー行列と差分なしに、全パラメーターの確認を主張 | フェーズ 3b の行列 + フェーズ 5 の差分 |
| 実行時サンプル 1 件から必須・任意を推論 | 複数サンプルを比較、または検証ルール・エラーを確認 |

---

<a id="two-layers-and-runtime-modes"></a>
## 2 つの層と実行時モード

| 層 | 出力 | 制限 |
|---|---|---|
| 静的解析（JS バンドル）| エンドポイントのパス、ルートの候補、ペイロード構築箇所のフィールド候補 | HTTP メソッドは不明。パラメーターはフェーズ 1b が必要。実行時に組み立てる URL は検出できない |
| 実行時解析（動作中のセッション）| メソッド、本文、レスポンス、動的 URL、WS/SSE。サンプル差分でパラメーターの不足を補完 | リクエストの発生にはページの描画が必要。1 サンプルだけで必須・任意は確定できない |

| モード | エンジン | 用途 |
|---|---|---|
| **depth** | `runtime_harvest.js` (Puppeteer) | API 一覧、メソッド・パラメーター・レスポンス、WS/SSE、繰り返せる一括実行 |
| **coverage** | ブラウザー + `preload.js` | タブ・ダイアログ・テーブルをクリックし、機能の確認範囲を深める |
| **both** | depth の後に coverage | 最も完全だが、最も時間がかかる |

パラメーターの解析方法（万能なスクリプトはありません）: パスには harvest・正規表現、パラメーターにはパス周辺の展開、UI の紐付けの連鎖、サンプル差分、エラーからの推論を使います。grep の例は [reference.md](api-recon-reference.md) の J 節を参照してください。

---

<a id="definition-of-completion"></a>
## 完了の定義

すべての条件を満たす場合のみ、完了を主張してください:

- [ ] 静的解析: フェーズ 1 で `api_static.txt`、`routes.txt`、`js/` を生成
- [ ] 実行時解析: 少なくとも depth または coverage。coverage/both では動作するフックと動的列挙ループが必要
- [ ] シェル: 業務パスが `/login` に戻らない（ハッシュルーティングも確認）
- [ ] パラメーター: coverage/both はトリガー行列と `param_samples.json` を完成させ、フェーズ 5 で `params_merged.json` を統合
- [ ] depth でモジュールが空の場合: フェーズ 4 で権限ツリーを復元し、locale/bootstrap 以外のモジュール API が現れるまで再実行
- [ ] 納品: フェーズ 5 の全出力。`insert_assets` に service と endpoint の資産を保存

---

<a id="scripts-and-gates"></a>
## スクリプトとゲート

`scripts/` は参照用テンプレートです。未調整の原本を実行し、最終版として扱ってはいけません。

読んで対象に合わせて調整し、`OUTDIR`（例: `recon/`）に書き出して `CHANGES.md` に記録します。不適切なら、方法に従って書き直し、構造だけを参考にしてください。

| ゲート | 時期 | テンプレート → OUTDIR のコピー | 主な変更 |
|---|---|---|---|
| A（静的解析）| フェーズ 0 後、最初の harvest/spider の実行前 | `harvest_static.py` / `spider_mpa.py` | 既定の正規表現は多くのサイトで動作。マニフェスト・構文の不一致がある場合のみ、エンドポイントの正規表現、webpack/Vite の `publicPath`、MPA の除外・Cookie を変更 |
| B（実行時解析）| フェーズ 2 後、depth/coverage の前 | `runtime_harvest.js` / `preload.js` + `config.json` | Cookie/localStorage のキー、成功値の補正、スタブ、ログインの正規表現、API の接頭辞、hash/history |

SPA の必須順序: スクリプト作成前の探索的な閲覧より、フェーズ番号を優先します。順序を変えないでください:

| 手順 | 必須 | 禁止 |
|---|---|---|
| フェーズ 0 後 | 次の Bash コマンド = `python3 OUTDIR/harvest_static.py <URL> OUTDIR` | メインの `index-*.js` を curl/grep/Read（通常 >500KB）|
| ゲート A | スクリプトをコピーし、必要な小変更を加え、即座に実行 | harvest の実行を決める前に、手作業で API を抽出 |
| フェーズ 1 の完了前 | `wc -l` で出力を確認。404 では harvest を修正して再試行 | 抽出スクリプトを作成、未ダウンロードの URL を繰り返し grep |
| フェーズ 1b 以降 | `OUTDIR/js/*.js` のみを grep | harvest の代わりにメインのバンドルを使う |

- 正しい例: `harvest_static.py` をコピーし、必要なら正規表現を調整して、即座に実行
- 誤った例: メインバンドルの curl、grep の反復、一時的な抽出器、最後に harvest
- MPA: フェーズ 0 後の次の Bash コマンド = `python3 OUTDIR/spider_mpa.py ...`

---

<a id="tool-and-output-limits"></a>
## ツールと出力の制限

| 制約 | 規則 |
|---|---|
| 大きなファイル | >100KB の `index-*.js` を Read/grep してコンテキストに入れない。OUTDIR のスクリプトで一括処理 |
| grep の出力 | `\| head -20` または `-m 5` が必須。バンドルの抜粋でなくパスの要約を保持 |
| 検証 | `wc -l`、`ls \| wc -l` を使用。ディレクトリ全体を Read しない |
| 正規表現のプレビュー | 任意で最大 1 回、≤50KB の chunk/HTML のみ。静的解析の正は harvest |
| 参照資料 | 手法・テンプレート・問題解決は [reference.md](api-recon-reference.md)。本文に繰り返さない |

---

<a id="execution-roadmap"></a>
## 実行の道筋

```
Phase 0 classification + OUTDIR
  → Gate A → Phase 1 harvest (run immediately)
  → Phase 1b parameter reconstruction
  → Phase 2 three authentication gates → config.json
  → Gate B → Phase 3 runtime + parameter matrix
  → Phase 4 permission tree if needed → rerun Phase 3
  → Phase 5 merge report + insert_assets for every discovered service/endpoint asset; omit none
```

順に確認し、前の手順が完了するまで先に進まないでください。

1. [ ] **フェーズ 0**: SPA/MPA を分類し、`OUTDIR` を作成 → [フェーズ 0](#phase-0-classification)
2. [ ] **ゲート A + フェーズ 1**: スクリプトをコピーし、即座に harvest、`wc -l` で検証 → [フェーズ 1](#phase-1-static-discovery)
3. [ ] **フェーズ 1b**: パス周辺の展開 + 紐付け層 → `param_candidates.json` → [フェーズ 1b](#phase-1b-parameter-reconstruction)
4. [ ] **フェーズ 2**: 認証の 3 ゲート → `config.json` → [フェーズ 2](#phase-2-three-authentication-gates)
5. [ ] **ゲート B**: 実行時スクリプトを調整 → [フェーズ 3](#phase-3-runtime)
6. [ ] **フェーズ 3**: depth / coverage / both。シェルへの進入を検証し、パラメーターのトリガー行列 → `param_samples.json`
7. [ ] 必要なら**フェーズ 4**: 権限ツリー → スタブ修正 → フェーズ 3 を再実行 → [フェーズ 4](#phase-4-permission-tree-reconstruction)
8. [ ] **フェーズ 5**: 出力の統合 + レポート + `insert_assets` → [フェーズ 5](#phase-5-merge-and-report)

---

<a id="phase-0-classification"></a>
## フェーズ 0: 分類

入口の HTML を取得し、`OUTDIR` を作成します。スキルの `scripts/` は変更しないでください:

- **SPA**: 空のシェル + `<div id=app>` + chunk → フェーズ 1～5
- **MPA**: SSR + `<form>`、エンドポイントのバンドルなし。ゲート A 後:

```bash
python3 recon/spider_mpa.py <BASE_URL> <OUTDIR> [--cookie "session=..."] [--max 300] [--depth 5] [--exclude "logout|delete|destroy"]
```

出力: `forms.txt`、`links.txt`、`api_inline.txt`。フォームがほぼない SPA は、フェーズ 1 に切り替えます。

---

<a id="phase-1-static-discovery"></a>
## フェーズ 1: 静的な検出

[スクリプトとゲート](#scripts-and-gates)、[ツールと出力の制限](#tool-and-output-limits)に従ってください。

```bash
python3 recon/harvest_static.py <BASE_URL> <OUTDIR>
```

harvest は HTML のスクリプトと webpack/Vite のマニフェストを解析し、遅延 chunk をすべてダウンロードして、`js/`、`api_static.txt`、`routes.txt`、`chunkmap.txt` を出力します。

```bash
wc -l OUTDIR/api_static.txt OUTDIR/routes.txt
ls OUTDIR/js | wc -l
```

- chunk 数をマニフェストと比較し、404 では harvest を修正して再試行。1 件ずつ curl しない
- `api_static.txt` の項目が少なすぎる場合: OUTDIR のエンドポイントの正規表現を広げて再実行（参照資料を確認）

<a id="phase-1b-parameter-reconstruction"></a>
### フェーズ 1b: パラメーターの再構成

パスはフェーズ 1 から取得し、パラメーターフィールドは別に調べます。[grep 出力の制限](#tool-and-output-limits)に従ってください。

完了条件: 重要な API について、フィールド名、送信上の位置、推定した型、必須状態、値の例、確信度を特定する。

<a id="1b0-transport-forms"></a>
#### 1b.0: 送信形式

| 形式 | パラメーターの位置 | 静的な証拠 |
|---|---|---|
| REST JSON | body + query | パスの近傍の `(params\|data\|body)\s*:\s*\{` |
| GraphQL | `variables` | gql テンプレート、`$page: Int` |
| 従来のフォーム | urlencoded | `<form>`, `FormData` |
| ファイルのアップロード | multipart | `FormData.append` |
| パスパラメーター | `/user/:id` | ルートテーブル + `useParams` / `$route.params` |
| 暗号化・署名 | `sign`/`data` で包む | 暗号化関数の入力をフック（参照資料 D）|

出力: 各 API に `transport: query|json|form|graphql|encrypted` を付けます。

<a id="1b1-expand-anchor-windows"></a>
#### 1b.1: パス周辺を展開

既知のパスの周囲を広げ、ペイロードを構築する箇所を探します:

```bash
grep -n '"/api/user/list"' OUTDIR/js/*.js | head -20
grep -rhoaE '.{0,120}("/api[^"]+").{0,200}' OUTDIR/js/*.js | head -20
grep -rhoaE '(params|data|body|payload)\s*:\s*\{' OUTDIR/js/*.js | head -20
```

| ラッパー | パラメーターの手掛かり |
|---|---|
| axios インスタンス | `data` / `params` |
| 共通のリクエストラッパー | インターセプターが注入する全体フィールド |
| OpenAPI クライアント | 生成されたメソッドのシグネチャー |
| React Query / SWR | フックの 2 番目の引数 |
| Vue composable | composable の引数 |

型の手掛かり: `yup`/`zod`/rules、`Form.Item name=`、埋め込まれた Swagger。

→ `param_candidates.json`: `{ path, fields[], source: "static-callsite", confidence }`

<a id="1b2-binding-layer"></a>
#### 1b.2: 紐付け層

```
Form field → onFinish/handleSubmit → transform → API payload
```

| 紐付け元 | 方法 |
|---|---|
| フォームの送信 | submit → transform → API を追跡 |
| テーブルの検索 | `getFieldsValue()` → `params` |
| ルート | `:id` / `?tab=` |
| インターセプター | 全体の `tenantId`、ページ送り、署名 |
| 列挙値の選択 | `options` → API の列挙値 |

DevTools の呼び出しスタックで、`fetch`/`XHR.send` から上へ、ペイロードの構築箇所を追ってください。

<a id="1b3-three-payload-questions-distinct-from-phase-2-auth-gates"></a>
#### 1b.3: ペイロードの 3 つの問い（フェーズ 2 の認証ゲートとは別）

| 問い | 必須の答え |
|---|---|
| 組み立て | ペイロードを構築・変換する場所 |
| 検証 | required、pattern、enum |
| 送信 | path / query / body / multipart / headers |

インターセプターのゲート（フェーズ 2）では、全体に注入するフィールド（Authorization、`X-Tenant-Id`、署名）も確認します。

<a id="1b4-connect-to-phase-3"></a>
#### 1b.4: フェーズ 3 への接続

候補は静的解析・紐付け解析から得ます。必須、任意、条件付きの依存関係は、フェーズ 3 の行列・差分とフェーズ 5 のエラー推論が必要です。

---

<a id="phase-2-three-authentication-gates"></a>
## フェーズ 2: 認証の 3 ゲート

`OUTDIR/js/` を grep して `head` で制限し、判明内容を `config.json` に記録します（参照資料の手法を参照）:

| ゲート | 問い | キーワード |
|---|---|---|
| 描画 | ログイン状態をどう判定するか？| `isLogin`、`getToken`、Cookie/localStorage |
| インターセプター | 何が `/login` への転送を起こすか？| `response_code`、`errno`、axios のインターセプター |
| 内容 | メニュー・権限はどこから来るか？| `menu`, `permission`, `role`, `acl`, `routes` |

localStorage のキー名を認証情報として扱わず、chunk・リクエストの連鎖で確認してください。

出口 = ゲート B: 結論を `config.json` に書き、`OUTDIR/runtime_harvest.js` / `preload.js` を調整します。

<a id="phase-2b-api-observation-optional"></a>
### フェーズ 2b: API の観測（任意）

OUTDIR の `preload.js` で、セッションのキー、Authorization、ネストした API URL を特定します:

| 設定 | 出力 |
|---|---|
| `recordDetail: true` | `__API_RECON_DETAIL__` |
| `observe.xhrHeaders: true` | ヘッダーの観測 |
| `extractUrlsFromResponse: true` | レスポンス内にネストした API |
| `observe.storageReads/cookieReads: true` | 設定を更新 |
| `neutralizeVueRouter: true` | `__API_RECON_ROUTES__` |

coverage の各回を出力します: `__API_RECON_LOG__`、`__API_RECON_DETAIL__`、`__API_RECON_ROUTES__`、`__API_RECON_OBSERVE__`。

---

<a id="phase-3-runtime"></a>
## フェーズ 3: 実行時解析

先にゲート B に合格してください。[境界](#boundaries-and-prohibitions-required-reading)と認証情報不要のモックの方針に従います。

`config.json` に `"runtimeMode": "depth" | "coverage" | "both"` を設定します（テンプレートは参照資料）。

<a id="hooks-and-stubs-shared-by-depth-and-coverage"></a>
### フックとスタブ（depth と coverage で共有）

| 層 | 範囲 | 目的 |
|---|---|---|
| L1 完全一致 | auth/permission/bootstrap のスタブ | 初期描画の認証を通過 |
| L2 否定結果の補正 | すべての JSON レスポンス | 未ログインのコード → 成功 |
| L3 フォールバック | L1 に一致しない `/api` など | 空の成功本文で UI を描画 |

- **depth**: 仮の認証 + `forward` による業務コードの補正 + `stubs`。`routes`（hash/history）を巡回し、`runtime_api.json` を出力
- **coverage**: document の開始時に `preload.js` を注入（CDP の `addScriptToEvaluateOnNewDocument` またはユーザースクリプト）

`window.__API_RECON_PRELOAD__` が存在し、業務パスが `/login` に戻らないことを確認してください。

```bash
cd recon && npm install
node runtime_harvest.js config.json
```

<a id="3b-coverage-dynamic-enumeration-required"></a>
### 3b: coverage の動的列挙（必須）

1. 主なナビゲーション・サイドバー: 各項目をクリックし、ネットワークを 1～3 秒待つ
2. タブ: `role=tab`、`.ant-tabs-tab`
3. テーブル: 先頭行の表示・編集・詳細
4. ツールバー: エクスポート、絞り込み、作成（不可逆の削除は避け、すべての禁止事項を引き続き守る）
5. 各モジュール: API・ルートを統合
6. SPA: `routes.txt` の未確認パスに対する制御された `pushState`（MPA では禁止）

必須のパラメータートリガー行列: モジュールごとに操作の種類を記録し、複数のサンプルを比較します:

| 操作 | 典型的な追加パラメーター |
|---|---|
| 初期一覧 | ページ送り + 既定の絞り込み |
| 検索 | keyword、filter |
| 詳細な絞り込み | さらに多くの任意フィールド |
| 作成・編集 | 完全なエンティティ |
| 一括・エクスポート・並べ替え | `ids[]`, `exportType`, `sortField` |

スタブ使用時も送信本文・ヘッダーは実際のものです。リクエストを証拠に使い、`scan_raw.json`、`param_samples.json`、`api_detail.json` を記録してください。

- **Vue**: `neutralizeVueRouter: true` + document 開始時の preload
- **React**: `routes.txt` + サイドバーのクリック + `pushState`
- **both**: 先に 3a の depth、その後 3b の coverage

---

<a id="phase-4-permission-tree-reconstruction"></a>
## フェーズ 4: 権限ツリーの再構成

開始条件: モジュールのページが空、または全ルートで初期化（locale など）しか発生しない場合は、内容のゲートを通過できていません。

| 症状 | 意味 |
|---|---|
| シェルに進入済み | 描画とインターセプターのゲートに合格 |
| サイドバーの項目がない・クリック結果が空 | スタブの形や権限コードが不完全 |
| 各ルートで少数の同じ API だけが発生 | `v-if permission` が失敗 |
| `routes.txt` がバンドル内のルートより大幅に少ない | 認証モジュールから補完 |

```bash
grep -rhoaE '"/api[^"]*(permission|perm|role|menu|acl)[^"]*"' OUTDIR/js/*.js | sort -u | head -30
grep -rhoaE 'userRouteAuth|getResultTree|routeMap|routeLink|menuList|authList' OUTDIR/js/*.js | head -20
```

典型的な連鎖: `role_permissions`（平坦なコード）+ `permissions/all`（ツリー）→ `getResultTree` → `userRouteAuth[CODE].url`。

```bash
python3 recon/extract_route_map.py recon/js recon/
python3 recon/build_perm_tree.py recon/js recon/ --config recon/config.json
```

中間出力: `route_map.json`、`userRouteAuth.json`、`permissions_tree.json`、`*_stub.json`、`perm_codes_all.txt`。

スタブを確認します。外側の `response_code` がインターセプターのゲートに一致し、平坦なコードがツリーと一致し、`routes` がすべての `route_map` のリンクを含む必要があります。

`config.json` の更新後、フェーズ 3 を再実行します。大きな SPA では `waitUntil`、`routeTimeout`、`perRouteMs` を調整できます（参照資料 A3/I）。

---

<a id="phase-5-merge-and-report"></a>
## フェーズ 5: 統合とレポート

<a id="outputs"></a>
### 出力

| ファイル | フェーズ | 内容 |
|---|---|---|
| `js/`, `api_static.txt`, `routes.txt`, `chunkmap.txt` | 1 | 静的なバンドルとパス |
| `param_candidates.json` | 1b | 静的なパラメーター候補 |
| `config.json` | 2 | 3 ゲート + 実行時設定 |
| `runtime_api.json` | 3a | WS/SSE を含む詳細な depth の記録 |
| `param_samples.json`, `scan_raw.json`, `api_detail.json` | 3b | 複数のサンプル、クリックログ、詳細 |
| `route_map.json` と関連ファイル | 4 | 使用した場合の権限ツリーの中間ファイル |
| `params_merged.json` | 5 | 統合したパラメーターフィールド + 確信度 |
| `api_merged.txt` | 5 | `METHOD /path [params] [static\|runtime\|both]` |
| `site_map.json` | 5 | ルート、API、パラメーター、機能、制限 |
| **insert_assets** | 5 | すべての service・endpoint 資産を保存 |

<a id="5b-merge-parameters"></a>
### 5b: パラメーターの統合

`param_samples.json` の差分を確認します。万能な統合スクリプトはありません。確信度の規則は参照資料 J7（高・中・低・トリガー待ち）にあります。

<a id="5c-infer-from-errors"></a>
### 5c: エラーからの推論

許可内で、不完全なリクエストの 400 エラーを、脆弱性テストではなくパラメーター検出のために確認できます。例: `field 'x' is required`、列挙値のエラー。`data` ラッパー、`variables`、暗号化前の `bizData` を考慮してください。

runtimeMode、静的・実行時 API 数、パラメーターの確信度、未対応のモジュール、テンプレートに対する `CHANGES.md` の要約を報告してください。

推奨する `site_map.json`:

```json
{
  "site": "https://example.com",
  "runtimeMode": "both",
  "appType": "vue-spa",
  "routeGuardStrategy": ["nav-neutralize", "L1-auth", "L2-patch", "forward"],
  "apisFromStatic": [],
  "apisFromRuntime": [],
  "apis": [],
  "params": [{ "method": "POST", "path": "/api/user/list", "transport": "json", "fields": [] }],
  "frontendRoutes": [],
  "routesVerifiedByClick": [],
  "featuresTriggered": [],
  "limitations": ""
}
```

追加のフィールドと grep の手法: [reference.md](api-recon-reference.md)。

---

<a id="general-notes"></a>
## 全般的な注意

- フレームワークに依存しない: webpack/Vite/Angular の遅延読み込みに同じ方法を使う
- 送信方式: REST/JSON、GraphQL、WebSocket、SSE。gRPC-web は対象外
- SSR: クライアントの fetch は記録可能。RSC/Server Actions は完全には列挙できない
- 死角: JSVMP、WASM、厳密な HMAC/mTLS → 静的解析と制限の明記
- パラメーターの死角: 条件付き操作、隠れたパラメーター、WASM の構築 → トリガー待ち・到達不能
- 静的解析へのフォールバック: 実行時解析が妨げられてもエンドポイントは検出可能

---

<a id="additional-resources"></a>
## 追加の資料

- grep の手法、`config.json` のテンプレート、問題解決、フック、パラメーターの再構成（J）、site_map のテンプレート: **[reference.md](api-recon-reference.md)**
- テンプレートのパス: [スクリプトとゲート](#scripts-and-gates)
