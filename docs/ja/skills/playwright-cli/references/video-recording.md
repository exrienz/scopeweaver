[English](../../../../../skills/playwright-cli/references/video-recording.md) | [한국어](../../../../ko/skills/playwright-cli/references/video-recording.md) | [简体中文](../../../../zh-Hans/skills/playwright-cli/references/video-recording.md) | [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/video-recording.md) | [日本語](video-recording.md) | [Español](../../../../es/skills/playwright-cli/references/video-recording.md)

<a id="video-recording"></a>
# 動画の記録



ブラウザー自動操作のセッションを、デバッグ、ドキュメント、検証用の動画として記録します。WebM（VP8/VP9 コーデック）を生成します。

<a id="basic-recording"></a>
## 基本的な記録

```bash
# Open browser first
playwright-cli open

# Start recording
playwright-cli video-start demo.webm

# Add a chapter marker for section transitions
playwright-cli video-chapter "Getting Started" --description="Opening the homepage" --duration=2000

# Navigate and perform actions
playwright-cli goto https://example.com
playwright-cli snapshot
playwright-cli click e1

# Add another chapter
playwright-cli video-chapter "Filling Form" --description="Entering test data" --duration=2000
playwright-cli fill e2 "test input"

# Stop and save
playwright-cli video-stop
```

<a id="best-practices"></a>
## 推奨する使い方

<a id="1-use-descriptive-filenames"></a>
### 1. 内容がわかるファイル名にする

```bash
# Include context in filename
playwright-cli video-start recordings/login-flow-2024-01-15.webm
playwright-cli video-start recordings/checkout-test-run-42.webm
```

<a id="2-record-entire-hero-scripts"></a>
### 2. 主なシナリオ全体を記録する

ユーザー向けや作業の証明用の動画では、コード片を作成して run-code で実行する方法を推奨します。操作間の適切な間隔や、動画への注釈を設定できます。そのための新しい Playwright API があります。

1) CLI でシナリオを実行し、すべてのロケーターと操作を控えます。強調表示用の境界領域を取得するためにロケーターが必要です。
2) 動画用のスクリプトファイルを作成します（下記）。自然な入力には delay 付きの pressSequentially を使い、適度な間隔を設けてください。
3) playwright-cli run-code --filename your-script.js を実行します。

**重要**: オーバーレイは `pointer-events: none` のため、ページの操作を妨げません。クリック、入力、その他の操作中も、固定のオーバーレイを安全に表示し続けられます。

```js
async page => {
  await page.screencast.start({ path: 'video.webm', size: { width: 1280, height: 800 } });
  await page.goto('https://demo.playwright.dev/todomvc');

  // Show a chapter card — blurs the page and shows a dialog.
  // Blocks until duration expires, then auto-removes.
  // Use this for simple use cases, but always feel free to hand-craft your own beautiful
  // overlay via await page.screencast.showOverlay().
  await page.screencast.showChapter('Adding Todo Items', {
    description: 'We will add several items to the todo list.',
    duration: 2000,
  });

  // Perform action
  await page.getByRole('textbox', { name: 'What needs to be done?' }).pressSequentially('Walk the dog', { delay: 60 });
  await page.getByRole('textbox', { name: 'What needs to be done?' }).press('Enter');
  await page.waitForTimeout(1000);

  // Show next chapter
  await page.screencast.showChapter('Verifying Results', {
    description: 'Checking the item appeared in the list.',
    duration: 2000,
  });

  // Add a sticky annotation that stays while you perform actions.
  // Overlays are pointer-events: none, so they won't block clicks.
  const annotation = await page.screencast.showOverlay(`
    <div style="position: absolute; top: 8px; right: 8px;
      padding: 6px 12px; background: rgba(0,0,0,0.7);
      border-radius: 8px; font-size: 13px; color: white;">
      ✓ Item added successfully
    </div>
  `);

  // Perform more actions while the annotation is visible
  await page.getByRole('textbox', { name: 'What needs to be done?' }).pressSequentially('Buy groceries', { delay: 60 });
  await page.getByRole('textbox', { name: 'What needs to be done?' }).press('Enter');
  await page.waitForTimeout(1500);

  // Remove the annotation when done
  await annotation.dispose();

  // You can also highlight relevant locators and provide contextual annotations.
  const bounds = await page.getByText('Walk the dog').boundingBox();
  await page.screencast.showOverlay(`
    <div style="position: absolute;
      top: ${bounds.y}px;
      left: ${bounds.x}px;
      width: ${bounds.width}px;
      height: ${bounds.height}px;
      border: 1px solid red;">
    </div>
    <div style="position: absolute;
      top: ${bounds.y + bounds.height + 5}px;
      left: ${bounds.x + bounds.width / 2}px;
      transform: translateX(-50%);
      padding: 6px;
      background: #808080;
      border-radius: 10px;
      font-size: 14px;
      color: white;">Check it out, it is right above this text
    </div>
  `, { duration: 2000 });

  await page.screencast.stop();
}
```

オーバーレイを生かし、自由な表現を試してください。

<a id="overlay-api-summary"></a>
### オーバーレイ API の概要

| 方法 | 用途 |
|--------|----------|
| `page.screencast.showChapter(title, { description?, duration?, styleSheet? })` | 背景をぼかした全画面の章タイトル。セクションの切り替えに最適 |
| `page.screencast.showOverlay(html, { duration? })` | 独自の HTML オーバーレイ。吹き出し、ラベル、強調表示に使用 |
| `disposable.dispose()` | duration なしで追加した固定のオーバーレイを削除 |
| `page.screencast.hideOverlays()` / `page.screencast.showOverlays()` | すべてのオーバーレイを一時的に非表示・表示 |

<a id="tracing-vs-video"></a>
## トレースと動画の比較

| 機能 | 動画 | トレース |
|---------|-------|---------|
| 出力 | WebM ファイル | トレースファイル（Trace Viewer で閲覧可能）|
| 表示するもの | 見た目の記録 | DOM スナップショット、ネットワーク、コンソール、操作 |
| 用途 | デモ、ドキュメント | デバッグ、分析 |
| サイズ | 大きい | 小さい |

<a id="limitations"></a>
## 制限

- 記録は自動操作の負荷を少し増やします。
- 大きな動画は大量のディスク領域を使う場合があります。
