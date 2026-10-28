[English](../../../../../skills/playwright-cli/references/tracing.md) · [한국어](../../../../ko/skills/playwright-cli/references/tracing.md) · [简体中文](../../../../zh-Hans/skills/playwright-cli/references/tracing.md) · 繁體中文 · [日本語](../../../../ja/skills/playwright-cli/references/tracing.md) · [Español](../../../../es/skills/playwright-cli/references/tracing.md)

<a id="tracing"></a>

# 追蹤

各語言版本請見本頁上方導覽。

擷取詳細執行追蹤以供偵錯與分析。追蹤包含 DOM 快照、畫面、網路活動與主控台記錄。

<a id="basic-usage"></a>

## 基本用法



```bash
# Start trace recording
playwright-cli tracing-start

# Perform actions
playwright-cli open https://example.com
playwright-cli click e1
playwright-cli fill e2 "test"

# Stop trace recording
playwright-cli tracing-stop
```



<a id="trace-output-files"></a>

## 追蹤輸出檔案

開始追蹤時，Playwright 會建立含多個檔案的 `traces/` 目錄：

<a id="trace-timestamptrace"></a>

### `trace-{timestamp}.trace`

**操作記錄**：主要追蹤檔包含：
- 每個操作（點選、填寫、導覽）
- 每次操作前後 DOM 快照
- 各步驟畫面
- 時序資訊
- 主控台訊息
- 原始碼位置

<a id="trace-timestampnetwork"></a>

### `trace-{timestamp}.network`

**網路記錄**：完整網路活動：
- 所有 HTTP 請求與回應
- 請求標頭與本文
- 回應標頭與本文
- 時序（DNS、連線、TLS、TTFB、下載）
- 資源大小
- 失敗請求與錯誤

<a id="resources"></a>

### `resources/`

**資源目錄**：快取資源：
- 圖片、字型、樣式表、指令碼
- 重播用回應本文
- 重建頁面狀態所需素材

<a id="what-traces-capture"></a>

## 追蹤擷取內容

| 類別 | 詳情 |
|----------|---------|
| **操作** | 點選、填寫、懸停、鍵盤輸入、導覽 |
| **DOM** | 每次操作前後完整 DOM 快照 |
| **畫面** | 各步驟視覺狀態 |
| **網路** | 全部請求、回應、標頭、本文與時序 |
| **主控台** | 全部 console.log、warn、error 訊息 |
| **時序** | 每項操作精確時間 |

<a id="use-cases"></a>

## 使用情境

<a id="debugging-failed-actions"></a>

### 偵錯失敗操作



```bash
playwright-cli tracing-start
playwright-cli open https://app.example.com

# This click fails - why?
playwright-cli click e5

playwright-cli tracing-stop
# Open trace to see DOM state when click was attempted
```



<a id="analyzing-performance"></a>

### 分析效能



```bash
playwright-cli tracing-start
playwright-cli open https://slow-site.com
playwright-cli tracing-stop

# View network waterfall to identify slow resources
```



<a id="capturing-evidence"></a>

### 擷取證據



```bash
# Record a complete user flow for documentation
playwright-cli tracing-start

playwright-cli open https://app.example.com/checkout
playwright-cli fill e1 "4111111111111111"
playwright-cli fill e2 "12/25"
playwright-cli fill e3 "123"
playwright-cli click e4

playwright-cli tracing-stop
# Trace shows exact sequence of events
```



<a id="trace-vs-video-vs-screenshot"></a>

## 追蹤、影片與畫面比較

| 功能 | 追蹤 | 影片 | 畫面 |
|---------|-------|-------|------------|
| **格式** | .trace 檔 | .webm 影片 | .png/.jpeg 圖片 |
| **DOM 檢查** | 是 | 否 | 否 |
| **網路詳情** | 是 | 否 | 否 |
| **逐步重播** | 是 | 連續 | 單張 |
| **檔案大小** | 中 | 大 | 小 |
| **最適用途** | 偵錯 | 示範 | 快速擷取 |

<a id="best-practices"></a>

## 最佳實務

<a id="1-start-tracing-before-the-problem"></a>

### 1. 在問題發生前開始追蹤



```bash
# Trace the entire flow, not just the failing step
playwright-cli tracing-start
playwright-cli open https://example.com
# ... all steps leading to the issue ...
playwright-cli tracing-stop
```



<a id="2-clean-up-old-traces"></a>

### 2. 清理舊追蹤

追蹤可能佔用大量磁碟空間：



```bash
# Remove traces older than 7 days
find .playwright-cli/traces -mtime +7 -delete
```



<a id="limitations"></a>

## 限制

- 追蹤增加自動化開銷。
- 大型追蹤可能佔用大量磁碟空間。
- 部分動態內容可能無法完整重播。
