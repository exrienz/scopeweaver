# OpenCode SDK 桥接器

[English](README.md) · [한국어](README.ko.md) · [简体中文](README.zh-Hans.md) · [繁體中文](README.zh-Hant.md) · [日本語](README.ja.md) · [Español](README.es.md)

可选的仅限回环地址的桥接器，支持固定的 **OpenCode 原生 CLI／客户端／SDK 2.1.4**，以及单独测试的 **旧版 SDK／CLI 1.19.4**。ScopeWeaver 继续管理对话、工具执行、审批、范围检查、重试与压缩。桥接器只流式传送一次模型步骤并收集工具请求，不执行 ScopeWeaver 工具或 OpenCode 内置工具。现有 API 与 OpenCode Go 供应商无需此桥接器。

## 启动

已测试的 CI 运行时是 **Bun 1.3.14**；更新版本尚未验证。

```sh
cd adapters/opencode-sdk
bun install --frozen-lockfile --ignore-scripts
export SCOPEWEAVER_OPENCODE_TOKEN="$(openssl rand -hex 32)"
bun run start
```

桥接令牌必须至少包含 **32 个非空白字符**。原生私有 CLI 服务器使用每次调用独立生成的密码（`OPENCODE_PASSWORD`）和 HTTP Basic 认证；该密码、桥接令牌与上游供应商密钥彼此独立。

在 ScopeWeaver 供应商配置中选择 `opencode-sdk`，地址设为 `http://127.0.0.1:4097`，填入上述**桥接令牌**。它不是云端 API 密钥；不要复用 OpenCode Go／Console 令牌。读取模型后选择准确的 `providerID/modelID`。

默认运行时不登录账户，使用隔离的目录与公开模型发现。发现只发送公开目录 GET 请求，**不执行推理**；仅显示当前公开列表中、固定运行时识别的零价格 OpenCode 模型。目录错误会明确返回。列入目录不代表具备资格、可用配额或推理成功；免费模型可能改变或消失。默认不启用付费模型或凭证回退。

**2026-10-25（KST）的先前验证：** `opencode/deepseek-v4.1-flash-free` 曾通过产品的原生 CLI／客户端 2.1.4 桥接器完成真实 ScopeWeaver 对话。三次实际推理覆盖界面连接测试、文件产生影响前经审核批准的宿主工具请求，以及界面最终答案 `FREE_OK`。模型来自默认零价格目录，未使用 API 密钥、账户导入、模型／响应模拟数据、身份伪装或付费回退。原生工具保持阻止状态，宿主操作由 ScopeWeaver 执行。这是该日期的既有证据，不是本次重跑结果。

原生推理使用真正的固定版本 CLI 和官方客户端 API，无推理的模型发现使用内嵌 SDK。此前仅收集工具的内嵌 SDK 路径收到 HTTP 403 `FreeTierError`。旧版的真实托管免费访问尚未验证。模型及服务资格可能变化；该日期的测试不保证数值配额或持续可用性。结构化拒绝返回 `free_tier_restricted`，不切换到付费模型。

`SCOPEWEAVER_OPENCODE_PORT` 可修改默认端口 4097；`SCOPEWEAVER_OPENCODE_HOST` 必须为 `127.0.0.1`。拒绝浏览器 `Origin` 请求及非回环 URL。所有 API 路由均要求令牌；请像密码一样保护它。

`SCOPEWEAVER_OPENCODE_TIMEOUT_MS` 的完整调用时限为 **0–86,400,000 毫秒**；默认 **0** 由宿主控制取消，没有隐藏的两分钟推理限制。取消与关闭另有 **1.5 秒**强制终止期限。公开目录请求限制为 **10 秒**。请求体上限 **16 MiB**，单个工具参数 JSON 上限 **2,000,000 字符**，原生模型文本／思考／参数合计上限 **8,000,000 字符**，工作进程输出上限 **20,000,000 字节**。超限明确失败，不截断数据。

## 专用供应商配置

自定义 OpenAI 兼容模型或本地测试模型可通过 `SCOPEWEAVER_OPENCODE_PROVIDER_CONFIG` 指定运营者管理的 JSON 文件。放在检出目录之外，或使用 Git 忽略的名称 `provider-config.local.json`：

```json
{
  "providers": {
    "local": {
      "name": "Local model",
      "package": "@opencode/ai/providers/openai-compatible",
      "settings": {
        "baseURL": "http://127.0.0.1:8000/v1",
        "apiKey": "explicit-provider-key-if-needed"
      },
      "models": {
        "my-model": {
          "name": "My model",
          "capabilities": { "tools": true },
          "limit": { "context": 200000, "output": 32000 },
          "compatibility": { "maxTokensField": "max_tokens" }
        }
      }
    }
  }
}
```

选择 `local/my-model`。文件会替代该桥接器的默认模型发现，不载入用户／项目的 OpenCode 配置、`.env`、插件、MCP 服务器或已保存凭证。仅将明确配置的 `apiKey` 传给对应供应商。远程地址可能收取推理费用；添加配置不会使其免费。任意 SDK 软件包和额外配置字段均被拒绝。

## 运行边界

每次调用使用独立的 Bun 工作进程、允许列表中的环境变量及自有临时 home／配置／数据路径。原生模式拥有私有官方 CLI 服务器、实际 CLI `run`、临时数据库和一次性认证的回环出口传输服务；官方客户端处理发现、会话事件和中断。不继承云端凭证、用户认证文件、代理或预加载设置。直接使用固定的可选平台二进制，因此支持 `--ignore-scripts`，无需全局安装；**不要省略可选依赖**。

保留 OpenCode 的真实内置工具定义及原始实现。所有原生权限为 ASK，CLI 不使用 `--auto`。公开的 `execute.before` 策略钩子独立阻止所有非宿主工具在执行前运行，包括生成的 `execute`。原生工具请求使桥接请求失败，不会变成 ScopeWeaver 工具请求。只有注册的宿主收集函数可暂停并成为宿主工具请求；移除其他代理。无冲突的 `sw_tool_N` 别名防止宿主工具覆盖内置定义，描述保留原始名称与内容，响应恢复原始宿主名称。

宿主 system 片段以两个换行连接，原样写入公开的自定义代理 system 设置；同时保留 OpenCode 生成的模型／环境／工具上下文。最后的策略说明只有标记 `Host tool:` 的函数可用。通过公开 context 钩子保留宿主角色、工具 ID／结果及每次调用的限制／设置，不保证 system 数组只含宿主片段。按模型传递 thinking／effort 字段，不支持的设置明确失败；模型声明的较小输出限额会限制请求上限。输入、输出及缓存用量分开记录。

阻止第二次模型请求。私有出口将实际 CLI 生成的请求转发到原始上游 URL，拒绝重定向，不替换地址、模型、身份或凭证。只有中断、删除会话及清理自有进程完成后才返回宿主工具请求。断连、DELETE、超时和关闭终止 POSIX 工作进程组，并在 **1.5 秒**后强制终止。该进程树清理边界未在 Windows 实现，**原生和旧版推理均不支持 Windows**。

CLI 的公开设置不能完全关闭系统／MDM 配置发现。macOS 上存在 `/Library/Application Support/opencode` 或用户／全局 `/Library/Managed Preferences` 下的 `ai.opencode.managed.plist`，或 Linux 上存在 `/etc/opencode`，原生模式便以 `configuration_error` 拒绝启动。无法访问的路径和损坏符号链接也明确拒绝；不读取或修改这些来源。

原生模式在完成响应、流响应和宿主工具历史重放中保留原始工具参数 JSON，包括 64 位整数；不执行 JavaScript 解析后的投影。原始参数之外的不安全数值、不支持的内容、未知原生工具、重复工具 ID 和格式错误响应均拒绝。工具 schema 支持内联定义、本地 JSON Pointer `$ref`／`$defs` 与联合类型；拒绝远程引用、`$id`、`$dynamicRef` 和无法解析的引用。初始化仍会加载原生供应商／目录代码；这是应用边界，**不是操作系统沙箱**。回环桥接器仍可调用远程模型，因此不满足 ScopeWeaver 的仅本地推理策略。

## 版本与旧版限制

| `SCOPEWEAVER_OPENCODE_RUNTIME` | 固定组合 | 能力响应 `runtime` |
| --- | --- | --- |
| `native`（默认） | 官方 CLI、`@opencode/client`、SDK **2.1.4** | `native` |
| `legacy` | `@opencode-ai/sdk/v2`、CLI **1.19.4** | `legacy` |

启动前设置 `SCOPEWEAVER_OPENCODE_RUNTIME=legacy` 可选择旧 API。两个组合的先前本地实际运行验证范围为 **macOS arm64**；Linux CI 已配置实际运行测试，但托管运行结果尚待验证。其他 SDK 版本在推理前被拒绝，不保证任意版本或平台兼容。

旧版直接运行匹配的官方可选平台二进制，支持 `--ignore-scripts`，不调用未初始化的包装器或全局 `opencode`。不要省略可选依赖。其 CLI 与回环 MCP 收集服务器随请求工作进程的 POSIX 进程组终止；Windows 不支持此清理边界。每次调用使用独立插件软件包副本及锁文件，不建立到共享依赖的可写链接。

旧版只支持结构有效、可安全表示的 JSON／历史。空白及等价 HTML／Unicode 转义可能被规范化，不承诺原始 JSON 字节。拒绝绝对值超过 **9,007,199,254,740,991** 的不安全整数、不透明／带签名的思考历史及不支持的供应商覆盖设置；需要原始 64 位参数保真时使用原生模式。旧版自定义供应商使用上面的显式 OpenAI 兼容配置，但只映射 `max_tokens` 兼容字段、支持的工具能力字段及上下文／输出限额；拒绝不支持的 `reasoningField` 和能力覆盖。超出任一 SDK 模式边界时，可继续使用 API／Go 配置。

## 协议与验证

协议为 **`scopeweaver-opencode-sdk`，版本 1**。认证路由：`GET /v1/capabilities`、`GET /v1/models`、`POST /v1/complete`、`POST /v1/stream`、`DELETE /v1/requests/{request_id}`。流使用 NDJSON，传递真实原生文本／思考增量、收集的工具请求及最终用量；成功时只发送一次 `message_stop`。错误使用有限机器代码并隐藏供应商消息。只有明确结构化的计费／配额耗尽才返回 `quota_exhausted`（完成请求 HTTP 402）；普通限流、认证及 5xx 不会改标为配额错误。结构化上下文溢出返回 `context_length_exceeded`（HTTP 400），保留宿主压缩信号；普通 400 或消息关键词不足以分类。结构化 `FreeTierError` 返回 `free_tier_restricted`（HTTP 403），与配额及普通禁止响应分开。桥接器不选择后备模型；后续行为由宿主配置的策略决定。

完成请求示例（流端点接受相同正文）：

```json
{
  "request_id": "unique-call-uuid",
  "session_id": "stable-host-conversation-uuid",
  "model": "local/my-model",
  "system": ["Host system instructions"],
  "messages": [{ "role": "user", "content": [{ "type": "text", "text": "Hello" }] }],
  "tools": [],
  "max_tokens": 64,
  "thinking": "disabled"
}
```

可选字段为 `session_id`、`temperature`、字符串数组 `stop`、`thinking`、`thinking_type`、`reasoning_effort`。宿主内容块支持 `text`、助手 `thinking`／`tool_use` 及带文本结果的用户 `tool_result`。工具声明使用 `name`、`description`、`input_schema`。完成响应包含 `message`、`stop_reason`（`end_turn`、`tool_use` 或 `max_tokens`）及分离输入／输出／缓存读取／缓存写入计数的 `usage`。公开契约与限额见 [protocol.mjs](src/protocol.mjs) 和 [bridge.mjs](src/bridge.mjs)。

对于 `opencode-go`，`session_id` 使每次调用的新 SDK 会话保持相同 `x-opencode-session`；缺省时使用 `request_id`。`opencode` 保留真实运行时生成的会话及身份头部。普通供应商不会收到网关专用会话头部。保留真实 CLI 的 User-Agent。

```sh
bun run check
bun test --timeout 30000
```

测试以固定的实际 CLI／客户端／SDK 和本地 SSE 模型验证无推理发现、宿主历史／设置、收集安全性、流／用量、原始数值参数、恶意原生调用、并发隔离及取消；不证明真实托管免费模型的使用资格。

## ScopeWeaver 配置与降级边界

空令牌仅在相同 SDK 地址保留已保存值。更改地址或格式不能复用旧令牌；`localhost`、`127.0.0.1`、`::1` 是不同凭证作用域。SDK 模板清除未保存的云端密钥、代理和自定义会话头部，模型留空，不发现、不推理、不保存、不启用。明确点击**读取模型**才发现模型；**测试连接**仅在点击时执行推理。

配置数据库 `CHECK` 约束增加 `opencode-sdk`；仍有 SDK 配置时，旧二进制无法启动。降级前备份数据库，并在当前版本中转换为旧版本支持的格式或移除 SDK 配置。此前迁移验证只使用临时数据库，没有迁移生产数据库。详见[供应商指南](../../docs/zh-Hans/llm-providers.md#opencode-sdk-bridge)。
