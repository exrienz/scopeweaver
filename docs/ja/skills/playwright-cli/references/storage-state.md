[English](../../../../../skills/playwright-cli/references/storage-state.md) | [한국어](../../../../ko/skills/playwright-cli/references/storage-state.md) | [简体中文](../../../../zh-Hans/skills/playwright-cli/references/storage-state.md) | [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/storage-state.md) | [日本語](storage-state.md) | [Español](../../../../es/skills/playwright-cli/references/storage-state.md)

<a id="storage-management"></a>
# ストレージの管理



Cookie、localStorage、sessionStorage、ブラウザーの保存状態を管理します。

<a id="storage-state"></a>
## 保存状態

Cookie とストレージを含む、ブラウザーの状態全体を保存・復元します。

<a id="save-storage-state"></a>
### 保存状態を書き出す

```bash
# Save to auto-generated filename (storage-state-{timestamp}.json)
playwright-cli state-save

# Save to specific filename
playwright-cli state-save my-auth-state.json
```

<a id="restore-storage-state"></a>
### 保存状態を復元する

```bash
# Load storage state from file
playwright-cli state-load my-auth-state.json

# Reload page to apply cookies
playwright-cli open https://example.com
```

<a id="storage-state-file-format"></a>
### 保存状態ファイルの形式

保存したファイルには次を含みます:

```json
{
  "cookies": [
    {
      "name": "session_id",
      "value": "abc123",
      "domain": "example.com",
      "path": "/",
      "expires": 1893456000,
      "httpOnly": true,
      "secure": true,
      "sameSite": "Lax"
    }
  ],
  "origins": [
    {
      "origin": "https://example.com",
      "localStorage": [
        { "name": "theme", "value": "dark" },
        { "name": "user_id", "value": "12345" }
      ]
    }
  ]
}
```

<a id="cookies"></a>
## Cookie

<a id="list-all-cookies"></a>
### すべての Cookie を一覧表示

```bash
playwright-cli cookie-list
```

<a id="filter-cookies-by-domain"></a>
### ドメインで Cookie を絞り込む

```bash
playwright-cli cookie-list --domain=example.com
```

<a id="filter-cookies-by-path"></a>
### パスで Cookie を絞り込む

```bash
playwright-cli cookie-list --path=/api
```

<a id="get-specific-cookie"></a>
### 特定の Cookie を取得

```bash
playwright-cli cookie-get session_id
```

<a id="set-a-cookie"></a>
### Cookie を設定

```bash
# Basic cookie
playwright-cli cookie-set session abc123

# Cookie with options
playwright-cli cookie-set session abc123 --domain=example.com --path=/ --httpOnly --secure --sameSite=Lax

# Cookie with expiration (Unix timestamp)
playwright-cli cookie-set remember_me token123 --expires=1893456000
```

<a id="delete-a-cookie"></a>
### Cookie を削除

```bash
playwright-cli cookie-delete session_id
```

<a id="clear-all-cookies"></a>
### すべての Cookie を消去

```bash
playwright-cli cookie-clear
```

<a id="advanced-multiple-cookies-or-custom-options"></a>
### 高度な操作: 複数の Cookie やカスタムオプション

複数の Cookie の同時追加など、複雑な場面では `run-code` を使います:

```bash
playwright-cli run-code "async page => {
  await page.context().addCookies([
    { name: 'session_id', value: 'sess_abc123', domain: 'example.com', path: '/', httpOnly: true },
    { name: 'preferences', value: JSON.stringify({ theme: 'dark' }), domain: 'example.com', path: '/' }
  ]);
}"
```

<a id="local-storage"></a>
## ローカルストレージ

<a id="list-all-localstorage-items"></a>
### localStorage のすべての項目を一覧表示

```bash
playwright-cli localstorage-list
```

<a id="get-single-value"></a>
### 値を 1 つ取得

```bash
playwright-cli localstorage-get token
```

<a id="set-value"></a>
### 値を設定

```bash
playwright-cli localstorage-set theme dark
```

<a id="set-json-value"></a>
### JSON の値を設定

```bash
playwright-cli localstorage-set user_settings '{"theme":"dark","language":"en"}'
```

<a id="delete-single-item"></a>
### 項目を 1 つ削除

```bash
playwright-cli localstorage-delete token
```

<a id="clear-all-localstorage"></a>
### localStorage をすべて消去

```bash
playwright-cli localstorage-clear
```

<a id="advanced-multiple-operations"></a>
### 高度な操作: 複数の操作

複数の値の同時設定など、複雑な場面では `run-code` を使います:

```bash
playwright-cli run-code "async page => {
  await page.evaluate(() => {
    localStorage.setItem('token', 'jwt_abc123');
    localStorage.setItem('user_id', '12345');
    localStorage.setItem('expires_at', Date.now() + 3600000);
  });
}"
```

<a id="session-storage"></a>
## セッションストレージ

<a id="list-all-sessionstorage-items"></a>
### sessionStorage のすべての項目を一覧表示

```bash
playwright-cli sessionstorage-list
```

<a id="get-single-value-1"></a>
### 値を 1 つ取得

```bash
playwright-cli sessionstorage-get form_data
```

<a id="set-value-1"></a>
### 値を設定

```bash
playwright-cli sessionstorage-set step 3
```

<a id="delete-single-item-1"></a>
### 項目を 1 つ削除

```bash
playwright-cli sessionstorage-delete step
```

<a id="clear-sessionstorage"></a>
### sessionStorage を消去

```bash
playwright-cli sessionstorage-clear
```

<a id="indexeddb"></a>
## IndexedDB

<a id="list-databases"></a>
### データベースを一覧表示

```bash
playwright-cli run-code "async page => {
  return await page.evaluate(async () => {
    const databases = await indexedDB.databases();
    return databases;
  });
}"
```

<a id="delete-database"></a>
### データベースを削除

```bash
playwright-cli run-code "async page => {
  await page.evaluate(() => {
    indexedDB.deleteDatabase('myDatabase');
  });
}"
```

<a id="common-patterns"></a>
## よく使うパターン

<a id="authentication-state-reuse"></a>
### 認証状態の再利用

```bash
# Step 1: Login and save state
playwright-cli open https://app.example.com/login
playwright-cli snapshot
playwright-cli fill e1 "user@example.com"
playwright-cli fill e2 "password123"
playwright-cli click e3

# Save the authenticated state
playwright-cli state-save auth.json

# Step 2: Later, restore state and skip login
playwright-cli state-load auth.json
playwright-cli open https://app.example.com/dashboard
# Already logged in!
```

<a id="save-and-restore-roundtrip"></a>
### 保存と復元の往復確認

```bash
# Set up authentication state
playwright-cli open https://example.com
playwright-cli eval "() => { document.cookie = 'session=abc123'; localStorage.setItem('user', 'john'); }"

# Save state to file
playwright-cli state-save my-session.json

# ... later, in a new session ...

# Restore state
playwright-cli state-load my-session.json
playwright-cli open https://example.com
# Cookies and localStorage are restored!
```

<a id="security-notes"></a>
## セキュリティ上の注意

- 認証トークンを含む保存状態ファイルをコミットしないでください。
- `*.auth-state.json` を `.gitignore` に追加してください。
- 自動操作が完了したら状態ファイルを削除してください。
- 機密データには環境変数を使ってください。
- 既定のセッションはメモリー上で動作し、機密性のある操作により安全です。
