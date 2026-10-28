[English](../../../../../skills/playwright-cli/references/test-generation.md) · [한국어](../../../../ko/skills/playwright-cli/references/test-generation.md) · 简体中文 · [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/test-generation.md) · [日本語](../../../../ja/skills/playwright-cli/references/test-generation.md) · [Español](../../../../es/skills/playwright-cli/references/test-generation.md)

<a id="test-generation-plan--generate--heal"></a>

# 测试生成（规划 → 生成 → 修复）

各语言版本请见本页上方导览。

以 `playwright-cli` 撰写与维护 Playwright 测试的端对端流程。每个操作都输出等效 Playwright TypeScript，作为测试原料。以下各节亦可独立使用：

- **生成原理**：操作转为 TypeScript，以及如何加入断言，为其他步骤的基础。
- **规划**：探索应用程序，产生描述测试内容的规格档。
- **生成**：将规格转为 Playwright 测试；模糊或过时时更新规格。
- **修复**：诊断失败测试、修正代码，让规格与现况一致。

规划／生成／修复共用同一机制：在背景运行 `npx playwright test --debug=cli`，再以 `playwright-cli attach tw-XXXX` 互动操作暂停页面。侦错／附加方式见 [playwright-tests.md](playwright-tests.md)。

---

<a id="0-how-generation-works"></a>

## 0. 生成原理

每个 `playwright-cli` 操作都产生相应 Playwright TypeScript，显示于输出，可直接复制至测试档。



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

### 创建测试档

将生成代码整理为 Playwright 测试：



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

### 使用语意 locator

生成代码尽可能使用较稳定的角色型 locator：



```typescript
// Generated (good - semantic)
await page.getByRole('button', { name: 'Submit' }).click();

// Avoid (fragile - CSS selectors)
await page.locator('#submit-btn').click();
```



<a id="explore-before-recording"></a>

### 记录前先探索

记录操作前，先取得快照了解页面结构：



```bash
playwright-cli open https://example.com
playwright-cli snapshot
# Review the element structure
playwright-cli click e5
```



<a id="add-assertions-manually"></a>

### 手动加入断言

生成代码只捕获操作，不包含断言。请在测试加入下列建议 matcher：

- `toBeVisible()`：元素已渲染且可见。
- `toHaveText(text)`：元素文字内容相符。
- `toHaveValue(value) / toBeEmpty()`：输入／选择栏值相符。
- `toBeChecked() / toBeUnchecked()`：核取方块状态相符。
- `toMatchAriaSnapshot(snapshot)`：页面或 locator 符合部分无障碍快照。

以 `playwright-cli generate-locator <target>` 产生断言的 locator 运算式，并用 snapshot／eval 捕获预期值。

断言文字内容时，locator 不应包含元素自身文字。`getByTestId()` 或 `getByLabel()` 通常适合文字断言；文字型 locator 则优先使用 `toBeVisible()`。

比对快照无须包含全部信息，只捕获断言所需内容。不稳定值可用规则运算式。



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

## 1. 规划

目标：产生规格档（例如 `specs/<feature>.plan.md`）列出测试情境。**一律**将规格写入文件。

<a id="11-prerequisite-workspace"></a>

### 1.1 前置条件：工作区

先检查工作区是否已安装 Playwright：



```bash
# Either of these confirms a workspace:
test -f playwright.config.ts || test -f playwright.config.js
npx --no-install playwright --version
```



若未安装，请初始化并让用户选择默认值：



```bash
npm init playwright@latest
```



<a id="12-prerequisite-seed-test"></a>

### 1.2 前置条件：种子测试

**种子测试**是让页面到达所有情境起始状态的最小测试，包括导览、必要登录、功能旗标等。情境假设种子完成后为全新起点。`--debug=cli` 在测试**内部**暂停，因此每次规划与生成都从种子开始。

最小可用种子：



```ts
// tests/seed.spec.ts
import { test } from '@playwright/test';

test('seed', async ({ page }) => {
  await page.goto('https://example.com/');
});
```



建议将导览放进 fixture，供情境测试重用：



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



若无种子，创建至少能导览至应用程序的种子测试。

<a id="13-explore-the-app"></a>

### 1.3 探索应用程序

在背景通过种子启动应用程序并附加：



```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test tests/seed.spec.ts --debug=cli
# wait for "Debugging Instructions" and the session name tw-XXXX
playwright-cli attach tw-XXXX
```



继续运行种子，再探查应用程序：



```bash
playwright-cli resume                   # resume so that seed test runs fully
playwright-cli snapshot                 # inventory of interactive elements
playwright-cli click e5                 # follow a flow
playwright-cli eval "location.href"     # read URL / state
playwright-cli show --annotate          # ask the user to point at something
```



整理以下项目：

- 互动界面（表单、按钮、清单、筛选、对话框）。
- 主要端对端使用流程。
- 边界：空状态、验证错误、极长输入、边界值。
- 持久化：刷新、本地／会话保存空间、URL fragment。
- 导览：哪些控制改变 URL、上一页／下一页行为。

**重要**：不得只用 `playwright-cli` 开启 URL；一律经由测试，才能包含自定义配置。
**重要**：探索完成后停止背景测试。

<a id="14-write-the-spec-file"></a>

### 1.4 撰写规格档

保存于 `specs/<feature>.plan.md`，采以下结构：



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



准则：

- 每个情境独立，从种子的全新状态开始；不得串接情境。
- 情境名称采 kebab-case，与测试档名一致（`should-add-single-todo` → `should-add-single-todo.spec.ts`）。
- 涵盖正常、边界、验证、失败流程与持久化。
- 以用户层级写步骤（在输入栏输入 Buy milk），勿写 API 层级（呼叫 `fill`）。
- 在 `- expect:` 项目写可观察结果，生成时各自成为断言。

---

<a id="2-generate"></a>

## 2. 生成

目标：将规格档转为 Playwright 测试档；规格有偏差时可更新。

<a id="21-inputs"></a>

### 2.1 输入

- **规格档**，如 `specs/basic-operations.plan.md`。
- **目标**：单一情境（如 `1.2`）、整组（`1`）或全部。
- **种子档**：取自该情境群组的 `**Seed:**` 行。

<a id="22-generate-one-scenario"></a>

### 2.2 生成一个情境

依序处理各目标情境；不得平行，因共用种子会话：



```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test <seed-file> --debug=cli   # background
playwright-cli attach tw-XXXX
# resume
```



**不得**只用 playwright-cli 开启 URL；一律经由测试，以纳入其自定义配置。

以 `playwright-cli` 逐步运行情境 `Steps:`，以规格为计划、实际应用程序为事实来源。若步骤模糊（哪个按钮）、参照已不存在元素或与实际行为矛盾，依判断更新规格以反映真实行为，再继续。生成中修改规格属预期行为。

每次操作印出等效 Playwright TypeScript（见[生成原理](#0-how-generation-works)）：



```bash
playwright-cli snapshot                         # find refs
playwright-cli fill e3 "John Doe"               # -> page.getByRole('textbox', {...}).fill(...)
playwright-cli press Enter
playwright-cli click e7
```



为每个 `- expect:` 加上明确断言；详见[生成原理](#0-how-generation-works)。

整理生成代码，写入规格指定的测试档路径：



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



规则：

- **每档一个测试。** 文件路径、describe 名称与测试名称逐字取自规格（不含序号）。
- 每个编号步骤的操作前加 `// N. <step text>` 注解。
- describe 群组名逐字沿用规格，不含 `1.` 序号。
- 专案有 `./fixtures` 时从该处导入，否则用 `@playwright/test`。
- **重要**：下一情境前，关闭 CLI 会话并停止背景测试。

<a id="23-generate-multiple-scenarios"></a>

### 2.3 生成多个情境

逐一对目标情境重复 2.2，每次重启种子，让各测试从干净页面开始。独特的生成会话名称允许安全平行化，但务必停止每次测试运行。

<a id="24-run-generated-tests"></a>

### 2.4 运行生成测试

生成后运行一次新测试：



```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test tests/<group>/<scenario>.spec.ts
```



任何失败皆进入第 3 节。

---

<a id="3-heal"></a>

## 3. 修复

目标：修复失败测试；若应用程序预期行为已改变，更新规格。

<a id="31-find-failing-tests"></a>

### 3.1 寻找失败测试



```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test
```



记录失败的 `<file>:<line>` 清单，逐一处理。不得平行修复；共用状态与单一 CLI 会话会使结果不稳定。

<a id="32-debug-one-failure"></a>

### 3.2 侦错一个失败

在背景以侦错模式运行单一失败测试，再附加：



```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test tests/<group>/<scenario>.spec.ts:<line> --debug=cli
# wait for "Debugging Instructions" and the tw-XXXX session name
playwright-cli attach tw-XXXX
```



测试于起点暂停。逐步运行或运行至失败操作／断言前，再诊断：



```bash
playwright-cli snapshot                # did the element change / move / rename?
playwright-cli console                 # app-side errors?
playwright-cli requests                # failed request? wrong payload?
playwright-cli show --annotate         # ask the user to point somewhere
```



常见原因：选择器偏差、新包装元素、标签／ARIA 更名、时序（转场、异步加载）、应用程序断言文字更新、测试数据跨运行泄漏。

以 `playwright-cli` 演练修正互动，将输出生成代码贴回测试。

<a id="33-apply-the-fix"></a>

### 3.3 应用修正

编辑测试档，更新 locator、断言、步骤顺序或输入以符合修正后行为。停止背景侦错，再重跑单一测试确认通过。

绝不可略过 hook 或以 sleep 修正；绝不使用 `networkidle`。

<a id="34-reconcile-with-the-spec"></a>

### 3.4 与规格核对

开启测试 `// spec:` 头部所参照规格，找到对应情境。

- **纯技术修正**（locator 偏差、较佳断言形式）且用户行为仍符合应用程序 → 不改规格。
- **修正改变规格描述的可见步骤、输入、顺序或预期结果** → 更新规格反映现况。情境 ID 与文件路径不变，只改步骤／expect。
- **不确定变更是刻意设计（规格过时）或回归（测试正确、应用程序错误）** → **停止并询问用户**，提供情境 ID（如 `2.3`）、不再相符的规格行，以及观察到的行为（快照片段或具体结果）。

用户回答后，才能更新规格（刻意变更）或将测试标记／回报为涵盖错误（回归）。

<a id="35-iteration-and-giving-up"></a>

### 3.5 迭代与停止

- 一次修一个失败，每次重跑。
- 充分调查后确信测试正确、应用程序错误，且**用户已确认为错误**时，可用 `test.fixme(...)` 标记并注解链接用户决定或 issue。不得默默略过。

---

<a id="cross-references"></a>

## 交叉参照

| 需求 | 参考 |
|---|---|
| `--debug=cli`／附加机制 | [playwright-tests.md](playwright-tests.md) |
| 探索／生成中模拟请求 | [request-mocking.md](request-mocking.md) |
| 管理 CLI 浏览器会话 | [session-management.md](session-management.md) |
