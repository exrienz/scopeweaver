[English](../../../../skills/playwright-cli/SKILL.md) · [한국어](../../../ko/skills/playwright-cli/README.md) · 简体中文 · [繁體中文](../../../zh-Hant/skills/playwright-cli/README.md) · [日本語](../../../ja/skills/playwright-cli/README.md) · [Español](../../../es/skills/playwright-cli/README.md)

---
name: playwright-cli
description: 自动化浏览器互动、测试网页并使用 Playwright 测试。
allowed-tools: Bash(playwright-cli:*) Bash(npx:*) Bash(npm:*)
---

<a id="browser-automation-with-playwright-cli"></a>

# 使用 playwright-cli 自动操作浏览器

各语言版本请见本页上方导览。

<a id="quick-start"></a>

## 快速开始



```bash
# open new browser
playwright-cli open
# navigate to a page
playwright-cli goto https://playwright.dev
# interact with the page using refs from the snapshot
playwright-cli click e15
playwright-cli type "page.click"
playwright-cli press Enter
# take a screenshot (rarely used, as snapshot is more common)
playwright-cli screenshot
# close the browser
playwright-cli close
```



<a id="commands"></a>

## 命令

<a id="core"></a>

### 核心



```bash
playwright-cli open
# open and navigate right away
playwright-cli open https://example.com/
playwright-cli goto https://playwright.dev
playwright-cli type "search query"
playwright-cli click e3
playwright-cli dblclick e7
# --submit presses Enter after filling the element
playwright-cli fill e5 "user@example.com"  --submit
playwright-cli drag e2 e8
# drop files or data onto an element (from outside the page)
playwright-cli drop e4 --path=./image.png
playwright-cli drop e4 --data="text/plain=hello world"
playwright-cli hover e4
playwright-cli select e9 "option-value"
playwright-cli upload ./document.pdf
playwright-cli check e12
playwright-cli uncheck e12
playwright-cli snapshot
# search the snapshot for text or a regexp, returns matching nodes with surrounding context
playwright-cli find "Sign in"
playwright-cli find --regex "Sign (in|up)"
# wrap the regexp in slashes to add flags, e.g. /i for case-insensitive
playwright-cli find --regex "/sign (in|up)/i"
playwright-cli eval "document.title"
playwright-cli eval "el => el.textContent" e5
# get element id, class, or any attribute not visible in the snapshot
playwright-cli eval "el => el.id" e5
playwright-cli eval "el => el.getAttribute('data-testid')" e5
playwright-cli dialog-accept
playwright-cli dialog-accept "confirmation text"
playwright-cli dialog-dismiss
playwright-cli resize 1920 1080
playwright-cli close
```



<a id="navigation"></a>

### 导览



```bash
playwright-cli go-back
playwright-cli go-forward
playwright-cli reload
```



<a id="keyboard"></a>

### 键盘



```bash
playwright-cli press Enter
playwright-cli press ArrowDown
playwright-cli keydown Shift
playwright-cli keyup Shift
```



<a id="mouse"></a>

### 滑鼠



```bash
playwright-cli mousemove 150 300
playwright-cli mousedown
playwright-cli mousedown right
playwright-cli mouseup
playwright-cli mouseup right
playwright-cli mousewheel 0 100
```



<a id="save-as"></a>

### 另存



```bash
playwright-cli screenshot
playwright-cli screenshot e5
playwright-cli screenshot --filename=page.png
playwright-cli screenshot --hires
playwright-cli pdf --filename=page.pdf
```



<a id="tabs"></a>

### 分页



```bash
playwright-cli tab-list
playwright-cli tab-new
playwright-cli tab-new https://example.com/page
playwright-cli tab-close
playwright-cli tab-close 2
playwright-cli tab-select 0
```



<a id="storage"></a>

### 保存空间



```bash
playwright-cli state-save
playwright-cli state-save auth.json
playwright-cli state-load auth.json

# Cookies
playwright-cli cookie-list
playwright-cli cookie-list --domain=example.com
playwright-cli cookie-get session_id
playwright-cli cookie-set session_id abc123
playwright-cli cookie-set session_id abc123 --domain=example.com --httpOnly --secure
playwright-cli cookie-delete session_id
playwright-cli cookie-clear

# LocalStorage
playwright-cli localstorage-list
playwright-cli localstorage-get theme
playwright-cli localstorage-set theme dark
playwright-cli localstorage-delete theme
playwright-cli localstorage-clear

# SessionStorage
playwright-cli sessionstorage-list
playwright-cli sessionstorage-get step
playwright-cli sessionstorage-set step 3
playwright-cli sessionstorage-delete step
playwright-cli sessionstorage-clear
```



<a id="network"></a>

### 网络



```bash
playwright-cli route "**/*.jpg" --status=404
playwright-cli route "https://api.example.com/**" --body='{"mock": true}'
playwright-cli route-list
playwright-cli unroute "**/*.jpg"
playwright-cli unroute
```



<a id="devtools"></a>

### 开发工具



```bash
playwright-cli console
playwright-cli console warning
playwright-cli requests
playwright-cli request 5
playwright-cli run-code "async page => await page.context().grantPermissions(['geolocation'])"
playwright-cli run-code --filename=script.js
playwright-cli tracing-start
playwright-cli tracing-stop
playwright-cli video-start video.webm
playwright-cli video-chapter "Chapter Title" --description="Details" --duration=2000
playwright-cli video-stop

# annotate each subsequent action (click, type, ...) with a callout naming the action and highlighting the target
playwright-cli video-show-actions --duration=600 --position=top-right
playwright-cli video-hide-actions

# launch the dashboard for UI review / design feedback — user annotates the page, you receive the annotated screenshot, snapshot, and notes
playwright-cli show --annotate

# generate a Playwright locator for an element from its ref or selector
playwright-cli generate-locator e5 --raw

# show a persistent highlight overlay for an element, optionally with a custom style
playwright-cli highlight e5
playwright-cli highlight e5 --style="outline: 3px dashed red"
# hide a single element highlight, or all page highlights when no target is given
playwright-cli highlight e5 --hide
playwright-cli highlight --hide
```



<a id="raw-output"></a>

## 原始输出

全域 `--raw` 选项会移除输出中的页面状态、生成代码与快照区段，仅传回结果值。可用来将命令输出管线传给其他工具。原本没有输出的命令不传回任何内容。



```bash
playwright-cli --raw eval "JSON.stringify(performance.timing)" | jq '.loadEventEnd - .navigationStart'
playwright-cli --raw eval "JSON.stringify([...document.querySelectorAll('a')].map(a => a.href))" > links.json
playwright-cli --raw snapshot > before.yml
playwright-cli click e5
playwright-cli --raw snapshot > after.yml
diff before.yml after.yml
TOKEN=$(playwright-cli --raw cookie-get session_id)
playwright-cli --raw localstorage-get theme
```



若要将每次回复包成 JSON 结构化输出，请加上 --json。

```bash
playwright-cli list --json
```



<a id="open-parameters"></a>

## 开启参数

```bash
# Use specific browser when creating session
playwright-cli open --browser=chrome
playwright-cli open --browser=firefox
playwright-cli open --browser=webkit
playwright-cli open --browser=msedge

# Emulate a generic mobile device (Pixel 10 for Chromium, iPhone 17 for WebKit).
# Prefer this when a mobile layout is acceptable: mobile pages are usually
# lighter, so snapshots are smaller and cheaper.
playwright-cli open --mobile
playwright-cli open --device="iPhone 15"

# Use persistent profile (by default profile is in-memory)
playwright-cli open --persistent
# Use persistent profile with custom directory
playwright-cli open --profile=/path/to/profile

# Connect to browser via Playwright Extension
playwright-cli attach --extension=chrome

# Connect to a running Chrome or Edge by channel name
playwright-cli attach --cdp=chrome
playwright-cli attach --cdp=msedge

# Connect to a running browser via CDP endpoint
playwright-cli attach --cdp=http://localhost:9222

# Start with config file
playwright-cli open --config=my-config.json

# Close the browser
playwright-cli close
# Detach from an attached browser (leaves the external browser running)
playwright-cli -s=msedge detach
# Delete user data for the default session
playwright-cli delete-data
```



<a id="urls-with--on-windows"></a>

## Windows 中含 `&` 的 URL

Windows 的 `cmd.exe` 与 PowerShell 将 `&` 视为命令分隔符，因此含多个查询参数的 URL 会在 `playwright-cli` 运行前被截断。`cmd.exe` 请以 `^&` 跳脱 `&`，PowerShell 请使用 `--%`：



```batch
playwright-cli goto "https://example.com/?a=1^&b=2"
```





```powershell
playwright-cli --% goto "https://example.com/?a=1&b=2"
```



<a id="snapshots"></a>

## 快照

每个命令后，playwright-cli 都会提供目前浏览器状态快照。



```bash
> playwright-cli goto https://example.com
### Page
- Page URL: https://example.com/
- Page Title: Example Domain
### Snapshot
[Snapshot](.playwright-cli/page-2026-02-14T19-22-42-679Z.yml)
```



亦可随时以 `playwright-cli snapshot` 取得快照；下列选项可依需要组合。



```bash
# default - save to a file with timestamp-based name
playwright-cli snapshot

# save to file, use when snapshot is a part of the workflow result
playwright-cli snapshot --filename=after-click.yaml

# snapshot an element instead of the whole page
playwright-cli snapshot "#main"

# limit snapshot depth for efficiency, take a partial snapshot afterwards
playwright-cli snapshot --depth=4
playwright-cli snapshot e34

# include each element's bounding box as [box=x,y,width,height]
playwright-cli snapshot --boxes

# search a large snapshot instead of capturing it all — returns matching nodes
# with 3 lines of context around each match (like grep -C)
playwright-cli find "Add to cart"
playwright-cli find --regex "\\$[0-9]+\\.[0-9]{2}"
```



<a id="targeting-elements"></a>

## 指定元素

默认使用快照的 ref 与页面元素互动。



```bash
# get snapshot with refs
playwright-cli snapshot

# interact using a ref
playwright-cli click e15
```



亦可使用 CSS 选择器或 Playwright locator。



```bash
# css selector
playwright-cli click "#main > button.submit"

# role locator
playwright-cli click "getByRole('button', { name: 'Submit' })"

# test id
playwright-cli click "getByTestId('submit-button')"
```



<a id="browser-sessions"></a>

## 浏览器会话



```bash
# create new browser session named "mysession" with persistent profile
playwright-cli -s=mysession open example.com --persistent
# same with manually specified profile directory (use when requested explicitly)
playwright-cli -s=mysession open example.com --profile=/path/to/profile
playwright-cli -s=mysession click e6
playwright-cli -s=mysession close  # stop a named browser
playwright-cli -s=mysession delete-data  # delete user data for persistent session

playwright-cli list
# Close all browsers
playwright-cli close-all
# Forcefully kill all browser processes
playwright-cli kill-all
```



<a id="installation"></a>

## 安装

若无全域 `playwright-cli` 命令，请尝试本地版本 `npx playwright cli`：



```bash
npx --no-install playwright --version
```



若本地版本可用，全部命令使用 `npx playwright cli`；否则将 `playwright-cli` 安装为全域命令：



```bash
npm install -g @playwright/cli@latest
```



<a id="example-form-submission"></a>

## 示例：提交表单



```bash
playwright-cli open https://example.com/form
playwright-cli snapshot

playwright-cli fill e1 "user@example.com"
playwright-cli fill e2 "password123"
playwright-cli click e3
playwright-cli snapshot
playwright-cli close
```



<a id="example-multi-tab-workflow"></a>

## 示例：多分页工作流



```bash
playwright-cli open https://example.com
playwright-cli tab-new https://example.com/other
playwright-cli tab-list
playwright-cli tab-select 0
playwright-cli snapshot
playwright-cli close
```



<a id="example-debugging-with-devtools"></a>

## 示例：以开发工具侦错



```bash
playwright-cli open https://example.com
playwright-cli click e4
playwright-cli fill e7 "test"
playwright-cli console
playwright-cli requests
playwright-cli close
```





```bash
playwright-cli open https://example.com
playwright-cli tracing-start
playwright-cli click e4
playwright-cli fill e7 "test"
playwright-cli tracing-stop
playwright-cli close
```



<a id="example-interactive-session"></a>

## 示例：互动会话

请用户提供界面审查或设计回馈。用户在实际页面画框并输入意见；您会收到注记画面、标记区域快照与备注。用户要求「界面审查」、「设计回馈」或「询问用户想法／需求／含义」时，请使用此功能：



```bash
playwright-cli open https://example.com
playwright-cli show --annotate
```



<a id="specific-tasks"></a>

## 特定任务

* **运行与侦错 Playwright 测试**：[references/playwright-tests.md](references/playwright-tests.md)
* **请求模拟**：[references/request-mocking.md](references/request-mocking.md)
* **运行 Playwright 代码**：[references/running-code.md](references/running-code.md)
* **浏览器会话管理**：[references/session-management.md](references/session-management.md)
* **保存状态（cookie、localStorage）**：[references/storage-state.md](references/storage-state.md)
* **测试生成（规划／生成／修复）**：[references/test-generation.md](references/test-generation.md)
* **追踪**：[references/tracing.md](references/tracing.md)
* **影片录制**：[references/video-recording.md](references/video-recording.md)
* **检查元素属性**：[references/element-attributes.md](references/element-attributes.md)
