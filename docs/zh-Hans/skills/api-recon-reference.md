[English](../../../skills/api-recon/reference.md) · [한국어](../../ko/skills/api-recon-reference.md) · 简体中文 · [繁體中文](../../zh-Hant/skills/api-recon-reference.md) · [日本語](../../ja/skills/api-recon-reference.md) · [Español](../../es/skills/api-recon-reference.md)

<a id="api-recon-reference"></a>

# api-recon 参考

各语言版本请见本页上方导览。

Grep 配方、`config.json` 模板与排错。在 `js/` 运行 grep。单行 bundle 可选用 `js-beautify` 或 `sed 's/}/}\n/g'`；一般原始 grep 搭配上下文视窗已足够。

<a id="script-notes"></a>

## 脚本注意事项

`scripts/` 全部文件为参考模板，运行前依目标调整。常见修改：

| 脚本 | 常见调整 |
|---|---|
| `harvest_static.py` | 端点 regex、webpack／Vite manifest、微前端 publicPath、重试／并行 |
| `runtime_harvest.js` | 中和字段名称／成功值、stub 比对／本文形状、路由来源、WS 捕获、`waitUntil`／`routeTimeout`／`proxy` |
| `preload.js` | `loginPathRe`、L1 stub、`neutralize.fields`、`apiPattern`、是否启用 L3、`recordDetail`、`observe.*`、`neutralizeVueRouter` |
| `spider_mpa.py` | `--exclude` 破坏性链接、cookie、深度／上限、同源筛选 |
| `extract_route_map.py` | `routeMap`／`routeLink` regex、KEY 命名模式 |
| `build_perm_tree.py` | `userRouteAuth` 剖析、`ROOTS`／`PREFIX_PARENT` 阶层启发式、外层 stub 字段名称 |
| `config.json` | 上述网站特定参数的集中配置 |

调整文件留在任务目录（如 `recon/`），并报告相对模板的精确差异。

---

<a id="a-reconstruct-the-three-gates"></a>

## A. 重建三个关卡

<a id="a1-rendering-how-is-login-state-determined"></a>

### A1. 渲染：如何判定登录状态？



```bash
grep -rhoaE '.{0,40}(isLogin|isAuthenticated|loggedIn|hasLogin|requireAuth)\b.{0,80}' js | head
grep -rhoaE 'function (getUser|getToken|getAuth)[0-9]?\([^)]*\)\{.{0,200}' js | head
grep -rhoaE '(localStorage|sessionStorage)\.getItem\("[^"]+"\)' js | sort -u
grep -rhoaE '(Cookies?|cookie)\.(get|load)\("[^"]+"\)' js | sort -u
grep -rhoaE '\batob\(|JSON\.parse\(|jwt|decode' js | head
```



追踪 `isLogin = f(getUser())` → `getUser = decode(storage.read(KEY))`，识别保存键、容器（Cookie／localStorage）与编码：

| 编码 | 配置中的模拟值 |
|---|---|
| 纯字串／`"1"`／token | `"value": "anything-truthy"` |
| `JSON.parse(x)` | `"value": "json:{\"id\":1,\"username\":\"admin\"}"` |
| `JSON.parse(atob(x))` | `"value": "b64json:{\"id\":1,\"username\":\"admin\"}"` |
| JWT | 未签章／`alg:none` JWT，或使用 bundle 内密钥签章，仅供客户端模拟守卫 |
| 加密（SM2／AES／RSA） | 寻找写死密钥；仅在渲染守卫只需可解码 blob 时模拟，否则回退静态分析 |

写入 `cookies`／`localStorage`。

<a id="a2-interceptor-what-triggers-login-redirects"></a>

### A2. 拦截器：什么触发 /login 重新导向？



```bash
grep -rhoaE '.{0,60}(interceptors\.response|axios|request\.use).{0,120}' js | head
grep -rhoaE '.{0,40}(response_code|errcode|errno|\bcode\b|\bret\b|\bstatus\b)\s*[=!]==?\s*[\-0-9]{1,4}.{0,60}' js | head -20
grep -rhoaE '.{0,40}(未登录|请重新登录|登录已过期|unauthorized|登录失效|授权|token.{0,10}invalid).{0,40}' js | head
grep -rhoaE '.{0,30}(location\.href|router\.(push|replace)|navigate)\([^)]*login[^)]*\)' js | head
```



识别字段名、成功值（通常 `0` 或 `200`）及触发重新导向的失败值，以无效会话验证：



```bash
curl -sk -X POST -H 'Cookie: <fakekey>=junk' https://target/api/<protected> -d '{}' -H 'Content-Type: application/json'
```



写入 `neutralize.fields` + `neutralize.success`。

<a id="a3-content-where-do-menus-and-permissions-come-from"></a>

### A3. 内容：选单与权限从哪里来？



```bash
grep -rhoaE '"/api[^"]*(permission|perm|role|menu|acl|resource|nav)[^"]*"' js | sort -u
grep -rhoaE '.{0,30}(menus|permissions|menuList|routeList|authList|role_permissions)\b.{0,120}' js | head
grep -rhoaE 'userRouteAuth|getResultTree|routeMap|routeLink|hasPermission|checkAuth' js | head
grep -rhoaE '([A-Z_][A-Z0-9_]*):\{name:"[^"]*",link:"/[^"]+"\}' js | head
```



业务管理界面常见两层数据：

| API | 典型 payload | 使用端 |
|---|---|---|
| `.../role_permissions` | `{ permissions: string[], role_type }` | 路由守卫、按钮层 ACL |
| `.../permissions/all` | `tree[{ code, position, children }]` | 侧栏选单渲染 |
| bundle 中的 `userRouteAuth` | `{ CODE: { url, name? } }` | code → 前端路径 |
| bundle 中的 `routeMap` | `{ KEY: { name, link } }` | 别名解析（webpack `o.DASHBOARD`） |

阅读使用端代码，确认 `getResultTree(tree, permissions)` 如何筛选，以及 `v-if`／`hasAuth(code)` 检查哪些字段。

小型网站可在 `stubs` 手动创建宽松模拟 payload。

大型网站侧栏／子模组空白时，依 I 节重建完整权限树。

---

<a id="b-configjson-template"></a>

## B. config.json 模板



```json
{
  "baseUrl": "https://target/",
  "runtimeMode": "both",
  "chromium": "/usr/bin/chromium",

  "cookies": [
    { "name": "auth", "value": "b64json:{\"id\":1,\"username\":\"admin\",\"role\":\"admin\",\"func\":{},\"permissions\":[\"*\"]}" }
  ],
  "localStorage": { "token": "faketoken", "isLogin": "1" },

  "neutralize": {
    "fields": ["response_code", "code", "errno", "ret", "status"],
    "success": 0,
    "flags": { "success": true, "message": "ok" }
  },
  "forward": true,
  "loginUrlPattern": "/login",
  "apiPattern": "/api/|/rest/|/graphql",

  "mockTier": "L1+L2",
  "recordDetail": true,
  "observe": {
    "storageReads": false,
    "cookieReads": false,
    "xhrHeaders": true
  },
  "neutralizeVueRouter": true,
  "stubs": [
    {
      "match": "permissions/all|/menu|role_permissions",
      "body": {
        "response_code": 0, "code": 0,
        "data": {
          "permissions": ["*"],
          "menus": [
            { "name": "dashboard", "path": "/dashboard", "show": true, "children": [] },
            { "name": "alert", "path": "/alert", "show": true, "children": [] }
          ]
        }
      }
    }
  ],

  "explore": {
    "clickTabs": true,
    "clickTables": true,
    "pushStateFallback": true,
    "maxMenuItems": 50
  },

  "routes": ["/dashboard", "/alert", "/asset", "/device", "/report", "/config", "/system"],
  "waitMs": 1500, "perRouteMs": 900, "headless": true,
  "waitUntil": "domcontentloaded",
  "routeTimeout": 12000,
  "proxy": "",

  "captureResponses": true, "recordWs": true, "respMax": 600
}
```



字段：
- `runtimeMode`：`depth`（Puppeteer）、`coverage`（browser MCP）、`both`。
- `cookies[].value` 前缀：`b64json:` → base64(JSON)；`json:` → 原始 JSON；无 → 字面值。
- `forward: true` 转发真实请求并改写状态码字段；`false` 使用离线 stub。
- `mockTier`：coverage preload 层，例如 `L1+L2`、`L1+L2+L3`。
- `routes` 来自 `routes.txt`；模拟选单后，测试架构附加 `<a href>` 路由。
- `captureResponses`／`recordWs` 仅适用 depth。
- `waitUntil`：大型 SPA 用 `domcontentloaded`，避免 `networkidle2` 卡住。
- `routeTimeout`：每个路由的 `page.goto` 逾时毫秒数。
- `proxy`：Puppeteer `--proxy-server`；亦可用 `HTTP_PROXY`／`HTTPS_PROXY`。

<a id="b1-two-stub-template-role_permissions--permissionsall"></a>

### B1. 双 stub 模板（role_permissions + permissions/all）



```json
"stubs": [
  {
    "match": "role_permissions",
    "body": {
      "response_code": 0,
      "data": {
        "permissions": ["MONITOR", "MONITOR_ALERT", "THREAT", "ASSETS_RISK"],
        "role_type": "SUPER_ADMIN"
      }
    }
  },
  {
    "match": "permissions/all",
    "body": {
      "response_code": 0,
      "data": [
        {
          "code": "MONITOR",
          "position": 1,
          "children": [
            { "code": "MONITOR_ALERT", "position": 1, "children": [] }
          ]
        }
      ]
    }
  }
]
```



外层字段（`response_code`／`code`／`data`）须符合 A2；`permissions` 必须涵盖树中全部叶代码。

---

<a id="c-coverage-preload-configuration"></a>

## C. Coverage preload 配置

编辑副本 `preload.js` 顶部 `CONFIG`，或在 CDP 注入前替换：



```javascript
const CONFIG = {
  loginPathRe: /\/(login|signin)(\/|$|\?)/i,
  mockTier: 'L1+L2',
  forward: true,
  recordDetail: true,
  extractUrlsFromResponse: true,
  neutralizeVueRouter: true,
  observe: { storageReads: false, cookieReads: false, xhrHeaders: true },
  neutralize: { fields: ['response_code', 'code'], success: 0 },
  stubs: [ /* Same as config.json stubs */ ],
  apiPattern: /\/(api|apis|v\d+|dev|internal|graphql)\//i,
};
```



验证 `window.__API_RECON_PRELOAD__ === true` 且 pathname 稳定。

导出捕获结果：



```javascript
JSON.stringify({
  apis: [...window.__API_RECON_LOG__],
  detail: window.__API_RECON_DETAIL__,
  routes: [...(window.__API_RECON_ROUTES__ || [])],
  observe: window.__API_RECON_OBSERVE__,
}, null, 2)
```



---

<a id="d-preloadruntime-hook-coverage"></a>

## D. Preload／runtime hook 覆盖

preload（coverage）与 runtime_harvest（depth）内建浏览器 hook：

| Hook | 探索价值 | 覆盖 |
|---|---|---|
| Hook fetch / XHR.open | 捕获 URL／方法 | ✅ `recordDetail` + `__API_RECON_LOG__` |
| Hook XHR.setRequestHeader | 探索 Authorization 等头部 | ✅ `observe.xhrHeaders` |
| Hook localStorage／cookie 读取 | 识别会话键 | 选用 `observe.storageReads/cookieReads` |
| Vue 路由搜集 | 补全 frontendRoutes | ✅ `__API_RECON_ROUTES__`（已加载路由） |
| 中和 Vue 守卫／封锁登录重新导向 | 渲染模组以触发 API | ✅ `neutralizeVueRouter` + 原生重新导向中和 |
| React 路由搜集 | 补全路由 | 静态 + 点选；无专用 hook |
| 封锁页面登录重新导向 | 保持页面以供分析 | 仅封锁登录路径，保留业务导览 |
| Hook 加密函数库（CryptoJS／SM 等） | 加密参数 → 明文 API 本文 | 须手动 hook 加密输入，将结果记入配置 |
| 略过反侦错 | 否则运行时无法捕获 API | 须手动处理；仍可静态分析 |

---

<a id="e-endpoint-regex-when-static-results-are-sparse"></a>

## E. 静态结果稀少时的端点 regex

扩大 `harvest_static.py` 的 `extract_endpoints`，或在必要 harvest 阶段后手动检查：



```bash
grep -rhoaE '"/[a-z][A-Za-z0-9_/\-]{3,}"' js | sort -u
grep -rhoaE '/api/[a-zA-Z0-9_./-]+' js | sort -u
```



---

<a id="f-troubleshooting"></a>

## F. 疑难排解

| 症状 | 原因与处理 |
|---|---|
| 静态 API 少 | 端点语法不符，扩大 regex（E 节） |
| chunk 数远少于 manifest | 仅 CSS 或未部署 chunk；重试 404 |
| 运行时仍显示登录 | 渲染守卫：重查 A1 键、容器、编码、网域 |
| 外壳正常、模组空白 | 内容关卡：模拟选单（A3），检查 `routes` |
| 各路由只有 bootstrap／locale | 缺权限代码：重建树（I），检查 `role_permissions` 与 `permissions/all` stub |
| 有侧栏、子页空白 | 缺中间树节点，或代码与 `userRouteAuth` 不符 |
| 每个 API 都导向登录 | 拦截器：检查 `neutralize`，扩展巢状字段遍历 |
| 无 WS frame | 订阅需互动；增加 `perRouteMs` |
| 响应本文为空 | 真实响应需 `forward: true` |
| 缺 Chromium | 安装 Chromium 或设 `config.chromium`／`CHROMIUM` |
| Mock 仍回登录 | Hook 太晚或缺 `location.href` setter；在文件开始注入 |
| 全部清单空白 | L3 空阵列正常；继续分页／配置／详情 |
| Redux action 误当路由 | 筛除含 get／set／change／clear／toggle／upload 的内部路径 |
| Vue 仍导向登录 | 文件开始注入 preload；若 `neutralizeVueRouter: false`，手动清除客户端守卫 |
| 记录缺响应 URL | 启用 `extractUrlsFromResponse` 或从 `__API_RECON_DETAIL__` 捕获 |
| 不知授权头部名 | 启用 `observe.xhrHeaders` 或检查 DevTools 请求头 |
| 运行缓慢／逾时 | 用 `waitUntil: domcontentloaded`、降低 `routeTimeout`、避免 `networkidle2` |
| 代理连接失败 | 检查 `proxy`／环境，对齐 Puppeteer 与 curl 代理连接埠 |

---

<a id="g-hardened-targets"></a>

## G. 强化目标

服务器在整个流程验证会话（无法模拟的签章 cookie、无法 stub 的服务器选单）时，运行时止于外壳。预期行为：

- 静态分析仍可枚举端点，因模组路径仍在代码。
- 上游参考亦描述使用明确获授权真实会话搭配同一测试架构（`forward: true`，不 neutralize）捕获实际方法／参数／响应。这超出本技能无凭证默认范围，不能凌驾禁止索取凭证或真实登录的规则。

---

<a id="h-per-task-checklist"></a>

## H. 各任务检查清单

1. 确认授权范围。
2. 阅读 `scripts/harvest_static.py`、调整、运行并检查 `api_static.txt`、`routes.txt`。
3. 阶段 1b：路径锚点视窗 + 绑定层 → `param_candidates.json`（J）。
4. 重建 A1／A2／A3 → 网站专用 `config.json`。
5. 运行前阅读并调整 `runtime_harvest.js`／`preload.js`。
6. `runtimeMode=depth`：`npm install` → 运行调整后 harvest。
7. `runtimeMode=coverage/both`：文件开始注入调整后 preload → browser MCP 枚举 + 参数触发矩阵。
8. 模组未渲染 → 权限树重建（I）→ 修补 stub → 重跑。
9. 多参数样本差异 + 错误推断 → `params_merged.json`。
10. 合并 → `site_map.json` + `api_merged.txt`；准确说明复盖、缺口与脚本变更。

---

<a id="i-permission-tree-reconstruction-phase-4-detail"></a>

## I. 权限树重建（阶段 4 详情）

简单 `menus: [{ path, show: true }]` mock 无法挂载子模组时使用。

<a id="i1-locate-auth-modules"></a>

### I1. 寻找验证模组



```bash
grep -l 'userRouteAuth' js/*.js
grep -l 'routeMap\|routeLink' js/*.js
grep -rhoaE 'getResultTree|role_permissions|permissions/all' js | head
```



记录权限 API 路径、响应字段名与使用它的 chunk 档名。

<a id="i2-extract-routemap"></a>

### I2. 捕获 routeMap



```bash
python3 scripts/extract_route_map.py recon/js recon/
# Output recon/route_map.json
```



出现 `[!] no routeMap pattern found` 时，扩大 `extract_route_map.py` regex 或以 grep 检查：



```bash
grep -rhoaE '([A-Z_][A-Z0-9_]*):\{name:"[^"]*",link:"/[^"]+"\}' js | head -20
```



<a id="i3-build-permission-tree-and-stubs"></a>

### I3. 创建权限树与 stub



```bash
python3 scripts/build_perm_tree.py recon/js recon/ --config recon/config.json
```



脚本逻辑：
1. 剖析 `userRouteAuth={MONITOR:{url:...},...}`，含 `He=o.DASHBOARD` 等 webpack 别名。
2. 通过 `route_map.json` 将别名解析为真实路径。
3. 从代码前缀推断父节点（`MONITOR_ALERT` → `MONITOR`）。
4. 输出 `permissions_tree.json`、`permissions_all_stub.json`、`role_permissions_stub.json`。
5. 加 `--config` 时更新 `config.json` 的 `stubs` 并扩展 `routes`。

在脚本副本顶端调整：
- `DEFAULT_ROOTS`：顶层模组代码。
- `DEFAULT_PREFIX_PARENT`：`PREFIX_` → 父节点对应。
- `DEFAULT_EXTRA_PARENT`：无前缀关系的孤立节点。

<a id="i4-verify-stub-consistency"></a>

### I4. 验证 stub 一致性



```bash
# Permission count should approximate userRouteAuth entries
wc -l recon/perm_codes_all.txt
# routes should cover all route_map links
python3 -c "import json; m=json.load(open('recon/route_map.json')); r=set(json.load(open('recon/config.json'))['routes']); print('missing', [v['link'] for v in m.values() if v['link'] not in r])"
```



<a id="i5-rerun-runtime-and-compare"></a>

### I5. 重跑运行时并比较



```bash
node recon/runtime_harvest.js recon/config.json
# Compare runtime_api.json counts before/after mocks; check module APIs under /attack, /asset, etc.
```



| 模拟前 | 成功模拟后 |
|---|---|
| 每路由相同 3–5 个启动请求 | 不同路由触发不同模组 API |
| 只有 `/api/locale/language` | 出现 `/api/web/...` 等模组端点 |
| `routes.txt` 路由个位数 | 从 route_map 取得 80–110+ 个 `routes` |

<a id="i6-if-it-still-fails"></a>

### I6. 仍失败时

- Coverage：点侧栏与分页；权限请求可能需互动。
- Stub 字段：与既有获授权 API 响应比较巢状结构。上游示例使用真实会话 curl；本无凭证技能不得取得凭证或登录。
- 额外守卫：grep `hasPermission|checkRole|func.` 等按钮检查，扩展 `role_permissions.permissions`。
- 静态回退：模组路径仍在 `api_static.txt`；运行时补上方法／本文。保留 `param_candidates.json` 与捕获样本。

---

<a id="j-parameter-reconstruction-phase-1b--5b--5c"></a>

## J. 参数重建（阶段 1b／5b／5c）

这是方法，不是通用脚本。路径用 regex；参数用锚点视窗、界面绑定、样本差异与错误推断。

<a id="j1-anchor-windows-path-to-payload-builder"></a>

### J1. 锚点视窗：路径至 payload 组装器



```bash
# Anchor on known Phase 1 paths
grep -n '"/api/user/list"' js/*.js
grep -rhoaE '.{0,120}("/api[^"]+").{0,200}' js | head
grep -rhoaE '(params|data|body|payload)\s*:\s*\{' js | head
grep -rhoaE '(get|post|put|delete|patch)\([^,]+,\s*\{' js | head
```



<a id="j2-wrappers-and-transport-forms"></a>

### J2. 包装层与传输形式



```bash
# axios / shared request wrapper
grep -rhoaE '(axios|request)\.(get|post|put|delete|patch)\(' js | head
grep -rhoaE 'interceptors\.(request|response)' js | head

# GraphQL
grep -rhoaE '(query|mutation)\s+\w+|gql`|graphql\(' js | head
grep -rhoaE '\$[a-zA-Z_]+\s*:\s*(Int|String|Boolean|\[)' js | head

# FormData / multipart
grep -rhoaE 'FormData|\.append\(' js | head

# Path parameters
grep -rhoaE 'path:\s*"/[^"]*:[^"]+"' js | head
grep -rhoaE 'useParams|route\.params|\$route\.params' js | head
```



<a id="j3-validation-required-fields-formats-enums"></a>

### J3. 验证：必填字段、格式、枚举



```bash
grep -rhoaE '(required|message|pattern|enum|validator)\s*:' js | head
grep -rhoaE 'yup\.|zod\.|async-validator|Form\.Item|a-form-item|el-form-item' js | head
grep -rhoaE 'rules\s*:\s*\[|name:\s*["\'][a-zA-Z_]+["\']' js | head
grep -rhoaE 'label.*value|options\s*:\s*\[' js | head
```



<a id="j4-binding-layer-form-to-api"></a>

### J4. 绑定层：表单至 API



```bash
grep -rhoaE 'onFinish|handleSubmit|getFieldsValue|validateFields' js | head
grep -rhoaE '(pick|omit|transform|dayjs|moment)\(' js | head
```



运行时证据：DevTools → Network → request → Initiator（呼叫堆叠）；从 `fetch`／`send` 向上追至 payload 组装。

<a id="j5-encrypted-parameters"></a>

### J5. 加密参数



```bash
grep -rhoaE 'encrypt|decrypt|sign|CryptoJS|sm2|sm3|sm4|RSA|AES' js | head
```



绝不从密文猜字段。Hook 加密函数输入、记录加密前明文 payload，并将结论保存至 `config.json`／`param_candidates.json`。

<a id="j6-parameter-trigger-matrix-required-in-phase-3"></a>

### J6. 参数触发矩阵（阶段 3 必要）

依模组记录各操作，比较请求本文／查询差异：

| 操作 | 检查 |
|---|---|
| 初始清单 | 分页默认值 |
| 搜寻 | keyword、filters |
| 进阶筛选 | 选填字段 |
| 新增／编辑 | 完整实体 |
| 批次／导出 | `ids[]`、`exportType` |
| 排序／翻页 | `sortField`、`order` |

输出 `param_samples.json`：`[{ "path", "method", "action": "search", "body", "query", "headers" }]`

<a id="j7-confidence-rules"></a>

### J7. 信心规则

| 信心 | 条件 |
|---|---|
| 高 | 静态呼叫点与至少两个相符运行样本 |
| 中 | 仅静态或单一运行样本 |
| 低 | 从响应／错误推断，未二次验证 |
| 待触发 | 已知静态字段，但界面／权限路径未到达 |

<a id="j8-scenario-recipes"></a>

### J8. 情境配方

| 情境 | 顺序 |
|---|---|
| REST 清单页 | J1 payload 对象 → J6 四次差异 → J3 规则 |
| 新增／编辑表单 | J3 表单名称 → J4 提交链 → 获授权、非破坏性运行提交与空字段 400 检查 |
| GraphQL | J2 变量声明 → 捕获各运行操作的 variables |
| 加密本文 | J5 hook 输入 → 加密前字段即实际参数 |

<a id="j9-mapping-to-api-recon-phases"></a>

### J9. 与 api-recon 阶段对应

| api-recon | 参数侦察 |
|---|---|
| 阶段 1 静态 | J1 锚点视窗 |
| 阶段 2 A2 拦截器 | 全域注入字段（tenantId、sign） |
| 阶段 3 运行时 | J6 触发矩阵 + `param_samples.json` |
| 阶段 4 权限树 | 模组表单不同；足够模拟权限才能显示全部字段 |
| 阶段 5 合并 | `params_merged.json` + 信心；不可由单一样本判断必填 |

<a id="j10-troubleshooting"></a>

### J10. 疑难排解

| 症状 | 处理 |
|---|---|
| 静态字段从未在运行时出现 | 标记待触发；补全权限树、进阶筛选、连动选单 |
| 相同路径、不同本文形状 | 正常；依 `action` 分别记录，勿强制合并结构 |
| Mock 响应下仍需参数 | 检查送出请求本文／头部，勿从 stub 响应推断 |
| 400 指出巢状字段 | 检查外层 `data`／`bizData`／`variables` |
| 只见 GraphQL 操作名称 | 展开 `variables` JSON；静态寻找 `$var: Type` |

---
