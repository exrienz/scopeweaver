[English](../../../../skills/playwright-cli/SKILL.md) | [한국어](../../../ko/skills/playwright-cli/README.md) | [简体中文](../../../zh-Hans/skills/playwright-cli/README.md) | [繁體中文](../../../zh-Hant/skills/playwright-cli/README.md) | [日本語](README.md) | [Español](../../../es/skills/playwright-cli/README.md)

---
name: playwright-cli
description: ブラウザー操作の自動化、ウェブページの検証、Playwright テストの操作を行います。
allowed-tools: Bash(playwright-cli:*) Bash(npx:*) Bash(npm:*)
---

<a id="browser-automation-with-playwright-cli"></a>
# playwright-cli によるブラウザーの自動操作


<a id="quick-start"></a>
## クイックスタート

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
## コマンド

<a id="core"></a>
### 基本操作

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
### ページ移動

```bash
playwright-cli go-back
playwright-cli go-forward
playwright-cli reload
```

<a id="keyboard"></a>
### キーボード

```bash
playwright-cli press Enter
playwright-cli press ArrowDown
playwright-cli keydown Shift
playwright-cli keyup Shift
```

<a id="mouse"></a>
### マウス

```bash
playwright-cli mousemove 150 300
playwright-cli mousedown
playwright-cli mousedown right
playwright-cli mouseup
playwright-cli mouseup right
playwright-cli mousewheel 0 100
```

<a id="save-as"></a>
### 保存

```bash
playwright-cli screenshot
playwright-cli screenshot e5
playwright-cli screenshot --filename=page.png
playwright-cli screenshot --hires
playwright-cli pdf --filename=page.pdf
```

<a id="tabs"></a>
### タブ

```bash
playwright-cli tab-list
playwright-cli tab-new
playwright-cli tab-new https://example.com/page
playwright-cli tab-close
playwright-cli tab-close 2
playwright-cli tab-select 0
```

<a id="storage"></a>
### ストレージ

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
### ネットワーク

```bash
playwright-cli route "**/*.jpg" --status=404
playwright-cli route "https://api.example.com/**" --body='{"mock": true}'
playwright-cli route-list
playwright-cli unroute "**/*.jpg"
playwright-cli unroute
```

<a id="devtools"></a>
### 開発者ツール

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
## 生の出力

全体オプション `--raw` は、ページの状態、生成コード、スナップショットの節を出力から除き、結果の値だけを返します。コマンドの出力を他のツールに渡す場合に使ってください。出力しないコマンドは何も返しません。

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

すべての応答を JSON で包んだ構造化出力には、--json を指定します。
```bash
playwright-cli list --json
```

<a id="open-parameters"></a>
## open のパラメーター
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
## Windows で `&` を含む URL

Windows の `cmd.exe` と PowerShell は `&` をコマンドの区切りとして扱うため、複数のクエリパラメーターを持つ URL は、`playwright-cli` の実行前に切られます。`cmd.exe` では `&` を `^&` でエスケープし、PowerShell では `--%` を使ってください:

```batch
playwright-cli goto "https://example.com/?a=1^&b=2"
```

```powershell
playwright-cli --% goto "https://example.com/?a=1&b=2"
```

<a id="snapshots"></a>
## スナップショット

各コマンド後、playwright-cli は現在のブラウザー状態のスナップショットを提供します。

```bash
> playwright-cli goto https://example.com
### Page
- Page URL: https://example.com/
- Page Title: Example Domain
### Snapshot
[Snapshot](.playwright-cli/page-2026-02-14T19-22-42-679Z.yml)
```

`playwright-cli snapshot` で、必要なときに取得することもできます。下記のオプションは必要に応じて併用できます。

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
## 要素の指定

既定では、スナップショットの参照を使ってページ要素を操作してください。

```bash
# get snapshot with refs
playwright-cli snapshot

# interact using a ref
playwright-cli click e15
```

CSS セレクターや Playwright のロケーターも利用できます。

```bash
# css selector
playwright-cli click "#main > button.submit"

# role locator
playwright-cli click "getByRole('button', { name: 'Submit' })"

# test id
playwright-cli click "getByTestId('submit-button')"
```

<a id="browser-sessions"></a>
## ブラウザーのセッション

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
## インストール

全体の `playwright-cli` コマンドが利用できない場合は、`npx playwright cli` でローカル版を試してください:

```bash
npx --no-install playwright --version
```

ローカル版がある場合は、すべてのコマンドで `npx playwright cli` を使います。なければ `playwright-cli` をグローバルコマンドとしてインストールしてください:

```bash
npm install -g @playwright/cli@latest
```

<a id="example-form-submission"></a>
## 例: フォームの送信

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
## 例: 複数タブの作業フロー

```bash
playwright-cli open https://example.com
playwright-cli tab-new https://example.com/other
playwright-cli tab-list
playwright-cli tab-select 0
playwright-cli snapshot
playwright-cli close
```

<a id="example-debugging-with-devtools"></a>
## 例: 開発者ツールでのデバッグ

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
## 例: 対話型セッション

UI のレビューやデザインの意見をユーザーに求められます。ユーザーが実際のページ上に枠を描き、コメントを入力すると、注釈付きスクリーンショット、指定領域のスナップショット、ユーザーのメモを受け取ります。「UI レビュー」「デザインの意見」「ユーザーの考え・希望・意図を聞いて」と要求された場合に使ってください:

```bash
playwright-cli open https://example.com
playwright-cli show --annotate
```

<a id="specific-tasks"></a>
## 目的別のガイド

* **Playwright テストの実行とデバッグ** [references/playwright-tests.md](references/playwright-tests.md)
* **リクエストのモック** [references/request-mocking.md](references/request-mocking.md)
* **Playwright コードの実行** [references/running-code.md](references/running-code.md)
* **ブラウザーセッションの管理** [references/session-management.md](references/session-management.md)
* **保存状態（Cookie、localStorage）** [references/storage-state.md](references/storage-state.md)
* **テストの生成（計画 / 生成 / 修復）** [references/test-generation.md](references/test-generation.md)
* **トレース** [references/tracing.md](references/tracing.md)
* **動画の記録** [references/video-recording.md](references/video-recording.md)
* **要素の属性の確認** [references/element-attributes.md](references/element-attributes.md)
