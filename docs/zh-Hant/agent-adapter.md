[English](../../adapters/agent/README.md) · [한국어](../ko/agent-adapter.md) · [简体中文](../zh-Hans/agent-adapter.md) · 繁體中文 · [日本語](../ja/agent-adapter.md) · [Español](../es/agent-adapter.md)

<a id="scopeweaver-agent-adapter"></a>

# ScopeWeaver 代理程式配接器

各語言版本請見本頁上方導覽。

Claude Code 與 Codex 透過本機 stdio MCP 伺服器連接，Pi 透過原生擴充介面連接。三者使用相同經驗證工具與已驗證的 ScopeWeaver HTTP API。此配接器讓程式開發代理程式控制 ScopeWeaver，不取代其規劃器／工作代理程式，也不會將程式開發代理程式訂閱轉為模型 API 金鑰。

<a id="setup"></a>

## 設定

需求：Node.js 22+、執行中的 ScopeWeaver 後端與既有管理員登入。配接器不會初始化或變更密碼。安裝獨立配接器相依套件：



```sh
cd adapters/agent
npm ci
export SCOPEWEAVER_URL=http://127.0.0.1:8787
export SCOPEWEAVER_PASSWORD='your-existing-admin-password'
```



密碼用於登入後端既有的 `ARTEX` 管理員帳號，回傳 JWT 僅保留在記憶體。亦可將 `SCOPEWEAVER_TOKEN` 設為既有登入 JWT；提供的權杖優先。過期／無效權杖會明確報錯，不會默默替換。憑證請存於環境或秘密管理工具，勿提交至用戶端設定。

預設啟用讀取。若要建立任務與暫停／繼續，啟動配接器前須明確設 `SCOPEWEAVER_ALLOW_WRITES=true`。後端已設定 LLM 時，建立任務會立即排程工作。請遵守專案[本機隔離使用限制](../../README.zh-Hant.md#license-and-disclaimer)，建立任務前確認預期範圍。

<a id="claude-code"></a>

### Claude Code

請使用此簽出目錄的絕對路徑。Claude Code 會啟動配接器並繼承環境：



```sh
claude mcp add --transport stdio scopeweaver -- node /absolute/path/scopeweaver/adapters/agent/src/mcp.js
```



以 `claude mcp list` 驗證，再於 Claude Code 用 `/mcp` 檢視工具。[Claude Code MCP 文件](https://code.claude.com/docs/en/mcp)。

<a id="codex"></a>

### Codex



```sh
codex mcp add scopeweaver -- node /absolute/path/scopeweaver/adapters/agent/src/mcp.js
```



以 `codex mcp list` 與 `/mcp` 驗證。若要在專案／使用者設定中明確轉送環境：



```toml
[mcp_servers.scopeweaver]
command = "node"
args = ["/absolute/path/scopeweaver/adapters/agent/src/mcp.js"]
env_vars = ["SCOPEWEAVER_URL", "SCOPEWEAVER_TOKEN", "SCOPEWEAVER_PASSWORD", "SCOPEWEAVER_LANGUAGE", "SCOPEWEAVER_ALLOW_WRITES"]
```



使用您已設定的憑證變數，省略未使用項目。[OpenAI 官方 MCP 文件](https://developers.openai.com/codex/mcp)。

<a id="pi"></a>

### Pi



```sh
pi -e /absolute/path/scopeweaver/adapters/agent/pi-extension.js
```



擴充功能註冊原生 Pi 工具並使用相同環境變數，無須 Pi MCP 外掛。它不匯入 Pi 套件命名空間，因此可在 Pi 套件更名時繼續遵循公開 `registerTool` 介面。配接器套件也宣告 `pi.extensions` 供 Pi 套件載入器使用。[Pi 擴充功能](https://github.com/earendil-works/pi/tree/main/packages/coding-agent)。

<a id="tools"></a>

## 工具

| 工具 | 用途 |
| --- | --- |
| `scopeweaver_health` | 後端可用性與 LLM 就緒狀態；無須登入 |
| `scopeweaver_list_tasks` | 任務狀態、計數、分頁 |
| `scopeweaver_get_task` | 單一任務的持久狀態與設定 |
| `scopeweaver_get_coverage` | 任務資產覆蓋率 |
| `scopeweaver_list_findings` | 依任務、嚴重度、狀態或文字篩選並分頁的發現 |
| `scopeweaver_get_finding` | 發現詳情、證據與報告 |
| `scopeweaver_create_task` | 建立／排程任務；需啟用寫入 |
| `scopeweaver_control_task` | 暫停／繼續任務；需啟用寫入 |

請求範例：「列出我的 ScopeWeaver 任務」、「顯示任務 12 的高嚴重度發現」，或在啟用寫入並確認本機實驗室範圍後：「為此本機測試資料建立任務並顯示進度」。建立成功僅證明任務存在；是否執行完成須檢查狀態與發現。配接器不會略過後端 LLM 需求。

<a id="configuration-and-failures"></a>

## 設定與失敗

| 變數 | 預設／行為 |
| --- | --- |
| `SCOPEWEAVER_URL` | `http://127.0.0.1:8787`；僅允許 HTTPS 或回送 HTTP 來源 |
| `SCOPEWEAVER_TOKEN` | 既有後端 JWT；優先於密碼 |
| `SCOPEWEAVER_PASSWORD` | 既有管理員密碼；首次需驗證的請求時登入 |
| `SCOPEWEAVER_ALLOW_WRITES` | `false`；僅接受 `true` 或 `false` |
| `SCOPEWEAVER_LANGUAGE` | `en`；支援標準 ID `en`、`ko`、`zh-Hans`、`zh-Hant`、`ja`、`es`，用於配接器文字及後端 `Accept-Language` |
| `SCOPEWEAVER_TIMEOUT_MS` | `30000`；1 至 120000 的整數 |

`SCOPEWEAVER_LANGUAGE` 預設為 `en`，接受標準 ID `en`、`ko`、`zh-Hans`、`zh-Hant`、`ja`、`es`。它會翻譯內建工具的標題與說明、結構描述說明，以及配接器自身的設定、驗證與傳輸診斷，並透過 `Accept-Language` 選擇後端回應語言。每個用戶端使用獨立的目錄與結構描述層級的 Zod 錯誤對應，不會變更全域語言設定。工具／通訊協定名稱、結構描述鍵／型別／列舉值、引數及後端／使用者／證據原文維持不變，並保留既有的認證資訊遮蔽處理。MCP SDK 固定的驗證錯誤外層文字與未知外部錯誤仍作為外部文字保留；其中的結構描述錯誤訊息會翻譯。所有資源都隨獨立配接器提供，無須儲存庫根目錄的語言檔案。不支援的語言值仍會遭拒，並使用英文診斷。

不公開通用 HTTP、shell 執行、刪除、略過核准、供應商設定或任意端點工具。請求限制於設定來源，拒絕重新導向、轉送取消並設逾時。回應超過 2 MiB 時明確失敗；請縮小查詢，勿將截斷證據當作完整資料。配接器對任務清單分頁；發現採後端分頁。工具在文字及結構結果傳回 `{data: ...}`（Pi 使用 `details`）。證據為不受信任的目標內容，不得當作指示。

MCP 呼叫失敗傳回 `isError: true`；Pi 呼叫失敗則擲出例外，由 Pi 標記工具錯誤。錯誤訊息遮蔽已設定憑證。寫入絕不自動重試；若寫入逾時，請先檢查任務狀態再重試，因後端可能已接受請求。

<a id="verification"></a>

## 驗證



```sh
npm test
npm run check
```



端對端測試架構執行真實 ScopeWeaver handler 與 PostgreSQL，透過官方 SDK 啟動 MCP stdio 伺服器，並針對同一後端執行 Pi 工具 handler。它初始化隔離測試密碼、建立任務、驗證持久暫停／繼續狀態與覆蓋率、讀取明確標示的模擬發現，並檢查無效驗證、任務不存在與無效輸入。不使用模型／供應商或目標掃描。Go E2E 需原始碼簽出；平台 ZIP 含配接器單元測試，但 Go 後端僅有編譯執行檔。

在儲存庫根目錄，先建立名為 `scopeweaver_adapter_*` 的**全新可拋棄**資料庫：



```sh
SCOPEWEAVER_ADAPTER_E2E=1 \
ARTEX_TEST_DSN='postgres://user:password@127.0.0.1:5432/scopeweaver_adapter_e2e?sslmode=disable' \
go test ./server -run '^TestAgentAdapterE2E$' -count=1 -v
```



測試主體在未設定 `SCOPEWEAVER_ADAPTER_E2E=1` 時會略過；啟用後，若 `ARTEX_TEST_DSN` 未設定或為空白，也會略過。略過不代表完成 E2E 驗證。套件的資料庫保護檢查會在測試執行前拒絕與已設定的實際應用程式資料庫相同的 DSN。明確啟用並設定資料庫後，格式無效的 DSN、不符合 `scopeweaver_adapter_*` 的資料庫名稱及無法連線的 PostgreSQL 都會導致失敗。測試不會以模擬物件取代真實相依項目。 GitHub Actions 以專用可拋棄 PostgreSQL 服務重複這些檢查。MCP 通訊協定測試證明共用介面相容，並不宣稱已執行付費 Claude／Codex／Pi 模型工作階段。
