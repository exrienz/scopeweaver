[English](../../PRODUCT.md) · [한국어](../ko/PRODUCT.md) · [简体中文](../zh-Hans/PRODUCT.md) · 繁體中文 · [日本語](../ja/PRODUCT.md) · [Español](../es/PRODUCT.md)

<a id="scopeweaver-product"></a>

# ScopeWeaver 產品

各語言版本請見本頁上方導覽。

<!-- impeccable:product-schema 1 -->

<a id="platform"></a>

## 平台
網頁

<a id="stack"></a>

## 技術組合
應用程式使用 Go 後端、內嵌 Next.js 前端及 PostgreSQL。另行核准的 GitHub Pages 介紹頁採純 HTML、CSS 與 JavaScript，必須不依賴應用程式後端即可執行。

<a id="users"></a>

## 使用者
評估 AI 輔助工作流程、進行本機隔離研究與驗證的開發者及安全研究者。此受眾與六語應用程式／文件範圍均已獲擁有者核准。

<a id="product-purpose"></a>

## 產品目的
在同一自行代管的研究控制台中，協調目標規劃、代理程式、資產範圍、覆蓋率、發現記錄與人工審查。介紹頁協助訪客理解產品、檢視如實呈現的證據，並找到下載／設定文件。

<a id="positioning"></a>

## 定位
共用資產圖與各任務探索圖將目標連結至代理程式方向、觀察及證據。外部代理程式控制配接器向 Claude Code、Codex、Pi 提供任務與發現工作流程。

<a id="operating-context"></a>

## 操作情境
自行代管應用程式，需要 PostgreSQL 與另行設定的模型供應商才能執行代理程式。GitHub Pages 僅代管公開靜態介紹，不提供即時安全測試服務或後端。

<a id="capabilities-and-constraints"></a>

## 能力與限制
功能與設定依據 README.md、adapters/agent/README.md 及程式碼。v0.1.0 提供五個平台封存檔與多架構 Docker 映像檔。配接器為控制介面，不取代內部代理程式，也不會將程式開發訂閱轉為 API 金鑰。儲存庫使用限制允許本機隔離研究／驗證，禁止線上目標測試。歷史遺留的後端範圍概覽不一致問題已在提交 `75a706d` 中修復。驗證證據與限制請參閱[VERIFICATION.md](VERIFICATION.md)；這不代表目前全部測試皆已通過，也不代表已可供正式環境使用。

<a id="brand-commitments"></a>

## 品牌承諾
保留 ScopeWeaver 名稱與既有範圍格線標誌。應用程式畫面建立精確、中性的控制台風格，僅使用節制的藍色重點。預設英文，可選韓文、簡體中文、繁體中文、日文與西班牙文。尊重 ARTEX 作者歸屬與 AGPL-3.0。

<a id="evidence-on-hand"></a>

## 現有證據
儲存庫的 screenshots/en 與 screenshots/ko 畫面展示虛構示範資料，必須標示為示範畫面。既有標誌：web/public/scopeweaver.svg。README、來源、驗證說明、配接器文件與發行成品為主要證據。已在登入的本機介面檢視擁有者實際執行的對話，記錄了建立任務、檢查探索圖與工作代理程式追蹤及韓文進度更新。公開頁面只可使用此保護隱私的流程摘要，不得包含目標名稱、原始訊息、漏洞宣稱、token 數或私人畫面。這是協作流程證據，並非獨立安全驗證。未提供經驗證客戶推薦、採用統計、效能基準或外部稽核安全宣稱。

<a id="product-principles"></a>

## 產品原則
產品宣稱必須可查證。區分公開介紹、示範資料與實際執行環境。入口附近說明設定與模型需求。保留範圍／核准機制。絕不發布憑證、目標詳情或私人對話。

<a id="accessibility--inclusion"></a>

## 無障礙與包容
支援桌面與行動網頁的回應式設計、可用鍵盤操作的控制、可讀對比、減少動態效果，以及完整英文、韓文、簡體中文、繁體中文、日文與西班牙文文案。
