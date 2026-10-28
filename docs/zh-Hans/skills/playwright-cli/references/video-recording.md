[English](../../../../../skills/playwright-cli/references/video-recording.md) · [한국어](../../../../ko/skills/playwright-cli/references/video-recording.md) · 简体中文 · [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/video-recording.md) · [日本語](../../../../ja/skills/playwright-cli/references/video-recording.md) · [Español](../../../../es/skills/playwright-cli/references/video-recording.md)

<a id="video-recording"></a>

# 影片录制

各语言版本请见本页上方导览。

将浏览器自动化会话录制为影片，用于侦错、文件或验证。产生 WebM（VP8／VP9 编码）。

<a id="basic-recording"></a>

## 基本录制



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

## 最佳实务

<a id="1-use-descriptive-filenames"></a>

### 1. 使用具描述性的档名



```bash
# Include context in filename
playwright-cli video-start recordings/login-flow-2024-01-15.webm
playwright-cli video-start recordings/checkout-test-run-42.webm
```



<a id="2-record-entire-hero-scripts"></a>

### 2. 录制完整主要流程

为用户或工作证据录影时，建议先创建程序片段，再以 run-code 运行。如此可在操作间加入适当暂停与注记；新的 Playwright API 支持此功能。

1) 先以 CLI 运行情境，记录全部 locator 与操作；需要这些 locator 取得元素边界框以醒目标示。
2) 创建预定录影脚本档（如下）。使用 pressSequentially 搭配 delay 呈现自然输入，加入合理暂停。
3) 运行 playwright-cli run-code --filename your-script.js。

**重要**：覆盖层为 `pointer-events: none`，不干扰页面互动。点选、填写或其他操作时可安全保持固定覆盖层显示。



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



善用创意；覆盖层具有丰富表达能力。

<a id="overlay-api-summary"></a>

### 覆盖层 API 摘要

| 方法 | 用途 |
|--------|----------|
| `page.screencast.showChapter(title, { description?, duration?, styleSheet? })` | 模糊背景的全屏幕章节卡，适合区段转场 |
| `page.screencast.showOverlay(html, { duration? })` | 自定义 HTML 覆盖层，用于提示、标签、重点标示 |
| `disposable.dispose()` | 移除未设 duration 的固定覆盖层 |
| `page.screencast.hideOverlays()` / `page.screencast.showOverlays()` | 暂时隐藏／显示全部覆盖层 |

<a id="tracing-vs-video"></a>

## 追踪与影片比较

| 功能 | 影片 | 追踪 |
|---------|-------|---------|
| 输出 | WebM 档 | 追踪档（可在 Trace Viewer 检视） |
| 内容 | 视觉录影 | DOM 快照、网络、控制台、操作 |
| 用途 | 示范、文件 | 侦错、分析 |
| 大小 | 较大 | 较小 |

<a id="limitations"></a>

## 限制

- 录影增加少量自动化开销。
- 大型录影可能占用大量磁碟空间。
