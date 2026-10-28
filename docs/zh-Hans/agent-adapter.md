[English](../../adapters/agent/README.md) · [한국어](../ko/agent-adapter.md) · 简体中文 · [繁體中文](../zh-Hant/agent-adapter.md) · [日本語](../ja/agent-adapter.md) · [Español](../es/agent-adapter.md)

<a id="scopeweaver-agent-adapter"></a>

# ScopeWeaver 智能体适配器

各语言版本请见本页上方导览。

Claude Code 与 Codex 通过本地 stdio MCP 服务器连接，Pi 通过原生扩展界面连接。三者使用相同经验证工具与已验证的 ScopeWeaver HTTP API。此适配器让编程智能体控制 ScopeWeaver，不取代其规划器／工作智能体，也不会将编程智能体订阅转为模型 API 密钥。

<a id="setup"></a>

## 配置

需求：Node.js 22+、运行中的 ScopeWeaver 后端与既有管理员登录。适配器不会初始化或变更密码。安装独立适配器依赖包：



```sh
cd adapters/agent
npm ci
export SCOPEWEAVER_URL=http://127.0.0.1:8787
export SCOPEWEAVER_PASSWORD='your-existing-admin-password'
```



密码用于登录后端既有的 `ARTEX` 管理员帐号，返回 JWT 仅保留在内存。亦可将 `SCOPEWEAVER_TOKEN` 设为既有登录 JWT；提供的令牌优先。过期／无效令牌会明确报错，不会默默替换。凭证请存于环境或秘密管理工具，勿提交至客户端配置。

默认启用读取。若要创建任务与暂停／继续，启动适配器前须明确设 `SCOPEWEAVER_ALLOW_WRITES=true`。后端已配置 LLM 时，创建任务会立即排程工作。请遵守专案[本地隔离使用限制](../../README.zh-Hans.md#license-and-disclaimer)，创建任务前确认预期范围。

<a id="claude-code"></a>

### Claude Code

请使用此签出目录的绝对路径。Claude Code 会启动适配器并继承环境：



```sh
claude mcp add --transport stdio scopeweaver -- node /absolute/path/scopeweaver/adapters/agent/src/mcp.js
```



以 `claude mcp list` 验证，再于 Claude Code 用 `/mcp` 检视工具。[Claude Code MCP 文件](https://code.claude.com/docs/en/mcp)。

<a id="codex"></a>

### Codex



```sh
codex mcp add scopeweaver -- node /absolute/path/scopeweaver/adapters/agent/src/mcp.js
```



以 `codex mcp list` 与 `/mcp` 验证。若要在专案／用户配置中明确转发环境：



```toml
[mcp_servers.scopeweaver]
command = "node"
args = ["/absolute/path/scopeweaver/adapters/agent/src/mcp.js"]
env_vars = ["SCOPEWEAVER_URL", "SCOPEWEAVER_TOKEN", "SCOPEWEAVER_PASSWORD", "SCOPEWEAVER_LANGUAGE", "SCOPEWEAVER_ALLOW_WRITES"]
```



使用您已配置的凭证变量，省略未使用项目。[OpenAI 官方 MCP 文件](https://developers.openai.com/codex/mcp)。

<a id="pi"></a>

### Pi



```sh
pi -e /absolute/path/scopeweaver/adapters/agent/pi-extension.js
```



扩展功能注册原生 Pi 工具并使用相同环境变量，无须 Pi MCP 外挂。它不导入 Pi 包命名空间，因此可在 Pi 包更名时继续遵循公开 `registerTool` 界面。适配器包也声明 `pi.extensions` 供 Pi 包加载器使用。[Pi 扩展功能](https://github.com/earendil-works/pi/tree/main/packages/coding-agent)。

<a id="tools"></a>

## 工具

| 工具 | 用途 |
| --- | --- |
| `scopeweaver_health` | 后端可用性与 LLM 就绪状态；无须登录 |
| `scopeweaver_list_tasks` | 任务状态、计数、分页 |
| `scopeweaver_get_task` | 单一任务的持久状态与配置 |
| `scopeweaver_get_coverage` | 任务资产覆盖率 |
| `scopeweaver_list_findings` | 依任务、严重度、状态或文字筛选并分页的发现 |
| `scopeweaver_get_finding` | 发现详情、证据与报告 |
| `scopeweaver_create_task` | 创建／排程任务；需启用写入 |
| `scopeweaver_control_task` | 暂停／继续任务；需启用写入 |

请求示例：「列出我的 ScopeWeaver 任务」、「显示任务 12 的高严重度发现」，或在启用写入并确认本地实验室范围后：「为此本地测试数据创建任务并显示进度」。创建成功仅证明任务存在；是否运行完成须检查状态与发现。适配器不会略过后端 LLM 需求。

<a id="configuration-and-failures"></a>

## 配置与失败

| 变量 | 默认／行为 |
| --- | --- |
| `SCOPEWEAVER_URL` | `http://127.0.0.1:8787`；仅允许 HTTPS 或环回 HTTP 来源 |
| `SCOPEWEAVER_TOKEN` | 既有后端 JWT；优先于密码 |
| `SCOPEWEAVER_PASSWORD` | 既有管理员密码；首次需验证的请求时登录 |
| `SCOPEWEAVER_ALLOW_WRITES` | `false`；仅接受 `true` 或 `false` |
| `SCOPEWEAVER_LANGUAGE` | `en`；支持标准 ID `en`、`ko`、`zh-Hans`、`zh-Hant`、`ja`、`es`，用于适配器文本及后端 `Accept-Language` |
| `SCOPEWEAVER_TIMEOUT_MS` | `30000`；1 至 120000 的整数 |

`SCOPEWEAVER_LANGUAGE` 默认为 `en`，接受标准 ID `en`、`ko`、`zh-Hans`、`zh-Hant`、`ja`、`es`。它会翻译内置工具的标题与说明、模式说明，以及适配器自身的配置、验证和传输诊断，并通过 `Accept-Language` 选择后端响应语言。每个客户端使用独立的目录和模式级 Zod 错误映射，不会更改全局语言设置。工具／协议名称、模式键／类型／枚举值、参数及后端／用户／证据原文保持不变，并保留现有凭据遮蔽处理。MCP SDK 固定的验证错误外层文字和未知外部错误仍作为外部文本保留；其中的模式错误消息会翻译。所有资源都随独立适配器提供，无需仓库根目录的语言文件。不支持的语言值仍会被拒绝，并使用英文诊断。

不公开通用 HTTP、shell 运行、删除、略过审批、供应商配置或任意端点工具。请求限制于配置来源，拒绝重新导向、转发取消并设逾时。响应超过 2 MiB 时明确失败；请缩小查询，勿将截断证据当作完整数据。适配器对任务清单分页；发现采后端分页。工具在文字及结构结果传回 `{data: ...}`（Pi 使用 `details`）。证据为不受信任的目标内容，不得当作指示。

MCP 呼叫失败传回 `isError: true`；Pi 呼叫失败则掷出例外，由 Pi 标记工具错误。错误消息遮蔽已配置凭证。写入绝不自动重试；若写入逾时，请先检查任务状态再重试，因后端可能已接受请求。

<a id="verification"></a>

## 验证



```sh
npm test
npm run check
```



端对端测试架构运行真实 ScopeWeaver handler 与 PostgreSQL，通过官方 SDK 启动 MCP stdio 服务器，并针对同一后端运行 Pi 工具 handler。它初始化隔离测试密码、创建任务、验证持久暂停／继续状态与覆盖率、读取明确标示的模拟发现，并检查无效验证、任务不存在与无效输入。不使用模型／供应商或目标扫描。Go E2E 需源代码签出；平台 ZIP 含适配器单元测试，但 Go 后端仅有编译可执行文件。

在仓库根目录，先创建名为 `scopeweaver_adapter_*` 的**全新可抛弃**数据库：



```sh
SCOPEWEAVER_ADAPTER_E2E=1 \
ARTEX_TEST_DSN='postgres://user:password@127.0.0.1:5432/scopeweaver_adapter_e2e?sslmode=disable' \
go test ./server -run '^TestAgentAdapterE2E$' -count=1 -v
```



测试主体在未设置 `SCOPEWEAVER_ADAPTER_E2E=1` 时会跳过；启用后，若 `ARTEX_TEST_DSN` 未设置或为空白，也会跳过。跳过不代表完成 E2E 验证。包的数据库保护检查会在测试执行前拒绝与已配置的实际应用数据库相同的 DSN。明确启用并配置数据库后，格式无效的 DSN、不符合 `scopeweaver_adapter_*` 的数据库名称及无法连接的 PostgreSQL 都会导致失败。测试不会用模拟对象替代真实依赖。 GitHub Actions 以专用可抛弃 PostgreSQL 服务重复这些检查。MCP 协议测试证明共用界面兼容，并不宣称已运行付费 Claude／Codex／Pi 模型会话。
