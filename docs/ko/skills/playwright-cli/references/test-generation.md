# 테스트 생성 (계획 → 생성 → 수정)

[English](../../../../../skills/playwright-cli/references/test-generation.md) · 한국어 · [简体中文](../../../../zh-Hans/skills/playwright-cli/references/test-generation.md) · [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/test-generation.md) · [日本語](../../../../ja/skills/playwright-cli/references/test-generation.md) · [Español](../../../../es/skills/playwright-cli/references/test-generation.md)

`playwright-cli`로 Playwright 테스트를 작성하고 유지하는 전체 흐름입니다. 각 조작은 대응하는 TypeScript를 출력하며 이를 테스트 작성에 사용합니다. 다음 절은 각각 사용할 수 있습니다:

- 생성 원리: 조작을 TypeScript로 바꾸고 단언을 추가하는 방법.
- 계획: 앱을 탐색하고 검사할 내용을 명세 파일로 작성.
- 생성: 명세를 테스트 파일로 변환. 모호하거나 오래된 명세 갱신.
- 수정: 실패 진단, 코드 수정, 실제 동작과 명세 일치.

모든 단계는 `npx playwright test --debug=cli`를 백그라운드에서 실행하고 `playwright-cli attach tw-XXXX`로 멈춘 페이지에 연결하는 방식을 사용합니다. 연결 방법은 [playwright-tests.md](playwright-tests.md)를 참고하세요.

---

## 0. 생성 원리

`playwright-cli`의 각 조작은 해당 Playwright TypeScript 코드를 생성합니다. This code appears in the output and can be copied directly into your test files.

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

### 테스트 파일 작성

생성 코드를 Playwright 테스트에 모읍니다:

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

### 의미 기반 locator 사용

가능하면 생성 코드는 변경에 강한 역할 기반 locator를 사용합니다:

```typescript
// Generated (good - semantic)
await page.getByRole('button', { name: 'Submit' }).click();

// Avoid (fragile - CSS selectors)
await page.locator('#submit-btn').click();
```

### 기록 전에 탐색

조작 기록 전에 스냅샷으로 페이지 구조를 파악합니다:

```bash
playwright-cli open https://example.com
playwright-cli snapshot
# Review the element structure
playwright-cli click e5
```

### 단언 직접 추가

생성 코드는 조작만 기록하고 단언은 넣지 않습니다. 다음 matcher로 기대값을 추가하세요:

- `toBeVisible()`: 요소가 렌더링되고 보임
- `toHaveText(text)`: 요소 텍스트 일치
- `toHaveValue(value) / toBeEmpty()`: 입력/select 값 일치
- `toBeChecked() / toBeUnchecked()`: 체크 상태 일치
- `toMatchAriaSnapshot(snapshot)`: 페이지 또는 locator가 부분 접근성 스냅샷과 일치

`playwright-cli generate-locator <target>`으로 단언용 locator를 만들고 snapshot/eval로 기대값을 확인합니다.

텍스트 단언 시 locator에 그 요소의 텍스트 자체가 포함되지 않게 합니다. `getByTestId()`나 `getByLabel()`이 적합합니다. 텍스트 기반 locator는 `toBeVisible()`을 우선합니다.

비교 스냅샷은 단언에 필요한 정보만 포함하면 됩니다. 변동 값은 정규식을 사용할 수 있습니다.

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

## 1. 계획

목표는 검사 시나리오를 나열한 `specs/<feature>.plan.md` 등의 명세입니다. 반드시 파일로 작성합니다.

### 1.1 사전 조건: 작업 공간

먼저 작업 공간에 Playwright가 설치되어 있는지 확인합니다:

```bash
# Either of these confirms a workspace:
test -f playwright.config.ts || test -f playwright.config.js
npx --no-install playwright --version
```

없으면 초기화하고 사용자가 기본값을 선택하게 합니다:

```bash
npm init playwright@latest
```

### 1.2 사전 조건: 시작 테스트

시작 테스트(seed)는 앱 이동, 필요한 로그인, 기능 플래그 등 모든 시나리오의 초기 상태를 만드는 최소 테스트입니다. 시나리오는 seed 이후의 새 상태에서 시작합니다. `--debug=cli`가 이 테스트 안에서 멈추므로 계획/생성 세션의 시작점입니다.

최소 seed:

```ts
// tests/seed.spec.ts
import { test } from '@playwright/test';

test('seed', async ({ page }) => {
  await page.goto('https://example.com/');
});
```

권장 방식은 이동을 fixture에 넣어 시나리오가 재사용하게 하는 것입니다:

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

seed가 없으면 최소한 앱으로 이동하는 테스트를 만듭니다.

### 1.3 앱 탐색

seed를 백그라운드에서 실행하고 연결합니다:

```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test tests/seed.spec.ts --debug=cli
# wait for "Debugging Instructions" and the session name tw-XXXX
playwright-cli attach tw-XXXX
```

실행을 재개해 seed를 수행한 뒤 앱을 확인합니다:

```bash
playwright-cli resume                   # resume so that seed test runs fully
playwright-cli snapshot                 # inventory of interactive elements
playwright-cli click e5                 # follow a flow
playwright-cli eval "location.href"     # read URL / state
playwright-cli show --annotate          # ask the user to point at something
```

다음을 파악합니다:

- 조작 요소: 폼, 버튼, 목록, 필터, 모달.
- 주요 사용자 흐름 전체.
- 경계 사례: 빈 상태, 검증 오류, 긴 입력, 경계값.
- 지속성: 새로고침, 로컬/세션 저장소, URL fragment.
- 이동: URL을 바꾸는 컨트롤, 뒤로/앞으로 동작.

앱 URL만 직접 열지 말고 사용자 지정 준비를 반영하도록 반드시 테스트를 거쳐 실행합니다.
탐색을 마치면 백그라운드 테스트를 중지합니다.

### 1.4 명세 파일 작성

`specs/<feature>.plan.md`에 다음 구조로 저장합니다:

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

지침:

- 각 시나리오는 독립적이며 seed의 새 상태에서 시작합니다. 시나리오를 연결하지 않습니다.
- 시나리오 이름은 kebab-case이며 파일명과 일치합니다 (`should-add-single-todo` → `should-add-single-todo.spec.ts`).
- 정상, 경계, 검증, 실패 흐름, 지속성을 포함합니다.
- `fill` 호출 같은 API 설명 대신 “입력창에 Buy milk 입력”처럼 사용자 행동으로 씁니다.
- 관찰 가능한 결과를 `- expect:`에 쓰며 생성 시 각각 단언이 됩니다.

---

## 2. 생성

명세를 Playwright 테스트 파일로 만듭니다. 실제와 달라졌으면 명세도 갱신합니다.

### 2.1 입력

- 명세 파일, 예: `specs/basic-operations.plan.md`.
- 대상: 단일 시나리오(`1.2`), 그룹(`1`), 전체.
- seed 파일: 시나리오 그룹의 `**Seed:**`에서 확인.

### 2.2 시나리오 하나 생성

대상 시나리오를 순차 처리합니다. 같은 seed 세션을 공유하는 시나리오는 병렬 실행하지 않습니다:

```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test <seed-file> --debug=cli   # background
playwright-cli attach tw-XXXX
# resume
```

앱 URL만 열지 말고 사용자 지정 준비가 반영되도록 테스트를 거칩니다.

명세를 계획, 실제 앱을 증거로 삼아 `Steps:`를 하나씩 수행합니다. 버튼 지정이 모호하거나 요소가 없어졌거나 실제 동작과 다르면 판단하여 명세를 수정한 뒤 계속합니다. 생성 중 명세 수정은 정상 절차입니다.

각 조작은 해당 Playwright TypeScript를 출력합니다 ([생성 원리](#0-생성-원리)):

```bash
playwright-cli snapshot                         # find refs
playwright-cli fill e3 "John Doe"               # -> page.getByRole('textbox', {...}).fill(...)
playwright-cli press Enter
playwright-cli click e7
```

각 `- expect:`마다 명시적 단언을 추가합니다. [생성 원리](#0-생성-원리)를 참고하세요.

생성 코드를 모아 명세에 지정된 경로에 테스트를 씁니다:

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

규칙:

- 파일당 테스트 하나입니다. 파일 경로, describe 이름, 테스트 이름은 순번을 제외하고 명세 그대로 사용합니다.
- 각 단계 조작 앞에 `// N. <step text>` 주석을 넣습니다.
- describe 그룹 이름은 `1.` 같은 순번 없이 명세 그대로 사용합니다.
- 프로젝트에 `./fixtures`가 있으면 거기서, 없으면 `@playwright/test`에서 import합니다.
- 다음 시나리오 전에 CLI 세션을 닫고 백그라운드 테스트를 중지합니다.

### 2.3 여러 시나리오 생성

2.2를 대상마다 반복하고 사이에 seed를 재시작해 새 페이지에서 시작합니다. 서로 다른 생성 세션 이름으로 완전히 분리된 실행은 병렬화할 수 있지만 각 테스트를 반드시 중지해야 합니다. 같은 seed 세션은 공유하지 않습니다.

### 2.4 생성 테스트 실행

생성 후 새 테스트를 한 번 실행합니다:

```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test tests/<group>/<scenario>.spec.ts
```

실패하면 3절로 진행합니다.

---

## 3. 수정

실패한 테스트를 수정하고 앱의 의도된 동작이 바뀌었다면 명세를 갱신합니다.

### 3.1 실패 테스트 찾기

```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test
```

실패한 `<file>:<line>` 목록을 기록하고 하나씩 처리합니다. 공유 상태와 단일 CLI 세션 때문에 병렬 수정하지 않습니다.

### 3.2 실패 하나 디버깅

실패 테스트 하나를 디버그 모드로 백그라운드 실행한 뒤 연결합니다:

```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test tests/<group>/<scenario>.spec.ts:<line> --debug=cli
# wait for "Debugging Instructions" and the tw-XXXX session name
playwright-cli attach tw-XXXX
```

시작 지점에서 멈춘 테스트를 실패 조작/단언 직전까지 진행해 진단합니다:

```bash
playwright-cli snapshot                # did the element change / move / rename?
playwright-cli console                 # app-side errors?
playwright-cli requests                # failed request? wrong payload?
playwright-cli show --annotate         # ask the user to point somewhere
```

일반 원인: 선택자 변경, 래퍼 추가, label/ARIA 이름 변경, 전환/비동기 타이밍, 앱 텍스트 변경, 실행 간 테스트 데이터 누출.

`playwright-cli`로 수정 조작을 확인하고 출력 코드를 테스트에 반영합니다.

### 3.3 수정 적용

올바른 동작에 맞게 locator, 단언, 단계 순서, 입력을 수정합니다. 백그라운드 디버그 실행을 중지하고 해당 테스트를 다시 실행해 확인합니다.

수정 목적으로 hook을 건너뛰거나 sleep을 추가하지 않습니다. `networkidle`도 사용하지 않습니다.

### 3.4 명세와 일치시키기

테스트의 `// spec:` 헤더가 가리키는 명세에서 해당 시나리오를 찾습니다.

- locator 변경이나 단언 개선 등 기술 수정이며 사용자 동작이 같으면 명세는 유지합니다.
- 사용자 단계, 입력, 순서, 기대 결과가 바뀌면 명세를 갱신합니다. 시나리오 ID와 파일 경로는 유지하고 step/expect만 바꿉니다.
- 의도된 앱 변경인지 회귀 오류인지 불명확하면 중지하고 사용자에게 확인합니다. 다음을 제공합니다:
  - 시나리오 ID (예: `2.3`),
  - 일치하지 않는 명세 줄,
  - 관찰한 동작 (스냅샷 일부 또는 구체 결과).

사용자 답변 후 의도된 변경이면 명세를 갱신하고 회귀면 해당 버그를 다루는 테스트로 기록합니다.

### 3.5 반복과 중단

- 실패 하나씩 수정하고 매번 다시 실행합니다.
- 충분한 조사로 테스트가 맞고 앱이 틀렸으며 사용자도 버그임을 확인한 경우 `test.fixme(...)`에 사용자 결정이나 이슈 링크 주석을 남깁니다. 조용히 건너뛰지 않습니다.

---

## 관련 문서

| 항목 | 참고 |
|---|---|
| `--debug=cli` / 연결 방법 | [playwright-tests.md](playwright-tests.md) |
| 탐색/생성 중 요청 mock | [request-mocking.md](request-mocking.md) |
| CLI 브라우저 세션 관리 | [session-management.md](session-management.md) |
