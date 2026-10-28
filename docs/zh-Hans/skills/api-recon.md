[English](../../../skills/api-recon/SKILL.md) · [한국어](../../ko/skills/api-recon.md) · 简体中文 · [繁體中文](../../zh-Hant/skills/api-recon.md) · [日本語](../../ja/skills/api-recon.md) · [Español](../../es/skills/api-recon.md)

---
name: api-recon
description: 在获授权范围内盘点网站 API 界面。
---

<a id="api-recon-frontend-interface-discovery"></a>

# API 侦察（前端界面探索）

各语言版本请见本页上方导览。

在获得授权的范围内，尽可能完整探索后端 API（路径、方法、参数、响应本文）、前端路由与界面触发点（分页、对话框、表格操作）。

---

<a id="boundaries-and-prohibitions-required-reading"></a>

## 边界与禁止事项（必读）

本技能仅盘点 API 与参数，不进行漏洞探索或利用。

<a id="task-boundaries"></a>

### 任务边界

| 范围 | 允许 | 禁止 |
|---|---|---|
| 目标 | 枚举路径、方法、参数、路由与界面触发点 | SQLi／XSS、存取控制攻击、暴力破解、漏洞模糊测试、窜改请求攻击、破坏性操作 |
| 验证 | Hook 并以 stub／mock 模拟客户端登录守卫 | 索取或猜测凭证；提交真实登录表单 |
| 运行时 | 无凭证 hook；以 mock 呈现登录后 SPA 外壳 | 需要真实后端会话的流程 |

<a id="dynamic-analysis-without-credentials-phase-3-default"></a>

### 无凭证动态分析（阶段 3 默认）

1. 以 `preload.js`／`runtime_harvest.js` 拦截并 stub 登录、权限、选单与其他启动 API。
2. 传回形状与成功代码正确的模拟查询本文，数据可为空。
3. 无后端或遇 401 时仍呈现登录后页面，以触发更多 XHR／fetch／WebSocket 请求。
4. 空数据、空表格与预留界面属预期，勿转为真实登录或漏洞测试。

Mock 仅用来挂载前端路由／元件并记录送出请求。目标是探索前端还会传送什么，不是取得真实后端数据。

<a id="workflow-prohibitions"></a>

### 工作流禁止事项

| 禁止 | 必须采用的替代方式 |
|---|---|
| 阶段 1 完成前，grep／curl／Read 主 `index-*.js` 捕获 API 路径 | 运行 `OUTDIR/harvest_static.py` |
| 手写 `extract_apis.py` 等替代程序 | 调整并重跑 `OUTDIR/harvest_static.py` |
| 至少失败两次后重复相同 grep／命令 | 改策略：读 tool_logs、调整 harvest、查参考 |
| 略过关卡 A／B，直接运行原始 `scripts/` 模板 | 复制至 OUTDIR 并依目标调整 |
| 真实凭证、OTP 或 OAuth 验证 | 依上文使用 stub／mock |
| 略过 stub，以存取控制／注入测试取得真实数据 | 仅在侦察范围记录送出请求 |
| 删除、导出敏感数据、批次写入或其他不可逆操作 | 覆盖率点选亦受此限制 |
| 无运行时与动态枚举却宣称全部页面／API | 满足完成定义或说明限制 |
| 无触发矩阵与差异却宣称全部参数 | 完成阶段 3b 矩阵 + 阶段 5 差异 |
| 由单一运行样本推断必填／选填 | 比较样本或检查验证规则／错误 |

---

<a id="two-layers-and-runtime-modes"></a>

## 两层分析与运行模式

| 层级 | 输出 | 限制 |
|---|---|---|
| 静态（JS bundle） | 端点路径、路由草稿、payload 组装处的候选字段 | 无 HTTP 方法；参数需阶段 1b；漏掉运行时组合 URL |
| 运行时（实际会话） | 方法、本文、响应、动态 URL、WS／SSE；样本差异补足参数 | 页面必须渲染才发出请求；单样本无法确定必填／选填 |

| 模式 | 引擎 | 用途 |
|---|---|---|
| **depth** | `runtime_harvest.js`（Puppeteer） | API 盘点、方法／参数／响应、WS／SSE、可重复批次运行 |
| **coverage** | 浏览器 + `preload.js` | 点选分页／对话框／表格，深入功能覆盖 |
| **both** | 先 depth 再 coverage | 最完整，也最耗时 |

参数方法（无通用脚本）：路径用 harvest／regex；参数用锚点视窗扩展、界面绑定链、样本差异与错误推断。[reference.md](api-recon-reference.md) J 节提供 grep 配方。

---

<a id="definition-of-completion"></a>

## 完成定义

全部条件满足后才能宣称完成：

- [ ] 静态：阶段 1 产生 `api_static.txt`、`routes.txt`、`js/`。
- [ ] 运行时：至少 depth 或 coverage；coverage／both 必须有可用 hook 与动态枚举循环。
- [ ] 外壳：业务路径保持在 `/login` 以外（检查 hash 路由）。
- [ ] 参数：coverage／both 完成触发矩阵与 `param_samples.json`；阶段 5 合并 `params_merged.json`。
- [ ] depth 模组空白时：阶段 4 恢复权限树并重跑，直到出现 locale／启动 API 以外的模组 API。
- [ ] 交付：全部阶段 5 输出；`insert_assets` 保存服务与端点资产。

---

<a id="scripts-and-gates"></a>

## 脚本与关卡

`scripts/` 内为参考模板，绝不可未调整便运行并当作最终结果。

先阅读、依目标调整，写入 `OUTDIR`（如 `recon/`），并记录 `CHANGES.md`。不适用时以相同方法重写，仅借用结构。

| 关卡 | 时机 | 模板 → OUTDIR 副本 | 常见调整 |
|---|---|---|---|
| A（静态） | 阶段 0 后，首次 harvest／spider 前 | `harvest_static.py` / `spider_mpa.py` | 默认 regex 适用多数网站；仅 manifest／语法不符时改端点 regex、webpack／Vite `publicPath`、MPA 排除／cookie |
| B（运行时） | 阶段 2 后，depth／coverage 前 | `runtime_harvest.js` / `preload.js` + `config.json` | Cookie／localStorage 键、中和成功值、stub、登录 regex、API 前缀、hash／history |

SPA 必须依此顺序；阶段编号优先于撰写脚本前的探索浏览，不得重排：

| 步骤 | 必须 | 禁止 |
|---|---|---|
| 阶段 0 后 | 下一个 Bash 命令 = `python3 OUTDIR/harvest_static.py <URL> OUTDIR` | curl／grep／Read 主 `index-*.js`（通常 >500KB） |
| 关卡 A | 复制脚本、必要小幅修改、立即运行 | 决定是否 harvest 前手动捕获 API |
| 阶段 1 完成前 | `wc -l` 检查输出，404 时调整 harvest 并重试 | 撰写捕获程序；反复 grep 未下载 URL |
| 阶段 1b 起 | 仅 grep `OUTDIR/js/*.js` | 以主 bundle 取代 harvest |

- 正确：复制 `harvest_static.py`、视需要调整 regex、立即运行。
- 错误：curl 主 bundle、反复 grep、暂时捕获程序、最后才 harvest。
- MPA：阶段 0 后下一个 Bash 命令 = `python3 OUTDIR/spider_mpa.py ...`。

---

<a id="tool-and-output-limits"></a>

## 工具与输出限制

| 限制 | 规则 |
|---|---|
| 大型文件 | 不得将 >100KB 的 `index-*.js` Read／grep 至上下文；以 OUTDIR 脚本批次处理 |
| grep 输出 | 必须加 `\| head -20` 或 `-m 5`；保留路径摘要，不贴 bundle 片段 |
| 验证 | 使用 `wc -l`、`ls \| wc -l`；勿 Read 整个目录 |
| Regex 预览 | 选用，最多一次，仅限 ≤50KB chunk／HTML；harvest 为静态事实来源 |
| 参考 | [reference.md](api-recon-reference.md) 提供配方／模板／排错，勿在行内重复 |

---

<a id="execution-roadmap"></a>

## 运行路线



```
Phase 0 classification + OUTDIR
  → Gate A → Phase 1 harvest (run immediately)
  → Phase 1b parameter reconstruction
  → Phase 2 three authentication gates → config.json
  → Gate B → Phase 3 runtime + parameter matrix
  → Phase 4 permission tree if needed → rerun Phase 3
  → Phase 5 merge report + insert_assets for every discovered service/endpoint asset; omit none
```



依序检查，前一步完成前不得前进。

1. [ ] **阶段 0**：分类 SPA／MPA，创建 `OUTDIR` → [阶段 0](#phase-0-classification)
2. [ ] **关卡 A + 阶段 1**：复制脚本、立即 harvest、`wc -l` 验证 → [阶段 1](#phase-1-static-discovery)
3. [ ] **阶段 1b**：锚点视窗 + 绑定层 → `param_candidates.json` → [阶段 1b](#phase-1b-parameter-reconstruction)
4. [ ] **阶段 2**：三个验证关卡 → `config.json` → [阶段 2](#phase-2-three-authentication-gates)
5. [ ] **关卡 B**：调整运行时脚本 → [阶段 3](#phase-3-runtime)
6. [ ] **阶段 3**：depth／coverage／both；验证进入外壳；参数触发矩阵 → `param_samples.json`。
7. [ ] 必要时**阶段 4**：权限树 → 修补 stub → 重跑阶段 3 → [阶段 4](#phase-4-permission-tree-reconstruction)
8. [ ] **阶段 5**：合并输出 + 报告 + `insert_assets` → [阶段 5](#phase-5-merge-and-report)

---

<a id="phase-0-classification"></a>

## 阶段 0：分类

取得入口 HTML 并创建 `OUTDIR`；勿修改技能的 `scripts/`：

- **SPA**：空外壳 + `<div id=app>` + chunk → 阶段 1–5。
- **MPA**：SSR + `<form>`，无端点 bundle；关卡 A 后：



```bash
python3 recon/spider_mpa.py <BASE_URL> <OUTDIR> [--cookie "session=..."] [--max 300] [--depth 5] [--exclude "logout|delete|destroy"]
```



输出：`forms.txt`、`links.txt`、`api_inline.txt`。SPA 表单约为零时，转阶段 1。

---

<a id="phase-1-static-discovery"></a>

## 阶段 1：静态探索

遵循[脚本与关卡](#scripts-and-gates)及[工具／输出限制](#tool-and-output-limits)。



```bash
python3 recon/harvest_static.py <BASE_URL> <OUTDIR>
```



Harvest 剖析 HTML script 与 webpack／Vite manifest，下载全部延迟 chunk，输出 `js/`、`api_static.txt`、`routes.txt`、`chunkmap.txt`。



```bash
wc -l OUTDIR/api_static.txt OUTDIR/routes.txt
ls OUTDIR/js | wc -l
```



- 比较 chunk 数与 manifest；404 时修正 harvest 并重试，勿逐个 curl。
- `api_static.txt` 太少：扩大 OUTDIR 端点 regex 并重跑，详见参考。

<a id="phase-1b-parameter-reconstruction"></a>

### 阶段 1b：参数重建

路径来自阶段 1；参数字段另行调查，遵守 [grep 输出限制](#tool-and-output-limits)。

完成标准：重要 API 须识别字段名、传输位置、推断类型、必填状态、示例值与信心程度。

<a id="1b0-transport-forms"></a>

#### 1b.0：传输形式

| 形式 | 参数位置 | 静态证据 |
|---|---|---|
| REST JSON | body + query | 路径锚点附近 `(params\|data\|body)\s*:\s*\{` |
| GraphQL | `variables` | gql 模板、`$page: Int` |
| 传统表单 | urlencoded | `<form>`、`FormData` |
| 文件上传 | multipart | `FormData.append` |
| 路径参数 | `/user/:id` | 路由表 + `useParams` / `$route.params` |
| 加密／签章 | 封装于 `sign`／`data` | Hook 加密函数输入（参考 D） |

输出：各 API 标注 `transport: query|json|form|graphql|encrypted`。

<a id="1b1-expand-anchor-windows"></a>

#### 1b.1：扩展锚点视窗

在已知路径周围扩展，寻找 payload 组装：



```bash
grep -n '"/api/user/list"' OUTDIR/js/*.js | head -20
grep -rhoaE '.{0,120}("/api[^"]+").{0,200}' OUTDIR/js/*.js | head -20
grep -rhoaE '(params|data|body|payload)\s*:\s*\{' OUTDIR/js/*.js | head -20
```



| 包装层 | 参数线索 |
|---|---|
| axios 运行个体 | `data` / `params` |
| 共用请求包装层 | 拦截器注入的全域字段 |
| OpenAPI 客户端 | 生成的方法签章 |
| React Query / SWR | 第二个 hook 参数 |
| Vue composable | composable 参数 |

类型线索：`yup`／`zod`／rules、`Form.Item name=`、内嵌 Swagger。

→ `param_candidates.json`：`{ path, fields[], source: "static-callsite", confidence }`

<a id="1b2-binding-layer"></a>

#### 1b.2：绑定层



```
Form field → onFinish/handleSubmit → transform → API payload
```



| 绑定来源 | 方法 |
|---|---|
| 表单提交 | 追踪 submit → transform → API |
| 表格搜寻 | `getFieldsValue()` → `params` |
| 路由 | `:id` / `?tab=` |
| 拦截器 | 全域 `tenantId`、分页、sign |
| 枚举选单 | `options` → API 枚举值 |

从 `fetch`／`XHR.send` 向上追踪 DevTools 呼叫堆叠至 payload 组装器。

<a id="1b3-three-payload-questions-distinct-from-phase-2-auth-gates"></a>

#### 1b.3：三个 payload 问题（不同于阶段 2 验证关卡）

| 问题 | 必须回答 |
|---|---|
| 组装 | payload 在何处创建与转换 |
| 验证 | required、pattern、enum |
| 传输 | path / query / body / multipart / headers |

拦截器关卡（阶段 2）亦需检查全域注入字段（Authorization、`X-Tenant-Id`、sign）。

<a id="1b4-connect-to-phase-3"></a>

#### 1b.4：衔接阶段 3

候选字段来自静态／绑定分析。必填、选填与条件依赖需阶段 3 矩阵／差异及阶段 5 错误推断。

---

<a id="phase-2-three-authentication-gates"></a>

## 阶段 2：三个验证关卡

以 `head` 限制 grep `OUTDIR/js/`，将结果记入 `config.json`，见参考配方：

| 关卡 | 问题 | 关键字 |
|---|---|---|
| 渲染 | 如何判定登录状态？ | `isLogin`、`getToken`、Cookie／localStorage |
| 拦截器 | 什么触发 `/login` 重新导向？ | `response_code`、`errno`、axios interceptor |
| 内容 | 选单／权限从哪里来？ | `menu`、`permission`、`role`、`acl`、`routes` |

勿将 localStorage 键名当作凭证；须通过 chunk／请求链验证。

出口为关卡 B：结论写入 `config.json`，调整 `OUTDIR/runtime_harvest.js`／`preload.js`。

<a id="phase-2b-api-observation-optional"></a>

### 阶段 2b：API 观察（选用）

以 OUTDIR 的 `preload.js` 识别会话键、Authorization 与巢状 API URL：

| 配置 | 输出 |
|---|---|
| `recordDetail: true` | `__API_RECON_DETAIL__` |
| `observe.xhrHeaders: true` | 头部观察 |
| `extractUrlsFromResponse: true` | 响应中的巢状 API |
| `observe.storageReads/cookieReads: true` | 更新配置 |
| `neutralizeVueRouter: true` | `__API_RECON_ROUTES__` |

导出每轮 coverage：`__API_RECON_LOG__`、`__API_RECON_DETAIL__`、`__API_RECON_ROUTES__`、`__API_RECON_OBSERVE__`。

---

<a id="phase-3-runtime"></a>

## 阶段 3：运行时

先通过关卡 B，遵守[边界](#boundaries-and-prohibitions-required-reading)与无凭证 mock 策略。

配置 `config.json` 的 `"runtimeMode": "depth" | "coverage" | "both"`；模板见参考。

<a id="hooks-and-stubs-shared-by-depth-and-coverage"></a>

### Hook 与 stub（depth／coverage 共用）

| 层级 | 范围 | 目的 |
|---|---|---|
| L1 精确 | auth／permission／bootstrap stub | 通过初始渲染验证 |
| L2 失败结果修正 | 全部 JSON 响应 | 未登录代码 → 成功 |
| L3 回退 | L1 未比对的 `/api` 等 | 空成功本文以呈现界面 |

- **depth**：假验证 + `forward` 业务代码修补 + `stubs`；遍历 `routes`（hash／history），输出 `runtime_api.json`。
- **coverage**：文件开始时注入 `preload.js`（CDP `addScriptToEvaluateOnNewDocument` 或 userscript）。

验证 `window.__API_RECON_PRELOAD__` 存在，业务路径不会回到 `/login`。



```bash
cd recon && npm install
node runtime_harvest.js config.json
```



<a id="3b-coverage-dynamic-enumeration-required"></a>

### 3b：coverage 动态枚举（必要）

1. 主导览／侧栏：逐项点选，等待网络 1–3 秒。
2. 分页：`role=tab`、`.ant-tabs-tab`。
3. 表格：第一列查看／编辑／详情。
4. 工具列：导出、筛选、新增（避免不可逆删除；全部禁令仍适用）。
5. 各模组：合并 API／路由。
6. SPA：对 `routes.txt` 未覆盖路径受控使用 `pushState`；MPA 禁止。

必要参数触发矩阵：依模组记录各操作类型并比较多个样本：

| 操作 | 常见新增参数 |
|---|---|
| 初始清单 | 分页 + 默认筛选 |
| 搜寻 | keyword、filter |
| 进阶筛选 | 更多选填字段 |
| 新增／编辑 | 完整实体 |
| 批次／导出／排序 | `ids[]`、`exportType`、`sortField` |

Stub 下送出本文／头部仍为真实值，应以请求作证据。记录 `scan_raw.json`、`param_samples.json`、`api_detail.json`。

- **Vue**：`neutralizeVueRouter: true` + 文件开始 preload。
- **React**：`routes.txt` + 侧栏点选 + `pushState`。
- **both**：先 3a depth，再 3b coverage。

---

<a id="phase-4-permission-tree-reconstruction"></a>

## 阶段 4：权限树重建

触发条件：模组页面空白，或所有路由只有 locale 等启动请求，表示内容关卡未通过。

| 症状 | 含义 |
|---|---|
| 已进入外壳 | 渲染与拦截器关卡通过 |
| 侧栏缺项／点选空白 | Stub 形状或权限代码不完整 |
| 各路由仅少量相同 API | `v-if permission` 失败 |
| `routes.txt` 远少于 bundle 路由 | 从验证模组补全 |



```bash
grep -rhoaE '"/api[^"]*(permission|perm|role|menu|acl)[^"]*"' OUTDIR/js/*.js | sort -u | head -30
grep -rhoaE 'userRouteAuth|getResultTree|routeMap|routeLink|menuList|authList' OUTDIR/js/*.js | head -20
```



典型链：`role_permissions`（平面代码）+ `permissions/all`（树）→ `getResultTree` → `userRouteAuth[CODE].url`。



```bash
python3 recon/extract_route_map.py recon/js recon/
python3 recon/build_perm_tree.py recon/js recon/ --config recon/config.json
```



中间输出：`route_map.json`、`userRouteAuth.json`、`permissions_tree.json`、`*_stub.json`、`perm_codes_all.txt`。

检查 stub：外层 `response_code` 符合拦截器关卡；平面代码符合树；`routes` 涵盖每个 `route_map` 链接。

更新 `config.json` 后重跑阶段 3。大型 SPA 可调整 `waitUntil`、`routeTimeout`、`perRouteMs`（参考 A3／I）。

---

<a id="phase-5-merge-and-report"></a>

## 阶段 5：合并与报告

<a id="outputs"></a>

### 输出

| 文件 | 阶段 | 内容 |
|---|---|---|
| `js/`, `api_static.txt`, `routes.txt`, `chunkmap.txt` | 1 | 静态 bundle 与路径 |
| `param_candidates.json` | 1b | 静态参数候选 |
| `config.json` | 2 | 三关卡 + 运行时配置 |
| `runtime_api.json` | 3a | depth 详细捕获，含 WS／SSE |
| `param_samples.json`, `scan_raw.json`, `api_detail.json` | 3b | 多样本、点选记录、详情 |
| `route_map.json` 等 | 4 | 使用时的权限树中间档 |
| `params_merged.json` | 5 | 合并参数字段 + 信心程度 |
| `api_merged.txt` | 5 | `METHOD /path [params] [static\|runtime\|both]` |
| `site_map.json` | 5 | 路由、API、参数、功能、限制 |
| **insert_assets** | 5 | 保存全部服务与端点资产 |

<a id="5b-merge-parameters"></a>

### 5b：合并参数

比较 `param_samples.json`；没有通用合并脚本。信心规则见参考 J7（高／中／低／待触发）。

<a id="5c-infer-from-errors"></a>

### 5c：从错误推断

在授权范围内，可用不完整请求检查 400 错误来探索参数，不作漏洞测试，例如 `field 'x' is required`、枚举错误等。须考量 `data` 包装、`variables` 与加密前 `bizData`。

报告 runtimeMode、静态／运行时 API 数、参数信心、未覆盖模组，以及相对模板的 `CHANGES.md` 摘要。

建议的 `site_map.json`：



```json
{
  "site": "https://example.com",
  "runtimeMode": "both",
  "appType": "vue-spa",
  "routeGuardStrategy": ["nav-neutralize", "L1-auth", "L2-patch", "forward"],
  "apisFromStatic": [],
  "apisFromRuntime": [],
  "apis": [],
  "params": [{ "method": "POST", "path": "/api/user/list", "transport": "json", "fields": [] }],
  "frontendRoutes": [],
  "routesVerifiedByClick": [],
  "featuresTriggered": [],
  "limitations": ""
}
```



更多字段与 grep 配方见 [reference.md](api-recon-reference.md)。

---

<a id="general-notes"></a>

## 一般注意事项

- 框架无关：webpack／Vite／Angular 延迟加载采相同方法。
- 传输：REST／JSON、GraphQL、WebSocket、SSE；不含 gRPC-web。
- SSR：可捕获客户端 fetch；无法完整枚举 RSC／Server Actions。
- 盲点：JSVMP、WASM、严格 HMAC／mTLS → 静态分析并明示限制。
- 参数盲点：条件互动、隐藏参数、WASM 组装 → 待触发／无法到达。
- 静态回退：运行时受阻时仍可探索端点。

---

<a id="additional-resources"></a>

## 其他资源

- Grep 配方、`config.json` 模板、排错、hook、参数重建（J）、site_map 模板：**[reference.md](api-recon-reference.md)**。
- 模板路径：[脚本与关卡](#scripts-and-gates)。
