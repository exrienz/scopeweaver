[English](../../../sidequestion/VALIDATION.md) · [한국어](../../ko/sidequestion/VALIDATION.md) · [简体中文](../../zh-Hans/sidequestion/VALIDATION.md) · 繁體中文 · [日本語](../../ja/sidequestion/VALIDATION.md) · [Español](../../es/sidequestion/VALIDATION.md)

<a id="btw-verification-record"></a>

# `/btw` 驗證記錄

各語言版本請見本頁上方導覽。

上游歷史記錄。日期：2026-09-10。分支：`codex/btw-side-question`。基準：`8dae851b9b622f2ff2631f332fde9719d0b16fba`。這些並非新的 ScopeWeaver 結果。

使用隔離 PostgreSQL 測試資料庫與資料目錄。真實模型憑證僅注入該環境，未寫入原始碼或本記錄。產品預設模型未變更。Go 1.26.3、norma v0.3.6、Next.js 16.2.9。

模型對話、回傳物件、工程斷言與 Qwen 審查存於 [validation-2026-09-10.json](../../../sidequestion/validation-2026-09-10.json)，不含 API 憑證。敘述文字已翻譯，識別碼與實測值保留。

<a id="engineering-checks"></a>

## 工程檢查

| 範圍 | 結果 | 證據 |
| --- | --- | --- |
| 結構化訊息與工具參數深層複製 | 通過 | `TestCheckpointDeepCopyAndBoundaries` |
| 摘要／壓縮請求不覆寫；完整回覆與終止狀態發布；排除部分回覆 | 通過 | `TestCheckpointDeepCopyAndBoundaries`, `TestSnapshotExcludesPartialStreamAndSelectsPoolMember` |
| 實際模型池成員身分 | 通過 | `TestSnapshotExcludesPartialStreamAndSelectsPoolMember` |
| 工具配對、20 次交換重播、預算裁減與溢位錯誤 | 通過 | `TestBuildRequestCompactionToolPairingAndBudget` |
| 主／側並行與雙向取消隔離 | 通過 | 阻塞供應商，`TestMainSideConcurrencyAndIndependentCancellation` |
| 無工具執行、串流／非串流、失敗時可取得用量 | 通過 | `TestServiceNoToolsAndUsageOnFailure` |
| 真實 norma ChatAgent + 本機 Read 工具；主要對話記錄／活動隔離 | 通過 | `TestSideActualChatCheckpointToolResultAndTranscriptIsolation`, 串流與非串流子測試 |
| 持久化、分頁、冪等性、重啟後部分回答 | 通過 | `TestSideHistoryIdempotencyPagingAndRecovery` |
| 清除／延遲寫入競態、父資源刪除、版本比較 | 通過 | `TestSideClearLateWritersAndDeletedParent` |
| MainAgent／Worker 封存與還原，v1／v2／v3 | 通過 | `TestSideTaskArchiveVersions` |
| 三種父 API、驗證、擁有權、Worker 邏輯刪除 | 通過 | `TestSideHTTPGlobalLimitTaskWorkerAndDeletion`, `TestSideCheckpointPersistsBeforeAdmissionAndRestart` |
| 忙碌主工作階段中的側問；獨立 SSE 重連／斷線、取消與清除 | 通過 | `TestSideHTTPBusyIsolationClearAndReconnect` |
| 每個父資源一個並行請求／全域四個 | 通過 | 兩個 `TestSideHTTP…` 測試 |
| 准入前儲存快照、重啟後追問、不捏造舊工作階段快照 | 通過 | `TestSideCheckpointPersistsBeforeAdmissionAndRestart` |
| 拒絕已刪快取設定檔或已變更模型 | 通過 | `TestSideRejectsDeletedOrChangedCachedProfile` |
| 封存前取消並等待最終回答／用量持久化 | 通過 | `TestSideTaskDrainPersistsBeforeArchive` |
| 串流消費者提早取消時僅記一次用量與側問歸屬 | 通過 | `TestSideUsageRecordedOnceOnConsumerCancellation` |
| 重啟還原的 Worker／deadline 執行上下文發布新快照 | 通過 | `TestSideRestoredWorkerRuntimePublishesNewCheckpoint` |
| 相關套件競態檢查 | 通過 | 下列指令 |
| TypeScript 與正式建置 | 通過 | `npx tsc --noEmit`, `npm run build` |
| 新增前端模組的 Biome | 通過 | `biome check`, 三個新模組 |

重現自動檢查時，將 `ARTEX_PG_DSN` 設為獨立可拋棄資料庫，絕不用正式資料庫：



```sh
go test -race ./agent ./db ./server ./sidequestion ./llmrec ./llmpool \
  -run 'Test(Side|Checkpoint|Snapshot|BuildRequest|Service|MainSide|CaptureRun|TaskArchive|CompleteForwards|StopIntent|CancelIntent)' -count=1
cd web
npx tsc --noEmit
npx biome check src/lib/side-questions.ts src/hooks/use-side-questions.ts src/components/side-question-workspace.tsx
npm run build
```



完整 Go 迴歸並未全數通過。兩個既有 `server` 測試在暫存目錄清理時因 `TempDir RemoveAll … directory not empty` 失敗：

- `TestInheritedActivityDetailAndRelationDeletion`
- `TestTaskMetadataPatchReturnsRenameAndPin`

匯出未修改基準，在同一隔離環境重跑 `server`，亦重現兩項清理失敗。基準另在 `TestCoreTaskLifecyclePG` 目標節點數斷言失敗；最終修改版伺服器迴歸沒有此失敗。其他套件、側問案例與競態檢查通過。基準問題未算作驗收通過，也未修改既有斷言來隱藏問題。

Next.js 顯示既有多 lockfile／工作區根目錄推斷警告；建置與全部頁面生成仍完成。

<a id="browser-checks"></a>

## 瀏覽器檢查

Codex 應用程式內瀏覽器連至隔離本機 Go 與 Next.js 開發伺服器。桌面及 390 × 844 窄螢幕檢查使用瀏覽器自動化與畫面／記錄審查：

- 一般對話中，`/btw` 在桌面側欄同時顯示主要與側問內容。
- 追問正常；停止後保留部分側問輸出，主流程持續。
- 面板關閉時請求持續；重開還原完整回答，重新整理後空白 `/btw` 還原歷史。
- 窄螢幕 Drawer 輸入、按鈕、歷史與關閉皆正常，無水平溢出。
- 清除需確認，移除歷史但保留主要對話記錄／快照。
- MainAgent 與兩個 Worker 間切換問題時，標籤與歷史各自分離。
- 阻塞本機模型測試資料保持 Worker 執行。從主輸入提交 `/btw` 並停止側問後，Worker 仍顯示即時執行與自己的暫停按鈕，部分側問回答亦持久化。
- 瀏覽器錯誤／警告記錄為空。

受控測試資料獨立於模型速度驗證精確並行時序。最初兩次 Worker 檢查因任務／回答已結束而缺少有效並行視窗。修正測試資料後重複檢查通過；前兩次未算通過。

<a id="real-model-conversations"></a>

## 真實模型對話

優先模型 `grok-4.6` 在 OpenAI 相容端點 `http://127.0.0.1:12580/tingly/openai` 的探測於 2.82 秒傳回 HTTP 200、模型 `grok-4.6` 與 `READY`。既然可用，便未使用或驗證 Tingly `glm` 或智譜 `glm-5.3` 備援。

| 情境 | 實際結果 |
| --- | --- |
| 主要執行期間詢問資產、目標與標記 | 回答 `redhaze.top`、首頁閱讀／摘要目標、`BTW-REAL-0910`；16.97 秒完成 |
| 讀取首頁後詢問工具證據 | 正確回答 WebFetch 200、curl 301 → 302 → 200 重新導向與標題；7.24 秒 |
| 要求側問透過 Bash 建立檔案 | 拒絕；未建立目標檔案；7.74 秒 |
| 側問完成後主要上下文不變 | 主要對話記錄 SHA-256 與活動不變；側問工具執行為零 |
| 實際停止／重啟 Go 後追問 | 保留前三筆側問記錄；無須重跑主要代理程式，即可從持久快照回答資產、標記與標題 |
| 新對話使用非串流 Grok | 正確資產與 `ATOMIC-0910`；回傳／儲存用量：input 11734、output 138、cache_read 11520 |

主要對話使用 WebFetch 與 Bash／curl 讀取公開首頁，最終到達 `https://id.redhaze.top/home`，標題為「RedHaze Technology RedHaze Group · Global integrated group portal」（譯文）。Bash 儲存本機暫存回應檔，沒有遠端寫入。此事已與側問無工具執行分開驗證。

主要對話記錄校驗值：`e7e61f135a4a120954b539f357e8c4205d7d5cd7460dcaf3dc0fd066463e1d00`。

用量限制：Tingly Grok 串流未回傳用量。直接以 `stream_options.include_usage=true` 探測，傳回 HTTP 200、12 個資料框及零個用量框。因此串流的零表示端點未提供用量，不表示免費。非串流用量及測試資料的失敗／取消用量均正確持久化。

<a id="qwen-review"></a>

## Qwen 審查

審查模型 `qwen-flash` 在 OpenAI 相容端點 `https://dashscope.aliyuncs.com/compatible-mode/v1` 傳回 HTTP 200。它接收前三次真實側問對話、主要工具證據與工程斷言，回傳 `verdict: accept`、`concerns: []`；判定回答與資產、標記、頁面證據一致，拒絕工具亦符合限制。用量：prompt 6625、completion 312、total 6937。

審查未涵蓋後續重啟與非串流測試。Qwen 的「無寫入」描述過於寬泛：如上所述，主要工作階段 curl 確實建立本機回應檔。工程斷言證明並行、零側問工具執行與對話記錄隔離；模型審查僅輔助評估回答品質。
