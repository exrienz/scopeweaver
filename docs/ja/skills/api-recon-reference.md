[English](../../../skills/api-recon/reference.md) | [한국어](../../ko/skills/api-recon-reference.md) | [简体中文](../../zh-Hans/skills/api-recon-reference.md) | [繁體中文](../../zh-Hant/skills/api-recon-reference.md) | [日本語](api-recon-reference.md) | [Español](../../es/skills/api-recon-reference.md)

<a id="api-recon-reference"></a>
# api-recon の参照資料



grep の手法、`config.json` のテンプレート、問題の解決方法です。grep は `js/` に対して実行してください。1 行のバンドルでは、任意で `js-beautify` または `sed 's/}/}\n/g'` を使えますが、通常は周辺範囲付きの生の grep で十分です。

<a id="script-notes"></a>
## スクリプトの注意

`scripts/` の全ファイルは参照用テンプレートです。実行前に対象に合わせて調整してください。主な変更:

| スクリプト | 主な調整 |
|---|---|
| `harvest_static.py` | エンドポイントの正規表現、webpack/Vite のマニフェスト、マイクロフロントエンドの publicPath、再試行・同時実行数 |
| `runtime_harvest.js` | neutralize のフィールド名・成功値、スタブの照合・本文の形、ルートの取得元、WS の記録、`waitUntil`/`routeTimeout`/`proxy` |
| `preload.js` | `loginPathRe`、L1 スタブ、`neutralize.fields`、`apiPattern`、L3 の有効化、`recordDetail`、`observe.*`、`neutralizeVueRouter` |
| `spider_mpa.py` | `--exclude` による破壊的リンクの除外、Cookie、depth/max、同一オリジンの絞り込み |
| `extract_route_map.py` | `routeMap` / `routeLink` の正規表現、KEY の命名パターン |
| `build_perm_tree.py` | `userRouteAuth` の解析、`ROOTS`/`PREFIX_PARENT` の階層の推定、外側のスタブのフィールド名 |
| `config.json` | 上記のサイト固有のパラメーターすべての集約設定 |

調整したファイルは `recon/` などのタスクのディレクトリに保持し、テンプレートとの差を正確に報告してください。

---

<a id="a-reconstruct-the-three-gates"></a>
## A. 3 つのゲートの再構成

<a id="a1-rendering-how-is-login-state-determined"></a>
### A1. 描画: ログイン状態をどう判定するか？

```bash
grep -rhoaE '.{0,40}(isLogin|isAuthenticated|loggedIn|hasLogin|requireAuth)\b.{0,80}' js | head
grep -rhoaE 'function (getUser|getToken|getAuth)[0-9]?\([^)]*\)\{.{0,200}' js | head
grep -rhoaE '(localStorage|sessionStorage)\.getItem\("[^"]+"\)' js | sort -u
grep -rhoaE '(Cookies?|cookie)\.(get|load)\("[^"]+"\)' js | sort -u
grep -rhoaE '\batob\(|JSON\.parse\(|jwt|decode' js | head
```

`isLogin = f(getUser())` → `getUser = decode(storage.read(KEY))` を追い、保存キー、場所（Cookie/localStorage）、エンコードを特定します:

| エンコード | config のモック値 |
|---|---|
| 通常の文字列 / `"1"` / トークン | `"value": "anything-truthy"` |
| `JSON.parse(x)` | `"value": "json:{\"id\":1,\"username\":\"admin\"}"` |
| `JSON.parse(atob(x))` | `"value": "b64json:{\"id\":1,\"username\":\"admin\"}"` |
| JWT | クライアント側のモックガードのみに、未署名・`alg:none` の JWT、またはバンドル内のキーによる署名を使用 |
| 暗号化（SM2/AES/RSA）| ハードコードされたキーを探す。描画ガードが復号可能な Blob だけを必要とする場合のみモックし、それ以外は静的解析に戻る |

`cookies` / `localStorage` に書き込みます。

<a id="a2-interceptor-what-triggers-login-redirects"></a>
### A2. インターセプター: 何が /login への転送を起こすか？

```bash
grep -rhoaE '.{0,60}(interceptors\.response|axios|request\.use).{0,120}' js | head
grep -rhoaE '.{0,40}(response_code|errcode|errno|\bcode\b|\bret\b|\bstatus\b)\s*[=!]==?\s*[\-0-9]{1,4}.{0,60}' js | head -20
grep -rhoaE '.{0,40}(未登录|请重新登录|登录已过期|unauthorized|登录失效|授权|token.{0,10}invalid).{0,40}' js | head
grep -rhoaE '.{0,30}(location\.href|router\.(push|replace)|navigate)\([^)]*login[^)]*\)' js | head
```

フィールド名、成功値（通常 `0` または `200`）、転送を起こす失敗値を特定します。仮のセッションで検証してください:

```bash
curl -sk -X POST -H 'Cookie: <fakekey>=junk' https://target/api/<protected> -d '{}' -H 'Content-Type: application/json'
```

`neutralize.fields` + `neutralize.success` に書き込みます。

<a id="a3-content-where-do-menus-and-permissions-come-from"></a>
### A3. 内容: メニューや権限はどこから来るか？

```bash
grep -rhoaE '"/api[^"]*(permission|perm|role|menu|acl|resource|nav)[^"]*"' js | sort -u
grep -rhoaE '.{0,30}(menus|permissions|menuList|routeList|authList|role_permissions)\b.{0,120}' js | head
grep -rhoaE 'userRouteAuth|getResultTree|routeMap|routeLink|hasPermission|checkAuth' js | head
grep -rhoaE '([A-Z_][A-Z0-9_]*):\{name:"[^"]*",link:"/[^"]+"\}' js | head
```

業務用の管理画面によくある 2 つのデータ層:

| API | 典型的なペイロード | 利用側 |
|---|---|---|
| `.../role_permissions` | `{ permissions: string[], role_type }` | ルートガード、ボタン単位の ACL |
| `.../permissions/all` | `tree[{ code, position, children }]` | サイドバーのメニュー描画 |
| バンドル内: `userRouteAuth` | `{ CODE: { url, name? } }` | code → フロントエンドのパス |
| バンドル内: `routeMap` | `{ KEY: { name, link } }` | 別名の解決（webpack の `o.DASHBOARD`）|

利用側のコードを読み、`getResultTree(tree, permissions)` の絞り込みと、`v-if` / `hasAuth(code)` が確認するフィールドを特定してください。

小さなサイトでは、許可するモックのペイロードを `stubs` に手動で作成します。

サイドバー・サブモジュールが空の大きなサイトでは、I 節で完全な権限ツリーを再構成してください。

---

<a id="b-configjson-template"></a>
## B. config.json のテンプレート

```json
{
  "baseUrl": "https://target/",
  "runtimeMode": "both",
  "chromium": "/usr/bin/chromium",

  "cookies": [
    { "name": "auth", "value": "b64json:{\"id\":1,\"username\":\"admin\",\"role\":\"admin\",\"func\":{},\"permissions\":[\"*\"]}" }
  ],
  "localStorage": { "token": "faketoken", "isLogin": "1" },

  "neutralize": {
    "fields": ["response_code", "code", "errno", "ret", "status"],
    "success": 0,
    "flags": { "success": true, "message": "ok" }
  },
  "forward": true,
  "loginUrlPattern": "/login",
  "apiPattern": "/api/|/rest/|/graphql",

  "mockTier": "L1+L2",
  "recordDetail": true,
  "observe": {
    "storageReads": false,
    "cookieReads": false,
    "xhrHeaders": true
  },
  "neutralizeVueRouter": true,
  "stubs": [
    {
      "match": "permissions/all|/menu|role_permissions",
      "body": {
        "response_code": 0, "code": 0,
        "data": {
          "permissions": ["*"],
          "menus": [
            { "name": "dashboard", "path": "/dashboard", "show": true, "children": [] },
            { "name": "alert", "path": "/alert", "show": true, "children": [] }
          ]
        }
      }
    }
  ],

  "explore": {
    "clickTabs": true,
    "clickTables": true,
    "pushStateFallback": true,
    "maxMenuItems": 50
  },

  "routes": ["/dashboard", "/alert", "/asset", "/device", "/report", "/config", "/system"],
  "waitMs": 1500, "perRouteMs": 900, "headless": true,
  "waitUntil": "domcontentloaded",
  "routeTimeout": 12000,
  "proxy": "",

  "captureResponses": true, "recordWs": true, "respMax": 600
}
```

フィールド:
- `runtimeMode`: `depth`（Puppeteer）、`coverage`（ブラウザー MCP）、`both`
- `cookies[].value` の接頭辞: `b64json:` → base64(JSON)、`json:` → 生の JSON、なし → 文字列をそのまま使用
- `forward: true` は実際のリクエストを転送し、ステータスコードのフィールドを書き換えます。`false` はオフラインのスタブを使います。
- `mockTier`: coverage の preload 層。例: `L1+L2`、`L1+L2+L3`
- `routes` は `routes.txt` から取得。メニューのモック後、ハーネスが `<a href>` のルートを追記
- `captureResponses` / `recordWs` は depth のみに適用
- `waitUntil`: 大きな SPA は `domcontentloaded` を使い、`networkidle2` の停止を回避
- `routeTimeout`: ルートごとの `page.goto` のタイムアウト（ミリ秒）
- `proxy`: Puppeteer の `--proxy-server`。代わりに `HTTP_PROXY` / `HTTPS_PROXY` も使用可能

<a id="b1-two-stub-template-role_permissions--permissionsall"></a>
### B1. 2 スタブのテンプレート（role_permissions + permissions/all）

```json
"stubs": [
  {
    "match": "role_permissions",
    "body": {
      "response_code": 0,
      "data": {
        "permissions": ["MONITOR", "MONITOR_ALERT", "THREAT", "ASSETS_RISK"],
        "role_type": "SUPER_ADMIN"
      }
    }
  },
  {
    "match": "permissions/all",
    "body": {
      "response_code": 0,
      "data": [
        {
          "code": "MONITOR",
          "position": 1,
          "children": [
            { "code": "MONITOR_ALERT", "position": 1, "children": [] }
          ]
        }
      ]
    }
  }
]
```

外側のフィールド（`response_code` / `code` / `data`）はゲート A2 に一致し、`permissions` はツリーのすべての末端コードを含める必要があります。

---

<a id="c-coverage-preload-configuration"></a>
## C. coverage の preload 設定

コピーした `preload.js` 冒頭の `CONFIG` を編集するか、CDP での注入前に置き換えてください:

```javascript
const CONFIG = {
  loginPathRe: /\/(login|signin)(\/|$|\?)/i,
  mockTier: 'L1+L2',
  forward: true,
  recordDetail: true,
  extractUrlsFromResponse: true,
  neutralizeVueRouter: true,
  observe: { storageReads: false, cookieReads: false, xhrHeaders: true },
  neutralize: { fields: ['response_code', 'code'], success: 0 },
  stubs: [ /* Same as config.json stubs */ ],
  apiPattern: /\/(api|apis|v\d+|dev|internal|graphql)\//i,
};
```

`window.__API_RECON_PRELOAD__ === true` と、パス名が安定していることを検証してください。

記録した結果を出力します:

```javascript
JSON.stringify({
  apis: [...window.__API_RECON_LOG__],
  detail: window.__API_RECON_DETAIL__,
  routes: [...(window.__API_RECON_ROUTES__ || [])],
  observe: window.__API_RECON_OBSERVE__,
}, null, 2)
```

---

<a id="d-preloadruntime-hook-coverage"></a>
## D. preload/runtime のフックの対象範囲

preload（coverage）と runtime_harvest（depth）に組み込まれたブラウザーフック:

| フック | 検出できるもの | 対応範囲 |
|---|---|---|
| fetch / XHR.open のフック | URL・メソッドを記録 | ✅ `recordDetail` + `__API_RECON_LOG__` |
| XHR.setRequestHeader のフック | Authorization とその他のヘッダーを検出 | ✅ `observe.xhrHeaders` |
| localStorage/Cookie の読み取りのフック | セッションのキーを特定 | 任意の `observe.storageReads/cookieReads` |
| Vue のルート収集 | frontendRoutes を補完 | ✅ `__API_RECON_ROUTES__`（読み込まれたルート）|
| Vue のガードの無効化 / ログイン転送のブロック | モジュールを描画して API を発生 | ✅ `neutralizeVueRouter` + ネイティブの転送の無効化 |
| React のルート収集 | ルートを補完 | 静的解析 + クリック。専用フックなし |
| ページの転送をブロック（ログインのパス）| 分析のためページに留まる | ログインのパスだけをブロックし、業務のページ移動を保持 |
| 暗号ライブラリーのフック（CryptoJS/SM など）| 暗号化されたパラメーター → 平文の API 本文 | 暗号化入力の手動フックが必要。結果を config に記録 |
| デバッグ妨害の回避 | 回避しないと実行時に API を記録できない | 手動対応が必要。静的解析は引き続き利用可能 |

---

<a id="e-endpoint-regex-when-static-results-are-sparse"></a>
## E. 静的な結果が少ない場合のエンドポイント正規表現

`harvest_static.py` の `extract_endpoints` を広げるか、必須の harvest フェーズ後に手動で確認します:

```bash
grep -rhoaE '"/[a-z][A-Za-z0-9_/\-]{3,}"' js | sort -u
grep -rhoaE '/api/[a-zA-Z0-9_./-]+' js | sort -u
```

---

<a id="f-troubleshooting"></a>
## F. 問題の解決

| 症状 | 原因と対応 |
|---|---|
| 静的な API が少ない | エンドポイント構文の不一致。正規表現を広げる（E 節）|
| chunk 数がマニフェストより大幅に少ない | CSS のみ、または未配置の chunk。404 を再試行 |
| 実行時にログインがまだ表示される | 描画ガード。A1 のキー、保存場所、エンコード、ドメインを再確認 |
| シェルは描画されるがモジュールは空 | 内容のゲート。メニューをモック（A3）し、`routes` のパスを確認 |
| 各ルートで bootstrap/locale のみ | 権限コードが不足。ツリーを再構成（I）し、`role_permissions` と `permissions/all` の両スタブを確認 |
| サイドバーはあるがサブページは空 | ツリーの中間ノードがないか、コードが `userRouteAuth` と不一致 |
| 全 API がログインへ転送 | インターセプター。`neutralize` を確認し、ネストしたフィールドへ走査を拡張 |
| WS フレームが 0 | 購読には操作が必要。`perRouteMs` を増やす |
| レスポンス本文が空 | 実際の応答には `forward: true` が必要 |
| Chromium がない | Chromium をインストールするか、`config.chromium` / `CHROMIUM` を設定 |
| モックでもログインへ戻る | フックが遅いか `location.href` の setter がない。document 開始時に preload |
| 一覧がすべて空 | 空の L3 配列は正常。タブ・設定・詳細を続ける |
| Redux の action をルートと誤認 | get/set/change/clear/toggle/upload を含む内部パスを除外 |
| Vue がまだログインへ転送 | document 開始時に preload を注入。`neutralizeVueRouter: false` ならクライアントのガードを手動で解除 |
| 応答の URL がログにない | `extractUrlsFromResponse` を有効にするか、`__API_RECON_DETAIL__` から抽出 |
| 認証ヘッダー名が不明 | `observe.xhrHeaders` を有効にするか、DevTools のリクエストヘッダーを確認 |
| 実行時解析が遅い・タイムアウト | `waitUntil: domcontentloaded` を使い、`routeTimeout` を下げ、`networkidle2` を避ける |
| プロキシの接続失敗 | `proxy`・環境を確認し、Puppeteer と curl のプロキシポートを一致させる |

---

<a id="g-hardened-targets"></a>
## G. 強化された対象

サーバーが処理全体でセッションを検証する場合（モックできない署名付き Cookie、スタブ化できないサーバー描画のメニュー）、実行時解析はシェルで停止します。想定される動作:

- コードにモジュールのパスが残るため、静的解析では引き続きエンドポイントを列挙できる。
- 上流の参照資料は、同じハーネス（`forward: true`、neutralize なし）で明示的に許可された実際のセッションを使い、実際のメソッド・パラメーター・レスポンスを記録する方法も示しています。これはこのスキルの認証情報不要の既定範囲外であり、認証情報の要求や実際のログインの禁止を上書きしてはいけません。

---

<a id="h-per-task-checklist"></a>
## H. タスクごとのチェックリスト

1. 許可されたスコープを確認
2. `scripts/harvest_static.py` を読み、調整・実行し、`api_static.txt` と `routes.txt` を確認
3. フェーズ 1b: パス周辺の範囲 + 紐付け層 → `param_candidates.json`（J）
4. A1/A2/A3 を再構成 → サイト固有の `config.json`
5. 実行前に `runtime_harvest.js` / `preload.js` を読み、調整
6. `runtimeMode=depth`: `npm install` → 調整した harvest スクリプトを実行
7. `runtimeMode=coverage/both`: document 開始時に調整した preload を注入 → ブラウザー MCP の列挙 + パラメーターのトリガー行列
8. モジュールが描画されない → 権限ツリーの再構成（I）→ スタブ修正 → 再実行
9. 複数のパラメーターサンプルの差分 + エラー推論 → `params_merged.json`
10. 統合 → `site_map.json` + `api_merged.txt`。確認範囲、不足、スクリプトの変更を正確に示す

---

<a id="i-permission-tree-reconstruction-phase-4-detail"></a>
## I. 権限ツリーの再構成（フェーズ 4 の詳細）

単純な `menus: [{ path, show: true }]` のモックではサブモジュールがマウントされない場合に使います。

<a id="i1-locate-auth-modules"></a>
### I1. 認証モジュールの特定

```bash
grep -l 'userRouteAuth' js/*.js
grep -l 'routeMap\|routeLink' js/*.js
grep -rhoaE 'getResultTree|role_permissions|permissions/all' js | head
```

権限 API のパス、応答のフィールド名、利用側の chunk のファイル名を記録します。

<a id="i2-extract-routemap"></a>
### I2. routeMap の抽出

```bash
python3 scripts/extract_route_map.py recon/js recon/
# Output recon/route_map.json
```

`[!] no routeMap pattern found` の場合、`extract_route_map.py` の正規表現を広げるか grep で確認してください:

```bash
grep -rhoaE '([A-Z_][A-Z0-9_]*):\{name:"[^"]*",link:"/[^"]+"\}' js | head -20
```

<a id="i3-build-permission-tree-and-stubs"></a>
### I3. 権限ツリーとスタブを構築

```bash
python3 scripts/build_perm_tree.py recon/js recon/ --config recon/config.json
```

スクリプトの処理:
1. `He=o.DASHBOARD` などの webpack の別名を含め、`userRouteAuth={MONITOR:{url:...},...}` を解析
2. `route_map.json` で別名を実際のパスに解決
3. コードの接頭辞から親を推定（`MONITOR_ALERT` → `MONITOR`）
4. `permissions_tree.json`、`permissions_all_stub.json`、`role_permissions_stub.json` を出力
5. `--config` を付けると、`config.json` の `stubs` を更新し、`routes` を拡張

コピーしたスクリプトの冒頭を調整します:
- `DEFAULT_ROOTS`: 最上位のモジュールコード
- `DEFAULT_PREFIX_PARENT`: `PREFIX_` → 親の対応
- `DEFAULT_EXTRA_PARENT`: 接頭辞による関係のない孤立ノード

<a id="i4-verify-stub-consistency"></a>
### I4. スタブの整合性を検証

```bash
# Permission count should approximate userRouteAuth entries
wc -l recon/perm_codes_all.txt
# routes should cover all route_map links
python3 -c "import json; m=json.load(open('recon/route_map.json')); r=set(json.load(open('recon/config.json'))['routes']); print('missing', [v['link'] for v in m.values() if v['link'] not in r])"
```

<a id="i5-rerun-runtime-and-compare"></a>
### I5. 実行時解析を再実行して比較

```bash
node recon/runtime_harvest.js recon/config.json
# Compare runtime_api.json counts before/after mocks; check module APIs under /attack, /asset, etc.
```

| モック前 | モック成功後 |
|---|---|
| 各ルートで同じ 3～5 件の初期化リクエスト | 異なるルートで異なるモジュール API が発生 |
| `/api/locale/language` のみ | `/api/web/...` などのモジュールのエンドポイント |
| `routes.txt` のルートが 1 桁 | route_map から 80～110 件以上の `routes` |

<a id="i6-if-it-still-fails"></a>
### I6. まだ失敗する場合

- Coverage: サイドバーとタブをクリック。権限関連の要求には操作が必要な場合があります。
- スタブのフィールド: 許可済みの API 応答とネストを比較。上流の例は実際のセッションで curl を使っていますが、認証情報不要のこのスキルでは、認証情報の取得やログインをしないでください。
- 追加のガード: `hasPermission|checkRole|func.` などのボタンの確認を grep し、`role_permissions.permissions` を拡張。
- 静的解析へのフォールバック: モジュールのパスは `api_static.txt` に残り、実行時解析がメソッド・本文を補います。`param_candidates.json` と記録したサンプルを保持してください。

---

<a id="j-parameter-reconstruction-phase-1b--5b--5c"></a>
## J. パラメーターの再構成（フェーズ 1b / 5b / 5c）

これは方法であり、万能なスクリプトではありません。パスには正規表現、パラメーターにはパス周辺の範囲、UI の紐付け、サンプル差分、エラー推論を使います。

<a id="j1-anchor-windows-path-to-payload-builder"></a>
### J1. パス周辺の範囲: パスからペイロード構築へ

```bash
# Anchor on known Phase 1 paths
grep -n '"/api/user/list"' js/*.js
grep -rhoaE '.{0,120}("/api[^"]+").{0,200}' js | head
grep -rhoaE '(params|data|body|payload)\s*:\s*\{' js | head
grep -rhoaE '(get|post|put|delete|patch)\([^,]+,\s*\{' js | head
```

<a id="j2-wrappers-and-transport-forms"></a>
### J2. ラッパーと送信形式

```bash
# axios / shared request wrapper
grep -rhoaE '(axios|request)\.(get|post|put|delete|patch)\(' js | head
grep -rhoaE 'interceptors\.(request|response)' js | head

# GraphQL
grep -rhoaE '(query|mutation)\s+\w+|gql`|graphql\(' js | head
grep -rhoaE '\$[a-zA-Z_]+\s*:\s*(Int|String|Boolean|\[)' js | head

# FormData / multipart
grep -rhoaE 'FormData|\.append\(' js | head

# Path parameters
grep -rhoaE 'path:\s*"/[^"]*:[^"]+"' js | head
grep -rhoaE 'useParams|route\.params|\$route\.params' js | head
```

<a id="j3-validation-required-fields-formats-enums"></a>
### J3. 検証: 必須フィールド、形式、列挙値

```bash
grep -rhoaE '(required|message|pattern|enum|validator)\s*:' js | head
grep -rhoaE 'yup\.|zod\.|async-validator|Form\.Item|a-form-item|el-form-item' js | head
grep -rhoaE 'rules\s*:\s*\[|name:\s*["\'][a-zA-Z_]+["\']' js | head
grep -rhoaE 'label.*value|options\s*:\s*\[' js | head
```

<a id="j4-binding-layer-form-to-api"></a>
### J4. 紐付け層: フォームから API へ

```bash
grep -rhoaE 'onFinish|handleSubmit|getFieldsValue|validateFields' js | head
grep -rhoaE '(pick|omit|transform|dayjs|moment)\(' js | head
```

実行時の証拠: DevTools → Network → request → Initiator（呼び出しスタック）。`fetch`/`send` から上へ、ペイロードの構築を追跡します。

<a id="j5-encrypted-parameters"></a>
### J5. 暗号化されたパラメーター

```bash
grep -rhoaE 'encrypt|decrypt|sign|CryptoJS|sm2|sm3|sm4|RSA|AES' js | head
```

暗号文からフィールドを推測してはいけません。暗号化関数の入力をフックして暗号化前の平文ペイロードを記録し、結論を `config.json` / `param_candidates.json` に保存してください。

<a id="j6-parameter-trigger-matrix-required-in-phase-3"></a>
### J6. パラメーターのトリガー行列（フェーズ 3 で必須）

モジュールごとに各操作を記録し、リクエスト本文・クエリの差分を確認します:

| 操作 | 確認するもの |
|---|---|
| 初期一覧 | ページ送りの既定値 |
| 検索 | keyword、絞り込み |
| 詳細な絞り込み | 任意フィールド |
| 作成・編集 | 完全なエンティティ |
| 一括・エクスポート | `ids[]`, `exportType` |
| 並べ替え・ページ送り | `sortField`, `order` |

出力 `param_samples.json`: `[{ "path", "method", "action": "search", "body", "query", "headers" }]`

<a id="j7-confidence-rules"></a>
### J7. 確信度の規則

| 確信度 | 条件 |
|---|---|
| 高 | 静的な呼び出し箇所と、一致する実行時サンプル 2 件以上 |
| 中 | 静的解析のみ、または実行時サンプル 1 件 |
| 低 | 再検証せず、レスポンス・エラーから推論 |
| トリガー待ち | 静的なフィールドは既知だが、UI・権限の経路に未到達 |

<a id="j8-scenario-recipes"></a>
### J8. シナリオ別の手法

| シナリオ | 順序 |
|---|---|
| REST の一覧ページ | J1 ペイロードのオブジェクト → J6 の 4 つの差分 → J3 の規則 |
| 作成・編集フォーム | J3 フォームの名前 → J4 送信の連鎖 → 許可された非破壊の実行時送信と、空フィールドの 400 の確認 |
| GraphQL | J2 変数宣言 → 実行時の操作ごとの variables を記録 |
| 暗号化された本文 | J5 入力のフック → 暗号化前のフィールドが実際のパラメーター |

<a id="j9-mapping-to-api-recon-phases"></a>
### J9. api-recon のフェーズとの対応

| api-recon | パラメーター解析 |
|---|---|
| フェーズ 1 静的解析 | J1 パス周辺の範囲 |
| フェーズ 2 A2 インターセプター | 全体に注入するフィールド（tenantId、sign）|
| フェーズ 3 実行時解析 | J6 トリガー行列 + `param_samples.json` |
| フェーズ 4 権限ツリー | モジュールごとにフォームが異なり、十分なモック権限ですべてのフィールドを表示 |
| フェーズ 5 統合 | `params_merged.json` + 確信度。1 サンプルから必須フィールドの結論を出さない |

<a id="j10-troubleshooting"></a>
### J10. 問題の解決

| 症状 | 対応 |
|---|---|
| 静的なフィールドが実行時に現れない | トリガー待ちとし、権限ツリー、詳細な絞り込み、連動する選択肢を補完 |
| 同じパスで本文の形が異なる | 正常。`action` ごとに記録し、無理にスキーマを統合しない |
| モック応答でもパラメーターが必要 | 推測したスタブの応答ではなく、送信リクエストの本文・ヘッダーを確認 |
| 400 がネストしたフィールドを示す | 外側の `data`/`bizData`/`variables` のラッパーを確認 |
| GraphQL の操作名しか見えない | `variables` の JSON を展開し、静的解析で `$var: Type` を探す |

---
