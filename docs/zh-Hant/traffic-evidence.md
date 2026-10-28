[English](../traffic-evidence.md) · [한국어](../ko/traffic-evidence.md) · [简体中文](../zh-Hans/traffic-evidence.md) · 繁體中文 · [日本語](../ja/traffic-evidence.md) · [Español](../es/traffic-evidence.md)

<a id="multiple-traffic-records-as-finding-evidence"></a>

# 多筆流量記錄作為發現證據

各語言版本請見本頁上方導覽。

發現詳情支援跨頁選取流量、設定角色與備註、排序及移除綁定。流量頁亦可將多筆記錄附加至既有發現。繼承自其他任務的證據為唯讀，請到來源任務編輯。

「代理程式自動綁定流量」預設關閉。透過 `/api/settings` 讀取或修改 `agent_traffic_binding`。啟用會增加請求／回應審查、工具呼叫與指示的 token 成本；下一個代理程式回合採用新設定。關閉時隱藏自動綁定參數與補充綁定工具、不注入自動綁定指示，並拒絕既有執行中工作階段提出的新自動綁定。手動綁定、擷取、讀取已儲存證據與匯出仍可使用。

啟用時一般流程為：記錄發現、自動啟動報告代理程式、驗證並綁定流量，再依最新證據版本撰寫報告。回報代理程式在 `evidence` 保留驗證指令、重要輸出、既有真實流量 ID 與用途。報告代理程式檢查發現與執行記錄，透過 `traffic_search`／`traffic_get` 驗證流量、呼叫 `bind_finding_traffic`，再讀取最新 `version` 後儲存。停用時不會提供額外原始流量搜尋／讀取工具或自動綁定，但仍可讀取手動綁定快照並撰寫報告。

為相容性，呼叫端仍可透過 `report_finding.traffic_refs`／`evidence_hint_id` 明確立即綁定。綁定為選填：TCP 等非 HTTP 發現、未擷取流量或缺少精確記錄時仍可回報。保留指令、記錄及其他可驗證證據，適當說明缺漏；不新增必填欄位。提交的每個 ID 均須有效且本文完整。任何失敗都會復原整個綁定操作；回報時的明確綁定失敗會復原整次回報。再次附加既有快照不會重複綁定或覆寫備註。

<a id="agent-identifiers-and-report-versions"></a>

## 代理程式識別碼與報告版本

`report_finding` 接受下列選填參數，陣列順序決定初始證據順序。



```json
{
  "traffic_refs": [
    {"traffic_id": "real-traffic-id", "role": "baseline", "note": "Normal account request"},
    {"traffic_id": "another-real-traffic-id", "role": "proof", "note": "Reproduction request"}
  ]
}
```



角色為 `baseline`（正常比較）、`proof`（發現證明）、`verification`（額外驗證）與 `supporting`（預設佐證）。先用 `traffic_search`／`traffic_get` 驗證真實記錄。網域與時間僅縮小候選範圍，不能證明任務歸屬。

指示參考 [CyberStrikeAI 發現回報指南](https://github.com/RuoJi6/CyberStrikeAI/blob/54d56774b8bd285817d16d48b70a4a5e6e0963f7/internal/app/vulnerability_tools.go)，並適配本專案選填綁定契約。不強制填寫缺少流量的原因。絕不猜測 ID，亦不得只為取得擷取記錄而重複探測。

回應第一行維持 `finding recorded: <exploration node ID>`。後續 JSON 提供獨立發現記錄 `finding_id`、探索節點 `finding_node_id` 與綁定摘要。

- `get_finding_traffic(finding_id)` 接受獨立發現記錄 ID，傳回排序清單與 `version`。提供 `binding_id`、`side=request|response`、`offset`、`length`，可讀取每段最多 8192 位元組。
- `update_finding_report.finding_id` 仍接受探索節點 ID。`evidence_version` 應設為實際讀取版本。生成期間證據若變更，舊版本寫入將遭拒；須重新讀取並生成。
- 未提供版本的舊報告呼叫不宣稱涵蓋既有流量證據。綁定、備註、角色或排序變更都會將既有報告標記為待更新。

啟用自動綁定時，`add_hint`／`add_task_hint` 可在單一提示或批次 `hints` 陣列的各項儲存 `traffic_refs`。規劃器代表其他代理程式回報時，可用 `evidence_hint_id` 從本任務提示選取參照，絕不可使用繼承提示。系統不會從網域、時間或瀏覽歷史推斷綁定。提交失敗不會建立部分發現，也不會提前啟動報告生成。

使用 `bind_finding_traffic(finding_id, traffic_refs)` 為既有發現補上缺漏綁定，無須再次登記。`list_findings`、`list_task_findings`、`node_detail`、`get_task_node_detail` 同時提供 `finding_id` 與 `finding_node_id`；舊版 `id` 保留探索節點含義。

啟動時僅為舊工具結構新增選填屬性。原始預設流量工具綁定延伸至報告代理程式，預設亦提供補充綁定工具。自訂綁定清單、提示詞、描述與啟用狀態保持不變。最終工具組裝完成後才附加指示：回報角色交接既有證據，報告代理程式驗證、綁定並撰寫。無任務上下文的平台對話應將結構化提示交給任務代理程式，勿直接回報。宣告任務完成前先交接現有證據；缺少擷取不強迫等待。`report_finding` 失敗不啟動報告代理程式。

<a id="api"></a>

## API

基底路徑：`/api/exploration/findings/{finding_id}/traffic`，使用獨立發現 ID。沿用既有驗證；`context_task` 檢查任務可見性與繼承唯讀存取。

| 方法／相對路徑 | 請求／回應 |
| --- | --- |
| `GET` | 排序摘要、證據版本、報告證據版本 |
| `POST` | 附加整批 `{"traffic_refs":[...]}` |
| `PATCH /{binding_id}` | `{"version":1,"role":"proof","note":"Description"}` |
| `DELETE /{binding_id}` | `{"version":1}` |
| `PUT /order` | `{"version":1,"binding_ids":["2","1"]}`；須包含完整清單 |
| `GET /{binding_id}` | 快照中繼資料與有界本文預覽 |
| `GET /{binding_id}/body` | `side`、`offset`、`length`；`download=1` 下載全部原始位元組 |

版本／排序集合衝突及封存期間寫入傳回 `409`；繼承寫入傳回 `403`；綁定不存在或不屬於該發現傳回 `404`。流量／附件讀取與驗證失敗皆明確報錯。

<a id="storage-and-migration"></a>

## 儲存與遷移

啟動時冪等遷移 PostgreSQL：`traffic_evidence_snapshots`、`finding_traffic_bindings` 及 `findings.evidence_version`／`report_evidence_version`（預設 0）。不從歷史敘述推斷綁定。

快照儲存原始流量 ID、擷取時間、URL、方法、狀態、請求／回應標頭、本文長度與 SHA-256。本文位於 `<data>/evidence/blobs/<first-two-characters>/<hash>.bin`，獨立於可拋棄的 `data/traffic`。多個發現可共用快照／本文。不提供更新快照內容 API；完整性不符會使讀取與匯出失敗。

完整本文（含大型 blob 與舊版目錄記錄）在原流量寫入鎖下讀取。先持久化並驗證檔案，再由單一 PostgreSQL 交易寫入探索節點、意圖關係、發現、快照與綁定。提交後才通知規劃器。失敗可能留下未參照檔案，但不會留下部分業務記錄。

PostgreSQL advisory lock `7337741004` 協調證據檔案與 SQL 參照。封存排入佇列後，任務列鎖禁止修改證據。還原從安裝本文到提交中繼資料期間持有證據鎖。刪除發現會串連刪除其綁定。

清理器每小時執行，僅回收至少 24 小時、無參照且不在進行中操作內的內容。一般流量清理不動證據目錄。備份熱資料時，請一併備份 PostgreSQL 與 `data/evidence`。

<a id="export-and-archives"></a>

## 匯出與封存

Markdown 包含排序證據清單與版本；JSON 包含中繼資料；CSV 增加數量與綁定 ID。`md-zip` 保留發現 Markdown 並包含：



```text
evidence/<finding_id>/<binding_id>/
  manifest.json
  request.http
  response.http
  request.bin
  response.bin
```



Markdown 以相對路徑連結訊息。送出下載前，伺服器複製附件、驗證雜湊、壓縮並同步封存檔，再讀取每個 ZIP 項目檢查 CRC。內容缺漏或損壞會使整次下載失敗。完整附件保留原始二進位位元組。

封存 v3 依發現綁定蒐集快照與本文，不依賴原始流量或網域。封存驗證完成後才清理熱資料；共用證據保留。還原先驗證並安裝本文，再以交易還原中繼資料與綁定，支援失敗重試。v1／v2 仍可還原，缺少的新欄位明確設為 0。

<a id="verification-and-boundaries"></a>

## 驗證與邊界

各套件透過 `ARTEX_PG_DSN` 使用獨立且全新建立的 PostgreSQL 測試資料庫，避免殘留任務／模型資料啟動背景工作。執行完整相關套件，確認未因缺少設定而略過：



```sh
# Set ARTEX_PG_DSN to the package's isolated database before each run.
# Explicit configuration failures must fail the test.
go test ./<package> -count=1
go test -race -p 1 ./evidence ./db ./agent ./server -run 'TestEvidence|TestFindingTraffic|TestFindingEvidence|TestReportFindingAtomicContract|TestTaskArchive'
```



前端檢查包括 `npx tsc --noEmit`、受影響檔案的 Biome、Webpack 建置與 `NEXT_EXPORT=1` 靜態匯出。使用獨立快取目錄，避免覆寫執行中的開發伺服器。

本機端對端驗收使用專用連接埠、受控 HTTP／以網域為基礎的 HTTPS 目標及暫存資料目錄。涵蓋兩個綁定入口、跨頁選取、排序／備註、錯誤、繼承唯讀、下載，以及刪除原始流量、封存、回收熱本文、還原與雜湊驗證後的匯出。

初版採全域證據協調鎖。大型綁定／匯出或緩慢附件下載可能使其他證據操作等待。不得捏造缺少的擷取或不完整本文。本功能不改變擷取設定，也不改變直接連至 IP 位址的 HTTPS 憑證處理。
