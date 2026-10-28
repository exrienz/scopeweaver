[English](../../../skills/api-recon/reference.md) · [한국어](../../ko/skills/api-recon-reference.md) · [简体中文](../../zh-Hans/skills/api-recon-reference.md) · 繁體中文 · [日本語](../../ja/skills/api-recon-reference.md) · [Español](../../es/skills/api-recon-reference.md)

<a id="api-recon-reference"></a>

# api-recon 參考

各語言版本請見本頁上方導覽。

Grep 配方、`config.json` 範本與排錯。在 `js/` 執行 grep。單行 bundle 可選用 `js-beautify` 或 `sed 's/}/}\n/g'`；一般原始 grep 搭配上下文視窗已足夠。

<a id="script-notes"></a>

## 指令碼注意事項

`scripts/` 全部檔案為參考範本，執行前依目標調整。常見修改：

| 指令碼 | 常見調整 |
|---|---|
| `harvest_static.py` | 端點 regex、webpack／Vite manifest、微前端 publicPath、重試／並行 |
| `runtime_harvest.js` | 中和欄位名稱／成功值、stub 比對／本文形狀、路由來源、WS 擷取、`waitUntil`／`routeTimeout`／`proxy` |
| `preload.js` | `loginPathRe`、L1 stub、`neutralize.fields`、`apiPattern`、是否啟用 L3、`recordDetail`、`observe.*`、`neutralizeVueRouter` |
| `spider_mpa.py` | `--exclude` 破壞性連結、cookie、深度／上限、同源篩選 |
| `extract_route_map.py` | `routeMap`／`routeLink` regex、KEY 命名模式 |
| `build_perm_tree.py` | `userRouteAuth` 剖析、`ROOTS`／`PREFIX_PARENT` 階層啟發式、外層 stub 欄位名稱 |
| `config.json` | 上述網站特定參數的集中設定 |

調整檔案留在任務目錄（如 `recon/`），並報告相對範本的精確差異。

---

<a id="a-reconstruct-the-three-gates"></a>

## A. 重建三個閘門

<a id="a1-rendering-how-is-login-state-determined"></a>

### A1. 轉譯：如何判定登入狀態？



```bash
grep -rhoaE '.{0,40}(isLogin|isAuthenticated|loggedIn|hasLogin|requireAuth)\b.{0,80}' js | head
grep -rhoaE 'function (getUser|getToken|getAuth)[0-9]?\([^)]*\)\{.{0,200}' js | head
grep -rhoaE '(localStorage|sessionStorage)\.getItem\("[^"]+"\)' js | sort -u
grep -rhoaE '(Cookies?|cookie)\.(get|load)\("[^"]+"\)' js | sort -u
grep -rhoaE '\batob\(|JSON\.parse\(|jwt|decode' js | head
```



追蹤 `isLogin = f(getUser())` → `getUser = decode(storage.read(KEY))`，識別儲存鍵、容器（Cookie／localStorage）與編碼：

| 編碼 | 設定中的模擬值 |
|---|---|
| 純字串／`"1"`／token | `"value": "anything-truthy"` |
| `JSON.parse(x)` | `"value": "json:{\"id\":1,\"username\":\"admin\"}"` |
| `JSON.parse(atob(x))` | `"value": "b64json:{\"id\":1,\"username\":\"admin\"}"` |
| JWT | 未簽章／`alg:none` JWT，或使用 bundle 內金鑰簽章，僅供用戶端模擬守衛 |
| 加密（SM2／AES／RSA） | 尋找寫死金鑰；僅在轉譯守衛只需可解碼 blob 時模擬，否則回退靜態分析 |

寫入 `cookies`／`localStorage`。

<a id="a2-interceptor-what-triggers-login-redirects"></a>

### A2. 攔截器：什麼觸發 /login 重新導向？



```bash
grep -rhoaE '.{0,60}(interceptors\.response|axios|request\.use).{0,120}' js | head
grep -rhoaE '.{0,40}(response_code|errcode|errno|\bcode\b|\bret\b|\bstatus\b)\s*[=!]==?\s*[\-0-9]{1,4}.{0,60}' js | head -20
grep -rhoaE '.{0,40}(未登录|请重新登录|登录已过期|unauthorized|登录失效|授权|token.{0,10}invalid).{0,40}' js | head
grep -rhoaE '.{0,30}(location\.href|router\.(push|replace)|navigate)\([^)]*login[^)]*\)' js | head
```



識別欄位名、成功值（通常 `0` 或 `200`）及觸發重新導向的失敗值，以無效工作階段驗證：



```bash
curl -sk -X POST -H 'Cookie: <fakekey>=junk' https://target/api/<protected> -d '{}' -H 'Content-Type: application/json'
```



寫入 `neutralize.fields` + `neutralize.success`。

<a id="a3-content-where-do-menus-and-permissions-come-from"></a>

### A3. 內容：選單與權限從哪裡來？



```bash
grep -rhoaE '"/api[^"]*(permission|perm|role|menu|acl|resource|nav)[^"]*"' js | sort -u
grep -rhoaE '.{0,30}(menus|permissions|menuList|routeList|authList|role_permissions)\b.{0,120}' js | head
grep -rhoaE 'userRouteAuth|getResultTree|routeMap|routeLink|hasPermission|checkAuth' js | head
grep -rhoaE '([A-Z_][A-Z0-9_]*):\{name:"[^"]*",link:"/[^"]+"\}' js | head
```



業務管理介面常見兩層資料：

| API | 典型 payload | 使用端 |
|---|---|---|
| `.../role_permissions` | `{ permissions: string[], role_type }` | 路由守衛、按鈕層 ACL |
| `.../permissions/all` | `tree[{ code, position, children }]` | 側欄選單轉譯 |
| bundle 中的 `userRouteAuth` | `{ CODE: { url, name? } }` | code → 前端路徑 |
| bundle 中的 `routeMap` | `{ KEY: { name, link } }` | 別名解析（webpack `o.DASHBOARD`） |

閱讀使用端程式碼，確認 `getResultTree(tree, permissions)` 如何篩選，以及 `v-if`／`hasAuth(code)` 檢查哪些欄位。

小型網站可在 `stubs` 手動建立寬鬆模擬 payload。

大型網站側欄／子模組空白時，依 I 節重建完整權限樹。

---

<a id="b-configjson-template"></a>

## B. config.json 範本



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



欄位：
- `runtimeMode`：`depth`（Puppeteer）、`coverage`（browser MCP）、`both`。
- `cookies[].value` 前綴：`b64json:` → base64(JSON)；`json:` → 原始 JSON；無 → 字面值。
- `forward: true` 轉送真實請求並改寫狀態碼欄位；`false` 使用離線 stub。
- `mockTier`：coverage preload 層，例如 `L1+L2`、`L1+L2+L3`。
- `routes` 來自 `routes.txt`；模擬選單後，測試架構附加 `<a href>` 路由。
- `captureResponses`／`recordWs` 僅適用 depth。
- `waitUntil`：大型 SPA 用 `domcontentloaded`，避免 `networkidle2` 卡住。
- `routeTimeout`：每個路由的 `page.goto` 逾時毫秒數。
- `proxy`：Puppeteer `--proxy-server`；亦可用 `HTTP_PROXY`／`HTTPS_PROXY`。

<a id="b1-two-stub-template-role_permissions--permissionsall"></a>

### B1. 雙 stub 範本（role_permissions + permissions/all）



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



外層欄位（`response_code`／`code`／`data`）須符合 A2；`permissions` 必須涵蓋樹中全部葉代碼。

---

<a id="c-coverage-preload-configuration"></a>

## C. Coverage preload 設定

編輯副本 `preload.js` 頂部 `CONFIG`，或在 CDP 注入前替換：



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



驗證 `window.__API_RECON_PRELOAD__ === true` 且 pathname 穩定。

匯出擷取結果：



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

## D. Preload／runtime hook 覆蓋

preload（coverage）與 runtime_harvest（depth）內建瀏覽器 hook：

| Hook | 探索價值 | 覆蓋 |
|---|---|---|
| Hook fetch / XHR.open | 擷取 URL／方法 | ✅ `recordDetail` + `__API_RECON_LOG__` |
| Hook XHR.setRequestHeader | 探索 Authorization 等標頭 | ✅ `observe.xhrHeaders` |
| Hook localStorage／cookie 讀取 | 識別工作階段鍵 | 選用 `observe.storageReads/cookieReads` |
| Vue 路由蒐集 | 補全 frontendRoutes | ✅ `__API_RECON_ROUTES__`（已載入路由） |
| 中和 Vue 守衛／封鎖登入重新導向 | 轉譯模組以觸發 API | ✅ `neutralizeVueRouter` + 原生重新導向中和 |
| React 路由蒐集 | 補全路由 | 靜態 + 點選；無專用 hook |
| 封鎖頁面登入重新導向 | 保持頁面以供分析 | 僅封鎖登入路徑，保留業務導覽 |
| Hook 加密函式庫（CryptoJS／SM 等） | 加密參數 → 明文 API 本文 | 須手動 hook 加密輸入，將結果記入設定 |
| 略過反偵錯 | 否則執行時無法擷取 API | 須手動處理；仍可靜態分析 |

---

<a id="e-endpoint-regex-when-static-results-are-sparse"></a>

## E. 靜態結果稀少時的端點 regex

擴大 `harvest_static.py` 的 `extract_endpoints`，或在必要 harvest 階段後手動檢查：



```bash
grep -rhoaE '"/[a-z][A-Za-z0-9_/\-]{3,}"' js | sort -u
grep -rhoaE '/api/[a-zA-Z0-9_./-]+' js | sort -u
```



---

<a id="f-troubleshooting"></a>

## F. 疑難排解

| 症狀 | 原因與處理 |
|---|---|
| 靜態 API 少 | 端點語法不符，擴大 regex（E 節） |
| chunk 數遠少於 manifest | 僅 CSS 或未部署 chunk；重試 404 |
| 執行時仍顯示登入 | 轉譯守衛：重查 A1 鍵、容器、編碼、網域 |
| 外殼正常、模組空白 | 內容閘門：模擬選單（A3），檢查 `routes` |
| 各路由只有 bootstrap／locale | 缺權限代碼：重建樹（I），檢查 `role_permissions` 與 `permissions/all` stub |
| 有側欄、子頁空白 | 缺中間樹節點，或代碼與 `userRouteAuth` 不符 |
| 每個 API 都導向登入 | 攔截器：檢查 `neutralize`，擴展巢狀欄位遍歷 |
| 無 WS frame | 訂閱需互動；增加 `perRouteMs` |
| 回應本文為空 | 真實回應需 `forward: true` |
| 缺 Chromium | 安裝 Chromium 或設 `config.chromium`／`CHROMIUM` |
| Mock 仍回登入 | Hook 太晚或缺 `location.href` setter；在文件開始注入 |
| 全部清單空白 | L3 空陣列正常；繼續分頁／設定／詳情 |
| Redux action 誤當路由 | 篩除含 get／set／change／clear／toggle／upload 的內部路徑 |
| Vue 仍導向登入 | 文件開始注入 preload；若 `neutralizeVueRouter: false`，手動清除用戶端守衛 |
| 記錄缺回應 URL | 啟用 `extractUrlsFromResponse` 或從 `__API_RECON_DETAIL__` 擷取 |
| 不知授權標頭名 | 啟用 `observe.xhrHeaders` 或檢查 DevTools 請求標頭 |
| 執行緩慢／逾時 | 用 `waitUntil: domcontentloaded`、降低 `routeTimeout`、避免 `networkidle2` |
| 代理連線失敗 | 檢查 `proxy`／環境，對齊 Puppeteer 與 curl 代理連接埠 |

---

<a id="g-hardened-targets"></a>

## G. 強化目標

伺服器在整個流程驗證工作階段（無法模擬的簽章 cookie、無法 stub 的伺服器選單）時，執行時止於外殼。預期行為：

- 靜態分析仍可列舉端點，因模組路徑仍在程式碼。
- 上游參考亦描述使用明確獲授權真實工作階段搭配同一測試架構（`forward: true`，不 neutralize）擷取實際方法／參數／回應。這超出本技能無憑證預設範圍，不能凌駕禁止索取憑證或真實登入的規則。

---

<a id="h-per-task-checklist"></a>

## H. 各任務檢查清單

1. 確認授權範圍。
2. 閱讀 `scripts/harvest_static.py`、調整、執行並檢查 `api_static.txt`、`routes.txt`。
3. 階段 1b：路徑錨點視窗 + 綁定層 → `param_candidates.json`（J）。
4. 重建 A1／A2／A3 → 網站專用 `config.json`。
5. 執行前閱讀並調整 `runtime_harvest.js`／`preload.js`。
6. `runtimeMode=depth`：`npm install` → 執行調整後 harvest。
7. `runtimeMode=coverage/both`：文件開始注入調整後 preload → browser MCP 列舉 + 參數觸發矩陣。
8. 模組未轉譯 → 權限樹重建（I）→ 修補 stub → 重跑。
9. 多參數樣本差異 + 錯誤推斷 → `params_merged.json`。
10. 合併 → `site_map.json` + `api_merged.txt`；準確說明覆蓋、缺口與指令碼變更。

---

<a id="i-permission-tree-reconstruction-phase-4-detail"></a>

## I. 權限樹重建（階段 4 詳情）

簡單 `menus: [{ path, show: true }]` mock 無法掛載子模組時使用。

<a id="i1-locate-auth-modules"></a>

### I1. 尋找驗證模組



```bash
grep -l 'userRouteAuth' js/*.js
grep -l 'routeMap\|routeLink' js/*.js
grep -rhoaE 'getResultTree|role_permissions|permissions/all' js | head
```



記錄權限 API 路徑、回應欄位名與使用它的 chunk 檔名。

<a id="i2-extract-routemap"></a>

### I2. 擷取 routeMap



```bash
python3 scripts/extract_route_map.py recon/js recon/
# Output recon/route_map.json
```



出現 `[!] no routeMap pattern found` 時，擴大 `extract_route_map.py` regex 或以 grep 檢查：



```bash
grep -rhoaE '([A-Z_][A-Z0-9_]*):\{name:"[^"]*",link:"/[^"]+"\}' js | head -20
```



<a id="i3-build-permission-tree-and-stubs"></a>

### I3. 建立權限樹與 stub



```bash
python3 scripts/build_perm_tree.py recon/js recon/ --config recon/config.json
```



指令碼邏輯：
1. 剖析 `userRouteAuth={MONITOR:{url:...},...}`，含 `He=o.DASHBOARD` 等 webpack 別名。
2. 透過 `route_map.json` 將別名解析為真實路徑。
3. 從代碼前綴推斷父節點（`MONITOR_ALERT` → `MONITOR`）。
4. 輸出 `permissions_tree.json`、`permissions_all_stub.json`、`role_permissions_stub.json`。
5. 加 `--config` 時更新 `config.json` 的 `stubs` 並擴展 `routes`。

在指令碼副本頂端調整：
- `DEFAULT_ROOTS`：頂層模組代碼。
- `DEFAULT_PREFIX_PARENT`：`PREFIX_` → 父節點對應。
- `DEFAULT_EXTRA_PARENT`：無前綴關係的孤立節點。

<a id="i4-verify-stub-consistency"></a>

### I4. 驗證 stub 一致性



```bash
# Permission count should approximate userRouteAuth entries
wc -l recon/perm_codes_all.txt
# routes should cover all route_map links
python3 -c "import json; m=json.load(open('recon/route_map.json')); r=set(json.load(open('recon/config.json'))['routes']); print('missing', [v['link'] for v in m.values() if v['link'] not in r])"
```



<a id="i5-rerun-runtime-and-compare"></a>

### I5. 重跑執行時並比較



```bash
node recon/runtime_harvest.js recon/config.json
# Compare runtime_api.json counts before/after mocks; check module APIs under /attack, /asset, etc.
```



| 模擬前 | 成功模擬後 |
|---|---|
| 每路由相同 3–5 個啟動請求 | 不同路由觸發不同模組 API |
| 只有 `/api/locale/language` | 出現 `/api/web/...` 等模組端點 |
| `routes.txt` 路由個位數 | 從 route_map 取得 80–110+ 個 `routes` |

<a id="i6-if-it-still-fails"></a>

### I6. 仍失敗時

- Coverage：點側欄與分頁；權限請求可能需互動。
- Stub 欄位：與既有獲授權 API 回應比較巢狀結構。上游範例使用真實工作階段 curl；本無憑證技能不得取得憑證或登入。
- 額外守衛：grep `hasPermission|checkRole|func.` 等按鈕檢查，擴展 `role_permissions.permissions`。
- 靜態備援：模組路徑仍在 `api_static.txt`；執行時補上方法／本文。保留 `param_candidates.json` 與擷取樣本。

---

<a id="j-parameter-reconstruction-phase-1b--5b--5c"></a>

## J. 參數重建（階段 1b／5b／5c）

這是方法，不是通用指令碼。路徑用 regex；參數用錨點視窗、介面綁定、樣本差異與錯誤推斷。

<a id="j1-anchor-windows-path-to-payload-builder"></a>

### J1. 錨點視窗：路徑至 payload 組裝器



```bash
# Anchor on known Phase 1 paths
grep -n '"/api/user/list"' js/*.js
grep -rhoaE '.{0,120}("/api[^"]+").{0,200}' js | head
grep -rhoaE '(params|data|body|payload)\s*:\s*\{' js | head
grep -rhoaE '(get|post|put|delete|patch)\([^,]+,\s*\{' js | head
```



<a id="j2-wrappers-and-transport-forms"></a>

### J2. 包裝層與傳輸形式



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

### J3. 驗證：必填欄位、格式、列舉



```bash
grep -rhoaE '(required|message|pattern|enum|validator)\s*:' js | head
grep -rhoaE 'yup\.|zod\.|async-validator|Form\.Item|a-form-item|el-form-item' js | head
grep -rhoaE 'rules\s*:\s*\[|name:\s*["\'][a-zA-Z_]+["\']' js | head
grep -rhoaE 'label.*value|options\s*:\s*\[' js | head
```



<a id="j4-binding-layer-form-to-api"></a>

### J4. 綁定層：表單至 API



```bash
grep -rhoaE 'onFinish|handleSubmit|getFieldsValue|validateFields' js | head
grep -rhoaE '(pick|omit|transform|dayjs|moment)\(' js | head
```



執行時證據：DevTools → Network → request → Initiator（呼叫堆疊）；從 `fetch`／`send` 向上追至 payload 組裝。

<a id="j5-encrypted-parameters"></a>

### J5. 加密參數



```bash
grep -rhoaE 'encrypt|decrypt|sign|CryptoJS|sm2|sm3|sm4|RSA|AES' js | head
```



絕不從密文猜欄位。Hook 加密函式輸入、記錄加密前明文 payload，並將結論儲存至 `config.json`／`param_candidates.json`。

<a id="j6-parameter-trigger-matrix-required-in-phase-3"></a>

### J6. 參數觸發矩陣（階段 3 必要）

依模組記錄各操作，比較請求本文／查詢差異：

| 操作 | 檢查 |
|---|---|
| 初始清單 | 分頁預設值 |
| 搜尋 | keyword、filters |
| 進階篩選 | 選填欄位 |
| 新增／編輯 | 完整實體 |
| 批次／匯出 | `ids[]`、`exportType` |
| 排序／翻頁 | `sortField`、`order` |

輸出 `param_samples.json`：`[{ "path", "method", "action": "search", "body", "query", "headers" }]`

<a id="j7-confidence-rules"></a>

### J7. 信心規則

| 信心 | 條件 |
|---|---|
| 高 | 靜態呼叫點與至少兩個相符執行樣本 |
| 中 | 僅靜態或單一執行樣本 |
| 低 | 從回應／錯誤推斷，未二次驗證 |
| 待觸發 | 已知靜態欄位，但介面／權限路徑未到達 |

<a id="j8-scenario-recipes"></a>

### J8. 情境配方

| 情境 | 順序 |
|---|---|
| REST 清單頁 | J1 payload 物件 → J6 四次差異 → J3 規則 |
| 新增／編輯表單 | J3 表單名稱 → J4 提交鏈 → 獲授權、非破壞性執行提交與空欄位 400 檢查 |
| GraphQL | J2 變數宣告 → 擷取各執行操作的 variables |
| 加密本文 | J5 hook 輸入 → 加密前欄位即實際參數 |

<a id="j9-mapping-to-api-recon-phases"></a>

### J9. 與 api-recon 階段對應

| api-recon | 參數偵察 |
|---|---|
| 階段 1 靜態 | J1 錨點視窗 |
| 階段 2 A2 攔截器 | 全域注入欄位（tenantId、sign） |
| 階段 3 執行時 | J6 觸發矩陣 + `param_samples.json` |
| 階段 4 權限樹 | 模組表單不同；足夠模擬權限才能顯示全部欄位 |
| 階段 5 合併 | `params_merged.json` + 信心；不可由單一樣本判斷必填 |

<a id="j10-troubleshooting"></a>

### J10. 疑難排解

| 症狀 | 處理 |
|---|---|
| 靜態欄位從未在執行時出現 | 標記待觸發；補全權限樹、進階篩選、連動選單 |
| 相同路徑、不同本文形狀 | 正常；依 `action` 分別記錄，勿強制合併結構 |
| Mock 回應下仍需參數 | 檢查送出請求本文／標頭，勿從 stub 回應推斷 |
| 400 指出巢狀欄位 | 檢查外層 `data`／`bizData`／`variables` |
| 只見 GraphQL 操作名稱 | 展開 `variables` JSON；靜態尋找 `$var: Type` |

---
