[English](../../../../../skills/playwright-cli/references/session-management.md) | [한국어](../../../../ko/skills/playwright-cli/references/session-management.md) | [简体中文](../../../../zh-Hans/skills/playwright-cli/references/session-management.md) | [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/session-management.md) | [日本語](session-management.md) | [Español](../../../../es/skills/playwright-cli/references/session-management.md)

<a id="browser-session-management"></a>
# ブラウザーのセッション管理



状態を永続化しながら、隔離した複数のブラウザーセッションを同時に実行できます。

<a id="named-browser-sessions"></a>
## 名前付きのブラウザーセッション

`-s` フラグでブラウザーのコンテキストを分離します:

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
## ブラウザーセッションで分離されるもの

各セッションは次を独立して持ちます:
- Cookie
- LocalStorage / SessionStorage
- IndexedDB
- キャッシュ
- 閲覧履歴
- 開いているタブ

<a id="browser-session-commands"></a>
## ブラウザーセッションのコマンド

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
## 環境変数

環境変数で既定のブラウザーセッション名を設定できます:

```bash
export PLAYWRIGHT_CLI_SESSION="mysession"
playwright-cli open example.com  # Uses "mysession" automatically
```

<a id="common-patterns"></a>
## よく使うパターン

<a id="concurrent-scraping"></a>
### 並列のスクレイピング

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
### A/B テストのセッション

```bash
# Test different user experiences
playwright-cli -s=variant-a open "https://app.com?variant=a"
playwright-cli -s=variant-b open "https://app.com?variant=b"

# Compare
playwright-cli -s=variant-a screenshot
playwright-cli -s=variant-b screenshot
```

<a id="persistent-profile"></a>
### 永続プロファイル

既定では、ブラウザーのプロファイルをメモリーにのみ保持します。ディスクに保存するには、`open` に `--persistent` フラグを指定してください:

```bash
# Use persistent profile (auto-generated location)
playwright-cli open https://example.com --persistent

# Use persistent profile with custom directory
playwright-cli open https://example.com --profile=/path/to/profile
```

<a id="attaching-to-a-running-browser"></a>
## 実行中のブラウザーに接続する

新しく起動せず、すでに動作中のブラウザーに接続するには `attach` を使います。

<a id="attach-by-channel-name"></a>
### チャンネル名で接続

チャンネル名で実行中の Chrome または Edge に接続します。ブラウザーでリモートデバッグを有効にする必要があります。対象ブラウザーで `chrome://inspect/#remote-debugging` を開き、「Allow remote debugging for this browser instance」にチェックを入れてください。

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

対応チャンネル: `chrome`、`chrome-beta`、`chrome-dev`、`chrome-canary`、`msedge`、`msedge-beta`、`msedge-dev`、`msedge-canary`。

`--session` を指定しない場合、セッションはチャンネルと同名になります（例: `--cdp=msedge` は `msedge` セッションを作成）。Chrome と Edge に同時接続しても `default` で衝突しません。上書きするには `--session=<name>` を指定してください。

<a id="attach-via-cdp-endpoint"></a>
### CDP エンドポイントで接続

Chrome DevTools Protocol のエンドポイントを公開するブラウザーに接続します:

```bash
playwright-cli attach --cdp=http://localhost:9222
```

<a id="attach-via-browser-extension"></a>
### ブラウザー拡張で接続

Playwright 拡張をインストールしたブラウザーに接続します:

```bash
playwright-cli attach --extension
```

<a id="detach"></a>
### 接続を解除

外部ブラウザーに影響を与えず、接続したセッションを終了します:

```bash
# Detach the default attached session
playwright-cli detach

# Detach a specific attached session
playwright-cli -s=msedge detach
```

`detach` は `attach` で作成したセッションにのみ使えます。`open` で作成したものには `close` を使ってください。

<a id="default-browser-session"></a>
## 既定のブラウザーセッション

`-s` を省略すると、コマンドは既定のセッションを使います:

```bash
# These use the same default browser session
playwright-cli open https://example.com
playwright-cli snapshot
playwright-cli close  # Stops default browser
```

<a id="browser-session-configuration"></a>
## ブラウザーセッションの設定

開くときに、セッション固有の設定を指定できます:

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
## 推奨する使い方

<a id="1-name-browser-sessions-semantically"></a>
### 1. セッション名で意味がわかるようにする

```bash
# GOOD: Clear purpose
playwright-cli -s=github-auth open https://github.com
playwright-cli -s=docs-scrape open https://docs.example.com

# AVOID: Generic names
playwright-cli -s=s1 open https://github.com
```

<a id="2-always-clean-up"></a>
### 2. 必ず後片付けをする

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
### 3. 古いブラウザーデータを削除する

```bash
# Remove old browser data to free disk space
playwright-cli -s=oldsession delete-data
```
