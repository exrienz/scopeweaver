[English](../../../../../skills/playwright-cli/references/request-mocking.md) · [한국어](../../../../ko/skills/playwright-cli/references/request-mocking.md) · 简体中文 · [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/request-mocking.md) · [日本語](../../../../ja/skills/playwright-cli/references/request-mocking.md) · [Español](../../../../es/skills/playwright-cli/references/request-mocking.md)

<a id="request-mocking"></a>

# 请求模拟

各语言版本请见本页上方导览。

拦截、模拟、修改与封锁网络请求。

<a id="cli-route-commands"></a>

## CLI 路由命令



```bash
# Mock with custom status
playwright-cli route "**/*.jpg" --status=404

# Mock with JSON body
playwright-cli route "**/api/users" --body='[{"id":1,"name":"Alice"}]' --content-type=application/json

# Mock with custom headers
playwright-cli route "**/api/data" --body='{"ok":true}' --header="X-Custom: value"

# Remove headers from requests
playwright-cli route "**/*" --remove-header=cookie,authorization

# List active routes
playwright-cli route-list

# Remove a route or all routes
playwright-cli unroute "**/*.jpg"
playwright-cli unroute
```



<a id="url-patterns"></a>

## URL 模式



```
**/api/users           - Exact path match
**/api/*/details       - Wildcard in path
**/*.{png,jpg,jpeg}    - Match file extensions
**/search?q=*          - Match query parameters
```



<a id="advanced-mocking-with-run-code"></a>

## 以 run-code 进阶模拟

适用于条件响应、检查请求本文、修改响应或延迟：

<a id="conditional-response-based-on-request"></a>

### 依请求条件响应



```bash
playwright-cli run-code "async page => {
  await page.route('**/api/login', route => {
    const body = route.request().postDataJSON();
    if (body.username === 'admin') {
      route.fulfill({ body: JSON.stringify({ token: 'mock-token' }) });
    } else {
      route.fulfill({ status: 401, body: JSON.stringify({ error: 'Invalid' }) });
    }
  });
}"
```



<a id="modify-real-response"></a>

### 修改真实响应



```bash
playwright-cli run-code "async page => {
  await page.route('**/api/user', async route => {
    const response = await route.fetch();
    const json = await response.json();
    json.isPremium = true;
    await route.fulfill({ response, json });
  });
}"
```



<a id="simulate-network-failures"></a>

### 模拟网络失败



```bash
playwright-cli run-code "async page => {
  await page.route('**/api/offline', route => route.abort('internetdisconnected'));
}"
# Options: connectionrefused, timedout, connectionreset, internetdisconnected
```



<a id="delayed-response"></a>

### 延迟响应



```bash
playwright-cli run-code "async page => {
  await page.route('**/api/slow', async route => {
    await new Promise(r => setTimeout(r, 3000));
    route.fulfill({ body: JSON.stringify({ data: 'loaded' }) });
  });
}"
```
