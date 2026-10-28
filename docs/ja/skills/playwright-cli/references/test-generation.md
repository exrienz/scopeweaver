[English](../../../../../skills/playwright-cli/references/test-generation.md) | [한국어](../../../../ko/skills/playwright-cli/references/test-generation.md) | [简体中文](../../../../zh-Hans/skills/playwright-cli/references/test-generation.md) | [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/test-generation.md) | [日本語](test-generation.md) | [Español](../../../../es/skills/playwright-cli/references/test-generation.md)

<a id="test-generation-plan--generate--heal"></a>
# テストの生成（計画 → 生成 → 修復）



`playwright-cli` で Playwright テストを作成・保守する、最初から最後までの作業フローです。すべての `playwright-cli` 操作は、対応する Playwright TypeScript を出力し、この生成コードがすべてのテストの素材になります。下記の各節は個別にも使えます:

- **生成の仕組み** — すべての基礎となる仕組み。操作を TypeScript に変換し、アサーションを追加する方法。
- **計画** — アプリを調べ、テスト内容を記述した仕様ファイルを作成。
- **生成** — 仕様から Playwright テストファイルを作成。曖昧・古い仕様は更新。
- **修復** — 失敗したテストを診断してコードを修正し、仕様を実際の動作と整合させる。

計画・生成・修復は同じ仕組みを使います。`npx playwright test --debug=cli` をバックグラウンドで実行し、`playwright-cli attach tw-XXXX` で停止中のページを対話的に操作します。デバッグと接続の仕組みは [playwright-tests.md](playwright-tests.md) を参照してください。

---

<a id="0-how-generation-works"></a>
## 0. 生成の仕組み

`playwright-cli` の各操作は、対応する Playwright の TypeScript コードを生成します。出力のコードを、そのままテストファイルにコピーできます。

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
### テストファイルの作成

生成したコードを Playwright のテストにまとめます:

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
### 意味に基づくロケーターを使う

生成コードは可能ならロールに基づく、変更に強いロケーターを使います:

```typescript
// Generated (good - semantic)
await page.getByRole('button', { name: 'Submit' }).click();

// Avoid (fragile - CSS selectors)
await page.locator('#submit-btn').click();
```

<a id="explore-before-recording"></a>
### 記録前に調べる

操作の記録前に、スナップショットでページの構造を理解します:

```bash
playwright-cli open https://example.com
playwright-cli snapshot
# Review the element structure
playwright-cli click e5
```

<a id="add-assertions-manually"></a>
### アサーションを手動で追加

生成コードは操作を記録しますが、アサーションは記録しません。推奨する以下のマッチャーで期待値を追加してください:

- `toBeVisible()` — 要素が描画され、見えている
- `toHaveText(text)` — 要素のテキストが一致する
- `toHaveValue(value) / toBeEmpty()` — 入力・選択の値が一致する
- `toBeChecked() / toBeUnchecked()` — チェックボックスの状態が一致する
- `toMatchAriaSnapshot(snapshot)` — ページまたはロケーターが部分的なアクセシビリティスナップショットに一致する

`playwright-cli generate-locator <target>` でアサーション用のロケーター式を生成し、snapshot/eval コマンドで期待値を取得してください。

テキスト内容を検査する際は、生成ロケーターにその要素自身のテキストが含まれないようにしてください。通常 `getByTestId()` や `getByLabel()` が適しています。テキストによるロケーターでは、代わりに `toBeVisible()` を優先してください。

照合するスナップショットは、すべての情報を含める必要はありません。アサーションに必要な内容だけを取得します。不安定な値には正規表現を使えます。

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
## 1. 計画

目的: テストするシナリオを列挙した仕様ファイル（例: `specs/<feature>.plan.md`）を作成します。**必ず**ファイルに保存してください。

<a id="11-prerequisite-workspace"></a>
### 1.1 前提条件: ワークスペース

最初に、ワークスペースに Playwright がインストールされているか確認します:

```bash
# Either of these confirms a workspace:
test -f playwright.config.ts || test -f playwright.config.js
npx --no-install playwright --version
```

なければ初期構成を作成し、既定値をユーザーに選んでもらってください:

```bash
npm init playwright@latest
```

<a id="12-prerequisite-seed-test"></a>
### 1.2 前提条件: シードテスト

**シードテスト**は、アプリへの移動、必要なログイン、機能フラグなど、各シナリオの開始状態を作る最小限のテストです。シナリオはシード実行後の新しい状態から始まると想定します。`--debug=cli` はこのテストの**内側**で停止するため、すべての計画・生成セッションはシードから始まります。

最小限のシード:

```ts
// tests/seed.spec.ts
import { test } from '@playwright/test';

test('seed', async ({ page }) => {
  await page.goto('https://example.com/');
});
```

推奨: シナリオが再利用できるよう、ページ移動を fixture に入れます:

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

シードがなければ、少なくともアプリへ移動するものを作成してください。

<a id="13-explore-the-app"></a>
### 1.3 アプリを調べる

シードからアプリをバックグラウンドで起動し、接続します:

```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test tests/seed.spec.ts --debug=cli
# wait for "Debugging Instructions" and the session name tw-XXXX
playwright-cli attach tw-XXXX
```

再開してシードを実行してから、アプリを調べます:

```bash
playwright-cli resume                   # resume so that seed test runs fully
playwright-cli snapshot                 # inventory of interactive elements
playwright-cli click e5                 # follow a flow
playwright-cli eval "location.href"     # read URL / state
playwright-cli show --annotate          # ask the user to point at something
```

次を整理します:

- 操作できる対象面（フォーム、ボタン、一覧、絞り込み、モーダル）。
- 主要なユーザーの一連の操作。
- 境界事例: 空の状態、入力エラー、非常に長い入力、境界値。
- 永続化: 再読み込み、local/session storage、URL のフラグメント。
- ページ移動: URL を変える操作、戻る・進むの動作。

**重要**: playwright-cli でアプリの URL を直接開くだけではなく、独自の初期設定を反映するため、必ずテストを経由してください。**重要**: 調査が終わったら、バックグラウンドのテストを停止してください。

<a id="14-write-the-spec-file"></a>
### 1.4 仕様ファイルを書く

`specs/<feature>.plan.md` に、次の構成で保存します:

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

指針:

- 各シナリオは独立し、シードの新しい状態から始めます。シナリオを連鎖させてはいけません。
- シナリオ名は kebab-case とし、テストファイル名と一致させます（`should-add-single-todo` → `should-add-single-todo.spec.ts`）。
- 正常系、境界事例、入力検証、異常系、永続化を含めてください。
- 手順は API の単位（「`fill` を呼ぶ」）ではなく、ユーザーの単位（「入力欄に『牛乳を買う』と入力」）で書いてください。
- 観測できる結果を `- expect:` の箇条書きにし、生成時にそれぞれアサーションにします。

---

<a id="2-generate"></a>
## 2. 生成

目的: 仕様ファイルから Playwright テストファイルを作成します。実際とずれていれば、必要に応じて仕様も更新します。

<a id="21-inputs"></a>
### 2.1 入力

- **仕様ファイル**。例: `specs/basic-operations.plan.md`。
- **対象**: シナリオ 1 件（例: `1.2`）、グループ全体（`1`）、すべてのいずれか。
- **シードファイル**: シナリオのグループの `**Seed:**` 行から読み取る。

<a id="22-generate-one-scenario"></a>
### 2.2 シナリオを 1 件生成

各対象シナリオを順に処理します（シードセッションを共有するため、並列にしない）:

```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test <seed-file> --debug=cli   # background
playwright-cli attach tw-XXXX
# resume
```

playwright-cli で URL を直接開くだけに**せず**、独自の初期設定を反映するため必ずテストを経由してください。

仕様を計画、実際のアプリを正とし、`playwright-cli` でシナリオの `Steps:` を 1 つずつ実行します。手順が曖昧（「ボタンをクリック」ではどのボタンか）、存在しない要素を参照する、実際の動作に矛盾する場合は、判断して仕様を実際の動作に合わせて更新し、続けてください。生成中の仕様編集は想定されています。

各操作は対応する Playwright TypeScript を表示します（[生成の仕組み](#0-how-generation-works)参照）:

```bash
playwright-cli snapshot                         # find refs
playwright-cli fill e3 "John Doe"               # -> page.getByRole('textbox', {...}).fill(...)
playwright-cli press Enter
playwright-cli click e7
```

各 `- expect:` 項目に明示的なアサーションを追加してください。詳細は[生成の仕組み](#0-how-generation-works)を参照。

生成コードをまとめ、仕様が示すパスにテストファイルを書き込みます:

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

規則:

- **1 ファイルに 1 テスト。** ファイルパス、describe 名、テスト名は、序数を除き仕様のとおりにします。
- 番号付きの各手順の操作前に、`// N. <step text>` のコメントを付けます。
- describe のグループ名は、仕様のとおり使います（`1.` の序数を除く）。
- プロジェクトに `./fixtures` があればそこから、なければ `@playwright/test` からインポートします。
- **重要**: 次のシナリオに進む前に、CLI セッションを閉じ、バックグラウンドのテストを停止してください。

<a id="23-generate-multiple-scenarios"></a>
### 2.3 複数シナリオの生成

対象シナリオごとに 2.2 を繰り返し、毎回シードを再起動して、各テストをきれいなページから始めます。生成されるセッション名は一意のため、各実行を必ず停止すれば、安全に並列化できます。

<a id="24-run-generated-tests"></a>
### 2.4 生成したテストを実行

生成後、新しいテストを 1 回実行します:

```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test tests/<group>/<scenario>.spec.ts
```

失敗は第 3 節で処理します。

---

<a id="3-heal"></a>
## 3. 修復

目的: 失敗したテストを修正し、アプリの意図した動作が変わった場合は仕様を更新します。

<a id="31-find-failing-tests"></a>
### 3.1 失敗したテストの特定

```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test
```

失敗した `<file>:<line>` の一覧を記録し、1 件ずつ処理します。共有状態と単一の CLI セッションでは壊れやすいため、並列で修正しないでください。

<a id="32-debug-one-failure"></a>
### 3.2 失敗を 1 件デバッグ

失敗した 1 テストをデバッグモードでバックグラウンド実行し、接続します:

```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test tests/<group>/<scenario>.spec.ts:<line> --debug=cli
# wait for "Debugging Instructions" and the tw-XXXX session name
playwright-cli attach tw-XXXX
```

テストは開始時に一時停止します。ステップ実行するか、失敗する操作・アサーションの直前まで進めて診断します:

```bash
playwright-cli snapshot                # did the element change / move / rename?
playwright-cli console                 # app-side errors?
playwright-cli requests                # failed request? wrong payload?
playwright-cli show --annotate         # ask the user to point somewhere
```

よくある原因: セレクターのずれ、新しいラッパー要素、ラベル・ARIA 名の変更、時間の問題（遷移、非同期読み込み）、アプリのアサーション対象テキストの更新、実行間のテストデータ漏れ。

`playwright-cli` で修正した操作を試し、出力の生成コードをテストに貼り戻します。

<a id="33-apply-the-fix"></a>
### 3.3 修正の適用

テストファイルを編集し、ロケーター、アサーション、手順の順序、入力を正しい動作に合わせます。バックグラウンドのデバッグ実行を停止し、単一テストを再実行して合格を確認してください。

修正のためにフックを省略したり、sleep を追加したりしてはいけません。`networkidle` は使わないでください。

<a id="34-reconcile-with-the-spec"></a>
### 3.4 仕様との整合

テストファイルの `// spec:` ヘッダーが参照する仕様を開き、対応するシナリオを見つけます。

- **技術的な修正だけ**（ロケーターのずれ、より適切なアサーション）で、仕様のユーザー向け動作がまだアプリと一致する → 仕様を変えない。
- **ユーザーに見える手順、入力、順序、期待する結果を変更した** → 仕様を実際に合わせて更新。シナリオ ID とファイルパスは保持し、step / expect の行だけを変更する。
- **意図した変更**（仕様が古い）か、**回帰**（テストは正しくアプリが誤り）か不明 → **停止してユーザーに質問**。次を示す:
  - シナリオ ID（例: `2.3`）、
  - 一致しなくなった仕様の行、
  - 観測したアプリの動作（スナップショットの抜粋または具体的な結果）。

ユーザーの回答後にのみ、仕様を更新するか（意図した変更）、バグを検出するテストとして報告・記録します（回帰）。

<a id="35-iteration-and-giving-up"></a>
### 3.5 反復と断念

- 失敗は 1 件ずつ修正し、その都度再実行します。
- 十分に調べてテストが正しくアプリが誤りだと確信し、**ユーザーがバグと確認した**場合は、判断や Issue のリンクをコメントして `test.fixme(...)` を付けます。黙ってスキップしてはいけません。

---

<a id="cross-references"></a>
## 関連資料

| 目的 | 参照 |
|---|---|
| `--debug=cli` / attach の仕組み | [playwright-tests.md](playwright-tests.md) |
| 調査・生成中のリクエストのモック | [request-mocking.md](request-mocking.md) |
| CLI のブラウザーセッションの管理 | [session-management.md](session-management.md) |
