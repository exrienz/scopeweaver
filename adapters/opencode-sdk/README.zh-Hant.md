# OpenCode SDK 橋接器

[English](README.md) · [한국어](README.ko.md) · [简体中文](README.zh-Hans.md) · [繁體中文](README.zh-Hant.md) · [日本語](README.ja.md) · [Español](README.es.md)

可選的僅限迴路地址的橋接器，支持固定的 **OpenCode 原生 CLI／客戶端／SDK 2.1.4**，以及單獨測試的 **舊版 SDK／CLI 1.19.4**。ScopeWeaver 繼續管理對話、工具執行、審批、範圍檢查、重試與壓縮。橋接器只流式傳送一次模型步驟並收集工具請求，不執行 ScopeWeaver 工具或 OpenCode 內建工具。現有 API 與 OpenCode Go 供應商無需此橋接器。

## 啓動

已測試的 CI 運行時是 **Bun 1.3.14**；更新版本尚未驗證。

```sh
cd adapters/opencode-sdk
bun install --frozen-lockfile --ignore-scripts
export SCOPEWEAVER_OPENCODE_TOKEN="$(openssl rand -hex 32)"
bun run start
```

橋接權杖必須至少包含 **32 個非空白字元**。原生私有 CLI 伺服器使用每次呼叫獨立產生的密碼（`OPENCODE_PASSWORD`）與 HTTP Basic 驗證；該密碼、橋接權杖與上游供應商金鑰彼此獨立。

在 ScopeWeaver 供應商設定中選擇 `opencode-sdk`，地址設為 `http://127.0.0.1:4097`，填入上述**橋接權杖**。它不是雲端 API 密鑰；不要復用 OpenCode Go／Console 權杖。讀取模型後選擇準確的 `providerID/modelID`。

預設運行時不登錄賬戶，使用隔離的目錄與公開模型發現。發現只發送公開目錄 GET 請求，**不執行推理**；僅顯示當前公開列表中、固定運行時識別的零價格 OpenCode 模型。目錄錯誤會明確返回。列入目錄不代表具備資格、可用配額或推理成功；免費模型可能改變或消失。預設不啓用付費模型或憑證回退。

**2026-10-25（KST）的先前驗證：** `opencode/deepseek-v4.1-flash-free` 曾通過產品的原生 CLI／客戶端 2.1.4 橋接器完成真實 ScopeWeaver 對話。三次實際推理覆蓋界面連接測試、文件產生影響前經審核批准的宿主工具請求，以及界面最終答案 `FREE_OK`。模型來自預設零價格目錄，未使用 API 密鑰、賬戶導入、模型／響應模擬資料、身份偽裝或付費回退。原生工具保持阻止狀態，宿主操作由 ScopeWeaver 執行。這是該日期的既有證據，不是本次重跑結果。

原生推理使用真正的固定版本 CLI 和官方客戶端 API，無推理的模型發現使用內嵌 SDK。此前僅收集工具的內嵌 SDK 路徑收到 HTTP 403 `FreeTierError`。舊版的真實托管免費訪問尚未驗證。模型及服務資格可能變化；該日期的測試不保證數值配額或持續可用性。結構化拒絕返回 `free_tier_restricted`，不切換到付費模型。

`SCOPEWEAVER_OPENCODE_PORT` 可修改預設端口 4097；`SCOPEWEAVER_OPENCODE_HOST` 必須為 `127.0.0.1`。拒絕瀏覽器 `Origin` 請求及非迴路 URL。所有 API 路由均要求權杖；請像密碼一樣保護它。

`SCOPEWEAVER_OPENCODE_TIMEOUT_MS` 的完整調用時限為 **0–86,400,000 毫秒**；預設 **0** 由宿主控制取消，沒有隱藏的兩分鐘推理限制。取消與關閉另有 **1.5 秒**強制終止期限。公開目錄請求限制為 **10 秒**。請求體上限 **16 MiB**，單個工具參數 JSON 上限 **2,000,000 字元**，原生模型文本／思考／參數合計上限 **8,000,000 字元**，工作進程輸出上限 **20,000,000 位元組**。超限明確失敗，不截斷資料。

## 專用供應商設定

自定義 OpenAI 兼容模型或本地測試模型可通過 `SCOPEWEAVER_OPENCODE_PROVIDER_CONFIG` 指定運營者管理的 JSON 文件。放在簽出目錄之外，或使用 Git 忽略的名稱 `provider-config.local.json`：

```json
{
  "providers": {
    "local": {
      "name": "Local model",
      "package": "@opencode/ai/providers/openai-compatible",
      "settings": {
        "baseURL": "http://127.0.0.1:8000/v1",
        "apiKey": "explicit-provider-key-if-needed"
      },
      "models": {
        "my-model": {
          "name": "My model",
          "capabilities": { "tools": true },
          "limit": { "context": 200000, "output": 32000 },
          "compatibility": { "maxTokensField": "max_tokens" }
        }
      }
    }
  }
}
```

選擇 `local/my-model`。文件會替代該橋接器的預設模型發現，不載入用戶／項目的 OpenCode 設定、`.env`、插件、MCP 服務器或已保存憑證。僅將明確設定的 `apiKey` 傳給對應供應商。遠程地址可能收取推理費用；添加設定不會使其免費。任意 SDK 套件和額外設定字段均被拒絕。

## 運行邊界

每次調用使用獨立的 Bun 工作進程、允許列表中的環境變量及自有臨時 home／設定／資料路徑。原生模式擁有私有官方 CLI 服務器、實際 CLI `run`、臨時資料庫和一次性認證的迴路出口傳輸服務；官方客戶端處理髮現、工作階段事件和中斷。不繼承雲端憑證、用戶認證文件、代理或預載入設置。直接使用固定的可選平台二進制，因此支持 `--ignore-scripts`，無需全局安裝；**不要省略可選依賴**。

保留 OpenCode 的真實內建工具定義及原始實現。所有原生權限為 ASK，CLI 不使用 `--auto`。公開的 `execute.before` 策略鈎子獨立阻止所有非宿主工具在執行前運行，包括生成的 `execute`。原生工具請求使橋接請求失敗，不會變成 ScopeWeaver 工具請求。只有註冊的宿主收集函數可暫停並成為宿主工具請求；移除其他代理。無衝突的 `sw_tool_N` 別名防止宿主工具覆蓋內建定義，描述保留原始名稱與內容，響應恢復原始宿主名稱。

宿主 system 片段以兩個換行連接，原樣寫入公開的自定義代理 system 設置；同時保留 OpenCode 生成的模型／環境／工具上下文。最後的策略說明只有標記 `Host tool:` 的函數可用。通過公開 context 鈎子保留宿主角色、工具 ID／結果及每次調用的限制／設置，不保證 system 數組只含宿主片段。按模型傳遞 thinking／effort 字段，不支持的設置明確失敗；模型聲明的較小輸出限額會限制請求上限。輸入、輸出及快取用量分開記錄。

阻止第二次模型請求。私有出口將實際 CLI 生成的請求轉發到原始上游 URL，拒絕重定向，不替換地址、模型、身份或憑證。只有中斷、刪除工作階段及清理自有進程完成後才返回宿主工具請求。斷連、DELETE、逾時和關閉終止 POSIX 工作進程組，並在 **1.5 秒**後強制終止。該進程樹清理邊界未在 Windows 實現，**原生和舊版推理均不支持 Windows**。

CLI 的公開設置不能完全關閉系統／MDM 設定發現。macOS 上存在 `/Library/Application Support/opencode` 或用戶／全局 `/Library/Managed Preferences` 下的 `ai.opencode.managed.plist`，或 Linux 上存在 `/etc/opencode`，原生模式便以 `configuration_error` 拒絕啓動。無法訪問的路徑和損壞符號鏈接也明確拒絕；不讀取或修改這些來源。

原生模式在完成響應、流響應和宿主工具歷史重放中保留原始工具參數 JSON，包括 64 位整數；不執行 JavaScript 解析後的投影。原始參數之外的不安全數值、不支持的內容、未知原生工具、重復工具 ID 和格式錯誤響應均拒絕。工具 schema 支持內聯定義、本地 JSON Pointer `$ref`／`$defs` 與聯合類型；拒絕遠程引用、`$id`、`$dynamicRef` 和無法解析的引用。初始化仍會載入原生供應商／目錄代碼；這是應用邊界，**不是操作系統沙箱**。迴路橋接器仍可調用遠程模型，因此不滿足 ScopeWeaver 的僅本地推理策略。

## 版本與舊版限制

| `SCOPEWEAVER_OPENCODE_RUNTIME` | 固定組合 | 能力響應 `runtime` |
| --- | --- | --- |
| `native`（預設） | 官方 CLI、`@opencode/client`、SDK **2.1.4** | `native` |
| `legacy` | `@opencode-ai/sdk/v2`、CLI **1.19.4** | `legacy` |

啓動前設置 `SCOPEWEAVER_OPENCODE_RUNTIME=legacy` 可選擇舊 API。兩個組合的先前本地實際運行驗證範圍為 **macOS arm64**；Linux CI 已設定實際運行測試，但托管運行結果尚待驗證。其他 SDK 版本在推理前被拒絕，不保證任意版本或平台兼容。

舊版直接運行匹配的官方可選平台二進制，支持 `--ignore-scripts`，不調用未初始化的包裝器或全局 `opencode`。不要省略可選依賴。其 CLI 與迴路 MCP 收集服務器隨請求工作進程的 POSIX 進程組終止；Windows 不支持此清理邊界。每次調用使用獨立插件套件副本及鎖文件，不建立到共享依賴的可寫鏈接。

舊版只支持結構有效、可安全表示的 JSON／歷史。空白及等價 HTML／Unicode 轉義可能被規範化，不承諾原始 JSON 位元組。拒絕絕對值超過 **9,007,199,254,740,991** 的不安全整數、不透明／帶簽名的思考歷史及不支持的供應商覆蓋設置；需要原始 64 位參數保真時使用原生模式。舊版自定義供應商使用上面的顯式 OpenAI 兼容設定，但只映射 `max_tokens` 兼容字段、支持的工具能力字段及上下文／輸出限額；拒絕不支持的 `reasoningField` 和能力覆蓋。超出任一 SDK 模式邊界時，可繼續使用 API／Go 設定。

## 協議與驗證

協議為 **`scopeweaver-opencode-sdk`，版本 1**。認證路由：`GET /v1/capabilities`、`GET /v1/models`、`POST /v1/complete`、`POST /v1/stream`、`DELETE /v1/requests/{request_id}`。流使用 NDJSON，傳遞真實原生文本／思考增量、收集的工具請求及最終用量；成功時只發送一次 `message_stop`。錯誤使用有限機器代碼並隱藏供應商消息。只有明確結構化的計費／配額耗盡才返回 `quota_exhausted`（完成請求 HTTP 402）；普通限流、認證及 5xx 不會改標為配額錯誤。結構化上下文溢出返回 `context_length_exceeded`（HTTP 400），保留宿主壓縮信號；普通 400 或消息關鍵詞不足以分類。結構化 `FreeTierError` 返回 `free_tier_restricted`（HTTP 403），與配額及普通禁止響應分開。橋接器不選擇後備模型；後續行為由宿主設定的策略決定。

完成請求示例（流端點接受相同正文）：

```json
{
  "request_id": "unique-call-uuid",
  "session_id": "stable-host-conversation-uuid",
  "model": "local/my-model",
  "system": ["Host system instructions"],
  "messages": [{ "role": "user", "content": [{ "type": "text", "text": "Hello" }] }],
  "tools": [],
  "max_tokens": 64,
  "thinking": "disabled"
}
```

可選字段為 `session_id`、`temperature`、字元串數組 `stop`、`thinking`、`thinking_type`、`reasoning_effort`。宿主內容塊支持 `text`、助手 `thinking`／`tool_use` 及帶文本結果的用戶 `tool_result`。工具聲明使用 `name`、`description`、`input_schema`。完成響應包含 `message`、`stop_reason`（`end_turn`、`tool_use` 或 `max_tokens`）及分離輸入／輸出／快取讀取／快取寫入計數的 `usage`。公開契約與限額見 [protocol.mjs](src/protocol.mjs) 和 [bridge.mjs](src/bridge.mjs)。

對於 `opencode-go`，`session_id` 使每次調用的新 SDK 工作階段保持相同 `x-opencode-session`；缺省時使用 `request_id`。`opencode` 保留真實運行時生成的工作階段及身份標頭。普通供應商不會收到網關專用工作階段標頭。保留真實 CLI 的 User-Agent。

```sh
bun run check
bun test --timeout 30000
```

測試以固定的實際 CLI／客戶端／SDK 和本地 SSE 模型驗證無推理髮現、宿主歷史／設置、收集安全性、流／用量、原始數值參數、惡意原生調用、並行隔離及取消；不證明真實托管免費模型的使用資格。

## ScopeWeaver 設定與降級邊界

空權杖僅在相同 SDK 地址保留已保存值。更改地址或格式不能復用舊權杖；`localhost`、`127.0.0.1`、`::1` 是不同憑證作用域。SDK 模板清除未保存的雲端密鑰、代理和自定義工作階段標頭，模型留空，不發現、不推理、不保存、不啓用。明確點擊**讀取模型**才發現模型；**測試連接**僅在點擊時執行推理。

設定資料庫 `CHECK` 約束增加 `opencode-sdk`；仍有 SDK 設定時，舊二進制無法啓動。降級前備份資料庫，並在當前版本中轉換為舊版本支持的格式或移除 SDK 設定。此前遷移驗證只使用臨時資料庫，沒有遷移生產資料庫。詳見[供應商指南](../../docs/zh-Hant/llm-providers.md#opencode-sdk-bridge)。
