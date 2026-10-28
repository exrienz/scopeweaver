[English](CONTRIBUTING.md) · [한국어](CONTRIBUTING.ko.md) · [简体中文](CONTRIBUTING.zh-Hans.md) · 繁體中文 · [日本語](CONTRIBUTING.ja.md) · [Español](CONTRIBUTING.es.md)

<a id="contributing-to-scopeweaver"></a>

# 貢獻 ScopeWeaver

各語言版本請見本頁上方導覽。

歡迎提交錯誤修正、ARTEX 上游回移、相依套件升級、文件與測試。

<a id="before-you-start"></a>

## 開始之前

- 閱讀[使用限制](README.zh-Hant.md#license-and-disclaimer)。貢獻不得新增以真實線上系統為目標的功能，亦不得削弱這些限制。
- 大型變更請先開 issue，以便就方案達成共識。
- 每個 PR 聚焦單一主題。較小的 PR 更容易快速審查。

<a id="requirements-for-a-pull-request"></a>

## PR 要求

- **完整支援已發布語言及對應文件。** 使用者可見文字與維護中的使用者／貢獻者指南，需提供英文、韓文、簡體中文（`zh-Hans`）、繁體中文（`zh-Hant`）、日文（`ja`）與西班牙文（`es`）：
  - 伺服器訊息：保持 `locale.Register(key, en, ko)` 相容，透過 `locale.RegisterAll` 與 `locale/` 中依語言索引的資源加入其他語言。
  - 網頁介面：EN／KO 表格與 `messages/{zh-hans,zh-hant,ja,es}.ts` 必須涵蓋全部目前英文鍵。
  - 文件：`README.md`、`CONTRIBUTING.md` 與維護中的指南須有 `ko`、`zh-Hans`、`zh-Hant`、`ja`、`es` 對應版本；保留可執行的英文技能入口。
- **測試變更行為。** 新增或更新在未套用變更時會失敗的測試。不得為通過測試而停用、略過或放寬既有檢查。
- **僅使用可拋棄資料庫。** 會寫入 PostgreSQL 的測試必須使用全新的 `scopeweaver_test_*` 資料庫（代理程式配接器 E2E 使用 `scopeweaver_adapter_*`）。
- **註明上游貢獻。** ARTEX 回移須加入 `Upstream-commit: mhtsec/ARTEX@<sha>` 與原作者的 `Co-authored-by:` trailer，並更新 `docs/PROVENANCE.md` 與 `docs/ko/PROVENANCE.md`。
- **授權。** 貢獻依儲存庫的 [AGPL-3.0](LICENSE) 授權接受。

<a id="adding-or-changing-a-locale"></a>

## 新增或修改語系

依照[語系指南](docs/zh-Hant/i18n.md)更新共用的 `locale/definitions.json`（標準 ID、原生名稱、Intl 對應與別名）、比對測試向量、完整前後端目錄、date-fns 對應、聚焦測試與瀏覽器基本測試。`zh`、`zh-CN`、`zh-SG` 對應 `zh-Hans`；`zh-TW`、`zh-HK`、`zh-MO` 對應 `zh-Hant`。明確文字系統優先於衝突地區。英文為鍵、含義、預留位置與 rich tag 的來源。後端執行時英文／鍵備援是相容性保護，不代表語言已可發布。不得翻譯使用者內容、草稿、原始證據、識別碼或歷史資料。請審查譯文並記錄疑慮；無證據時不得宣稱經母語者核准。六種語言的維護中功能指南均須完整，包括英文／韓文配對文件。 亦須同步獨立配接器的標準 ID 清單與根目錄 shell／批次訊息輔助。執行 `node e2e/script-locales.mjs`，並以原生 Windows 的 `node e2e/windows-launcher.mjs`／Windows CI 驗證；封裝配接器不得在執行時依賴儲存庫根目錄 JSON。

<a id="checks-to-run"></a>

## 必須執行的檢查

以下對應 [.github/workflows/ci.yml](.github/workflows/ci.yml) 的 CI 工作：



```bash
# patched Norma SDK (a separate Go module)
(cd third_party/norma && go test ./... -count=1)

# backend database-backed suites (db, server, agent): against a throwaway database
# that is created and dropped for you. They require ARTEX_TEST_DSN and skip (never
# fall back to the live DSN) without it — see docs/VERIFICATION.md.
make test-db

# backend: every Go package (set ARTEX_TEST_DSN to a disposable scopeweaver_test_* database)
ARTEX_TEST_DSN=postgres://sw_user_test@/scopeweaver_test_1?host=/tmp&sslmode=disable go test ./... -count=1

# frontend
cd web && npm ci && npm test && npm run build:static

# agent adapter (and its real-backend E2E test against a scopeweaver_adapter_* database)
cd adapters/agent && npm ci && npm test && npm run check
SCOPEWEAVER_ADAPTER_E2E=1 go test ./server -run '^TestAgentAdapterE2E$' -count=1
```



`e2e` 工作流程亦會在每個 PR 與合併後，針對真實執行檔執行必要瀏覽器基本測試。若要在本機針對執行中的伺服器及全新的 `scopeweaver_test_*` 資料庫執行，需要 `psql` 與同一簽出版本建置的內嵌執行檔：



```bash
cd e2e && npm ci && npx playwright install chromium
SCOPEWEAVER_URL=http://127.0.0.1:8787 ARTEX_PG_DSN=postgres://.../scopeweaver_test_e2e \
  SCOPEWEAVER_BINARY=/absolute/path/to/scopeweaver npm run smoke
```



既有基本檢查初始化密碼後，儀表板迴歸測試會針對同一測試資料庫，在空閒回送連接埠啟動該執行檔的獨立副本。它檢查英文／韓文 Legacy／New token 堆疊與設定檔篩選，之後停止伺服器並移除測試資料。儀表板證據與子伺服器記錄儲存在 `e2e/e2e-results/dashboard-token-chart/`。

必要基本測試涵蓋六種語言的偏好還原、設定／登入／任務／設定／對話草稿、首次 API 語言、SSE 重連／篩選保留，以及實際呈現的數字格式。

請在 PR 說明執行了哪些檢查，以及哪些無法執行。

<a id="review-and-merging"></a>

## 審查與合併

每個 `feat/*` 或 `fix/*` 分支皆由最新 `dev` 建立，PR 目標設為 `dev`。整合後，等待該確切 `dev` 提交上的所有檢查及 E2E，然後從本儲存庫 `dev` 向 `main` 開升版 PR。升版需使用合併提交，讓 `dev` 維持為 `main` 的祖先。升版後再次驗證檢查；後續工作仍從 `dev` 開始。`branch-policy` 會拒絕其他 PR 路由。

1. 目前候選版的全部必要檢查必須通過：`branch-policy`、`frontend`、`agent-adapter`、`backend`、`windows`、`chat-browser`、`smoke`。`dev` 與 `main` 合併後都執行 CI 及 E2E。首次貢獻者需由維護者核准 CI 執行。
2. 依差異檢查正確性、安全性、已發布語言完整度，以及六語文件對等性。
3. **每個 PR 合併前，維護者均須進行最終人工檢查。** 未經檢查不得合併。

版本由維護者在合併前後完成完整端對端測試後發布。
