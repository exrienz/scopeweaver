[English](../PROVENANCE.md) · [한국어](../ko/PROVENANCE.md) · [简体中文](../zh-Hans/PROVENANCE.md) · 繁體中文 · [日本語](../ja/PROVENANCE.md) · [Español](../es/PROVENANCE.md)

<a id="scopeweaver-provenance-and-modifications"></a>

# ScopeWeaver 來源與修改

各語言版本請見本頁上方導覽。

ScopeWeaver 為 [Autumn-27/ARTEX](https://github.com/Autumn-27/ARTEX) 的獨立衍生專案，基於提交 `160fe13`，另行維護於 [cskwork/scopeweaver](https://github.com/cskwork/scopeweaver)，不具 GitHub fork 網路關係。

**2026-10-02** 的 ScopeWeaver 修改包括產品名稱、英文預設／韓文可選的介面與訊息、文件與內附代理程式指引、設定／建置／啟動／更新／重設密碼提示、報告與代理程式語言處理及發行命名。執行檔為 `scopeweaver`（Windows 為 `scopeweaver.exe`）；封存檔命名為 `scopeweaver-<version>-<os>-<arch>.zip`。Docker Compose 在本機建置簽出的原始碼。發行工作流程以本儲存庫與 GHCR 為目標。

原始 `LICENSE` 保持不變。原作者、貢獻者致謝與上游變更歷史仍歸屬 ARTEX。Go 模組 `github.com/Autumn-27/artex`、來源入口 `./cmd/artex`、`ARTEX_*` 設定契約、`artex` 資料庫預設值與 `ARTEX` 登入名稱均為相容性而保留。

此修改日期並非版本標籤，也不表示已發布二進位版本或容器映像檔。`sidequestion/` 下的歷史驗證文件描述上游執行，並非新的 ScopeWeaver 驗證收據。歷史 JSON 翻譯會保留技術識別碼與測量值，並標示敘述文字為譯文。

**2026-10-15** 新增可選的原生 Ollama 設定檔格式（`/api/chat`，搭配 `num_ctx`、`keep_alive` 與 `num_predict`）、新增表單的三種 Ollama 範本、免金鑰本機設定檔、各設定檔並行數、Ollama 感知的預熱、壓縮與核准裁決預設值，以及選擇性啟用的 `local_llm_only` LLM 網路政策。新增 `SCOPEWEAVER_LLM_*` 與 `SCOPEWEAVER_LOCAL_LLM_*` 別名，優先於保留的 `ARTEX_*` 契約。既有 Anthropic、OpenAI、OpenAI Responses、自訂 URL、代理與 GLM 範本行為保持不變，政策預設關閉。

**2026-10-15** 後續原生強化候選版新增不搶占的緊急下一名額准入、依實際原生來源選擇裁決模型、結構約束輸出與 90 秒／ask 預設值，以及 64K 以內的 72% 壓縮門檻與 16/32 KiB 工具輸出上限。選擇性嚴格目的地縮限為回送位址與明確推論 CIDR（包括 LAN／Docker 遷移），拒絕重新導向／代理，封鎖所有代理程式搜尋與更新網路操作，但保留本機狀態／復原。另提供唯讀 GPU 記憶體估算、共用模型警告、安全載入裁決設定，以及英文／韓文／簡體中文介面文字。這些是應用程式控制，不保證整台主機離線或模型品質；未新增相依套件、遷移、自動拉取模型或發行。詳見[供應商指南](llm-providers.md)與[候選版驗證](VERIFICATION.md#native-hardening-candidate--2026-10-15)。

**2026-10-03** 額外修改完成伺服器翻譯稽核，並新增 GLM-5.3 設定範本及其必需的常啟推理相容性。範本使用既有供應商配接器，不會自動建立或啟用設定檔。

另已回移 ARTEX 提交 `e2c8a23`、`86729b6`、`a951e4a` 的修正：正式環境同源 SSE、獨立核准模型 token 計量，以及失敗時拒絕存取的驗證、原子化初始密碼建立與有界資料庫連線池。新訊息與部署指南提供英文及韓文。回移提交已註明原作者，其他 ScopeWeaver 修改保持不變。

<a id="norma-execution-and-compaction-fixes--2026-10-25"></a>

## Norma 執行與壓縮修正 — 2026-10-25

儲存庫自有的 `third_party/norma` 替代實現基於
[`52f17dd52d95b087265e5d5b306fe85c1ea05599`](https://github.com/Autumn-27/Norma/commit/52f17dd52d95b087265e5d5b306fe85c1ea05599)。
該提交僅為 v0.4.3（`320ae32989da97dd91469bb89164cbfe87d83d5d`）添加 Apache-2.0 授權條款：本地修改前共 211 個檔案／1,293,448 位元組。此快照沒有上游 NOTICE 檔案。

本地修正讓執行器在結算期間保留其原始超時上下文，補上重試測試缺少的用戶端，按小視窗調整預設壓縮節奏，並防止有實質收益的摘要合併一直得不到執行。修正還僅學習明確且更小的上下文限制，防止已學習上限增加，並區分輸出長度錯誤與上下文溢出。原始歷史與封存保持完整；無法再縮減的視圖仍明確失敗。過時的小視窗限制測試改為正向的合併測試。

`third_party/norma/README.scopeweaver.md` 列出全部十個修改過的上游檔案與四個新增檔案。每個修改過的上游檔案均帶有顯著的修改聲明。其他上游檔案、原始授權條款與模組中繼資料保持不變。匯出簽名與序列化格式不變；列出的行為修正屬於有意變更。

根模組替代配置與 Docker 依賴複製順序會選用此原始碼。完整 Apache 授權條款也已納入 `THIRD_PARTY_NOTICES.txt`，供既有封存檔／容器封裝使用。後端 CI 工作會單獨執行完整巢狀 SDK 測試套件；根套件遍歷不會包含它。不包含向上游儲存庫發佈或資料庫遷移。

主應用負責的 `enableNoa` 修正會轉發已配置的上下文視窗，包括 250K 與 1M，而非靜默替換為 SDK 的 200K 回退值。輸出 token 限制、停用 Noa 與原生 Ollama 策略相互獨立，均予以保留。詳見配對的[驗證指南](VERIFICATION.md)。
