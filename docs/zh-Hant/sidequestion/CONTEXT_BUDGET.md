[English](../../../sidequestion/CONTEXT_BUDGET.md) · [한국어](../../ko/sidequestion/CONTEXT_BUDGET.md) · [简体中文](../../zh-Hans/sidequestion/CONTEXT_BUDGET.md) · 繁體中文 · [日本語](../../ja/sidequestion/CONTEXT_BUDGET.md) · [Español](../../es/sidequestion/CONTEXT_BUDGET.md)

<a id="long-side-conversations-and-context-budgets"></a>

# 長側問對話與上下文預算

各語言版本請見本頁上方導覽。

上游歷史修正，2026-09-11。原實作將請求 JSON 字元當作 token，並繼承主任務 32K 輸出預留，因此當 HTML、JavaScript 與工具結果佔用大量上下文時，過早拒絕有效側問。

<a id="open-source-implementations-reviewed"></a>

## 已檢視的開源實作

- [Grok CLI 側問上下文](https://github.com/superagent-ai/grok-cli/blob/fb97af83f06dca873281d60168430f06c8de6324/src/agent/agent.ts#L739)：擷取最近使用者／助理文字，預算約 2000 字元、每則最多 400 字元。此路徑不維護持續側問歷史。
- [Grok CLI 獨立請求](https://github.com/superagent-ai/grok-cli/blob/fb97af83f06dca873281d60168430f06c8de6324/src/utils/side-question.ts)：獨立取消訊號、支援時採 2048 token 輸出上限、不含工具。
- [Grok CLI 主工作階段壓縮](https://github.com/superagent-ai/grok-cli/blob/fb97af83f06dca873281d60168430f06c8de6324/src/agent/compaction.ts)：估算 token、保留近期內容、以新資料更新舊摘要，並處理跨回合截斷。
- [OpenCode 工作階段壓縮](https://github.com/anomalyco/opencode/blob/b3f1a96c6dd7adeb28b36dd11add1998fc84d67b/packages/core/src/session/compaction.ts)：估算完整請求、預留輸出／緩衝空間、結合近期內容與滾動摘要，摘要請求不含工具。該實作預設近期上下文 8000 token、摘要輸出 4096 token。
- [OpenCode 溢位復原](https://github.com/anomalyco/opencode/blob/b3f1a96c6dd7adeb28b36dd11add1998fc84d67b/packages/core/src/session/runner/llm.ts)：僅在助理開始輸出前嘗試復原，復原呼叫不再次進入同一路徑。

ARTEX 借用獨立輸出預算、近期內容加滾動摘要及有界復原，保留 norma v0.3.6 結構化訊息與工具配對，而非 Grok 文字摘錄。不將 OpenCode 式主要工作階段壓縮事件寫入 ARTEX 主要對話記錄。ScopeWeaver 保留此設計；引用版本描述歷史實作。

<a id="request-budget-and-execution"></a>

## 請求預算與執行

- 訊息使用 norma 內容區塊 UTF-8 位元組估算，加 4/3 裕度，再加系統指示、工具結構與訊息封裝開銷。這是估算，不是精確模型 token 數。
- 側問輸出預設最多 8192 token，絕不超過主設定輸出上限。`ARTEX_BTW_MAX_OUTPUT_TOKENS` 接受 256 至 32768 的伺服器端上限，不變更預設模型或主任務參數。
- 輸入預算為上下文視窗減輸出上限及安全裕度。未知視窗採平台預設 200K。安全裕度為視窗 5%，限制在 128–8192 token。
- 成功交換依遞增序位載入，每批最多 20 次。最多保留 20 次原始交換，且不超過輸入預算四分之一或 16K token。
- 超額交換更新滾動摘要，保留歷史來源與上下文時間。舊助理回答不是新的工具證據；衝突時以最新主要快照優先。
- 主要上下文仍過長時，僅摘要副本中的舊訊息，保留最多 8K 近期 token。切分點保留工具呼叫／結果配對；過大群組整組納入摘要。
- 摘要輸入依剩餘視窗空間在 UTF-8 安全邊界切分，輸出上限 2048 token。空白、截斷、工具呼叫或超預算摘要不快取。每個側問在同一 120 秒逾時內最多 12 次摘要呼叫，達上限明確失敗，不無限迴圈。
- 首次模型上下文溢位且尚未輸出文字／工具呼叫時，允許一次更小請求重試。估算大小未減少時立即停止復原。其他模型錯誤與部分串流不觸發復原。
- 所有回傳用量（含摘要、失敗嘗試與取消）累計於同一側問請求。供應商未回傳用量時僅記 0；不得以估算冒充實測用量。

<a id="persistence-and-interface"></a>

## 持久化與介面

`side_question_sessions.memory` 儲存較舊交換摘要、涵蓋序位，以及依快照身分快取的主要上下文摘要。`side_question_requests.context_info` 儲存準備階段、實際重播次數、摘要使用情況與預算估算。

只有原請求仍執行且清除版本一致時才儲存摘要。清除亦移除快取，延遲寫入無法還原已清除資料。新快照不重用舊快照摘要。封存 v3 保留摘要欄位；還原缺少欄位的舊 v3 時填入空物件，v1／v2 保持相容。

POST 先接受並回傳請求，再於背景準備／壓縮；期間不持有准入鎖或資料庫交易。SSE／歷史顯示準備、整理交換、壓縮副本及回答階段。壓縮失敗持久化為側問終止失敗。前端保留錯誤，並將失敗問題還原為草稿，不在輸入欄上方顯示 toast。歷史輪詢不再清除提交錯誤。

<a id="historical-verification-record"></a>

## 歷史驗證記錄

以下為 2026-09-11 修正所記錄的上游結果，並非新的 ScopeWeaver 結果。

- 19／20／21／50 次交換重播、保留超過 20 次交換前的結論、重新啟動後摘要快取重用：自動檢查通過。
- 極長中文回答／程式碼上下文、分塊預算、工具配對、快照不變及新快照快取失效：自動檢查通過。
- 摘要失敗／取消／截斷／超長／工具回傳、清除競態、呼叫上限、單次溢位復原與部分串流後不重試：自動檢查通過。
- 隔離 PostgreSQL 分頁、重啟、v1／v2／v3 封存、摘要／預算中繼資料還原、舊 v3 缺欄位及 20 個父資源共用四個並行槽：通過。
- 候選 Go 建置、前端 TypeScript、變更元件 Biome 與隔離目錄的 Next.js 正式建置：通過。
- 應用程式內瀏覽器以 1280×720 及 390×844 隔離介面測試資料，驗證準備、摘要範圍提示、失敗草稿還原、無錯誤 toast、無水平溢出與主控台錯誤。暫存資料已移除。
- 唯讀重播既有本機 Worker 快照通過新預算檢查。例如 Worker #3 的 293085 字元快照不再被本機字元計數拒絕。此檢查未呼叫外部模型。
- 自動核准審查拒絕將私人 Worker 快照傳給 Grok 進行真實對話；該測試未執行，也未算通過。
- 應使用者要求，本機後端於 2026-09-11 00:37 使用原資料庫、資料目錄及登入設定重新啟動。執行檔與候選 SHA-256 一致；後端與前端代理的 `/api/health` 均回報健康。

驗證指令（僅限隔離測試資料庫）：



```sh
go test -race ./sidequestion ./db ./server -run 'TestSide|TestCheckpoint|TestSnapshot|TestBuildRequest|TestService|TestMainSide|TestTaskArchive' -count=1
go build ./cmd/artex
npx tsc --noEmit
npm run build -- --webpack
```



歷史前端正式建置使用獨立副本，避免覆寫執行中預覽的 `.next`。候選執行檔為 `/private/tmp/artex-btw-budget-candidate`，複製至 `/private/tmp/artex-btw-preview/artex` 並啟動；舊檔在該處備份為 `artex.before-context-budget`。這些路徑描述當時執行，並非隨 ScopeWeaver 發行的檔案。
