[English](../../DESIGN.md) · [한국어](../ko/DESIGN.md) · [简体中文](../zh-Hans/DESIGN.md) · 繁體中文 · [日本語](../ja/DESIGN.md) · [Español](../es/DESIGN.md)

<a id="scopeweaver-landing--designmd"></a>

# ScopeWeaver 介紹頁 — DESIGN.md

各語言版本請見本頁上方導覽。

<!-- impeccable:design-schema 1 -->

ScopeWeaver 的靜態 GitHub Pages 介紹頁。預設英文（`landing/index.html`）；韓文、簡體中文、繁體中文、日文與西班牙文為對等版本（`landing/{ko,zh-Hans,zh-Hant,ja,es}/index.html`）。純 HTML／CSS／JS，無框架、CDN、追蹤器或後端。在專案基底 `/scopeweaver/` 及其語言子目錄提供服務，使用相對素材路徑。

<a id="mode"></a>

## 模式

說服。此頁為產品展示櫥窗：開發者或安全研究者應能了解 ScopeWeaver、看到執行畫面、理解任務流程與程式開發代理程式的操作方式、信任限制說明，並找到發行版或設定指令。

<a id="visual-world"></a>

## 視覺風格

將應用程式控制台延伸為整頁尺度。

- **底色** `#fafafa`、**表面** `#ffffff`、凹陷區 `#f4f4f3`。
- **墨色** `#171717`（亦用於標誌方塊與滿版代理程式區塊），次要／第三層文字為 `#3f3f3f`／`#595959`。深色區塊使用 `#fafafa` 文字與 `#adb0b8` 次要文字（帶色中性色，在 `#171717` 上 ≥ 6:1）。
- **單一節制藍色**，僅用於連結、重點與焦點：淺色背景 `#2563eb`，懸停 `#1b45c9`；深色區塊 `#7cafff`／`#a8c8ff`。連結顏色使用 `--link`／`--link-hover`，深色區塊重新指定，維持懸停可讀性。
- **細線**：1px 的 `#e6e6e4`／`#d4d4d2` 分隔各區，不用粗重分隔線。
- **範圍格線圖樣**：呼應標誌 3×3 格子的淡雙軸格線，置於首屏（徑向遮罩）與代理程式區塊背景。這是已確認的品牌圖樣；偵測器預設標記裝飾格線，但此已確認風格在此優先。
- **深度**：克制的陰影，具有真實偏移與柔和模糊；不使用零偏移彩色光暈。畫面放在淺色應用程式邊框內。

<a id="type"></a>

## 字型

- **Schibsted Grotesk**（自行代管可變 woff2、字重 400–900、OFL）為主要字型：標題字重 640–660，字距 −0.028em，平衡換行；展示字級上限約 3.7rem。
- **JetBrains Mono**（自行代管可變 woff2、OFL）用於識別碼、指令、來源與標籤，啟用 `tnum`／`zero`。
- 僅自行代管拉丁字型；韓文版在拉丁展示字型之後加入系統韓文字型（`Apple SD Gothic Neo`、`Pretendard`、`Malgun Gothic`、`Noto Sans KR`），讓兩版 ASCII 維持同一風格。韓文使用 `word-break: keep-all` + `overflow-wrap: break-word`；正文中的行內 `<code>` 識別碼使用 `overflow-wrap: anywhere`，避免長測試名稱撐寬格線（堆疊格線使用 `minmax(0, 1fr)`）。
- 正文行寬約 33–68ch；功能文字 ≥ 11px。

<a id="structure"></a>

## 結構

以細線分隔的單欄區塊，刻意變化密度：

1. **首屏**：5/7 分欄；左側為產品文案與操作，右側為大型真實儀表板畫面邊框。
2. **操作導覽**：7/5 分欄。垂直 ARIA tablist 切換五個真實畫面（儀表板、任務、發現、系統設定、LLM 範本），更新固定畫面並展開各畫面的單行說明。雙圖、規劃器 → 單意圖工作代理程式 → 事實／發現、人工審查與複測皆以正文說明，不生成示意圖。
3. **代理程式橋接**：滿版墨色區塊。水平 ARIA tablist 切換 Claude Code／Codex／Pi 安裝指令與複製按鈕；八個配接器工具及「控制介面，不取代內部代理程式」的如實說明，與有邊框的行動版畫面並列。
4. **安裝**：一鍵指令碼、Docker Compose、發行封存檔三種具體方式，可複製指令、五個平台封存標籤與 GHCR。
5. **需求**：四個細線格：PostgreSQL、另設模型供應商、Node.js 22+ 配接器、本機實驗室目標。
6. **限制與致謝**：直接列出限制（僅本機隔離；歷史範圍概覽不一致已於 `75a706d` 修正（目前狀態另見[驗證文件](VERIFICATION.md)，不宣稱目前全部測試通過）；不宣稱正式環境／成效；Docker／Windows 未驗證；npm audit 問題；配接器測試用本機資料，程式開發代理程式至供應商的真實執行未獨立驗證，連結 VERIFICATION），旁列去識別化的擁有者流程與 ARTEX／AGPL-3.0、字型／畫面來源。
7. **頁尾**：品牌、資源連結、授權及「介紹頁，非即時服務」聲明。

<a id="signature-interaction"></a>

## 代表互動

操作導覽 tablist。選取畫面時，以 opacity 交叉淡化有邊框的畫面，並透過 `grid-template-rows` 收合動畫顯示說明；只有這個精心設計的互動，不讓每個區塊都入場。使用移動 tabindex 及 Up／Down／Home／End；水平代理程式 tablist 使用 Left／Right。

<a id="accessibility--states"></a>

## 無障礙與狀態

- 跳轉連結、地標區域、帶標籤導覽與明顯的主題焦點框（淺色底用藍，深色底用淺藍）。
- ARIA tabs 正確設定 `aria-selected`／`tabindex`／`hidden` 並可用鍵盤操作；複製按鈕播報本地化狀態（`Copy`／`Copied` · `복사`／`복사됨`），不支援剪貼簿時可手動選取。
- 行動導覽使用 `aria-expanded` 展開控制。頁首項目不縮小；≤420px 時縮減間距，支援至 320px。純圖示 GitHub 連結保留視覺隱藏的 GitHub 文字作為無障礙名稱。
- 瀏覽器表面配合主題：選取、捲軸、焦點框、連結底線偏移及等寬表格數字。
- `prefers-reduced-motion` 停用轉場與平滑捲動。
- 圖片具有 width／height 與 `aspect-ratio` 避免版面位移；首屏畫面設 `fetchpriority="high"`，其餘在首屏下方時延遲載入。

<a id="honesty--provenance"></a>

## 誠實呈現與來源

每張畫面均標示為虛構示範資料。擁有者工作流程段落僅為去識別化的協作證據，不含目標、發現、訊息、模型用量或憑證。宣稱可追溯至 README、配接器 README、PROVENANCE、VERIFICATION 與 v0.1.0 變更記錄。ScopeWeaver 衍生自 ARTEX，依 AGPL-3.0 授權；僅限本機隔離研究。

<a id="verification-performed-build-time-non-browser"></a>

## 已執行驗證（建置時，非瀏覽器）

- `tools/validate-landing.mjs`（位於 `landing/` 外）：素材存在、相對基底路徑安全、EN／KO id／tab／control 對等、語言互連、示範圖說、未發布規劃中繼資料、存在 `.nojekyll`，共 17/17。
- 對 JS 與驗證指令碼執行 `node --check`。
- Impeccable 偵測器：19 → 4 個殘留（單一重點懸停顏色與深色區塊靜態配對；已確認滿版區塊由內層 wrap 內縮），兩者實際轉譯正確。格線圖樣提示屬已確認品牌風格。
- 協調者在 Chrome 1440px 與 390px 檢視英文及韓文，套用一批修正後，確認兩語在 320px 也無水平溢出。畫面分頁、代理程式分頁、鍵盤切換、行動選單、語言連結與本地化複製回饋通過，未觀察到瀏覽器主控台錯誤；導覽時載入首屏下方的延遲畫面。
- Pages 僅部署 `landing/`；上傳前工作流程執行檔案完整性與 JavaScript 語法檢查。開發文件與私人執行資料不包含在公開成品中。

<a id="implementation-attribution"></a>

## 實作歸屬

原請求為 Claude Code 使用 `claude-opus-5-5`、medium effort。主要實作工作階段在供應商保護中斷後自動回退至 `claude-opus-4-8`；有界的瀏覽器 QA 修正階段全程使用 `claude-opus-5-5`、medium effort。因此不得將整頁完全歸屬 Opus 5.5。
