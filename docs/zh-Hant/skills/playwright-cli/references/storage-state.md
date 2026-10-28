[English](../../../../../skills/playwright-cli/references/storage-state.md) · [한국어](../../../../ko/skills/playwright-cli/references/storage-state.md) · [简体中文](../../../../zh-Hans/skills/playwright-cli/references/storage-state.md) · 繁體中文 · [日本語](../../../../ja/skills/playwright-cli/references/storage-state.md) · [Español](../../../../es/skills/playwright-cli/references/storage-state.md)

<a id="storage-management"></a>

# 儲存管理

各語言版本請見本頁上方導覽。

管理 cookie、localStorage、sessionStorage 與瀏覽器儲存狀態。

<a id="storage-state"></a>

## 儲存狀態

儲存並還原包括 cookie 與儲存空間的完整瀏覽器狀態。

<a id="save-storage-state"></a>

### 儲存狀態



```bash
# Save to auto-generated filename (storage-state-{timestamp}.json)
playwright-cli state-save

# Save to specific filename
playwright-cli state-save my-auth-state.json
```



<a id="restore-storage-state"></a>

### 還原狀態



```bash
# Load storage state from file
playwright-cli state-load my-auth-state.json

# Reload page to apply cookies
playwright-cli open https://example.com
```



<a id="storage-state-file-format"></a>

### 儲存狀態檔案格式

儲存的檔案包含：



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

### 列出全部 cookie



```bash
playwright-cli cookie-list
```



<a id="filter-cookies-by-domain"></a>

### 依網域篩選 cookie



```bash
playwright-cli cookie-list --domain=example.com
```



<a id="filter-cookies-by-path"></a>

### 依路徑篩選 cookie



```bash
playwright-cli cookie-list --path=/api
```



<a id="get-specific-cookie"></a>

### 取得特定 cookie



```bash
playwright-cli cookie-get session_id
```



<a id="set-a-cookie"></a>

### 設定 cookie



```bash
# Basic cookie
playwright-cli cookie-set session abc123

# Cookie with options
playwright-cli cookie-set session abc123 --domain=example.com --path=/ --httpOnly --secure --sameSite=Lax

# Cookie with expiration (Unix timestamp)
playwright-cli cookie-set remember_me token123 --expires=1893456000
```



<a id="delete-a-cookie"></a>

### 刪除 cookie



```bash
playwright-cli cookie-delete session_id
```



<a id="clear-all-cookies"></a>

### 清除全部 cookie



```bash
playwright-cli cookie-clear
```



<a id="advanced-multiple-cookies-or-custom-options"></a>

### 進階：多個 cookie 或自訂選項

一次新增多個 cookie 等複雜情境可使用 `run-code`：



```bash
playwright-cli run-code "async page => {
  await page.context().addCookies([
    { name: 'session_id', value: 'sess_abc123', domain: 'example.com', path: '/', httpOnly: true },
    { name: 'preferences', value: JSON.stringify({ theme: 'dark' }), domain: 'example.com', path: '/' }
  ]);
}"
```



<a id="local-storage"></a>

## 本機儲存空間

<a id="list-all-localstorage-items"></a>

### 列出全部 localStorage 項目



```bash
playwright-cli localstorage-list
```



<a id="get-single-value"></a>

### 取得單一值



```bash
playwright-cli localstorage-get token
```



<a id="set-value"></a>

### 設定值



```bash
playwright-cli localstorage-set theme dark
```



<a id="set-json-value"></a>

### 設定 JSON 值



```bash
playwright-cli localstorage-set user_settings '{"theme":"dark","language":"en"}'
```



<a id="delete-single-item"></a>

### 刪除單一項目



```bash
playwright-cli localstorage-delete token
```



<a id="clear-all-localstorage"></a>

### 清除全部 localStorage



```bash
playwright-cli localstorage-clear
```



<a id="advanced-multiple-operations"></a>

### 進階：多項操作

一次設定多個值等複雜情境可使用 `run-code`：



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

## 工作階段儲存空間

<a id="list-all-sessionstorage-items"></a>

### 列出全部 sessionStorage 項目



```bash
playwright-cli sessionstorage-list
```



<a id="get-single-value"></a>

### 取得單一值



```bash
playwright-cli sessionstorage-get form_data
```



<a id="set-value"></a>

### 設定值



```bash
playwright-cli sessionstorage-set step 3
```



<a id="delete-single-item"></a>

### 刪除單一項目



```bash
playwright-cli sessionstorage-delete step
```



<a id="clear-sessionstorage"></a>

### 清除 sessionStorage



```bash
playwright-cli sessionstorage-clear
```



<a id="indexeddb"></a>

## IndexedDB

<a id="list-databases"></a>

### 列出資料庫



```bash
playwright-cli run-code "async page => {
  return await page.evaluate(async () => {
    const databases = await indexedDB.databases();
    return databases;
  });
}"
```



<a id="delete-database"></a>

### 刪除資料庫



```bash
playwright-cli run-code "async page => {
  await page.evaluate(() => {
    indexedDB.deleteDatabase('myDatabase');
  });
}"
```



<a id="common-patterns"></a>

## 常見模式

<a id="authentication-state-reuse"></a>

### 重用驗證狀態



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

### 儲存與還原往返



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

## 安全注意事項

- 絕不提交包含驗證權杖的儲存狀態檔。
- 將 `*.auth-state.json` 加入 `.gitignore`。
- 自動化完成後刪除狀態檔。
- 敏感資料使用環境變數。
- 預設工作階段採記憶體模式，對敏感操作較安全。
