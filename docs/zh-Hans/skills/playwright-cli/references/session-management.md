[English](../../../../../skills/playwright-cli/references/session-management.md) · [한국어](../../../../ko/skills/playwright-cli/references/session-management.md) · 简体中文 · [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/session-management.md) · [日本語](../../../../ja/skills/playwright-cli/references/session-management.md) · [Español](../../../../es/skills/playwright-cli/references/session-management.md)

<a id="browser-session-management"></a>

# 浏览器会话管理

各语言版本请见本页上方导览。

同时运行多个隔离浏览器会话，并保留状态。

<a id="named-browser-sessions"></a>

## 具名浏览器会话

使用 `-s` 隔离浏览器上下文：



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

## 浏览器会话隔离属性

每个浏览器会话皆独立拥有：
- Cookie
- LocalStorage／SessionStorage
- IndexedDB
- 缓存
- 浏览历史
- 开启的分页

<a id="browser-session-commands"></a>

## 浏览器会话命令



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

## 环境变量

以环境变量指定默认浏览器会话名称：



```bash
export PLAYWRIGHT_CLI_SESSION="mysession"
playwright-cli open example.com  # Uses "mysession" automatically
```



<a id="common-patterns"></a>

## 常见模式

<a id="concurrent-scraping"></a>

### 并行捕获



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

### A/B 测试会话



```bash
# Test different user experiences
playwright-cli -s=variant-a open "https://app.com?variant=a"
playwright-cli -s=variant-b open "https://app.com?variant=b"

# Compare
playwright-cli -s=variant-a screenshot
playwright-cli -s=variant-b screenshot
```



<a id="persistent-profile"></a>

### 持久配置

默认浏览器配置仅存于内存。`open` 加 `--persistent` 可保存至磁碟：



```bash
# Use persistent profile (auto-generated location)
playwright-cli open https://example.com --persistent

# Use persistent profile with custom directory
playwright-cli open https://example.com --profile=/path/to/profile
```



<a id="attaching-to-a-running-browser"></a>

## 附加至运行中的浏览器

以 `attach` 连接已运行的浏览器，无须启动新的浏览器。

<a id="attach-by-channel-name"></a>

### 依通道名称附加

依通道名称连接运行中的 Chrome 或 Edge。浏览器必须启用远程侦错：在目标浏览器开启 `chrome://inspect/#remote-debugging`，勾选「Allow remote debugging for this browser instance」。



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



支持通道：`chrome`、`chrome-beta`、`chrome-dev`、`chrome-canary`、`msedge`、`msedge-beta`、`msedge-dev`、`msedge-canary`。

未提供 `--session` 时，以通道命名（例如 `--cdp=msedge` 创建 `msedge` 会话），避免平行附加 Chrome 与 Edge 共用 `default`。以 `--session=<name>` 覆写。

<a id="attach-via-cdp-endpoint"></a>

### 通过 CDP 端点附加

连接公开 Chrome DevTools Protocol 端点的浏览器：



```bash
playwright-cli attach --cdp=http://localhost:9222
```



<a id="attach-via-browser-extension"></a>

### 通过浏览器扩展功能附加

连接已安装 Playwright 扩展功能的浏览器：



```bash
playwright-cli attach --extension
```



<a id="detach"></a>

### 分离

结束附加的会话，不影响外部浏览器：



```bash
# Detach the default attached session
playwright-cli detach

# Detach a specific attached session
playwright-cli -s=msedge detach
```



`detach` 仅适用于通过 `attach` 创建的会话；通过 `open` 创建者请用 `close`。

<a id="default-browser-session"></a>

## 默认浏览器会话

省略 `-s` 时，命令使用默认浏览器会话：



```bash
# These use the same default browser session
playwright-cli open https://example.com
playwright-cli snapshot
playwright-cli close  # Stops default browser
```



<a id="browser-session-configuration"></a>

## 浏览器会话配置

开启时应用特定会话配置：



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

## 最佳实务

<a id="1-name-browser-sessions-semantically"></a>

### 1. 使用具语意的会话名称



```bash
# GOOD: Clear purpose
playwright-cli -s=github-auth open https://github.com
playwright-cli -s=docs-scrape open https://docs.example.com

# AVOID: Generic names
playwright-cli -s=s1 open https://github.com
```



<a id="2-always-clean-up"></a>

### 2. 务必清理



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

### 3. 删除过期浏览器数据



```bash
# Remove old browser data to free disk space
playwright-cli -s=oldsession delete-data
```
