[English](../../../../../skills/playwright-cli/references/test-generation.md) · [한국어](../../../../ko/skills/playwright-cli/references/test-generation.md) · [简体中文](../../../../zh-Hans/skills/playwright-cli/references/test-generation.md) · 繁體中文 · [日本語](../../../../ja/skills/playwright-cli/references/test-generation.md) · [Español](../../../../es/skills/playwright-cli/references/test-generation.md)

<a id="test-generation-plan--generate--heal"></a>

# 測試生成（規劃 → 生成 → 修復）

各語言版本請見本頁上方導覽。

以 `playwright-cli` 撰寫與維護 Playwright 測試的端對端流程。每個操作都輸出等效 Playwright TypeScript，作為測試原料。以下各節亦可獨立使用：

- **生成原理**：操作轉為 TypeScript，以及如何加入斷言，為其他步驟的基礎。
- **規劃**：探索應用程式，產生描述測試內容的規格檔。
- **生成**：將規格轉為 Playwright 測試；模糊或過時時更新規格。
- **修復**：診斷失敗測試、修正程式碼，讓規格與現況一致。

規劃／生成／修復共用同一機制：在背景執行 `npx playwright test --debug=cli`，再以 `playwright-cli attach tw-XXXX` 互動操作暫停頁面。偵錯／附加方式見 [playwright-tests.md](playwright-tests.md)。

---

<a id="0-how-generation-works"></a>

## 0. 生成原理

每個 `playwright-cli` 操作都產生相應 Playwright TypeScript，顯示於輸出，可直接複製至測試檔。



```bash
# Start a session
playwright-cli open https://example.com/login

# Take a snapshot to see elements
playwright-cli snapshot
# Output shows: e1 [textbox "Email"], e2 [textbox "Password"], e3 [button "Sign In"]

# Fill form fields - generates code automatically
playwright-cli fill e1 "user@example.com"
# Ran Playwright code:
# await page.getByRole('textbox', { name: 'Email' }).fill('user@example.com');

playwright-cli fill e2 "password123"
# Ran Playwright code:
# await page.getByRole('textbox', { name: 'Password' }).fill('password123');

playwright-cli click e3
# Ran Playwright code:
# await page.getByRole('button', { name: 'Sign In' }).click();
```



<a id="building-a-test-file"></a>

### 建立測試檔

將生成程式碼整理為 Playwright 測試：



```typescript
import { test, expect } from '@playwright/test';

test('login flow', async ({ page }) => {
  // Generated code from playwright-cli session:
  await page.goto('https://example.com/login');
  await page.getByRole('textbox', { name: 'Email' }).fill('user@example.com');
  await page.getByRole('textbox', { name: 'Password' }).fill('password123');
  await page.getByRole('button', { name: 'Sign In' }).click();

  // Add assertions
  await expect(page).toHaveURL(/.*dashboard/);
});
```



<a id="use-semantic-locators"></a>

### 使用語意 locator

生成程式碼盡可能使用較穩定的角色型 locator：



```typescript
// Generated (good - semantic)
await page.getByRole('button', { name: 'Submit' }).click();

// Avoid (fragile - CSS selectors)
await page.locator('#submit-btn').click();
```



<a id="explore-before-recording"></a>

### 記錄前先探索

記錄操作前，先取得快照了解頁面結構：



```bash
playwright-cli open https://example.com
playwright-cli snapshot
# Review the element structure
playwright-cli click e5
```



<a id="add-assertions-manually"></a>

### 手動加入斷言

生成程式碼只擷取操作，不包含斷言。請在測試加入下列建議 matcher：

- `toBeVisible()`：元素已轉譯且可見。
- `toHaveText(text)`：元素文字內容相符。
- `toHaveValue(value) / toBeEmpty()`：輸入／選擇欄值相符。
- `toBeChecked() / toBeUnchecked()`：核取方塊狀態相符。
- `toMatchAriaSnapshot(snapshot)`：頁面或 locator 符合部分無障礙快照。

以 `playwright-cli generate-locator <target>` 產生斷言的 locator 運算式，並用 snapshot／eval 擷取預期值。

斷言文字內容時，locator 不應包含元素自身文字。`getByTestId()` 或 `getByLabel()` 通常適合文字斷言；文字型 locator 則優先使用 `toBeVisible()`。

比對快照無須包含全部資訊，只擷取斷言所需內容。不穩定值可用規則運算式。



```bash
# Get a stable locator for an element ref to use in the assertion
playwright-cli --raw generate-locator e5
# getByRole('button', { name: 'Submit' })

# Capture expected text content for toHaveText
playwright-cli --raw eval "el => el.textContent" e5

# Capture expected input value for toHaveValue/toBeEmpty
playwright-cli --raw eval "el => el.value" e5

# Capture expected aria snapshot for toMatchAriaSnapshot/toBeChecked
# (whole page, or use a ref to scope to a region)
playwright-cli --raw snapshot
playwright-cli --raw snapshot e5
```





```typescript
// Generated action
await page.getByRole('button', { name: 'Submit' }).click();

// Manual assertions using the outputs above:
await expect(page.getByRole('alert', { name: 'Success' })).toBeVisible();
await expect(page.getByTestId('main-header')).toHaveText('Welcome, user');
await expect(page.getByRole('textbox', { name: 'Email' })).toHaveValue('user@example.com');
await expect(page.getByRole('checkbox', { name: 'Enable notifications' })).toBeChecked();

// toMatchAriaSnapshot on the whole page, finds a matching region
await expect(page).toMatchAriaSnapshot(`
  - heading "Welcome, user"
  - link /\\d+ new messages?/
  - button "Sign out"
`);

// toMatchAriaSnapshot scoped to a region
await expect(page.getByRole('navigation')).toMatchAriaSnapshot(`
  - link "Home"
  - link /\\d+ new messages?/
  - link "Profile"
`);
```



---

<a id="1-planning"></a>

## 1. 規劃

目標：產生規格檔（例如 `specs/<feature>.plan.md`）列出測試情境。**一律**將規格寫入檔案。

<a id="11-prerequisite-workspace"></a>

### 1.1 前置條件：工作區

先檢查工作區是否已安裝 Playwright：



```bash
# Either of these confirms a workspace:
test -f playwright.config.ts || test -f playwright.config.js
npx --no-install playwright --version
```



若未安裝，請初始化並讓使用者選擇預設值：



```bash
npm init playwright@latest
```



<a id="12-prerequisite-seed-test"></a>

### 1.2 前置條件：種子測試

**種子測試**是讓頁面到達所有情境起始狀態的最小測試，包括導覽、必要登入、功能旗標等。情境假設種子完成後為全新起點。`--debug=cli` 在測試**內部**暫停，因此每次規劃與生成都從種子開始。

最小可用種子：



```ts
// tests/seed.spec.ts
import { test } from '@playwright/test';

test('seed', async ({ page }) => {
  await page.goto('https://example.com/');
});
```



建議將導覽放進 fixture，供情境測試重用：



```ts
// tests/fixtures.ts
import { test as baseTest } from '@playwright/test';
export { expect } from '@playwright/test';

export const test = baseTest.extend({
  page: async ({ page }, use) => {
    await page.goto('https://example.com/');
    await use(page);
  },
});
```





```ts
// tests/seed.spec.ts
import { test } from './fixtures';

test('seed', async ({ page }) => {
  // Fixture already navigates. This empty body tells agents where to start.
});
```



若無種子，建立至少能導覽至應用程式的種子測試。

<a id="13-explore-the-app"></a>

### 1.3 探索應用程式

在背景透過種子啟動應用程式並附加：



```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test tests/seed.spec.ts --debug=cli
# wait for "Debugging Instructions" and the session name tw-XXXX
playwright-cli attach tw-XXXX
```



繼續執行種子，再探查應用程式：



```bash
playwright-cli resume                   # resume so that seed test runs fully
playwright-cli snapshot                 # inventory of interactive elements
playwright-cli click e5                 # follow a flow
playwright-cli eval "location.href"     # read URL / state
playwright-cli show --annotate          # ask the user to point at something
```



整理以下項目：

- 互動介面（表單、按鈕、清單、篩選、對話框）。
- 主要端對端使用流程。
- 邊界：空狀態、驗證錯誤、極長輸入、邊界值。
- 持久化：重新整理、本機／工作階段儲存空間、URL fragment。
- 導覽：哪些控制改變 URL、上一頁／下一頁行為。

**重要**：不得只用 `playwright-cli` 開啟 URL；一律經由測試，才能包含自訂設定。
**重要**：探索完成後停止背景測試。

<a id="14-write-the-spec-file"></a>

### 1.4 撰寫規格檔

儲存於 `specs/<feature>.plan.md`，採以下結構：



```markdown
# <Feature> Test Plan

## Application Overview

<One paragraph describing what the feature does and why it matters.>

## Test Scenarios

### 1. <Group Name>

**Seed:** `tests/seed.spec.ts`

#### 1.1. <kebab-case-scenario-name>

**File:** `tests/<group>/<kebab-case-scenario-name>.spec.ts`

**Steps:**
  1. <Concrete user step>
    - expect: <observable outcome>
    - expect: <another observable outcome>
  2. <Next step>
    - expect: <outcome>

#### 1.2. <next-scenario>
...

### 2. <Next Group>

**Seed:** `tests/seed.spec.ts`
...
```



準則：

- 每個情境獨立，從種子的全新狀態開始；不得串接情境。
- 情境名稱採 kebab-case，與測試檔名一致（`should-add-single-todo` → `should-add-single-todo.spec.ts`）。
- 涵蓋正常、邊界、驗證、失敗流程與持久化。
- 以使用者層級寫步驟（在輸入欄輸入 Buy milk），勿寫 API 層級（呼叫 `fill`）。
- 在 `- expect:` 項目寫可觀察結果，生成時各自成為斷言。

---

<a id="2-generate"></a>

## 2. 生成

目標：將規格檔轉為 Playwright 測試檔；規格有偏差時可更新。

<a id="21-inputs"></a>

### 2.1 輸入

- **規格檔**，如 `specs/basic-operations.plan.md`。
- **目標**：單一情境（如 `1.2`）、整組（`1`）或全部。
- **種子檔**：取自該情境群組的 `**Seed:**` 行。

<a id="22-generate-one-scenario"></a>

### 2.2 生成一個情境

依序處理各目標情境；不得平行，因共用種子工作階段：



```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test <seed-file> --debug=cli   # background
playwright-cli attach tw-XXXX
# resume
```



**不得**只用 playwright-cli 開啟 URL；一律經由測試，以納入其自訂設定。

以 `playwright-cli` 逐步執行情境 `Steps:`，以規格為計畫、實際應用程式為事實來源。若步驟模糊（哪個按鈕）、參照已不存在元素或與實際行為矛盾，依判斷更新規格以反映真實行為，再繼續。生成中修改規格屬預期行為。

每次操作印出等效 Playwright TypeScript（見[生成原理](#0-how-generation-works)）：



```bash
playwright-cli snapshot                         # find refs
playwright-cli fill e3 "John Doe"               # -> page.getByRole('textbox', {...}).fill(...)
playwright-cli press Enter
playwright-cli click e7
```



為每個 `- expect:` 加上明確斷言；詳見[生成原理](#0-how-generation-works)。

整理生成程式碼，寫入規格指定的測試檔路徑：



```ts
// spec: specs/basic-operations.plan.md
// seed: tests/seed.spec.ts
import { test, expect } from './fixtures';   // or '@playwright/test' if no fixtures file

test.describe('Signing in and out', () => {
  test('should sign in', async ({ page }) => {
    // 1. Navigate to the application
    // (handled by the seed fixture)

    // 2. Type 'John Doe' into the username field
    await page.getByRole('textbox', { name: 'username' }).fill('John Doe');

    // 3. Type password
    await page.getByRole('textbox', { name: 'password' }).fill('TestPassword');

    // 4. Press Enter to submit
    await page.getByRole('textbox', { name: 'password' }).press('Enter');

    await expect(page.getByRole('heading')).toContainText('Welcome, John Doe!');
  });
});
```



規則：

- **每檔一個測試。** 檔案路徑、describe 名稱與測試名稱逐字取自規格（不含序號）。
- 每個編號步驟的操作前加 `// N. <step text>` 註解。
- describe 群組名逐字沿用規格，不含 `1.` 序號。
- 專案有 `./fixtures` 時從該處匯入，否則用 `@playwright/test`。
- **重要**：下一情境前，關閉 CLI 工作階段並停止背景測試。

<a id="23-generate-multiple-scenarios"></a>

### 2.3 生成多個情境

逐一對目標情境重複 2.2，每次重新啟動種子，讓各測試從乾淨頁面開始。獨特的生成工作階段名稱允許安全平行化，但務必停止每次測試執行。

<a id="24-run-generated-tests"></a>

### 2.4 執行生成測試

生成後執行一次新測試：



```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test tests/<group>/<scenario>.spec.ts
```



任何失敗皆進入第 3 節。

---

<a id="3-heal"></a>

## 3. 修復

目標：修復失敗測試；若應用程式預期行為已改變，更新規格。

<a id="31-find-failing-tests"></a>

### 3.1 尋找失敗測試



```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test
```



記錄失敗的 `<file>:<line>` 清單，逐一處理。不得平行修復；共用狀態與單一 CLI 工作階段會使結果不穩定。

<a id="32-debug-one-failure"></a>

### 3.2 偵錯一個失敗

在背景以偵錯模式執行單一失敗測試，再附加：



```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test tests/<group>/<scenario>.spec.ts:<line> --debug=cli
# wait for "Debugging Instructions" and the tw-XXXX session name
playwright-cli attach tw-XXXX
```



測試於起點暫停。逐步執行或執行至失敗操作／斷言前，再診斷：



```bash
playwright-cli snapshot                # did the element change / move / rename?
playwright-cli console                 # app-side errors?
playwright-cli requests                # failed request? wrong payload?
playwright-cli show --annotate         # ask the user to point somewhere
```



常見原因：選擇器偏差、新包裝元素、標籤／ARIA 更名、時序（轉場、非同步載入）、應用程式斷言文字更新、測試資料跨執行洩漏。

以 `playwright-cli` 演練修正互動，將輸出生成程式碼貼回測試。

<a id="33-apply-the-fix"></a>

### 3.3 套用修正

編輯測試檔，更新 locator、斷言、步驟順序或輸入以符合修正後行為。停止背景偵錯，再重跑單一測試確認通過。

絕不可略過 hook 或以 sleep 修正；絕不使用 `networkidle`。

<a id="34-reconcile-with-the-spec"></a>

### 3.4 與規格核對

開啟測試 `// spec:` 標頭所參照規格，找到對應情境。

- **純技術修正**（locator 偏差、較佳斷言形式）且使用者行為仍符合應用程式 → 不改規格。
- **修正改變規格描述的可見步驟、輸入、順序或預期結果** → 更新規格反映現況。情境 ID 與檔案路徑不變，只改步驟／expect。
- **不確定變更是刻意設計（規格過時）或迴歸（測試正確、應用程式錯誤）** → **停止並詢問使用者**，提供情境 ID（如 `2.3`）、不再相符的規格行，以及觀察到的行為（快照片段或具體結果）。

使用者回答後，才能更新規格（刻意變更）或將測試標記／回報為涵蓋錯誤（迴歸）。

<a id="35-iteration-and-giving-up"></a>

### 3.5 迭代與停止

- 一次修一個失敗，每次重跑。
- 充分調查後確信測試正確、應用程式錯誤，且**使用者已確認為錯誤**時，可用 `test.fixme(...)` 標記並註解連結使用者決定或 issue。不得默默略過。

---

<a id="cross-references"></a>

## 交叉參照

| 需求 | 參考 |
|---|---|
| `--debug=cli`／附加機制 | [playwright-tests.md](playwright-tests.md) |
| 探索／生成中模擬請求 | [request-mocking.md](request-mocking.md) |
| 管理 CLI 瀏覽器工作階段 | [session-management.md](session-management.md) |
