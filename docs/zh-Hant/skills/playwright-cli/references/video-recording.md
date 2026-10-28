[English](../../../../../skills/playwright-cli/references/video-recording.md) · [한국어](../../../../ko/skills/playwright-cli/references/video-recording.md) · [简体中文](../../../../zh-Hans/skills/playwright-cli/references/video-recording.md) · 繁體中文 · [日本語](../../../../ja/skills/playwright-cli/references/video-recording.md) · [Español](../../../../es/skills/playwright-cli/references/video-recording.md)

<a id="video-recording"></a>

# 影片錄製

各語言版本請見本頁上方導覽。

將瀏覽器自動化工作階段錄製為影片，用於偵錯、文件或驗證。產生 WebM（VP8／VP9 編碼）。

<a id="basic-recording"></a>

## 基本錄製



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

## 最佳實務

<a id="1-use-descriptive-filenames"></a>

### 1. 使用具描述性的檔名



```bash
# Include context in filename
playwright-cli video-start recordings/login-flow-2024-01-15.webm
playwright-cli video-start recordings/checkout-test-run-42.webm
```



<a id="2-record-entire-hero-scripts"></a>

### 2. 錄製完整主要流程

為使用者或工作證據錄影時，建議先建立程式片段，再以 run-code 執行。如此可在操作間加入適當暫停與註記；新的 Playwright API 支援此功能。

1) 先以 CLI 執行情境，記錄全部 locator 與操作；需要這些 locator 取得元素邊界框以醒目標示。
2) 建立預定錄影指令碼檔（如下）。使用 pressSequentially 搭配 delay 呈現自然輸入，加入合理暫停。
3) 執行 playwright-cli run-code --filename your-script.js。

**重要**：覆蓋層為 `pointer-events: none`，不干擾頁面互動。點選、填寫或其他操作時可安全保持固定覆蓋層顯示。



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



善用創意；覆蓋層具有豐富表達能力。

<a id="overlay-api-summary"></a>

### 覆蓋層 API 摘要

| 方法 | 用途 |
|--------|----------|
| `page.screencast.showChapter(title, { description?, duration?, styleSheet? })` | 模糊背景的全螢幕章節卡，適合區段轉場 |
| `page.screencast.showOverlay(html, { duration? })` | 自訂 HTML 覆蓋層，用於提示、標籤、重點標示 |
| `disposable.dispose()` | 移除未設 duration 的固定覆蓋層 |
| `page.screencast.hideOverlays()` / `page.screencast.showOverlays()` | 暫時隱藏／顯示全部覆蓋層 |

<a id="tracing-vs-video"></a>

## 追蹤與影片比較

| 功能 | 影片 | 追蹤 |
|---------|-------|---------|
| 輸出 | WebM 檔 | 追蹤檔（可在 Trace Viewer 檢視） |
| 內容 | 視覺錄影 | DOM 快照、網路、主控台、操作 |
| 用途 | 示範、文件 | 偵錯、分析 |
| 大小 | 較大 | 較小 |

<a id="limitations"></a>

## 限制

- 錄影增加少量自動化開銷。
- 大型錄影可能佔用大量磁碟空間。
