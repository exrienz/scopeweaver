[English](../VERIFICATION.md) · [한국어](../ko/VERIFICATION.md) · [简体中文](../zh-Hans/VERIFICATION.md) · 繁體中文 · [日本語](../ja/VERIFICATION.md) · [Español](../es/VERIFICATION.md)

<a id="verification-and-known-limitations"></a>

# 驗證與已知限制

各語言版本請見本頁上方導覽。

<a id="current-dev-pr-40-reconstruction--2026-10-25"></a>

## 基於目前 dev 重建 PR #40 — 2026-10-25

最初僅將 `aebea3e..9b04754` 中預期的四個提交重建到 `f354c920` 上；未重放分叉的分支歷史。隨後，候選版變基到 `7af06f1`，保留其六語言整合。目前的整合要求恰好 **1,987** 個正式鍵具有英文、韓文、簡體中文、繁體中文、日文和西班牙文的原始譯文。既有 OpenCode 環境鍵隔離、工作階段標頭優先順序及 User-Agent 預設值均予以保留。

更廣的取消／競態測試覆蓋揭露了既有規劃器結果未同步的問題：流式工作階段可能在暫停時傳回，而真實的 `prove_goal` 處理程式仍在設定 `GoalMet` 及其原因。終結工具的寫入方與規劃器的結果快照現在共用一個私有互斥鎖。這保留了匯出欄位、真實工具／終結回執，以及目前暫停與完成之間的優先次序；不增加供應商請求，也不改變正式資料庫 schema。

終結證明關卡還要求**真實且成功的 SDK 工具結果回執**，且須與標記的工具使用 ID 及發起它的模型回應相符。私有捕獲觀察器在下一次壓縮或推理邊界之前執行；僅處理程式完成或 `PostToolUse` hook 本身不能滿足資格。對抗回歸使用真實的 `goal_met`、**30 ms** 執行預算與 **120 ms** 工具後 hook：舊關卡在收到中斷回執後吞掉了結算，使**供應商呼叫為 1 次而非 2 次**，而原生原因仍為 `timeout`，錯誤仍為 nil。修正後的路徑保留該中斷回執，允許真實結算呼叫及其壓縮，並拒絕延遲 hook 賦予資格。第二個同步回歸將 hook 保持等待，直到捕獲觀察到中斷，以證明真實處理程式在超時回執之前執行；正常成功證明、父上下文取消、反應式溢出與延遲來源防護仍在覆蓋範圍內。詳見[終結證明回歸](../../agent/terminal_proof_test.go)。

最初的重建仍有繼承的關閉缺口：在未改動的 `f354c920` 上，`TestMainServeFailureRunsShutdown` 在 **10 次獨立執行中失敗 3 次**，包括資料庫關閉後排程器仍寫入。後續修復在啓動前登記排程器歸屬。`Manager.Close` 在管理器鎖之外取消並等待排程器結束，最多 **5 秒**，之後才停止日誌寫入器並關閉資料庫。如果排程器 SQL 仍然阻塞，Close 傳回 `context.DeadlineExceeded` 並**保持資料庫開啟**；工作結束後，可以重試關閉。CLI 原本就會報告關閉失敗並以 1 退出。這不表示既有不帶上下文的 SQL 呼叫變得可取消。已預先取消的排程器不再初始化水位或啓動下一輪 tick。[阻塞 SQL 回歸](../../server/scheduler_shutdown_test.go)檢查並行關閉超時、工作仍活躍時資料庫保持開啟，以及解除阻塞後並行重試成功。首次並行全套件執行還出現日誌接收器測試的全域行數斷言失敗，當時其他套件的子處理程式使用同一個資料庫；跨包干擾仍屬假設，因此最終驗證遵循 CI，為每個套件使用獨立的可拋棄資料庫。這些歷史回執不代表最終確切提交的代管 CI、原生 Windows 或真實供應商驗證。

整合候選版在網頁端使用 Next.js **16.4.3** 與 Zod **4.6.7**，配接器使用 Zod **4.6.7**，並在鎖定檔案中以相容方式修復 body-parser **2.4.0**、fast-uri **3.1.9**、ip-address **10.7.4** 和 qs **6.16.1**。配接器 SDK 仍固定為 **1.32.0**。六語言變基後，兩個套件的全新 `npm ci` 均通過；網頁測試通過 **75/75**，冷啟動靜態建置匯出 **31 頁**，配接器測試通過 **18/18**，語法檢查也通過。先前 72/7 測試回執仍作為歷史證據保留。目前兩項 `npm audit --audit-level=low` 檢查均報告 **0 個漏洞**。這些結果取代了最初 Next.js 16.4.0 公告／建置失敗的限制；原始回執仍為歷史證據。瀏覽器原生貼上／撤銷按鍵現在使用 `ControlOrMeta+V` 與 `ControlOrMeta+Z`，保留斷言和超時。最終標準瀏覽器執行及確切提交的代管 CI 是獨立關卡，本節不宣稱已通過。

六語言整合後的相容性檢查必須覆蓋每一種已發佈來源語言及每一種請求輸出語言的舊版內建規劃器／工作器提示詞與 `add_intent` schema。新增 zh-Hant／ja／es 測試資料來自不可變的 PR40 之前 dev `7af06f1`。精確識別必須保留已編輯提示詞、自定義 schema 欄位／預設值、已存行與原始歷史。這擴展了下方原先的三語言驗證。

<a id="patched-sdk-and-model-specific-context-windows"></a>

### 已修補 SDK 與各模型的上下文視窗

後端 CI 工作必須執行巢狀的 Norma 測試套件。請在根套件之外單獨執行 `(cd third_party/norma && go test ./... -count=1)`；詳見[來源與修改聲明](PROVENANCE.md#norma-execution-and-compaction-fixes--2026-10-25)。

回歸集覆蓋原始的 30 ms / 120 ms 工具中斷競態、小視窗節奏與長工作階段邊界、有實質收益的摘要合併、真實溢出重試視圖、單調學習上限，以及不得觸發上下文恢復的輸出錯誤。既有數值邊界與原始歷史保持不變。舊斷言認為 40K 無法進入第二層，這在未改動的上游就已不成立；替代測試使用同一測試資料，要求真正執行合併、回收 token 並限制摘要大小。

`TestNoaRetainsConfiguredModelContextWindow` 實際執行 40K、128K、200K、250K、400K 與 1M 的 Noa 工作階段。主應用修正前，所有非 200K 情況都會變成 200K。`enableNoa` 現在先傳入已解析配置中的正數視窗，再清除被替換的壓縮器。其他測試分別覆蓋視窗缺失時的回退、停用 Noa、損壞狀態恢復，以及既有本地 Ollama 策略。`MaxTokens` 仍獨立限制輸出。200K 回退適用於未設定視窗的情況；不是通用模型上限。SDK 在 200K 及以上的預設值保持不變，而有實質收益的摘要按數量決定是否合併的條件則有意作了調整，該調整適用於所有上下文視窗，與視窗大小無關。

<a id="trigger-evidence-in-the-first-planner-request-2026-10-18"></a>

## 首次規劃器請求中的觸發證據（2026-10-18）

這是原始提交 `ffe260435fed6576394d613991aaf6e8010d3f48` 上的本地候選版，已回移共用 CLI 修正。通常，finding 或工作器完成事件會攜帶觸發摘要及精簡的 `graph_overview` 喚醒規劃器；`prove_goal` 仍要求每個目標條件均有已記錄證據。精簡條目缺少完整的 finding／fact 欄位，因此證明前讀取 `node_detail` 是合理的。

[Planner.Plan](../../agent/planner.go) 的實際變更節錄如下，省略了中間未變動的任務上下文設定：

```go
// Before
situational := renderTriggers(ts, triggers) + renderGraphOverview(tsx.graphOverviewData())
// After
overview := tsx.graphOverviewData()
situational := renderTriggers(ts, triggers) + renderGraphOverview(overview)
// After ensureRunDir and WithReviewContext:
situational += renderPlannerEvidence(ctx, overview, triggers, tools, ts.ID(), taskDir)
```

第一行新程式碼只捕獲一次既有的有界概覽；第二行呈現同一快照。最後一行在任務／審查上下文建立後，為具體 `finding`／`done` 觸發 intent 所產生、屬於目前 exploration 的 fact／finding 候選節點附上完整詳情。它不遍歷 `related_tasks`，也不擴展全域譜系。[附加實現](../../agent/planner_evidence.go)最多進行 **4 次讀取嘗試**，且僅當**整個附件不超過 32,768 個 UTF-8 位元組**時才納入完整節點，包含本地化標頭、JSON 外層與 `omitted_count`；過大的節點會被省略，不會截斷。重複候選會去重。

只有最終解析出的 `node_detail` 能力可以讀取：其 schema 必須接受輸入，`IsReadOnly` 必須成立，權限必須明確允許未改動的輸入。包裝器、預設值、計量、受限結果與錯誤均仍有效；不安全或不完整的結果會被省略，不會回退到原始輔助函數。標頭在 EN／KO／zh-Hans 中均完整，並將記錄內容標記為不可信觀察。既有顯式讀取（包括關聯的直接來源節點）保留原本的訪問規則。

具體資料行來自 **CLI 所屬的合成環回 R22 實驗**，並非原始正式環境事件：intent `3` 完成一次自有 GET 並持久化 fact `4` 與 finding `5` 後，目標 `2` 仍未達成。目標要求 finding 名稱精確為 `SYNTHETIC LAB ONLY public debug marker`、嚴重程度 `low`、標記與觀察內容。在 **2026-10-17 22:42:31+09:00**，規劃器為取得這些缺少的完整欄位，合理地讀取了 `node_detail` `5` 與 `4`。測試框架清理在證明前的 **22:42:54+09:00** 將其暫停；這不是人工暫停。R22 在未變動的 **420 秒任務預算**下，於**總計 470.50 秒**後失敗。底層模型／供應商延遲的原因**未知**。修正後，同一快照候選中符合資格的完整詳情會進入觸發回合的首次請求；詳情本身並不構成證明。

| 本地候選版檢查 | 觀察結果 |
| --- | --- |
| 原始完整 `agent` 包，無 overlay 的確切正式原始碼 | 598 個測試／子測試通過，0 失敗／略過；等效 CLI 為 607 通過，0 失敗／略過 |
| 三個 `planner_evidence*test.go` 檔案中的新增回歸 | 31 個測試／子測試通過；實際 `Planner.Plan` 流式／非流式首次請求相等、九種權限情況、數量／位元組邊界、去重／隔離、缺少／已刪／延遲／隱藏證據、顯式恢復、不自動證明與多個目標 |
| 原始聚焦伺服器證明／語言區域檢查及完整 `locale` 包 | 分別通過 5 與 19 項，0 失敗／略過；保留 **1,986 個鍵 × 3 種語言**的嚴格完整性，因新標頭將預期數量由 1,985 更新 |
| 原始 `go build ./...`、`go vet ./...` | 兩者原生退出碼均為 0 |
| Linux `-race`，目前等效的 CLI 原始碼 | 聚焦檢查 49 項與完整 `agent` 包 607 項通過，0 失敗／略過；兩者實際 Go 退出碼均為 0，無 `DATA RACE`。已驗證全部 1,175 個快照原始碼檔案；這不是原始模組執行 |
| 真實封裝 CLI 環回驗收，全部五種情況 | core 19.14s、GLM LOW 82.63s、Flash LOW 114.84s、OAuth LOW 108.12s 與 Flash HIGH 317.83s 原生退出碼均為 0，各自使用未變動的 420 秒任務預算。這些單次實驗執行不是效能比較 |

未改動的原始基線在流式及非流式缺少上下文斷言上失敗（兩個葉節點及其父項失敗），並非設定失敗。回歸測試資料中的受控供應商用量為合成資料，不是付費 token 證據。仍須使用真實的 `prove_goal`、真實用量／回執與既有終結證明關卡；階段、工作器、努力級別、截止時間與重試行為均不變。反例：停用、受限、缺少、過大、未觸發或繼承的節點不會自動獲得完整詳情。可用時，顯式讀取可以恢復被省略／新產生的證據；單憑讀取資料不能完成目標，證明多個目標中的一個也不能完成任務。

以上為本地工作樹結果，並非確切新提交的 CI。未改動的 Windows `TestChatBranchCopyRejectsDestinationSymlink` 權限失敗也在 CLI 基線 `650dc82910e5e36668161a5276455506f64713fe` 重現；未放寬任何檢查，也不宣稱完整原生伺服器成功。新候選版全部七項必要代管檢查仍待完成。PR #40 仍未合併，最終人工審查待完成；本節不宣稱已公開或發行。不引入公共 API、schema 或遷移變更。較早日期的章節保留其歷史結果。

<a id="chat-context-readiness-2026-10-17"></a>

## 對話上下文就緒狀態（2026-10-17）

這是基於 PR #40 提交 c9fb642 的本地界面候選版。檢查器可以在首次 controls 請求完成之前打開。先前它會將尚未解析的工作階段視為草稿，再將首次解析出的工作階段當作切換而關閉面板。在[檢查執行 39031048389](https://github.com/cskwork/scopeweaver/actions/runs/39031048389) 中，中文桌面任務案例因關閉按鈕先不穩定、後脫離文檔而失敗。該代管提交仍是失敗證據；本次新結果為本地結果。

重設邏輯節錄（binding 是請求的 controls 鍵；session 是解析出的工作階段識別）：
~~~ts
// Before: every change from draft to resolved identity ran the reset.
epoch.current++;
setOpen(false);
// After: qualify the change before the existing reset.
const previous = hydrated.current;
hydrated.current = { binding, session };
if (previous.binding === binding && !previous.session) return;
epoch.current++;
setOpen(false);
~~~
第一行新增程式碼讀取先前的 binding 與 session；第二行記住目前這組值。第三行僅在同一 binding 尚未解析出首個工作階段時保留已打開的面板。不同請求 binding 或後續已解析工作階段變化，仍會增加 epoch 並完整重設視圖／表單，包括焦點與保留數量。hook 暴露既有請求鍵；沒有伺服器 API 或翻譯標籤變更。

具體受控資料行：任務 1、exploration 1、segment 0 通過真實任務 API 暫停。其首次 controls 回應被保持等待，直到 /context 打開。先前釋放該真實回應會使觸發按鈕關閉，導致正常點擊在 15 秒後超時（Go 退出碼 1）；現在逐位元組相同的瀏覽器判定檢查會保留打開的觸發按鈕並正常關閉（退出碼 0），供應商呼叫為零。任務暫停有別於對話關卡的 paused 欄位。

| 凍結修正原始碼上的本地檢查 | 觀察結果 |
| --- | --- |
| 既有前端測試 | 65 通過，0 失敗／略過 |
| 全新靜態前端建置 | TypeScript 與 31 頁完成；已驗證原生退出碼 0 |
| 未改動的獨立首次回應延遲判定檢查 | 1 個 Go 測試通過，0 失敗／略過；供應商呼叫為零 |
| 標準上下文瀏覽器測試資料 | 全部 12 個 EN/KO/zh-Hans × 1440/390 × 獨立／任務案例，加上 hydration 與請求 binding 防護均通過（14 個瀏覽器步驟；1 個 Go 測試，0 失敗／略過） |

新增防護先使用真實的首次後端回應，再使用合成路由狀態來檢查後續已解析工作階段變化，以及被保持等待的舊上下文回應。每個替代回應均保持等待，直到舊內容斷言完成，避免新結果掩蓋洩漏。非預設焦點與保留數量 7 會重設為空焦點與預設值 2。反例：真實請求 binding 或後續已解析工作階段變化仍須重設；延遲結果不得恢復舊上下文。第二次真實任務查詢切換（完整執行中從任務 7 到 8）在新 controls 回應保持等待時觀察到了新 URL，保留同一個仍連接的檢查器觸發按鈕與文檔，重設焦點／保留數量，並在替代資料可能掩蓋問題前拒絕延遲舊上下文。這證明瞭已掛載狀態下請求 binding 的變化，而非重新掛載。延遲錯誤／壓縮成功分支仍僅經原始碼審查，未單獨執行。合成識別防護證明的是用戶端處理，不是伺服器 exploration 替換。其他上下文案例使用受控環回供應商；未呼叫任何付費／外部供應商。

最初的 shell 包裝嘗試因環境／捕獲設定而停止；最終建置／測試退出碼另行記錄，保留警告，且未以原始碼／斷言變通來通過。先前規劃器／關閉變更予以保留。配對檔案記錄這些本地結果；下一次已提交候選版仍須獨立最終審查及全部七項代管檢查。PR 就緒不等於維護者批准、合併或公開發行。

<a id="log-writer-shutdown-ownership-2026-10-17"></a>

## 日誌寫入器的關閉歸屬（2026-10-17）

伺服器將日誌保存在記憶體中，並通過異步資料庫寫入器持久化。先前取消伺服器上下文時不會等待該寫入器結束：使用後台上下文的插入可能在 `Manager.Close` 關閉連接池時仍然活躍。PR #40 提交 `14b5403` 的最近後端執行，在未改動的連接埠佔用關閉測試中揭露了此問題。

寫入器與管理器之間的流程節錄：
```go
// Before: writer, then manager shutdown
pg.InsertLog(level, tag, text) // background context
return m.pg.Close()           // no writer join
// After: writer, then manager shutdown
pg.InsertLogContext(writeCtx, level, tag, text)
logSink.StopDB(m.pg)
return m.pg.Close()
```
第一行新程式碼使實際插入可取消；首次歷史恢復也使用同一個上下文。第二行只解除這個確切連接池的綁定，取消未完成 SQL，並在接收器鎖之外等待寫入器結束。每次同一擁有者的停止操作都共用該完成信號；另一個管理器的連接池不能停止它。只有之後第三行才關閉連接池。既有不帶上下文的 DB 方法委派給同一 SQL；環形緩衝區、stderr 與 SSE 傳遞仍可用，入隊通道從不關閉。

一條觀察資料：後端執行 [39028154916](https://github.com/cskwork/scopeweaver/actions/runs/39028154916)，job `117383132313`，在 **2026-10-17 11:18:45 UTC** 的 `TestMainServeFailureRunsShutdown` 失敗。預期的連接埠佔用綁定失敗之後出現關閉及 `[logsink] db write: sql: database is closed`。另一個獨立真實 PostgreSQL 表鎖檢查，在未改動的 `14b5403` 上重現了 `Manager.Close` 傳回時其寫入器仍阻塞（Go 退出碼 1）。現在完全相同的測試資料會在鎖仍持有、連接池尚未關閉時通過；沒有過濾任何診斷。

| 原始本地 R19 檢查 | 觀察結果 |
| --- | --- |
| 未改動的獨立阻塞插入回歸 | 1 通過，0 失敗／略過；保留基線行為 RED |
| 真實 SQL 拒絕防護 | 1 通過，0 失敗／略過；可見 SQLSTATE `23514`，被拒行不存在，後續有效行已持久化 |
| 凍結的 `TestStopDB*` 測試資料 | 6 通過，0 失敗／略過：無綁定、阻塞插入、其他連接池、已取消上下文、並行停止與 stderr 連續性 |
| 未改動的連接埠佔用關閉測試 | 1 次執行加上 20 次重複執行通過；保留預期子處理程式失敗，父 Go 退出碼 0 |
| `go build ./...`、`go vet ./...` | 兩者退出碼均為 0 |

反例：真實的活躍上下文 SQL 拒絕仍必須報告；取消／等待結束並不會將它變成成功插入。這仍是盡力而為的日誌記錄，沒有為佇列中的行新增刷寫保證。並行測試資料檢查總體完成；每個等待方共用等待結束的行為也經原始碼審查驗證，不宣稱為單獨的執行時判定檢查。全部資料庫執行均使用可拋棄測試資料庫；獨立測試資料現在使用所提供可拋棄資料庫中的自有 schema，不需要 CREATE DATABASE 權限。正式權限、HTTP／schema 合約與既有規劃器測試不變。這些為本地結果，不是完整原生伺服器或發行聲明：繼承的 Windows 符號連結權限限制仍存在，下一次已提交候選版必須通過全部七項代管檢查。

<a id="terminal-goal-completion-2026-10-17"></a>

## 目標最終完成（2026-10-17）

以下為本地 R17 候選版結果；最終獨立原始碼審查與確切候選版的代管關卡另計。R17 共用修正讓已完成的證明回合沿正常 SDK 路徑走完，再於另一上游請求前在本地結束規劃器。通常，`prove_goal` 將已確認的 fact／finding 關聯到目標，並標記該目標已達成；只有全部目標均已證明才設定 `GoalMet`。先前 `Plan` 會等待 `captureRun` 完成後才將該標記傳回引擎，因此工作階段等待另一次模型回應時，引擎無法完成。

控制流程節錄（`...` 是未變動的活動回調）：
```go
// Before
_, _, err = captureRun(ctx, opts, input, ...)
return tsx.GoalMet, tsx.Reason, err
// After
installTerminalProofGateOn(&opts, tsx)
_, _, err = captureRun(ctx, opts, input, ...)
return tsx.GoalMet, tsx.Reason, err
```
第一行舊程式碼等待整個工作階段；第二行才對外提供目標結果。新增的一行在捕獲開始前安裝下一次請求的邊界。它讓目前回應、實際工具結果、延遲用量及對話記錄完成，再提供不含內容、上游用量為零的本地 `end_turn`，並跳過證明後不必要的 `Compactor.Pre`。它保留所選壓縮器、可選 `ContextView`、正常證明前／恢復行為，以及既有供應商／工具包裝器。原始 `ToolUseID` 及成功／失敗回應資格判定，防止延遲處理程式將舊失敗證明歸屬於後續成功回應。最新證明來源通過 compare-and-swap 保留；較舊或未知來源不能抹掉較新的有效證明。失敗記錄在產出錯誤前執行，即使其消費者立即停止亦然。

具體實驗資料行：在**獨立 CLI R11 候選版**中（並非原始正式環境事件），finding 節點 `5` 於 **2026-10-17 18:36:24+09:00** 證明目標 `2`。未變動的 420 秒測試框架在 84 秒後的 **18:37:48+09:00** 暫停時，模型輸出仍活躍。該次執行恰有一個 GET、一個 intent 與一個 finding。回執為 `scopeweaver-cli/dist/pi-glm-flash-fix-20261017/r11-flash-high-digest.txt`；它說明瞭完成等待，不能說明底層供應商延遲原因。反例：部分／被拒證明必須正常繼續；失敗證明回應保留其真實錯誤／恢復路徑，不能賦予本地完成資格。

原始原始碼證明使用受控供應商與可拋棄 PostgreSQL，不使用付費推理：

| 本地候選版檢查 | 觀察結果 |
| --- | --- |
| 完整 `agent` 包，R17 使用已修正的自有 fact 測試資料 | 567 個測試／子測試通過，0 失敗、0 略過；包含流式／非流式、部分／被拒證明、父上下文取消、錯誤／恢復、壓縮／視圖及延遲來源案例 |
| `TestWireTools`、內建執行時 S7 與實際引擎 S18 | 28 通過，0 失敗、0 略過；S18 在一次真實供應商呼叫後持久化 done，具有真實證明／終結回執，以及精確的輸入／輸出／快取讀取／快取寫入用量 **11/7/3/2**，原始對話記錄亦包含這些資料 |
| S18 中真實暫停／供應商錯誤防護 | 如實的錯誤／原因回執保留已發出用量：使用者暫停為 **11/0/3/0**，供應商失敗為 **11/7/3/2**；觀察證明後暫停與任務完成的狀態優先次序，未新增策略斷言 |
| 完整 `locale`、建置與靜態分析 | 19 個測試通過，0 失敗、0 略過；`go build ./...` 與 `go vet ./...` 退出碼 0 |

已保存的原始 S18 基線失敗，因為它嘗試了第 2 次供應商呼叫，且沒有終結結果／完成；其取消／供應商錯誤回執防護通過。獨立不可變 R14 回歸也曾失敗：舊失敗處理程式借用了後續回應的識別，導致 calls=2 而非 3；候選版保留了三次呼叫回歸。審查者隨後在原始 R16 候選版重現相反順序：舊的失敗 response-1 處理程式在有效 response-2 證明之後完成，抹掉了其來源，導致不必要的第三次請求。未改動的實際工作階段 overlay 在 R17 前以 calls=3 失敗，現在以 calls=2/origin=2 及一份真實新證明結果通過。兩種延遲處理程式順序均通過[邊界回歸](../../agent/terminal_boundary_test.go)；[實際引擎回歸](../../server/engine_terminal_proof_test.go)另行保留完成／回執／用量檢查。首次整合 agent 執行失敗，僅因新測試資料預期 `CreateExploration` 會創建一個來源 fact。R16 自行建立已確認 fact；這次測試資料修正沒有改變任何斷言或正式程式碼。較早失敗回執仍予以保留。

S18 現在復用 S7 在所提供可拋棄資料庫內的自有 schema 輔助函數，在刪除自己的 schema 前關閉存儲區；擁有資料庫的 NOCREATEDB 測試角色就足夠，正式權限保持不變。未改動的 `TestChatBranchCopyRejectsDestinationSymlink` 仍在本地 `server/chat_history_test.go:270` 失敗，因為 Windows 拒絕創建符號連結。不宣稱完整原生伺服器成功或已通過確切候選版全部七項代管檢查；CI／前端／瀏覽器交付仍待完成。根協調方負責另行進行競態／真實執行驗收；本節僅報告這些原始本地檢查。

<a id="plannertool-contract-and-stored-stock-compatibility-2026-10-17"></a>

## 規劃器／工具合約與已存內建內容相容性（2026-10-17）

這是 `fix/glm-agent-contracts` 基於 `dev` 提交 `d7365ec` 的本地 Windows 結果。規劃器的 `parent_ids` 指引現在與既有僅接受 fact／finding 的處理程式一致，批次 `add_intent` 中繼資料說明瞭欄位與必要的 `summary`。規劃器／工作器指引保留全新驗證與重測，要求每個目標條件都有證據，且僅在成功派發或確認最終目標證明後結束回合。EN／KO／zh-Hans 資源均有配對。

完全相同的舊版內建規劃器／工作器提示詞與 `add_intent` schema 會解析為修正後的執行時預設值。使用者編輯、綁定、啓用標記、已存行與提示詞歷史均不變；沒有正式資料庫遷移，也沒有處理程式語義變更。詳見[合約回歸](../../agent/planner_contract_test.go)、[內建內容相容性回歸](../../agent/legacy_stock_compat_test.go)及 [HTTP／執行時 PostgreSQL 證明](../../server/tools_stock_schema_test.go)。

測試先行的證明成功編譯，並在基線上產生 **8 個頂層 agent 斷言失敗**與 **9 個 S7 內建 schema 子案例失敗**。僅為編譯測試而臨時使用的未變動尾部提取與直通解析器，在應用修正前已移除。候選版結果：

| 本地檢查 | 觀察結果 |
| --- | --- |
| 完整 `agent` 套件與可拋棄 PostgreSQL | 549 個測試／子測試通過，0 失敗、0 略過 |
| 完整 `locale` 包 | 19 通過，0 失敗、0 略過 |
| 聚焦伺服器語言區域／工具／來源檢查 | 11 通過，0 失敗、0 略過；1,985 個正式鍵保留原始三語言完整性檢查 |
| `TestExplorationDirectSourceReadView` | 使用可拋棄 PostgreSQL 通過 |
| `TestStockMetadataRuntimePostgres` 加 `TestWireTools`，`-count=2` | 48 個測試／子測試通過，0 失敗、0 略過；實際環回 HTTP 處理程式、執行時解析及已存行／歷史回讀 |
| 已保存的公開工具情境 | S11：全部 10 個主智能體／規劃器回調案例通過；S12：歷史譜系、來源及無關／間接父節點拒絕通過；S3：無效 kind／空白 summary 拒絕、部分錯誤及省略／空值／null 來源關聯通過 |
| `go build ./...`、`go vet ./...`、差異空白檢查 | 通過 |

`TestStockMetadataRuntimePostgres` 與 `TestEngineTerminalProofPersistsAuthenticReceipt` 要求顯式提供選擇 `scopeweaver_test_*` 的可拋棄 `ARTEX_TEST_DSN`。擁有資料庫的測試角色可以具有 **NOCREATEDB**，與 `make test-db` 一致。每份測試資料創建自己的 `stock_*` schema，並將所有池化連接的 `search_path` 僅設定為該 schema，保留所選資料庫與其他連接選項。測試在使用 `CASCADE` 僅刪除自己的 schema 前關閉存儲區；清理錯誤會報告，但刪除 schema 本身無法偵測空閒洩漏連接。`TestStockFixtureIsolationAndCleanup` 檢查多個池化連接、父表不可見／保留及 schema 刪除。可用 `go test ./server -run '^Test(WireTools|StockMetadataRuntimePostgres|StockFixtureIsolationAndCleanup|EngineTerminalProofPersistsAuthenticReceipt)$' -count=2` 重現隔離。正式資料庫權限與 schema 不變。這項測試資料修復取代了先前對子資料庫／CREATE DATABASE 的要求。

首次 S3 測試框架嘗試在建立 hint 時使用無效狀態 `open` 而失敗；在通過的重跑前，測試資料修正為既有 `active` 狀態，沒有產品／斷言變更。最終 S7 設定在其他測試填入父資料庫後也通過了兩次。這些檢查確立合約與相容行為，不能說明先前真實 Flash 重複／超時的原因。此分支完整 `go test ./...`、前端／瀏覽器檢查及確切候選版全部七項必要 CI 工作仍待完成；不宣稱已合併或發行。

<a id="shared-runtime-fixes-from-standalone-cli-verification-2026-10-17"></a>

## 獨立 CLI 驗證發現的共用執行時修正（2026-10-17）

候選分支 `fix/cli-discovered-runtime-bugs`，基於 `dev` 提交 `3f0ef25`。以下為本機結果；PR 確切提交的代管檢查另計。此變更未將獨立 CLI 或訂閱供應商加入原始應用程式。

- HTTP 自訂工具現在使用 `Manager.ProxyAddr` 傳回的完整記錄代理 URL。先前 `"http://" + addr` 將測試的 `http://127.0.0.1:<port>` 變成 `http://http://127.0.0.1:<port>`，尚未抵達代理便失敗；現在 `proxyStr = addr` 保留 URL。真實直接、明確設定與自動代理路由皆通過；自動路由修正前失敗。
- 啟動時先確認既有對話記錄路徑是目錄再讀取。Windows 一般檔案迴歸修正前失敗，現會拒絕路徑並保留檔案；不存在的目錄仍有效。新診斷提供英文、韓文、簡體中文。
- 關閉時先取消工作、排空 HTTP，再等待已准入對話、側問與快照發布，最後關閉儲存區。先前僅排空 HTTP，可能讓背景結束處理在資料庫關閉後寫入。最終快照／上下文寫入錯誤現會向上傳遞、保留待寫項目，並使處理程序明確關閉失敗；已刪／衝突記錄仍視為無害。此有界排空不宣稱涵蓋所有排程器或引擎生產者。
- 取消用量測試資料現建立既有選用用量表，符合正式初始化。修正前重現缺表失敗；修正後全新資料庫測試保留精確用量斷言。

真實 PostgreSQL 關閉檢查通過 11 個測試／子測試，含關閉重開持久化與注入最終寫入失敗。主處理程序通過 4 項，涵蓋正常結束 0、重啟 75、有界失敗及真實連接埠佔用服務失敗的清理。聚焦代理／啟動／語系迴歸通過。完整 Windows 後端為 1,975 通過、2 失敗、15 略過；一項為未改變的符號連結權限限制，另一項揭露目錄預期數，因新增兩診斷由 1,980 改為 1,982，未移除完整性檢查。修正目錄另行驗證；其餘平台結果須由 Linux／代管檢查建立。

前端 65 測試與靜態匯出通過，既有 POSIX npm 指令碼使用 Git Bash。配接器 7 測試與語法檢查通過。聚焦後端靜態分析通過。未使用使用者資料庫、真實線上目標或通知；各寫入測試均建立並刪除獨立 `scopeweaver_test_*`。不宣稱合併或發行。

<a id="chat-controls-and-context-2026-10-16"></a>

## 對話控制與上下文（2026-10-16）

[PR #29](https://github.com/cskwork/scopeweaver/pull/29) 作者針對上游 `6366429` 回報以下檢查。這是 PR #29／#30 合併審查前的貢獻者歷史 Linux 結果，不證明後續候選狀態或 Windows 附件複製行為。

| 檢查 | 結果 |
| --- | --- |
| 可拋棄 PostgreSQL 17 競態套件（`db`、`server`、`agent`） | 667 頂層測試通過，無競態報告 |
| Go 套件與靜態分析 | `go test ./... -count=1`、`go vet ./...` 通過 |
| 前端測試、翻譯、稽核、靜態匯出 | 65 測試；3,263 EN／KO／zh-Hans 鍵對等；0 稽核問題；TypeScript 與 31 頁通過 |
| 後端翻譯 | 1,978 正式鍵在三語通過完整性、對等與 printf |
| 對話瀏覽器閘門 | EN／KO／zh-Hans、1440／390 px、獨立／任務對話共 60 情境通過，無未捕捉頁面錯誤 |
| 內嵌應用基本測試與儀表板 | Smoke E2E 與 14 儀表板情境通過，無未捕捉瀏覽器錯誤 |
| 代理程式配接器 | 7 單元、語法及 3 MCP／Pi 情境通過 |
| 安裝與封裝 | 8 安裝資料、11 Compose 案例、每語 EN／KO 各 5 封存檔、6 發行聲明測試通過 |

對話瀏覽器案例涵蓋佇列編輯／排序、重啟確認、重試／編輯／分支、上下文檢查／壓縮、工具引導與文件貼上。後端迴歸涵蓋已接受語言持久化、預檢錯誤、中斷工具交換、放棄分支準備與文件讀取歸屬。

可將模型目前上下文與計費對話用量分開檢視。壓縮保留您選擇的近期完整交換；完整顯示歷史與之前嘗試用量仍保留。

貢獻者回報一次審查找出六個整合缺陷，之後修復生成輸入准入與複測復原、壓縮後附件路徑、舊版顯示歷史及引導投遞控制。一項附件測試在子分支下次提示前刪父分支，另一項檢查 rebase 保留 manifest 字面文字。各修復前均回報失敗迴歸，之後完整本機重跑。儀表板推理總數斷言限縮至摘要列，因懸停提示有相同標籤；重跑回報 14 儀表板情境通過。

回報推論測試使用回送模型與可拋棄資料庫。Windows 執行、付費供應商、遠端 CI 需另驗證。聲明測試使用 Python 3，符合 CI 直譯器需求。

<a id="native-hardening-candidate--2026-10-15"></a>

## 原生強化候選版 — 2026-10-15

在基於 main `33b131e` 的 `feat/ollama-zero-egress-hardening` 進行本機與獨立檢查。這是候選結果，確切提交的遠端 CI 與 PR 交付仍待完成。以下歷史段落保留原始結果。

| 實際執行檢查 | 結果 |
| --- | --- |
| 獨立 Go 執行，20 套件各用隔離可拋棄 PostgreSQL | 1,366 測試／子測試通過、0 失敗；`enrich` 無測試；2 選擇性略過（`TestAgentAdapterE2E`、`TestLiveContextReview`） |
| 另測代理程式配接器 | 7 單元及 3 真實 MCP／Pi 情境通過；刻意不執行外部真實供應商審查 |
| 聚焦後端 `-race` | 傳輸、wire／用量、上下文、來源選擇／失敗、搜尋／更新、准入皆通過，無競態 |
| `go vet ./...` | 通過 |
| 前端單元／本地化與靜態匯出 | 45 通過；3,168 EN／KO／zh-Hans 鍵保留對等、預留位置、rich tag；TypeScript 與全部 31 頁通過 |
| 後端目錄 | 1,943 正式鍵通過原始三語完整性／對等與 printf |
| 安裝、Compose、聲明 | 8 安裝、11 Compose 憑證、EN／KO 每語 5 封裝測試封存的聲明檢查、Python 3.12 的 6 發行聲明測試通過 |
| 靜態建置與相依檢查 | macOS 內嵌與 Windows amd64 交叉建置通過；`go.mod`／`go.sum` 無漂移；未執行原生 Windows |
| 內嵌瀏覽器基本測試 | 最終 UI 重建後 60 通過（48 smoke + 12 儀表板）、0 失敗。EN／KO／zh-Hans 穩定桌面／行動畫面及水平邊界通過：390px sheet [0,390]，1440px [864,1440]，全部 20 項關鍵內容可容納。已重現並修正先前 420px 最小寬裁切 |

[原生 HTTP 優先級迴歸](../../server/ollama_priority_test.go) 將 worker 1 留在 `/api/chat`，排隊 worker 2／3，再呼叫真實接線裁決，不由呼叫端標記緊急。`-race` 通過，HTTP 順序 **worker 1 → judge → worker 2 → worker 3**，峰值並行 1，原生裁決 schema／verdict 有效，全部完成且無名額洩漏。既有佇列測試涵蓋 FIFO 角色、取消與降低上限。

[原生來源失敗迴歸](../../server/provider_origin_failure_test.go) 修正前失敗：原生工具待執行後串流出錯，可能繼承全域雲端 `allow`。修正後 Stream／Complete 與終止路由錯誤保留具體原生設定檔、90 秒／`ask` 模型政策、300 秒／`deny` 人工核准到期，不呼叫審查者。實際雲端容錯仍清除原生偏好。迴歸與受影響競態套件通過。

已驗證 API 對不轉送請求的本機代理切換嚴格模式 **false → true → false**。四種搜尋後端 `ddgs`、`brave-free`、`tavily`、`deepseek` 僅在關閉時到達該接收端，嚴格模式各新增請求為零。嚴格更新檢查／套用遭封鎖，本機狀態可用。[更新迴歸](../../server/update_policy_test.go) 亦通過快取、重新導向、即時本文讀取，以及已准入非同步下載被後續切換阻擋後的清理／失敗完成。這不代表整台主機出口控制。

主機 Ollama **0.32.15** 使用既有 `qwen3:4b`、**4,096 token 輸出預算**，通過原生診斷、decision／comment schema、結構工具呼叫／結果重播、用量與要求 32K（`/api/ps` **32,768**）。先前 256／512 預算失敗仍保留證據，未丟棄或算成功。載入／速度／卸載數值只是一次熱執行快照，非基準或駐留保證。未下載模型或改主機設定，不宣稱自主任務完成或安全評估品質。

瀏覽器涵蓋三語診斷／警告、嚴格更新狀態、裁決載入／失敗保護及既有儀表板／用量。視覺 QA 重現固定 420px 最小寬編輯器在 390px 視窗左溢 30px。最小寬現符合視窗；重建應用通過全部 60 項，六張桌面／行動畫面標籤可讀，穩定內容在視窗內。獨立內嵌重建與瀏覽器執行檔逐位元組一致。確切提交遠端 CI 與 PR 讀回另為交付閘門。不宣稱合併、發行、付費供應商呼叫、外部目標掃描或真實通知。

<a id="provider-authentication-follow-up--2026-10-15"></a>

## 供應商驗證後續 — 2026-10-15

基於已合併 main `38fde23` 的 `fix/provider-env-auth`，本機迴歸重現並修正兩個需驗證的本機閘道失敗。舊版 Anthropic、OpenAI Chat Completions、OpenAI Responses 的啟動供應商金鑰備援與空金鑰連線測試皆回 HTTP 401；相同資料在修正後通過。

| 檢查 | 結果 |
| --- | --- |
| [`FromEnv → NewProvider` 本機 HTTP 驗證](../../agent/provider_env_test.go) | 21 情境通過：18 一般格式涵蓋供應商金鑰、檔案金鑰、現代／舊版覆寫、明確空白；3 原生涵蓋雲端金鑰隔離、Ollama 權杖、明確空白 |
| [透過本機 HTTP 資料的連線測試 handler](../../server/llm_gateway_test.go) | 16 情境通過：15 一般格式涵蓋相符正規化服務、不同端點／格式／代理、保留非本機備援；1 原生拒絕繼承啟用雲端金鑰 |
| 兩聚焦組 `-race` | 通過，無競態 |

上述 **37 個案例**為葉情境，不含父測試。各修正前重現驗證失敗，之後同測試通過。非本機 handler 案例中的公開文件位址經明確本機測試代理，不聯絡公開供應商。

無 profile ID 的空金鑰連線測試，僅在一般格式、正規化基底 URL、代理全相符時重用目前本機服務金鑰，含新草稿。不同本機服務與原生 Ollama 保持隔離。空金鑰新設定檔不匯入環境憑證。

這些檢查建立已測協定與憑證邊界，不代表付費帳號可用或全部歷史功能。修正後完整套件、瀏覽器、CI、已合併 main、發行成品結果另為閘門。未呼叫付費供應商。

<a id="installed-model-selection-follow-up--2026-10-15"></a>

## 已安裝模型選擇後續 — 2026-10-15

在 `feat/ollama-native-flexible-providers` 本機驗證新設定檔 Ollama 預設選擇，取代先前範本行為：選 Ollama 現執行唯讀模型探索。下方歷史 37 檢查瀏覽器快照早於此行為；「無隱含請求」為歷史斷言，不是目前契約。

| 檢查 | 結果 |
| --- | --- |
| 前端單元與本地化（`npm --prefix web test`） | 45 通過、0 失敗 |
| 正式靜態匯出（`npm --prefix web run build:static`） | TypeScript 與全部 31 頁通過 |
| 瀏覽器與已驗證後端真實主機模型探索 | 未存新設定檔、同機 Ollama、空代理／金鑰通過 |
| 內嵌執行檔與全新 `scopeweaver_test_default_final` 完整擴展瀏覽器 smoke | 43 通過、0 失敗（31 smoke + 12 儀表板）；涵蓋 EN／KO 三範本安裝預設、直接格式選擇、雲端相容、僅中繼資料選擇、失敗／空探索、手改／拒絕延遲回應、重開草稿／既有設定檔，以及既有設定、登入、本地化、SSE、儀表板迴歸 |

主機 `/api/tags` 傳回 `qwen2.5-coder:7b`、`qwen3:4b`、`embeddinggemma:latest`。瀏覽器選 **`qwen2.5-coder:7b`**，第一個宣告 `tools` 的已安裝模型。中繼資料為 `completion`、`tools`、`insert`、7.6B、Q4_K_M；`qwen3:4b` 亦宣告 `tools`，`embeddinggemma:latest` 僅 `embedding`。

瀏覽器恰好送一次 LLM 操作 `POST /api/llm/models`，主機端點、無金鑰／代理。新草稿關閉，未儲存、啟用、測連線或推論。模型／機器設定不變。這僅證明安裝模型選擇與中繼資料探索，不代表工具執行或自主可靠性。

<a id="historical-native-ollama-branch--2026-10-15-branch-local-not-a-release"></a>

## 歷史原生 Ollama 分支 — 2026-10-15（僅分支，非發行）

`feat/ollama-native-flexible-providers` 開 PR 前的本機檢查。遠端 CI 另於 GitHub Actions 追蹤；本節不代表發行、合併或 CI 結果。

| 檢查 | 結果 |
| --- | --- |
| 20 個 Go 套件各用全新可拋棄 PostgreSQL 16 | 1,197 測試／子測試通過；2 選擇性略過（`TestAgentAdapterE2E`、`TestLiveContextReview`） |
| 另測配接器 | 3 真實 MCP／Pi 情境通過 |
| `go vet ./...` | 通過 |
| 前端單元／本地化 | 43 通過；英／韓／簡體中文各 3,159 唯一鍵 |
| 後端訊息目錄 | 三語各 1,938 唯一鍵；printf／對等通過 |
| TypeScript／正式靜態匯出 | 通過，最終 health UI 後重建 |
| `npm audit` | 0 漏洞 |
| 建置 | 最終 health UI 後重建 `CGO_ENABLED=0` 內嵌執行檔；Windows amd64 `CGO_ENABLED=0` 交叉建置通過，未執行原生 Windows |
| 真實內嵌執行檔與全新 `scopeweaver_test_ollama_final_health` 的 `npm run smoke` | 37 通過、0 失敗（25 smoke + 12 儀表板）。含 EN／KO 三 Ollama 範本無隱含請求、免金鑰儲存／啟用、本機免金鑰相容 health 與缺金鑰公開雲端對比、雲端格式／可編輯 GLM、模型中繼資料／工具警告、缺模型精確 pull 指令、工具／上下文／載入／速度診斷、預設關閉本機政策，以及既有三語設定、登入、語言、草稿、SSE、核准用量迴歸 |
| Compose 憑證 | 真實 `network_mode: none` 容器經環境與 `FromEnv`，11 案例通過 |
| 安裝與封裝 | 8 安裝資料共通過，含英／韓選擇；英／韓各 5 封存檔通過 |
| 發行聲明（Python 3.12） | 6 測試通過 |
| 獨立後端安全／正確性審查 | 通過，含 DeepSeek 本機政策、即時 CIDR、404 分類 |

<a id="real-ollama-run"></a>

### 真實 Ollama 執行

主機 Ollama 0.32.15、`qwen3:4b`，經原生串流 API：

- SDK 層級串流檢查的合成工具資料中，模型呼叫 `record_fact`，ScopeWeaver 在本機執行並重播結果，回合完成；這不是平台資料庫的 `record_fact`。
- 要求 32K，`/api/ps` 回報分配 32,768。
- 精確用量：串流回合輸入 186／輸出 750；完成回合輸入 826／輸出 193。
- 真實 API 免金鑰儲存／啟用通過。診斷 `tool_call_verified`、`context_verified` 皆 true，上下文 32,768、載入 2.225375 ms、29.231 token/s、延遲 18,958 ms。**僅一次熱執行，非冷啟基準或硬體泛化。**
- 缺模型的精確韓文 pull 提示及嚴格政策切換封鎖公開 IP 通過。
- **平台任務 Worker Bash 證據**：task 1 的 `qwen3:4b`（要求 32K、分配 32,768；並行 1；keep-alive 30m；思考關閉；上限 4,096）發出原生 Bash `call_219yv72x`。活動 9（call）、10（`tool_result`）同 ID、無錯誤，輸出含 `LOCAL_TASK_OLLAMA_OK`。帳本精確記 6,704 輸入／4,096 輸出；下一原生呼叫重播相同 `tool_call_id`。意圖透過既有 `db.AddIntent` 手動植入，非 LLM 建立。輸出亦含測試資料的字面 `\n` 跳脫，不據此推斷換行處理或模型錯誤。執行檔來自最終 UI／DTO 前的未提交分支原始碼。

<a id="autonomous-task-reliability-is-not-validated"></a>

### 自主任務可靠性未驗證

工具診斷通過僅證明傳輸路徑，不代表模型可完成任務。

- `qwen2.5-coder:7b` 宣告 `tools`，卻將工具形狀 JSON 寫入 `assistant.content`，不是傳輸 `tool_calls`，因此無 Bash／事實工具執行；ScopeWeaver 正確保留純文字。
- `qwen3:4b` 作主要代理時，長提示下反覆思考並以文字寫出預期標記。作任務 Worker 確實執行上述 Bash，但 360 秒時任務暫停，未 `record_fact` 或自主完成；一次成功工具呼叫不代表自主完成。

**不對任何本機模型宣稱端對端自主完成或安全評估品質，7B 級模型亦未驗證可作完整代理程式。** [硬體分級](llm-providers.md#hardware-tiers) 為估算，非基準測試。

<a id="not-exercised"></a>

### 未執行項目

無付費雲端呼叫、外部目標掃描、真實通知、發布、部署、發行或合併。Anthropic、OpenAI Chat／Responses、自訂閘道經傳輸契約測試、迴歸與可編輯瀏覽器流程驗證，不使用真實供應商帳號。測試資料庫皆可拋棄，主機 Ollama 設定未改。選用 `local_llm_only` 預設關閉；伺服器仍需另設 `OLLAMA_NO_CLOUD`。

<a id="running-database-backed-tests"></a>

## 執行資料庫測試

`db`、`server`、`agent` 套件需要 PostgreSQL，連線字串**只**從 `ARTEX_TEST_DSN` 取得，絕不備援應用程式的 `ARTEX_PG_DSN`／`config.json`。此設計防止 `go test` 修改執行中應用的即時／正式資料庫。

- **未設 `ARTEX_TEST_DSN`**：全部資料庫測試略過，不接觸資料庫。
- **`ARTEX_TEST_DSN` 設為可拋棄資料庫**：套件對該庫執行。
- **解析至與應用相同資料庫**（同主機、連接埠、庫名）：任何測試連線前即快速失敗，錯誤 `refusing to run tests against the live database; point ARTEX_TEST_DSN at a throwaway DB`。

最簡單的執行方式：



```bash
make test-db
```



`make test-db`（透過 `scripts/test-db.sh`）建立由 `sw_user_test` 角色擁有的可拋棄 `scopeweaver_test_<pid>`，將 `ARTEX_TEST_DSN` 指向該資料庫，再由 `ARTEX_TEST_DSN` 指定測試連線，執行 `go test ./db/ ./server/ ./agent/ -count=1`，並在結束時刪除資料庫，即使測試失敗亦同。可覆寫 `TEST_DB_ROLE`、`TEST_DB_HOST`、`TEST_DB_NAME` 配合本機 PostgreSQL。庫名必須以 `scopeweaver_test_` 開頭，配接器 E2E 可用 `scopeweaver_adapter_`。

若要針對自行管理資料庫執行部分測試，匯出 `ARTEX_TEST_DSN` 並直接呼叫 `go test`：



```bash
export ARTEX_TEST_DSN='postgres://sw_user_test@/scopeweaver_test_1?host=/tmp&sslmode=disable'
go test ./server/ -run TestSomething -count=1
```



<a id="v021-candidate--2026-10-11"></a>

## v0.2.1 候選版 — 2026-10-11

Windows 候選版以真實靜態前端建置，針對可拋棄 PostgreSQL 16.15 測試。結果取代下方歷史相依與 Windows 執行限制；舊記錄保留供追溯。

| 檢查 | 結果 |
| --- | --- |
| 前端相依稽核 | 全嚴重度 0 漏洞，含開發相依 |
| 前端迴歸／本地化 | 26 通過、無略過 |
| 正式靜態匯出 | TypeScript 通過，生成 31 頁 |
| 原生 Windows Go、18 套件 | 1,016 測試／子測試通過、0 失敗、13 既有平台／選擇性略過 |
| 原生 Windows 完整 server | 311 測試／子測試通過、3 既有選擇性／平台略過 |
| 配接器 | 7 單元／語法通過；真實已驗證後端與新庫的 3 MCP／Pi 情境通過 |
| 本機安裝 | 4 隔離資料皆通過：新安裝、舊檔替換、相依失敗、前端建置失敗 |
| 後端翻譯目錄 | 1,868 正式鍵通過英／韓完整性與 printf 參數 |
| 通知隔離 | 確定性舊 Worker 迴歸修正前失敗、之後通過；管理／通知選擇連續 5 次通過 |
| Windows 證據 | 完整 PostgreSQL 套件通過，含並行寫入、提交失敗、長路徑 |
| Windows Python／技能檔 | 真實 Python 與無法使用啟動器的備援通過；巢狀 Unicode 路徑通過清單／讀取 handler |
| 內嵌 Windows 瀏覽器 E2E | 8 通過：兩語設定服務中斷保護、設定／登入、同源 SSE、用量 API 隔離、兩語用量卡、無未捕捉錯誤 |

Go 數量含父測試與子測試。13 原生略過為十個 Unix 自我更新資料、一個缺符號連結權限、選擇性配接器整合及真實供應商。配接器另驗證，Unix 資料由 Linux CI 涵蓋。未為產生結果停用測試。

迴歸重現原 Windows 目錄同步失敗、本地化分隔符不符、不可用 Python 啟動器、通知 Worker 洩漏與設定檔清理順序。E2E SQL 輔助現將選項置連線 URL 前；直接中斷／還原檢查確認 Windows psql 執行陳述式。

發行閘門另要求合併前後的 `frontend`、`windows`、`backend`、`agent-adapter`、瀏覽器 `smoke`。發布結果、平台封存與校驗值見 [v0.2.1](https://github.com/cskwork/scopeweaver/releases/tag/v0.2.1)；此來源記錄不宣稱後續工作或下載資產已通過。

保留既有 Python 直譯器路徑。若舊安裝存入不可用啟動器別名，請在設定選可執行直譯器。原生 Windows 測試不證明 Windows 自我更新或 macOS 執行。未使用付費供應商、外部目標或真實通知收件人。

<a id="historical-verification--2026-10-03"></a>

## 歷史驗證 — 2026-10-03

**2026-10-03** 記錄的英／韓 ScopeWeaver 驗證，基於上游 ARTEX `160fe13`，使用可拋棄 PostgreSQL 與本機資料。此為原始碼交付，當時未發布二進位或容器映像。

| 檢查 | 結果 |
| --- | --- |
| 前端迴歸／本地化 | 21 通過，TypeScript 通過 |
| 正式靜態匯出／Go 內嵌執行檔 | 建置成功 |
| Go 迴歸、18 套件 | 986 測試／子測試通過，1 繼承失敗，1 外部供應商略過 |
| 完整 server | 285 測試／子測試通過，僅選擇性真實供應商略過 |
| 後端翻譯 | 1,865 正式鍵通過英／韓完整性與 printf |
| 自撰中文稽核 | 無未翻譯自撰來源／文件／Go 註解，保留刻意相容／資料例外 |
| 桌面瀏覽器 | 每語 24 路由，無缺標籤或頁面水平溢出 |
| 行動瀏覽器 | 兩語儀表板、任務、發現、設定，390px 無頁面水平溢出 |
| 真實應用畫面 | 兩語登入／通知表單檢查；設定驗證／初始化通過 |
| 語言切換 | 導覽／狀態更新，保留選擇與未送草稿 |
| HTTP 本地化 | Markdown／CSV 匯出、協商、偏好儲存、無效語言拒絕通過 |
| GLM-5.3 契約 | 本機 HTTP／SSE 21 聚焦測試／子測試通過，含兩端點、推理、工具重播 |
| GLM 範本瀏覽器 | 兩語兩範本，保留名稱／金鑰／代理；無儲存、啟用或供應商請求 |
| 封裝／啟動指令碼 | Shell 語法、雙語封裝資料、10 監督案例通過 |
| 來源／相依 | 原授權、相依版本、Go 模組不變；文件連結與差異檢查通過 |

Go 數量含父測試與子測試。各套件用獨立資料庫，初始化與一般啟動相同的補充計量表。原始測試與檢查保持啟用。

<a id="inherited-issues-at-the-time"></a>

### 當時繼承問題

`TestGraphOverviewExpandsAssociatedCompanyScope` 在 ScopeWeaver 與未改上游皆因 `task scope missing: <nil>` 失敗。測試預期圖概覽含詳細範圍，但上游 `06a43f3` 刻意精簡。`75a706d` 改為 `TestGraphOverviewKeepsAssociatedCompanyScopeCompact`，檢查精簡概覽並直接驗證儲存的公司範圍。此後完整後端 CI 已通過。

上游驗證曾在 `TestTaskMetadataPatchReturnsRenameAndPin` 出現間歇暫存目錄清理失敗；最終 ScopeWeaver server 執行通過，未以無關生命週期變更隱藏。

上游 mock 預覽無通知中繼資料 mock，因此通知頁報錯。**真實後端的通知頁與新增通道表單在兩語皆通過**，驗證未傳送通知。

v0.1.0 記錄 **14 個 npm audit 問題：1 嚴重、9 高、4 中**。框架／工具更新及移除未用 shadcn CLI 相依，已於上方 v0.2.1 候選解決。執行 `cd web && npm audit` 查看 npm 套件登錄庫的最新稽核報告；新公告可能改變結果。

<a id="historical-scope-and-limits"></a>

### 歷史範圍與限制

- 選擇性 `TestLiveContextReview` 需私人外部供應商設定，未執行。無付費模型、外部目標掃描或真實通知。
- GLM 測試僅證明本機契約，不代表帳號存取／供應商可用。Coding Plan 僅支援官方工具，ScopeWeaver 未列入，見[供應商設定](llm-providers.md)。
- 未驗證 Docker 映像／Windows 批次執行；封裝資料不代表真實 Windows 測試。
- 設定進入已驗證路由，但一次等待 window load 逾時；之後以真實 API 確認初始化與新密碼。
- 使用者文字、儲存證據、編輯提示詞、傳輸識別碼、舊剖析標記、Unicode 資料與上游原畫面均保留。英／韓應用不翻譯任意使用者資料或第三方工具輸出。
- `sidequestion/` 歷史驗證描述上游工作，不是新的 ScopeWeaver 執行。

目前遠端檢查見 [GitHub Actions](https://github.com/cskwork/scopeweaver/actions/workflows/ci.yml)。

<a id="chat-document-paste-2026-10-16"></a>

## 對話文件貼上（2026-10-16）

PR #29 記錄對可拋棄 PostgreSQL 17 與本機後端的以下貼上檢查。這是貢獻者於 2026-10-16 回報結果；目前整合證據另記：

- 前端 52 測試與正式靜態匯出通過。
- 剪貼簿瀏覽器 12 組通過：獨立／任務對話、EN／KO／zh-Hans、1440／390 px。
- 涵蓋單一草稿擁有者的快速文件貼上、伴隨文字、原生 3,000 字元插入／復原、Unicode 預覽、草稿還原、上傳重試、導覽、mention、IME 與兩種送出偏好。貼上未提交任何訊息。
- 貢獻者停用 React paste handler，確認因缺第二附件標籤而失敗，再還原重跑。
- 後端 smoke 通過剪貼簿與 Ollama 回送資料，無未捕捉頁面錯誤。

檔案案例透過 paste event 提供瀏覽器 File 物件；文字與復原案例使用 Chromium 剪貼簿與鍵盤貼上。
