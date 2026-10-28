[English](../../../skills/api-recon/SKILL.md) · [한국어](../../ko/skills/api-recon.md) · [简体中文](../../zh-Hans/skills/api-recon.md) · 繁體中文 · [日本語](../../ja/skills/api-recon.md) · [Español](../../es/skills/api-recon.md)

---
name: api-recon
description: 在獲授權範圍內盤點網站 API 介面。
---

<a id="api-recon-frontend-interface-discovery"></a>

# API 偵察（前端介面探索）

各語言版本請見本頁上方導覽。

在獲得授權的範圍內，盡可能完整探索後端 API（路徑、方法、參數、回應本文）、前端路由與介面觸發點（分頁、對話框、表格操作）。

---

<a id="boundaries-and-prohibitions-required-reading"></a>

## 邊界與禁止事項（必讀）

本技能僅盤點 API 與參數，不進行漏洞探索或利用。

<a id="task-boundaries"></a>

### 任務邊界

| 範圍 | 允許 | 禁止 |
|---|---|---|
| 目標 | 列舉路徑、方法、參數、路由與介面觸發點 | SQLi／XSS、存取控制攻擊、暴力破解、漏洞模糊測試、竄改請求攻擊、破壞性操作 |
| 驗證 | Hook 並以 stub／mock 模擬用戶端登入守衛 | 索取或猜測憑證；提交真實登入表單 |
| 執行時 | 無憑證 hook；以 mock 呈現登入後 SPA 外殼 | 需要真實後端工作階段的流程 |

<a id="dynamic-analysis-without-credentials-phase-3-default"></a>

### 無憑證動態分析（階段 3 預設）

1. 以 `preload.js`／`runtime_harvest.js` 攔截並 stub 登入、權限、選單與其他啟動 API。
2. 傳回形狀與成功代碼正確的模擬查詢本文，資料可為空。
3. 無後端或遇 401 時仍呈現登入後頁面，以觸發更多 XHR／fetch／WebSocket 請求。
4. 空資料、空表格與預留介面屬預期，勿轉為真實登入或漏洞測試。

Mock 僅用來掛載前端路由／元件並記錄送出請求。目標是探索前端還會傳送什麼，不是取得真實後端資料。

<a id="workflow-prohibitions"></a>

### 工作流程禁止事項

| 禁止 | 必須採用的替代方式 |
|---|---|
| 階段 1 完成前，grep／curl／Read 主 `index-*.js` 擷取 API 路徑 | 執行 `OUTDIR/harvest_static.py` |
| 手寫 `extract_apis.py` 等替代程式 | 調整並重跑 `OUTDIR/harvest_static.py` |
| 至少失敗兩次後重複相同 grep／指令 | 改策略：讀 tool_logs、調整 harvest、查參考 |
| 略過閘門 A／B，直接執行原始 `scripts/` 範本 | 複製至 OUTDIR 並依目標調整 |
| 真實憑證、OTP 或 OAuth 驗證 | 依上文使用 stub／mock |
| 略過 stub，以存取控制／注入測試取得真實資料 | 僅在偵察範圍記錄送出請求 |
| 刪除、匯出敏感資料、批次寫入或其他不可逆操作 | 覆蓋率點選亦受此限制 |
| 無執行時與動態列舉卻宣稱全部頁面／API | 滿足完成定義或說明限制 |
| 無觸發矩陣與差異卻宣稱全部參數 | 完成階段 3b 矩陣 + 階段 5 差異 |
| 由單一執行樣本推斷必填／選填 | 比較樣本或檢查驗證規則／錯誤 |

---

<a id="two-layers-and-runtime-modes"></a>

## 兩層分析與執行模式

| 層級 | 輸出 | 限制 |
|---|---|---|
| 靜態（JS bundle） | 端點路徑、路由草稿、payload 組裝處的候選欄位 | 無 HTTP 方法；參數需階段 1b；漏掉執行時組合 URL |
| 執行時（實際工作階段） | 方法、本文、回應、動態 URL、WS／SSE；樣本差異補足參數 | 頁面必須轉譯才發出請求；單樣本無法確定必填／選填 |

| 模式 | 引擎 | 用途 |
|---|---|---|
| **depth** | `runtime_harvest.js`（Puppeteer） | API 盤點、方法／參數／回應、WS／SSE、可重複批次執行 |
| **coverage** | 瀏覽器 + `preload.js` | 點選分頁／對話框／表格，深入功能覆蓋 |
| **both** | 先 depth 再 coverage | 最完整，也最耗時 |

參數方法（無通用指令碼）：路徑用 harvest／regex；參數用錨點視窗擴展、介面綁定鏈、樣本差異與錯誤推斷。[reference.md](api-recon-reference.md) J 節提供 grep 配方。

---

<a id="definition-of-completion"></a>

## 完成定義

全部條件滿足後才能宣稱完成：

- [ ] 靜態：階段 1 產生 `api_static.txt`、`routes.txt`、`js/`。
- [ ] 執行時：至少 depth 或 coverage；coverage／both 必須有可用 hook 與動態列舉迴圈。
- [ ] 外殼：業務路徑保持在 `/login` 以外（檢查 hash 路由）。
- [ ] 參數：coverage／both 完成觸發矩陣與 `param_samples.json`；階段 5 合併 `params_merged.json`。
- [ ] depth 模組空白時：階段 4 還原權限樹並重跑，直到出現 locale／啟動 API 以外的模組 API。
- [ ] 交付：全部階段 5 輸出；`insert_assets` 儲存服務與端點資產。

---

<a id="scripts-and-gates"></a>

## 指令碼與閘門

`scripts/` 內為參考範本，絕不可未調整便執行並當作最終結果。

先閱讀、依目標調整，寫入 `OUTDIR`（如 `recon/`），並記錄 `CHANGES.md`。不適用時以相同方法重寫，僅借用結構。

| 閘門 | 時機 | 範本 → OUTDIR 副本 | 常見調整 |
|---|---|---|---|
| A（靜態） | 階段 0 後，首次 harvest／spider 前 | `harvest_static.py` / `spider_mpa.py` | 預設 regex 適用多數網站；僅 manifest／語法不符時改端點 regex、webpack／Vite `publicPath`、MPA 排除／cookie |
| B（執行時） | 階段 2 後，depth／coverage 前 | `runtime_harvest.js` / `preload.js` + `config.json` | Cookie／localStorage 鍵、中和成功值、stub、登入 regex、API 前綴、hash／history |

SPA 必須依此順序；階段編號優先於撰寫指令碼前的探索瀏覽，不得重排：

| 步驟 | 必須 | 禁止 |
|---|---|---|
| 階段 0 後 | 下一個 Bash 指令 = `python3 OUTDIR/harvest_static.py <URL> OUTDIR` | curl／grep／Read 主 `index-*.js`（通常 >500KB） |
| 閘門 A | 複製指令碼、必要小幅修改、立即執行 | 決定是否 harvest 前手動擷取 API |
| 階段 1 完成前 | `wc -l` 檢查輸出，404 時調整 harvest 並重試 | 撰寫擷取程式；反覆 grep 未下載 URL |
| 階段 1b 起 | 僅 grep `OUTDIR/js/*.js` | 以主 bundle 取代 harvest |

- 正確：複製 `harvest_static.py`、視需要調整 regex、立即執行。
- 錯誤：curl 主 bundle、反覆 grep、暫時擷取程式、最後才 harvest。
- MPA：階段 0 後下一個 Bash 指令 = `python3 OUTDIR/spider_mpa.py ...`。

---

<a id="tool-and-output-limits"></a>

## 工具與輸出限制

| 限制 | 規則 |
|---|---|
| 大型檔案 | 不得將 >100KB 的 `index-*.js` Read／grep 至上下文；以 OUTDIR 指令碼批次處理 |
| grep 輸出 | 必須加 `\| head -20` 或 `-m 5`；保留路徑摘要，不貼 bundle 片段 |
| 驗證 | 使用 `wc -l`、`ls \| wc -l`；勿 Read 整個目錄 |
| Regex 預覽 | 選用，最多一次，僅限 ≤50KB chunk／HTML；harvest 為靜態事實來源 |
| 參考 | [reference.md](api-recon-reference.md) 提供配方／範本／排錯，勿在行內重複 |

---

<a id="execution-roadmap"></a>

## 執行路線



```
Phase 0 classification + OUTDIR
  → Gate A → Phase 1 harvest (run immediately)
  → Phase 1b parameter reconstruction
  → Phase 2 three authentication gates → config.json
  → Gate B → Phase 3 runtime + parameter matrix
  → Phase 4 permission tree if needed → rerun Phase 3
  → Phase 5 merge report + insert_assets for every discovered service/endpoint asset; omit none
```



依序檢查，前一步完成前不得前進。

1. [ ] **階段 0**：分類 SPA／MPA，建立 `OUTDIR` → [階段 0](#phase-0-classification)
2. [ ] **閘門 A + 階段 1**：複製指令碼、立即 harvest、`wc -l` 驗證 → [階段 1](#phase-1-static-discovery)
3. [ ] **階段 1b**：錨點視窗 + 綁定層 → `param_candidates.json` → [階段 1b](#phase-1b-parameter-reconstruction)
4. [ ] **階段 2**：三個驗證閘門 → `config.json` → [階段 2](#phase-2-three-authentication-gates)
5. [ ] **閘門 B**：調整執行時指令碼 → [階段 3](#phase-3-runtime)
6. [ ] **階段 3**：depth／coverage／both；驗證進入外殼；參數觸發矩陣 → `param_samples.json`。
7. [ ] 必要時**階段 4**：權限樹 → 修補 stub → 重跑階段 3 → [階段 4](#phase-4-permission-tree-reconstruction)
8. [ ] **階段 5**：合併輸出 + 報告 + `insert_assets` → [階段 5](#phase-5-merge-and-report)

---

<a id="phase-0-classification"></a>

## 階段 0：分類

取得入口 HTML 並建立 `OUTDIR`；勿修改技能的 `scripts/`：

- **SPA**：空外殼 + `<div id=app>` + chunk → 階段 1–5。
- **MPA**：SSR + `<form>`，無端點 bundle；閘門 A 後：



```bash
python3 recon/spider_mpa.py <BASE_URL> <OUTDIR> [--cookie "session=..."] [--max 300] [--depth 5] [--exclude "logout|delete|destroy"]
```



輸出：`forms.txt`、`links.txt`、`api_inline.txt`。SPA 表單約為零時，轉階段 1。

---

<a id="phase-1-static-discovery"></a>

## 階段 1：靜態探索

遵循[指令碼與閘門](#scripts-and-gates)及[工具／輸出限制](#tool-and-output-limits)。



```bash
python3 recon/harvest_static.py <BASE_URL> <OUTDIR>
```



Harvest 剖析 HTML script 與 webpack／Vite manifest，下載全部延遲 chunk，輸出 `js/`、`api_static.txt`、`routes.txt`、`chunkmap.txt`。



```bash
wc -l OUTDIR/api_static.txt OUTDIR/routes.txt
ls OUTDIR/js | wc -l
```



- 比較 chunk 數與 manifest；404 時修正 harvest 並重試，勿逐個 curl。
- `api_static.txt` 太少：擴大 OUTDIR 端點 regex 並重跑，詳見參考。

<a id="phase-1b-parameter-reconstruction"></a>

### 階段 1b：參數重建

路徑來自階段 1；參數欄位另行調查，遵守 [grep 輸出限制](#tool-and-output-limits)。

完成標準：重要 API 須識別欄位名、傳輸位置、推斷型別、必填狀態、範例值與信心程度。

<a id="1b0-transport-forms"></a>

#### 1b.0：傳輸形式

| 形式 | 參數位置 | 靜態證據 |
|---|---|---|
| REST JSON | body + query | 路徑錨點附近 `(params\|data\|body)\s*:\s*\{` |
| GraphQL | `variables` | gql 範本、`$page: Int` |
| 傳統表單 | urlencoded | `<form>`、`FormData` |
| 檔案上傳 | multipart | `FormData.append` |
| 路徑參數 | `/user/:id` | 路由表 + `useParams` / `$route.params` |
| 加密／簽章 | 封裝於 `sign`／`data` | Hook 加密函式輸入（參考 D） |

輸出：各 API 標註 `transport: query|json|form|graphql|encrypted`。

<a id="1b1-expand-anchor-windows"></a>

#### 1b.1：擴展錨點視窗

在已知路徑周圍擴展，尋找 payload 組裝：



```bash
grep -n '"/api/user/list"' OUTDIR/js/*.js | head -20
grep -rhoaE '.{0,120}("/api[^"]+").{0,200}' OUTDIR/js/*.js | head -20
grep -rhoaE '(params|data|body|payload)\s*:\s*\{' OUTDIR/js/*.js | head -20
```



| 包裝層 | 參數線索 |
|---|---|
| axios 執行個體 | `data` / `params` |
| 共用請求包裝層 | 攔截器注入的全域欄位 |
| OpenAPI 用戶端 | 生成的方法簽章 |
| React Query / SWR | 第二個 hook 參數 |
| Vue composable | composable 參數 |

型別線索：`yup`／`zod`／rules、`Form.Item name=`、內嵌 Swagger。

→ `param_candidates.json`：`{ path, fields[], source: "static-callsite", confidence }`

<a id="1b2-binding-layer"></a>

#### 1b.2：綁定層



```
Form field → onFinish/handleSubmit → transform → API payload
```



| 綁定來源 | 方法 |
|---|---|
| 表單提交 | 追蹤 submit → transform → API |
| 表格搜尋 | `getFieldsValue()` → `params` |
| 路由 | `:id` / `?tab=` |
| 攔截器 | 全域 `tenantId`、分頁、sign |
| 列舉選單 | `options` → API 列舉值 |

從 `fetch`／`XHR.send` 向上追蹤 DevTools 呼叫堆疊至 payload 組裝器。

<a id="1b3-three-payload-questions-distinct-from-phase-2-auth-gates"></a>

#### 1b.3：三個 payload 問題（不同於階段 2 驗證閘門）

| 問題 | 必須回答 |
|---|---|
| 組裝 | payload 在何處建立與轉換 |
| 驗證 | required、pattern、enum |
| 傳輸 | path / query / body / multipart / headers |

攔截器閘門（階段 2）亦需檢查全域注入欄位（Authorization、`X-Tenant-Id`、sign）。

<a id="1b4-connect-to-phase-3"></a>

#### 1b.4：銜接階段 3

候選欄位來自靜態／綁定分析。必填、選填與條件相依需階段 3 矩陣／差異及階段 5 錯誤推斷。

---

<a id="phase-2-three-authentication-gates"></a>

## 階段 2：三個驗證閘門

以 `head` 限制 grep `OUTDIR/js/`，將結果記入 `config.json`，見參考配方：

| 閘門 | 問題 | 關鍵字 |
|---|---|---|
| 轉譯 | 如何判定登入狀態？ | `isLogin`、`getToken`、Cookie／localStorage |
| 攔截器 | 什麼觸發 `/login` 重新導向？ | `response_code`、`errno`、axios interceptor |
| 內容 | 選單／權限從哪裡來？ | `menu`、`permission`、`role`、`acl`、`routes` |

勿將 localStorage 鍵名當作憑證；須透過 chunk／請求鏈驗證。

出口為閘門 B：結論寫入 `config.json`，調整 `OUTDIR/runtime_harvest.js`／`preload.js`。

<a id="phase-2b-api-observation-optional"></a>

### 階段 2b：API 觀察（選用）

以 OUTDIR 的 `preload.js` 識別工作階段鍵、Authorization 與巢狀 API URL：

| 設定 | 輸出 |
|---|---|
| `recordDetail: true` | `__API_RECON_DETAIL__` |
| `observe.xhrHeaders: true` | 標頭觀察 |
| `extractUrlsFromResponse: true` | 回應中的巢狀 API |
| `observe.storageReads/cookieReads: true` | 更新設定 |
| `neutralizeVueRouter: true` | `__API_RECON_ROUTES__` |

匯出每輪 coverage：`__API_RECON_LOG__`、`__API_RECON_DETAIL__`、`__API_RECON_ROUTES__`、`__API_RECON_OBSERVE__`。

---

<a id="phase-3-runtime"></a>

## 階段 3：執行時

先通過閘門 B，遵守[邊界](#boundaries-and-prohibitions-required-reading)與無憑證 mock 策略。

設定 `config.json` 的 `"runtimeMode": "depth" | "coverage" | "both"`；範本見參考。

<a id="hooks-and-stubs-shared-by-depth-and-coverage"></a>

### Hook 與 stub（depth／coverage 共用）

| 層級 | 範圍 | 目的 |
|---|---|---|
| L1 精確 | auth／permission／bootstrap stub | 通過初始轉譯驗證 |
| L2 失敗結果修正 | 全部 JSON 回應 | 未登入代碼 → 成功 |
| L3 備援 | L1 未比對的 `/api` 等 | 空成功本文以呈現介面 |

- **depth**：假驗證 + `forward` 業務代碼修補 + `stubs`；遍歷 `routes`（hash／history），輸出 `runtime_api.json`。
- **coverage**：文件開始時注入 `preload.js`（CDP `addScriptToEvaluateOnNewDocument` 或 userscript）。

驗證 `window.__API_RECON_PRELOAD__` 存在，業務路徑不會回到 `/login`。



```bash
cd recon && npm install
node runtime_harvest.js config.json
```



<a id="3b-coverage-dynamic-enumeration-required"></a>

### 3b：coverage 動態列舉（必要）

1. 主導覽／側欄：逐項點選，等待網路 1–3 秒。
2. 分頁：`role=tab`、`.ant-tabs-tab`。
3. 表格：第一列查看／編輯／詳情。
4. 工具列：匯出、篩選、新增（避免不可逆刪除；全部禁令仍適用）。
5. 各模組：合併 API／路由。
6. SPA：對 `routes.txt` 未覆蓋路徑受控使用 `pushState`；MPA 禁止。

必要參數觸發矩陣：依模組記錄各操作類型並比較多個樣本：

| 操作 | 常見新增參數 |
|---|---|
| 初始清單 | 分頁 + 預設篩選 |
| 搜尋 | keyword、filter |
| 進階篩選 | 更多選填欄位 |
| 新增／編輯 | 完整實體 |
| 批次／匯出／排序 | `ids[]`、`exportType`、`sortField` |

Stub 下送出本文／標頭仍為真實值，應以請求作證據。記錄 `scan_raw.json`、`param_samples.json`、`api_detail.json`。

- **Vue**：`neutralizeVueRouter: true` + 文件開始 preload。
- **React**：`routes.txt` + 側欄點選 + `pushState`。
- **both**：先 3a depth，再 3b coverage。

---

<a id="phase-4-permission-tree-reconstruction"></a>

## 階段 4：權限樹重建

觸發條件：模組頁面空白，或所有路由只有 locale 等啟動請求，表示內容閘門未通過。

| 症狀 | 含義 |
|---|---|
| 已進入外殼 | 轉譯與攔截器閘門通過 |
| 側欄缺項／點選空白 | Stub 形狀或權限代碼不完整 |
| 各路由僅少量相同 API | `v-if permission` 失敗 |
| `routes.txt` 遠少於 bundle 路由 | 從驗證模組補全 |



```bash
grep -rhoaE '"/api[^"]*(permission|perm|role|menu|acl)[^"]*"' OUTDIR/js/*.js | sort -u | head -30
grep -rhoaE 'userRouteAuth|getResultTree|routeMap|routeLink|menuList|authList' OUTDIR/js/*.js | head -20
```



典型鏈：`role_permissions`（平面代碼）+ `permissions/all`（樹）→ `getResultTree` → `userRouteAuth[CODE].url`。



```bash
python3 recon/extract_route_map.py recon/js recon/
python3 recon/build_perm_tree.py recon/js recon/ --config recon/config.json
```



中間輸出：`route_map.json`、`userRouteAuth.json`、`permissions_tree.json`、`*_stub.json`、`perm_codes_all.txt`。

檢查 stub：外層 `response_code` 符合攔截器閘門；平面代碼符合樹；`routes` 涵蓋每個 `route_map` 連結。

更新 `config.json` 後重跑階段 3。大型 SPA 可調整 `waitUntil`、`routeTimeout`、`perRouteMs`（參考 A3／I）。

---

<a id="phase-5-merge-and-report"></a>

## 階段 5：合併與報告

<a id="outputs"></a>

### 輸出

| 檔案 | 階段 | 內容 |
|---|---|---|
| `js/`, `api_static.txt`, `routes.txt`, `chunkmap.txt` | 1 | 靜態 bundle 與路徑 |
| `param_candidates.json` | 1b | 靜態參數候選 |
| `config.json` | 2 | 三閘門 + 執行時設定 |
| `runtime_api.json` | 3a | depth 詳細擷取，含 WS／SSE |
| `param_samples.json`, `scan_raw.json`, `api_detail.json` | 3b | 多樣本、點選記錄、詳情 |
| `route_map.json` 等 | 4 | 使用時的權限樹中間檔 |
| `params_merged.json` | 5 | 合併參數欄位 + 信心程度 |
| `api_merged.txt` | 5 | `METHOD /path [params] [static\|runtime\|both]` |
| `site_map.json` | 5 | 路由、API、參數、功能、限制 |
| **insert_assets** | 5 | 儲存全部服務與端點資產 |

<a id="5b-merge-parameters"></a>

### 5b：合併參數

比較 `param_samples.json`；沒有通用合併指令碼。信心規則見參考 J7（高／中／低／待觸發）。

<a id="5c-infer-from-errors"></a>

### 5c：從錯誤推斷

在授權範圍內，可用不完整請求檢查 400 錯誤來探索參數，不作漏洞測試，例如 `field 'x' is required`、列舉錯誤等。須考量 `data` 包裝、`variables` 與加密前 `bizData`。

報告 runtimeMode、靜態／執行時 API 數、參數信心、未覆蓋模組，以及相對範本的 `CHANGES.md` 摘要。

建議的 `site_map.json`：



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



更多欄位與 grep 配方見 [reference.md](api-recon-reference.md)。

---

<a id="general-notes"></a>

## 一般注意事項

- 框架無關：webpack／Vite／Angular 延遲載入採相同方法。
- 傳輸：REST／JSON、GraphQL、WebSocket、SSE；不含 gRPC-web。
- SSR：可擷取用戶端 fetch；無法完整列舉 RSC／Server Actions。
- 盲點：JSVMP、WASM、嚴格 HMAC／mTLS → 靜態分析並明示限制。
- 參數盲點：條件互動、隱藏參數、WASM 組裝 → 待觸發／無法到達。
- 靜態備援：執行時受阻時仍可探索端點。

---

<a id="additional-resources"></a>

## 其他資源

- Grep 配方、`config.json` 範本、排錯、hook、參數重建（J）、site_map 範本：**[reference.md](api-recon-reference.md)**。
- 範本路徑：[指令碼與閘門](#scripts-and-gates)。
