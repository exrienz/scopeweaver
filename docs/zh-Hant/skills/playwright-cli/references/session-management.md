[English](../../../../../skills/playwright-cli/references/session-management.md) · [한국어](../../../../ko/skills/playwright-cli/references/session-management.md) · [简体中文](../../../../zh-Hans/skills/playwright-cli/references/session-management.md) · 繁體中文 · [日本語](../../../../ja/skills/playwright-cli/references/session-management.md) · [Español](../../../../es/skills/playwright-cli/references/session-management.md)

<a id="browser-session-management"></a>

# 瀏覽器工作階段管理

各語言版本請見本頁上方導覽。

同時執行多個隔離瀏覽器工作階段，並保留狀態。

<a id="named-browser-sessions"></a>

## 具名瀏覽器工作階段

使用 `-s` 隔離瀏覽器上下文：



```bash
# Browser 1: Authentication flow
playwright-cli -s=auth open https://app.example.com/login

# Browser 2: Public browsing (separate cookies, storage)
playwright-cli -s=public open https://example.com

# Commands are isolated by browser session
playwright-cli -s=auth fill e1 "user@example.com"
playwright-cli -s=public snapshot
```



<a id="browser-session-isolation-properties"></a>

## 瀏覽器工作階段隔離屬性

每個瀏覽器工作階段皆獨立擁有：
- Cookie
- LocalStorage／SessionStorage
- IndexedDB
- 快取
- 瀏覽歷史
- 開啟的分頁

<a id="browser-session-commands"></a>

## 瀏覽器工作階段指令



```bash
# List all browser sessions
playwright-cli list

# Stop a browser session (close the browser)
playwright-cli close                # stop the default browser
playwright-cli -s=mysession close   # stop a named browser

# Stop all browser sessions
playwright-cli close-all

# Forcefully kill all daemon processes (for stale/zombie processes)
playwright-cli kill-all

# Delete browser session user data (profile directory)
playwright-cli delete-data                # delete default browser data
playwright-cli -s=mysession delete-data   # delete named browser data
```



<a id="environment-variable"></a>

## 環境變數

以環境變數指定預設瀏覽器工作階段名稱：



```bash
export PLAYWRIGHT_CLI_SESSION="mysession"
playwright-cli open example.com  # Uses "mysession" automatically
```



<a id="common-patterns"></a>

## 常見模式

<a id="concurrent-scraping"></a>

### 並行擷取



```bash
#!/bin/bash
# Scrape multiple sites concurrently

# Start all browsers
playwright-cli -s=site1 open https://site1.com &
playwright-cli -s=site2 open https://site2.com &
playwright-cli -s=site3 open https://site3.com &
wait

# Take snapshots from each
playwright-cli -s=site1 snapshot
playwright-cli -s=site2 snapshot
playwright-cli -s=site3 snapshot

# Cleanup
playwright-cli close-all
```



<a id="ab-testing-sessions"></a>

### A/B 測試工作階段



```bash
# Test different user experiences
playwright-cli -s=variant-a open "https://app.com?variant=a"
playwright-cli -s=variant-b open "https://app.com?variant=b"

# Compare
playwright-cli -s=variant-a screenshot
playwright-cli -s=variant-b screenshot
```



<a id="persistent-profile"></a>

### 持久設定檔

預設瀏覽器設定檔僅存於記憶體。`open` 加 `--persistent` 可儲存至磁碟：



```bash
# Use persistent profile (auto-generated location)
playwright-cli open https://example.com --persistent

# Use persistent profile with custom directory
playwright-cli open https://example.com --profile=/path/to/profile
```



<a id="attaching-to-a-running-browser"></a>

## 附加至執行中的瀏覽器

以 `attach` 連接已執行的瀏覽器，無須啟動新的瀏覽器。

<a id="attach-by-channel-name"></a>

### 依通道名稱附加

依通道名稱連接執行中的 Chrome 或 Edge。瀏覽器必須啟用遠端偵錯：在目標瀏覽器開啟 `chrome://inspect/#remote-debugging`，勾選「Allow remote debugging for this browser instance」。



```bash
# Attach to Chrome
playwright-cli attach --cdp=chrome

# Attach to Chrome Canary
playwright-cli attach --cdp=chrome-canary

# Attach to Microsoft Edge
playwright-cli attach --cdp=msedge

# Attach to Edge Dev
playwright-cli attach --cdp=msedge-dev
```



支援通道：`chrome`、`chrome-beta`、`chrome-dev`、`chrome-canary`、`msedge`、`msedge-beta`、`msedge-dev`、`msedge-canary`。

未提供 `--session` 時，以通道命名（例如 `--cdp=msedge` 建立 `msedge` 工作階段），避免平行附加 Chrome 與 Edge 共用 `default`。以 `--session=<name>` 覆寫。

<a id="attach-via-cdp-endpoint"></a>

### 透過 CDP 端點附加

連接公開 Chrome DevTools Protocol 端點的瀏覽器：



```bash
playwright-cli attach --cdp=http://localhost:9222
```



<a id="attach-via-browser-extension"></a>

### 透過瀏覽器擴充功能附加

連接已安裝 Playwright 擴充功能的瀏覽器：



```bash
playwright-cli attach --extension
```



<a id="detach"></a>

### 分離

結束附加的工作階段，不影響外部瀏覽器：



```bash
# Detach the default attached session
playwright-cli detach

# Detach a specific attached session
playwright-cli -s=msedge detach
```



`detach` 僅適用於透過 `attach` 建立的工作階段；透過 `open` 建立者請用 `close`。

<a id="default-browser-session"></a>

## 預設瀏覽器工作階段

省略 `-s` 時，指令使用預設瀏覽器工作階段：



```bash
# These use the same default browser session
playwright-cli open https://example.com
playwright-cli snapshot
playwright-cli close  # Stops default browser
```



<a id="browser-session-configuration"></a>

## 瀏覽器工作階段設定

開啟時套用特定工作階段設定：



```bash
# Open with config file
playwright-cli open https://example.com --config=.playwright/my-cli.json

# Open with specific browser
playwright-cli open https://example.com --browser=firefox

# Open in headed mode
playwright-cli open https://example.com --headed

# Open with persistent profile
playwright-cli open https://example.com --persistent
```



<a id="best-practices"></a>

## 最佳實務

<a id="1-name-browser-sessions-semantically"></a>

### 1. 使用具語意的工作階段名稱



```bash
# GOOD: Clear purpose
playwright-cli -s=github-auth open https://github.com
playwright-cli -s=docs-scrape open https://docs.example.com

# AVOID: Generic names
playwright-cli -s=s1 open https://github.com
```



<a id="2-always-clean-up"></a>

### 2. 務必清理



```bash
# Stop browsers when done
playwright-cli -s=auth close
playwright-cli -s=scrape close

# Or stop all at once
playwright-cli close-all

# If browsers become unresponsive or zombie processes remain
playwright-cli kill-all
```



<a id="3-delete-stale-browser-data"></a>

### 3. 刪除過期瀏覽器資料



```bash
# Remove old browser data to free disk space
playwright-cli -s=oldsession delete-data
```
