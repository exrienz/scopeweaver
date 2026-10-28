# 영상 기록

[English](../../../../../skills/playwright-cli/references/video-recording.md) · 한국어 · [简体中文](../../../../zh-Hans/skills/playwright-cli/references/video-recording.md) · [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/video-recording.md) · [日本語](../../../../ja/skills/playwright-cli/references/video-recording.md) · [Español](../../../../es/skills/playwright-cli/references/video-recording.md)

브라우저 자동화를 디버깅, 문서, 검증용 영상으로 기록합니다. 출력은 WebM(VP8/VP9)입니다.

## 기본 기록

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

## 권장 방법

### 1. 내용을 설명하는 파일 이름 사용

```bash
# Include context in filename
playwright-cli video-start recordings/login-flow-2024-01-15.webm
playwright-cli video-start recordings/checkout-test-run-42.webm
```

### 2. 전체 시연 흐름 기록

사용자용 영상이나 작업 증거를 만들 때는 코드 조각을 작성하고 run-code로 실행하는 방식이 적합합니다.
작업 사이 적절한 대기와 영상 설명을 넣을 수 있으며 이를 위한 Playwright API가 있습니다.

1) CLI로 시나리오를 실행하며 모든 locator와 조작을 기록합니다. 강조 영역 좌표를 조회할 때 locator가 필요합니다.
2) 아래처럼 영상용 스크립트 파일을 작성합니다. 지연을 둔 pressSequentially와 적절한 대기로 입력을 보여줍니다.
3) playwright-cli run-code --filename your-script.js를 실행합니다.

오버레이는 `pointer-events: none`이므로 페이지 조작을 방해하지 않습니다. 표시한 상태에서 클릭, 입력 등 작업을 수행할 수 있습니다.

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

오버레이를 활용해 시연 내용을 자유롭게 설명할 수 있습니다.

### 오버레이 API 요약

| 메서드 | 용도 |
|--------|----------|
| `page.screencast.showChapter(title, { description?, duration?, styleSheet? })` | 흐린 배경의 전체 화면 챕터 카드; 구간 전환에 적합 |
| `page.screencast.showOverlay(html, { duration? })` | 설명, 이름표, 강조용 HTML 오버레이 |
| `disposable.dispose()` | duration 없이 만든 고정 오버레이 제거 |
| `page.screencast.hideOverlays()` / `page.screencast.showOverlays()` | 모든 오버레이 잠시 숨기기/보이기 |

## 추적과 영상 비교

| 항목 | 영상 | 추적 |
|---------|-------|---------|
| 출력 | WebM 파일 | Trace Viewer로 보는 추적 파일 |
| 표시 | 시각 기록 | DOM, 네트워크, 콘솔, 조작 |
| 용도 | 시연, 문서 | 디버깅, 분석 |
| 크기 | 더 큼 | 더 작음 |

## 한계

- 영상 기록은 자동화에 약간의 부하를 줍니다
- 큰 영상은 많은 디스크 공간을 사용합니다
