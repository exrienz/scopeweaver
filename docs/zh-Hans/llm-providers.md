[English](../llm-providers.md) · [한국어](../ko/llm-providers.md) · 简体中文 · [繁體中文](../zh-Hant/llm-providers.md) · [日本語](../ja/llm-providers.md) · [Español](../es/llm-providers.md)

<a id="llm-provider-templates"></a>

# LLM 供应商模板

各语言版本请见本页上方导览。

ScopeWeaver 支持 **Anthropic**、**OpenAI Chat Completions**、**OpenAI Responses**、原生 **Ollama** 和可选的 **OpenCode SDK 桥接器**。原有四种格式继续保留自定义基础 URL、代理、会话头部、重试、速率与并行设置；云端、网关与自行托管的 OpenAI 兼容服务器保持原有行为。SDK 使用独立凭证及下文的限制，不替代 API／OpenCode Go 路径。

SDK 模板也不发送请求；模型发现必须明确点击**读取模型**。

模板只填写**新配置表单**，不保存凭证、不启用配置、不修改既有配置。Ollama 模板亦从所选服务器读取已安装模型；云端模板不发出请求。应用后格式、基础 URL 与模型均可编辑。

<a id="zai-glm-53"></a>

## Z.ai GLM-5.3

![GLM-5.3 新配置模板](../../screenshots/en/llm-templates.png)

预览未输入 API 密钥的模板，并非实际供应商连接。

开启「LLM → 新增 → 配置模板」。API 计费选「General API（建议）」；只有取得 Z.ai 对 ScopeWeaver 的授权后，才可使用「Coding Plan（参考）」。

| 字段 | General API | Coding Plan 参考 |
| --- | --- | --- |
| 格式 | OpenAI Chat Completions | OpenAI Chat Completions |
| 基础 URL | `https://api.z.ai/api/paas/v4` | `https://api.z.ai/api/coding/paas/v4` |
| 模型 | `glm-5.3` | `glm-5.3` |
| 思考 | `enabled` | `enabled` |
| 推理强度 | `low` | `low` |
| 上下文视窗（K） | `0`（默认 200K） | `0`（默认 200K） |
| 输出上限 | `16384` | `16384` |

两个端点与计费分开：General API 用量不属于 Coding Plan 订阅额度。详见 [Z.ai 连接配置](https://zcode.z.ai/en/docs/configuration)。测试前确认帐号模型权限与计费。

GLM-5.3 接受文字，上下文为 1M token，且必须保持推理启用。支持 `low`、`high`、`max`；模板采 `low`，在相同发现的小任务[实测](#measured-effect-glm-proxy)中，用量远低于 `max` 的一半。较难任务可提高强度。模型字段仍可编辑，选其他模型时请检查推理与上下文配置。[官方 GLM-5.3 指南](https://docs.z.ai/guides/llm/glm-5.3)。

**Coding Plan 支持边界：** Z.ai 仅允许官方支持工具使用 Coding Plan，ScopeWeaver 不在清单。端点模板不代表授权或支持；使用前须取得 Z.ai 授权。详见[工具整合](https://docs.z.ai/devpack/tool/others)与[使用政策](https://docs.z.ai/devpack/usage-policy)。ScopeWeaver 不冒充支持工具，也不自动切换计费端点。

输入配置名称与自己的 API 密钥。应用模板保留已输入名称、密钥、代理、会话头部与其他无关偏好。准备完成后保存；新配置须另行启用。既有「测试连接」与「加载模型」只有点选后才发出真实请求；此变更未进行实际供应商测试。

已于 **2026-10-03** 对照官方文档验证。供应商可用性、帐号资格与政策可能改变。

<a id="opencode-sdk-bridge"></a>

## OpenCode SDK 桥接器

`opencode-sdk` 是独立的可选配置格式，不是 OpenCode 的内置 HTTP API。ScopeWeaver 的 Norma 运行时继续管理对话、工具、审批、范围、重试与压缩；桥接器只完成一次模型步骤并收集宿主工具请求。

| `SCOPEWEAVER_OPENCODE_RUNTIME` | 固定的运行时 | 参数与历史边界 |
| --- | --- | --- |
| `native`（默认） | 官方 CLI、客户端、SDK **2.1.4** | 保留响应和宿主历史中的原始工具参数 JSON，包括 64 位整数；拒绝原始参数之外不安全的数值。 |
| `legacy` | SDK、CLI **1.19.4** | 只支持结构有效、安全可表示的 JSON；整数绝对值不超过 **9,007,199,254,740,991**。空白及等价 HTML／Unicode 转义可规范化，不保留原始字节；拒绝不安全整数和不透明／带签名思考历史。需要原始参数保真时用 `native`。 |

原生模式通过实际固定 CLI 的 `serve --stdio`、`run --server --session` 及官方客户端管理会话、事件和中断。每次调用拥有私有临时 home／配置／数据路径和数据库。保留真实内置工具定义及原始实现；原生权限为 ASK，不使用 `--auto`，公开 `execute.before` 钩子在执行前阻止所有非宿主工具。只有收集的宿主请求交回 Norma 执行和审批。OpenCode 生成的模型／环境／工具上下文与宿主 system、历史共同保留。遇到无法受控的系统／MDM 配置来源时拒绝启动，不读取或修改它们；细节见适配器指南。

旧版自定义供应商使用显式 OpenAI 兼容配置。**两个运行时均不支持 Windows**，因为必需的 POSIX 进程树清理边界尚未实现。本地运行时验证覆盖 **macOS arm64**。CI 在 **Linux x64** 上运行实际运行时测试；不声称已验证其他平台。其他 SDK 版本需单独验证。

按[适配器启动指南](../../adapters/opencode-sdk/README.zh-Hans.md)使用 **Bun 1.3.14**。桥接器使用自己的供应商配置，不导入用户 OpenCode 或云端已保存凭证。设置至少 **32 个非空白字符**的独立 `SCOPEWEAVER_OPENCODE_TOKEN`，在 **LLM → 新增 → 配置模板 → OpenCode SDK 桥接器**中填入；不要粘贴 OpenCode Go／Console API 密钥。

```sh
cd adapters/opencode-sdk
bun install --frozen-lockfile --ignore-scripts
export SCOPEWEAVER_OPENCODE_TOKEN="$(openssl rand -hex 32)"
bun run start
```

保留令牌供配置的**桥接令牌**字段使用。默认 `native`；启动前设 `SCOPEWEAVER_OPENCODE_RUNTIME=legacy` 可改用旧版。监听地址为 `127.0.0.1:4097`。通过 `SCOPEWEAVER_OPENCODE_PROVIDER_CONFIG` 指定专用 JSON 可配置显式供应商，必须使用适配器文档中的格式，不要指向用户认证文件。安装时保留可选平台依赖。

`SCOPEWEAVER_OPENCODE_TIMEOUT_MS` 为 **0–86,400,000 毫秒**，默认 **0** 由宿主取消控制。请求体上限 **16 MiB**；超限明确失败。其余输出与清理限额见适配器指南。

| 字段 | 值 |
| --- | --- |
| 格式 | `opencode-sdk` |
| 桥接 URL | `http://127.0.0.1:4097` |
| 模型 | 点击**读取模型**选择或手动输入 `providerID/modelID` |
| 桥接令牌 | 桥接器的 `SCOPEWEAVER_OPENCODE_TOKEN`，不是云端 API 密钥 |
| 流式传输 | 开启，使用所选固定运行时 |
| 思考／推理强度 | 模板省略，仅设置模型支持的值 |
| 输出上限 | `0` 使用宿主调用路径的现有默认值；桥接器按原生模型声明的较小输出上限限制请求 |
| 代理／自定义会话头部 | 不支持；模板清除两者 |

模板清除未保存的云端密钥，模型留空，**不发现、不推理、不保存、不启用**。填入令牌后明确点击**读取模型**，可不执行推理而查看目录；准备好后保存并单独启用。**测试连接**只在点击时发起模型请求，元数据发现不代表推理访问成功。

**先前免费模型验证 — 2026-10-25 KST：** 当目录列出时，可从**读取模型**中选择 `opencode/deepseek-v4.1-flash-free`。此前未登录的 macOS arm64 运行通过真正的原生 CLI 2.1.4 完成 UI 连接测试、手动 ASK 批准后的宿主 Bash 写入、精确的 `FREE_OK` 加换行文件内容及最终助手答案。共 **3 次推理 POST**，关闭重试，没有付费回退，也未观察到内置工具执行。这只证明当时一个小型隔离流程；不代表本次重跑、永久可用或无限免费。当前模型、免费资格及限额仍需确认。旧版托管免费访问尚未验证。

编辑时，空令牌只在同一 SDK 地址保留已保存值。更改 URL 会清除输入的令牌；更改地址或格式后留空不会复用已保存凭证。`localhost`、`127.0.0.1`、`::1` 是不同凭证作用域。模型 ID 与输出限制以当前元数据为准。思考设置依赖模型／供应商支持，不支持时明确失败。

**网络与账户限制：** 桥接器必须和 ScopeWeaver 位于同一网络命名空间；容器中的 localhost 指容器本身。下游仍可能是云服务，所以**仅本地 LLM** 模式拒绝 SDK 配置。桥接器不支持代理；下游网络在桥接器侧配置。免费模型 ID 不证明订阅权限、持续提供或无限用量；确认服务条款和当前限额。桥接器不选择付费回退、不导入账户；宿主已配置的故障转移策略另行管理。现有 API／OpenCode Go 配置保持独立。

**降级准备：** 数据库配置格式的 `CHECK` 约束增加 `opencode-sdk`。仍有 SDK 配置时旧二进制不能启动。降级前备份数据库，在当前版本将 SDK 配置转换为旧支持格式或移除。先前验证使用临时数据库，没有迁移生产数据库。

<a id="opencode-go-opencode-subscription-gateway"></a>

## OpenCode Go（opencode 订阅网关）

开启「LLM → 新增 → 配置模板」，选择「OpenCode Go · glm-5.3-flash」。

| 字段 | 值 |
| --- | --- |
| 格式 | OpenAI Chat Completions |
| 基础 URL | `https://opencode.ai/zen/go/v1` |
| 模型 | `glm-5.3-flash` |
| 思考 | 省略（模型默认） |
| 推理强度 | `max` |
| 上下文视窗（K） | `0`（默认 200K） |
| 输出上限 | `16384`，以 `max_tokens` 传送 |
| 会话头部 | `x-opencode-session`（网关必需；模板自动填入） |

网关兼容 OpenAI：opencode 1.18.5 的 `opencode-go` 定义以 `@ai-sdk/openai-compatible` 连至 `https://opencode.ai/zen/go/v1`，环境密钥为 `OPENCODE_API_KEY`（2026-10-20 检查）。GLM 类模型接受 `low`、`high`、`max`；本模板依操作人员要求刻意填 `max`，预期推理 token 用量最高。「加载模型」读取网关模型清单；模型栏仍可输入 `qwen3.7-plus` 等 ID。每个请求必须带 `x-opencode-session`，否则网关回 `400 MissingSessionID`。模板填入此头部名称，ScopeWeaver 以自己的会话 ID 作为值。

**凭证与条款：** 密钥属于 opencode 订阅，opencode 存于自身验证档；ScopeWeaver 绝不读取。请将密钥贴入配置或配置 `OPENCODE_API_KEY`。密钥受 opencode 订阅条款约束；保存前确认允许从其他工具使用。与 Z.ai Coding Plan 一样，此模板不构成授权。

环境配置：



```bash
SCOPEWEAVER_LLM_PROVIDER=opencode-go
SCOPEWEAVER_LLM_MODEL=glm-5.3-flash
# Key: OPENCODE_API_KEY is used when no generic key is set; SCOPEWEAVER_LLM_API_KEY overrides.
```



`SCOPEWEAVER_LLM_PROVIDER=opencode-go` 解析为 OpenAI Chat Completions 传输格式；未设基础 URL 与模型时采上表默认值。

<a id="controlling-token-use"></a>

## 控制 token 用量

每个智能体回合都重送系统提示词、工具结构与目前完整对话。因此输入逐回合增加，直到压缩缩短历史；供应商提示词缓存决定多少重复输入采较低缓存费率。本节响应云端配置的 [issue #24](https://github.com/cskwork/scopeweaver/issues/24)。原生 Ollama 另有[运行时保护](#runtime-safeguards)。

**证据状态：** 测量使用 Coding Plan 的 Z.ai GLM-5.3 作为 DeepSeek 代理。GLM 的 tokenizer、缓存计量与价格不同，因此 **DeepSeek 数字未验证**。DeepSeek、Claude、OpenAI、Qwen、Kimi、OpenRouter、OpenCode Go 支持以官方响应格式创建的测试数据验证，并非实际呼叫。下文 Claude 5.5 请求与重放亦同，未使用 Claude API 密钥。

<a id="economy-templates"></a>

### 节省用量模板

在「LLM → 新增 → 配置模板」选 *(economy)* 项目。六种均将输出上限设 `16384`、上下文视窗设 `0`（200K 压缩默认）。与 GLM 模板一样保留已输入密钥，不保存或启用配置。

| 模板 | 格式 | 基础 URL | 模型 | 思考 | 强度 | 输出字段 |
| --- | --- | --- | --- | --- | --- | --- |
| DeepSeek | OpenAI Chat Completions | `https://api.deepseek.com` | `deepseek-flash` | 省略（DeepSeek 默认开启） | `low` | `max_tokens` |
| Anthropic Claude | Anthropic | `https://api.anthropic.com` | `claude-haiku-5-5` | `disabled` | `low` | `max_tokens` |
| OpenAI | OpenAI Responses | `https://api.openai.com/v1` | `gpt-6-luna` | 省略 | `low` | `max_output_tokens` |
| Alibaba Qwen（国际） | OpenAI Chat Completions | `https://dashscope-intl.aliyuncs.com/compatible-mode/v1` | `qwen3.8-flash` | 省略 | `low` | `max_completion_tokens` |
| Moonshot Kimi | OpenAI Chat Completions | `https://api.moonshot.ai/v1` | `kimi-k3` | 省略 | `low` | `max_completion_tokens` |
| OpenRouter | OpenAI Chat Completions | `https://openrouter.ai/api/v1` | `deepseek/deepseek-v4.1-flash` | 省略 | `low` | `max_tokens` |

- 上列供应商的 **`low` 强度皆为未实测的节省默认**。结果截断或过浅时，先提高至供应商默认附近（DeepSeek `high`、GPT-6 Luna `medium`、Qwen `xhigh`、Kimi K3 `max`），再提高输出上限。
- Claude：Haiku 模板关闭思考以节省输出 token。Haiku 仅在 `low`、`medium`、`high` 接受停用思考；Sonnet、Opus 5.5 拒绝，因此无节省模板，请选 **Omit** 或 **Adaptive**，详见 [Claude 5.5 思考](#claude-55-thinking)。
- OpenAI：GPT-6 Luna 在 Chat Completions 仅于 `none` 强度接受工具，因此模板使用 Responses。
- OpenRouter：模型 ID 与 `reasoning_effort` 支持已于 2026-10-16 查公开模型清单。Claude 需 Chat Completions 无法传送的缓存标记，因此模板选自动缓存前缀的 DeepSeek Flash。
- Z.ai GLM-5.3：General 与 Coding Plan 模板现填 `low`、输出 `16384`、上下文 `0`，即下方实测节省配置。旧版为 `max`、上下文 `1000`、输出 `0`。GLM-5.3 必须保持思考启用。

<a id="claude-55-thinking"></a>

### Claude 5.5 思考

规则仅适用官方主机（`https://api.anthropic.com` 或空白基础 URL）上 `claude-…-5-5` 形式的模型 ID，含日期快照 `claude-…-5-5-YYYYMMDD`。

- 思考留空（**Omit**）现传送 `thinking: {type: adaptive, display: summarized}`；新增 **Adaptive** 同样传送此值；**On**（`enabled`）亦加 `display: summarized`。
- 工具循环重放工具呼叫及其思考区块，含空文字及 `redacted_thinking`。各供应商在内存保留最近 128 次工具呼叫、最多 4 MiB；含摘要文字的区块亦保存至历史，但工具循环中途重启后，空白／遮蔽区块无法恢复。
- 摘要与审批裁决不再传 `temperature`。Sonnet、Opus，以及 `xhigh`／`max` 的 Haiku 拒绝 `disabled`，因此这些呼叫移除停用配置，改用配置思考值。
- 配置 **Off**（`disabled`）在智能体回合仍送 `disabled`；Sonnet／Opus 5.5 应选 **Omit** 或 **Adaptive**。
- 第三方 Anthropic 兼容端点（DeepSeek、Kimi、Z.ai）及其他模型 ID 不变，空白思考仍省略。
- **仅以请求与流式测试数据验证**；真实 Claude 行为与计费未验证。

<a id="measured-effect-glm-proxy"></a>

### 实测影响（GLM 代理）

一个小任务：本地测试目标有三条路由与一个植入的未验证秘密。每种配置运行 3 次，各用全新数据库，并等待报告撰写对话完成。点数依 Z.ai 公布的 GLM-5.3 倍率计算：（未缓存输入 × 6.9 + 缓存输入 × 1.7 + 输出 × 24）/ 10,000，并非读取帐单。

| 配置 | 输入 + 输出 token 中位数 | 点数中位数 | 找到植入问题 |
| --- | --- | --- | --- |
| 基准：旧 Z.ai Coding 模板（`max`、上下文 `1000`、输出 `0`） | 209,233（172,576–216,452） | 75.1（64.3–88.6） | 3/3 |
| 节省：`low`、输出 `16384`、上下文 `0` | 96,023（95,060–105,023） | 26.6（25.0–27.4） | 3/3，皆由报告撰写智能体完成报告 |

- 相较基准，节省配置中位数减少 54% 输入 + 输出 token 与 65% 点数；每次节省运行皆低于最便宜基准。最大单次呼叫输入 13,287、输出 3,829 token，未碰到 200K 视窗或输出上限，因此此处节省来自 `low` 强度。输出与视窗对大型任务的限制效果未测量。最大／最小运行比为基准 1.25×、节省 1.10×。
- 各工作类别点数：基准 worker 29.6%、planner 29.3%、report writer 29.1%、goal setup 12.0%；节省为 report writer 38.8%、planner 34.1%、worker 20.2%、goal setup 6.9%。未发生压缩，无会话摘要呼叫；裁决与网页搜寻关闭。
- 输入缓存比例：基准 85.2%、节省 92.3%。含推理的输出占点数基准 39.3%、节省 23.4%；运行当时尚未分开记录推理 token。

GLM-5.3 一律思考，因此本测量不涵盖关闭思考的影响。

真实桥接检查（Claude 经 `claude -p`、OpenAI 经 `codex exec`）仅通过本地桥接测模型行为，不验证 API 传输格式或计费。

`codex exec` 桥接以 OpenAI 模板（gpt-6-luna、Responses）完成一个完整任务：工具循环完成、找到植入问题并由报告撰写智能体撰文，后续侧问亦成功回答，未标记中断。`llm_usage` 含推理与缓存读取 token，以及 `btw` 侧问列。另两次因桥接缓慢未在测试时间内完成；这是桥接速度限制，不是程序结果。`claude -p` 无法驱动工具协定，因此放弃。Claude／OpenAI 请求与用量处理由测试数据证明。

<a id="recommended-settings"></a>

### 建议配置

1. 从供应商节省模板开始；任务无需更多历史时保持上下文 `0`。
2. 保持正数输出上限；`0` 不会省略字段，详见下文。
3. 保持低强度，结果变差才提高。
4. 无需时关闭审批裁决、网页搜寻与 noa 上下文压缩，三者默认皆关。
5. 小任务可考虑少于默认 3 个工作智能体；效果未测量。
6. 无需每个发现的详细报告时，关闭 **Report writer** 的触发器（**System → Agent**）。
7. 首次任务后检查各工作类别用量，见[检查 token 去向](#checking-where-tokens-go)。

<a id="context-window-output-limit-thinking-and-effort"></a>

### 上下文视窗、输出上限、思考与强度

| 配置 | 云端配置行为 | 指引 |
| --- | --- | --- |
| 上下文视窗（K） | 仅本地压缩预算，不送供应商。`0` = 200K；值限制 32–1000。 | 越大越晚压缩，每次携带更多历史。200K 时约 115K token 开始清除旧工具结果，约 179K 摘要历史。 |
| 输出上限 | 每次回复上限。**`0` 不省略**：代理回合送 32768，直接 Anthropic 呼叫送 8192；原生 Ollama 采自身默认。 | 上限只截停长回复，不使回复自动变短；太低会截断。模板使用 16384。 |
| 思考 | `thinking.type`；空白省略，但官方 Claude 5.5 空白送 `adaptive`。推理以输出计费。 | DeepSeek 默认思考并支持 `disabled`，工具呼叫亦重送旧推理作输入。GLM-5.3 一律思考。Claude 模板关闭思考。 |
| 推理强度 | `low` 至 `max`；空白省略。 | 模板采 `low`，仅 GLM-5.3 实测；其他供应商为未测节省默认。支持值依供应商而异。 |

<a id="context-optimization"></a>

### 上下文最佳化

- **对话压缩**（内建、开启）：接近视窗时摘要旧消息，保留最新 8 则。摘要多一次呼叫，输出受上限限制、最多 20000（上限 `0` 时亦为 20000）。
- **旧工具输出清除**：超过视窗 60% 时，Read、Bash、Grep、Glob、WebFetch、WebSearch、Edit、Write 中超出最近五笔的结果会被替换；图、证据、自定义工具结果保留至下次摘要。
- **工具输出上限**：各文字区块截至 30,000 字节；完整输出存磁碟并提供指标。
- **图压缩**：探索图不活跃部分折成摘要，每个合格群组一次呼叫，输出上限 1500。
- **noa 上下文压缩**（系统配置、实验、默认关闭）取代内建压缩。
- 72% 触发与 16/32 KiB 输出上限仅适用原生 Ollama。

**提示词缓存：**

- Claude：planner、worker、main-agent、chat（含 report writer）现在一律标记系统提示词与其前工具结构可缓存。其他含系统提示词但无边界的 Anthropic 请求（目标配置、对话／图摘要、侧问准备、审批裁决）会标记完整系统提示词。低于模型最小可缓存长度时，即使标记也不缓存。各不同系统提示词首次写入以输入价 1.25× 计费；有效期内后续呼叫依缓存价读取。Worker 提示词依意图不同，每个意图写一次。标记位置经测试验证；**真实节省未验证**，因无 Claude 密钥。
- DeepSeek、Kimi、Z.ai 自动缓存前缀；DeepSeek／Kimi 文件说明 Anthropic 兼容端点忽略区块标记。OpenAI、Qwen、OpenRouter（DeepSeek Flash）也自动缓存相同前缀。
- 测试确认三种云端格式中，同一会话连续工具回合保持相同系统提示词与工具结构，仅附加历史。压缩、收尾、新解锁延后工具会变更前缀。自定义提示词模板若用 `{{.Now}}`，每次运行都改变系统提示词。

<a id="workers-retries-and-task-settings"></a>

### Worker、重试与任务配置

- **Worker concurrency · Worker Agent count**（系统配置，每任务默认 3）：各智能体领不同意图，运行自身会话并携带完整系统提示词与工具结构。
- **Max concurrent LLM requests**（默认 `0` 无上限）与配置 **Max concurrent requests** 只调节呼叫速度，不减少每次 token。
- 重试重送完整请求。默认：**Connection retries** 3、**Empty-response retries** 2（仅 OpenAI 格式）、**Same-provider safe-window retries** 2（任何输出前）、模型错误后意图重跑 2 次、仅推理回复后最多续接 2 次。在 LLM 页面、各配置或全域政策设 `-1` 可停用一层。失败是否计费取决供应商。
- 回合／时间预算用尽或任务收束时，收尾默认最多额外 10 回合。
- 供应商池容错（默认关闭）会在另一配置重放完整请求。

<a id="known-causes-of-extra-calls"></a>

### 额外呼叫的已知原因

- 长会话每回合重送全部历史，后期成本较高。
- Planner 每次唤醒（启动、意图完成、新发现、闲置心跳）创建新会话，各含多回合。
- 目标分解在规划前运行，最多 8 回合。
- **Report writer** 为预先创建的自定义智能体，每次 `report_finding` 皆有启用触发器。各发现启动独立对话，最多 5 个并行。
- **DeepSeek 网页搜寻**：每次搜寻为独立 DeepSeek 呼叫，使用目前配置密钥，输出上限 1024、每次呼叫最多 3 次搜寻。代理搜寻与配置页搜寻测试皆记入 `llm_usage`，类别 `web_search`，使用目前配置名称与模型；不在 LLM 录制中。DeepSeek 未说明搜寻响应用量格式，因此读取 Anthropic 式字段。**无用量框时记 0 token：呼叫有计数，token 未知。** 请到 DeepSeek 控制台查计费 token。
- 审批裁决默认关闭，无匹配规则的每次工具操作一次呼叫。侧问仅依要求运行，回答前最多 12 次摘要。
- **Test connection** 真实呼叫输出上限 8192，不记录。
- 已修正 Anthropic 格式流式在 `llm_usage` 高估输出：累计 10、20 以前误计 30，现为 20。只影响 ScopeWeaver 数字，不影响供应商帐单。
- 已修正内部会话与对话记录总数同样 10 + 20 误计 30，现为 20。这不是 `llm_usage`；高估只可能提早压缩。
- 已修正 OpenAI Responses 侧问因缺停止事件被误报中断。`response.completed` 现正常结束；`response.incomplete` 仍标记中断。

<a id="checking-where-tokens-go"></a>

### 检查 token 去向

- **Tasks → task → Overview → LLM token usage (by model)**：由常启 `llm_usage` 表显示各模型呼叫、输入、输出、**Reasoning (in output)**、缓存读取。仅显示标记该任务的列；report writer 列无任务。DeepSeek 搜寻列在目前配置模型下。
- **Dashboard** 与 **Interception rules → Approval token usage** 亦显示 **Reasoning (in output)**。
- **LLM recordings**：每次呼叫一列，含工作类别、token、延迟、原始请求／响应。默认关闭，任务前开启 **Recording**，会保存完整提示词与响应。
- 对 `llm_usage` 查某时间窗 SQL；类别位于 `worker` 栏：`planner`、`worker`、`goals`、`compactor`、`judge`、侧问 `btw`、`mainagent`、DeepSeek 搜寻 `web_search`；report writer、retest、自定义代理对话为空。对话摘要与收尾沿用智能体类别。



```sql
SELECT COALESCE(worker, '(none)') AS lane, model, COUNT(*) AS calls,
       SUM(input_tokens) AS input, SUM(cache_read) AS cache_read,
       SUM(cache_write) AS cache_write, SUM(output_tokens) AS output,
       SUM(reasoning_tokens) AS reasoning
FROM llm_usage
WHERE ts >= '2026-10-16 09:00' AND ts < '2026-10-16 10:00'  -- your task's time range
GROUP BY 1, 2
ORDER BY SUM(input_tokens) + SUM(output_tokens) DESC;
```



`cache_read`、`cache_write` 已包含于 `input`，`reasoning` 已包含于 `output`，不可再次相加。

缓存读取来自 `prompt_tokens_details.cached_tokens`、DeepSeek 顶层 `prompt_cache_hit_tokens`、Kimi 旧版顶层 `cached_tokens`。OpenAI Chat、OpenAI Responses、Kimi K3、OpenRouter 记 `cache_write_tokens`，Qwen 记 `cache_creation_input_tokens`，亦记 Claude 缓存写入。字段对应仅以测试数据验证。

<a id="accounting-gaps-and-follow-ups"></a>

### 计量缺口与后续

- 供应商无独立推理数字时记 0，不代表未推理。
- DeepSeek 搜寻用量字段未文件化，见[额外呼叫原因](#known-causes-of-extra-calls)。
- Gemini 尚无模板，需升级 Norma SDK v0.4.4（后续）。
- Claude 历史缓存（最后消息标记）需 SDK 支持（后续）。
- OpenRouter 部分模型仅接受统一 `reasoning.effort`，可能忽略模板 `reasoning_effort`；SDK 目前仅能送后者（后续）。

<a id="native-ollama-local-inference"></a>

## 原生 Ollama（本地推论）

ScopeWeaver 通过原生 `/api/chat` 与 Ollama 通讯。OpenAI 兼容端点无法配置上下文大小（[Ollama OpenAI 兼容性](https://docs.ollama.com/api/openai-compatibility)），因此指向 `/v1` 的配置无法保证 `num_ctx`。原生格式每次请求皆传送此值。以 `/v1`、`/api`、`/api/chat` 结尾的基础 URL 会正规化至服务器根路径。

Ollama 文件的默认上下文说明不一致：[FAQ](https://docs.ollama.com/faq) 为 4,096 token，[上下文长度页](https://docs.ollama.com/context-length) 则依 VRAM 决定（低于 24 GiB 约 4K，24–48 GiB 为 32K，更高为 256K），因此 ScopeWeaver 不依赖服务器默认。

<a id="templates"></a>

### 模板

开启「LLM → 新增 → 配置模板 → 本地推论（Ollama）」。

| 字段 | 同一台电脑 | Docker 存取主机 | 内附 Compose 服务 |
| --- | --- | --- | --- |
| 格式 | Ollama（原生 /api/chat） | Ollama（原生 /api/chat） | Ollama（原生 /api/chat） |
| 基础 URL | `http://127.0.0.1:11434` | `http://host.docker.internal:11434` | `http://ollama:11434` |
| 模型 | 此服务器已安装模型 | 此服务器已安装模型 | 此服务器已安装模型 |
| API 密钥 | 空白 | 空白 | 空白 |
| 上下文视窗（K） | `32`（`num_ctx` 32,768） | `32` | `32` |
| 最大并行请求 | `1` | `1` | `1` |
| Keep-alive | `30m` | `30m` | `30m` |
| 输出上限 | `4096` | `4096` | `4096` |
| 思考／强度 | 省略（模型默认） | 省略 | 省略 |

选取 Ollama 模板会从端点读 `/api/tags` 与模型元数据，再选已安装模型，优先声明 `tools` 者。新配置选 Ollama 格式亦同；空白端点或未修改的云端模板端点改用主机默认，自定义端点保持不变。探索期间模型留空；失败或无模型时保持空白并显示错误。请检查端点、选用令牌与代理，再按「加载模型」或手填标签。编辑模型／连接字段会取消待选，延迟响应不能覆写。开启既有配置保留已存模型。选取不会下载／运行模型、保存或启用配置。「测试连接」仅点选才运行。

**API 密钥行为：** Ollama 模板与切换至 Ollama 格式会在探索前清除未保存密钥，保留名称、代理、会话头部及其他偏好，但**清除所有未保存 API 密钥**，避免将云端凭证送往本地。GLM 模板仍保留输入密钥。本地 Ollama 应留空；仅前置反向代理需要时输入令牌，以 `Authorization: Bearer` 传送。免密钥 Ollama 可测试、保存、启用与容错；卡片显示「不需要密钥」。编辑已存原生 Ollama 配置时，空白密钥保留既有令牌；将云端改为 Ollama 且字段空白则删除旧云端密钥。

**其他格式密钥：** Anthropic／OpenAI 需要密钥，除非基础 URL 明确为本地，即主机只解析至环回或私有地址。因此 vLLM、LM Studio 等免密钥本地兼容服务器可留空；ScopeWeaver 送内部非秘密预留值，防止 SDK 继承环境 `OPENAI_API_KEY`／`ANTHROPIC_API_KEY`。非本地且无密钥会拒绝，错误为 *llm: API key required for non-local compatible endpoint*。

<a id="context-output-and-residency-controls"></a>

### 上下文、输出与驻留控制

| 配置 | Ollama 行为 | 云端格式 |
| --- | --- | --- |
| 上下文视窗（K） | 二进位 K，以 `options.num_ctx` 传送（32 = 32,768）；`0` 采 32，低于 16 采 16。亦配置压缩预算。 | 不变，仅压缩预算；`0` = 200K |
| 输出上限 | `options.num_predict`；`0` 采 4,096 | `0` 在代理回合送 32768，直接 Anthropic 送 8192 |
| Keep-alive | `keep_alive`；空白采 `30m`（Ollama 本身默认 5 分钟） | 不传送 |
| 最大并行请求 | `0` 采 1。每个 Ollama 服务器跨配置共用准入关卡，采最低已设上限 | `0` 继承「系统配置 → 最大并行 LLM 请求」 |

排队中的审批裁决、主要智能体、侧问先于背景工作取得下一名额；各组 FIFO，不中断进行中推论。诊断与预热仍为背景请求。

只有服务器 `OLLAMA_NUM_PARALLEL` 大于 1 才提高并行数。内存用量随 `OLLAMA_NUM_PARALLEL × context length` 增长（[FAQ](https://docs.ollama.com/faq)）。

<a id="runtime-safeguards"></a>

### 运行时保护

- **预热**：原生配置首次推论前，先呼叫 `/api/show`，拒绝未声明 `tools` 的模型，再以空 `/api/chat`、配置 `num_ctx` 与 `keep_alive` 加载模型。预热亦占同一并行名额。
- **压缩**：原生 Ollama 视窗 ≤64K 时，于 72% 触发历史压缩，保留完整工具呼叫／结果配对。即使启用实验 noa，此保护仍优先，略过 noa 并警告。≤32K 时给模型的各工具输出上限 16,384 **字节**（16 KiB，以 SDK `len(s)` 计，不是 Unicode 字符）；>32K 至 64K 为 32,768 字节（32 KiB）。完整输出保留于工具结果指向的溢出档。云端压缩与 noa 不变。
- **审批裁决**：明确裁决配置优先，否则优先原始代理／运行实际使用的原生配置（含绑定、池、容错）；云端或无来源时用目前启用配置。原生裁决默认 **90 秒**、失败动作 **ask**，已存覆写优先；用户审批仍于 300 秒到期并默认拒绝。原生输出受 decision／comment JSON schema 约束并再次验证。系统配置成功加载前，表单不得保存。

<a id="generic-openai-compatible-servers-vllm-lm-studio"></a>

### 一般 OpenAI 兼容服务器（vLLM、LM Studio）

这些服务器仍选既有 **OpenAI（Chat Completions）** 格式，基础 URL 指向 `/v1`。不得使用原生 Ollama 格式。

- 原生格式「加载模型」读 `/api/tags`，HTTP 404／405 时回退 `/v1/models`，所以可能显示一般服务器模型清单。
- 对话**绝不**默默切换：原生配置一律呼叫 `/api/chat`，一般服务器未提供。因此「找不到模型或端点」及 `ollama pull` 提示不适用；拉模型不能修复 vLLM／LM Studio。请改选 OpenAI 格式。

<a id="load-models-and-test-connection"></a>

### 加载模型与测试连接

「加载模型」读 `/api/tags`、`/api/show`，各列显示参数大小、量化与 `tools`／`thinking`／`vision` 标章。未声明 `tools` 时警告：「此模型未声明原生工具呼叫支持；ScopeWeaver 智能体需要工具呼叫以更新资产与探索图。」元数据读取失败显示未知，不当作缺少能力。

「测试连接」以 `/api/show` 检查模型，使用配置 `num_ctx`、`keep_alive`、`num_predict` 做单回合工具呼叫基本测试，再读 `/api/ps`。面板显示分配上下文是否涵盖请求、工具是否返回、冷启毫秒与 token/s。缺模型时显示服务器上应运行的精确命令，如 `ollama pull qwen2.5-coder:32b`。ScopeWeaver 不拉取模型或创建 Modelfile。

<a id="host-and-container-routing"></a>

### 主机与容器路由

| ScopeWeaver 位置 | Ollama 位置 | 基础 URL |
| --- | --- | --- |
| 主机 | 同一主机 | `http://127.0.0.1:11434` |
| Docker Compose | 主机 | `http://host.docker.internal:11434` |
| Docker Compose | 内附 `ollama` 服务 | `http://ollama:11434` |

容器内 `127.0.0.1` 指容器本身。Compose 将 `host.docker.internal` 对应主机网关 `host-gateway`。Ollama 默认绑定 `127.0.0.1:11434`；Linux 主机 Ollama 必须监听容器能到达的地址（如 `OLLAMA_HOST=0.0.0.0:11434`），这会对网络公开服务。请以防火墙限制 11434 连接埠。

**选用 Compose 服务：** `docker compose --profile local-llm up -d` 启动 `ollama`，配置 `OLLAMA_NO_CLOUD=1`、`OLLAMA_KEEP_ALIVE=30m`、`OLLAMA_NUM_PARALLEL=1`、`OLLAMA_CONTEXT_LENGTH=32768` 与 `ollama_data` 磁碟区。以 `docker compose exec ollama ollama pull qwen2.5-coder:14b` 拉取模型。未启用 profile 时 Compose 不变。此服务不请求 GPU。**macOS Docker 无法使用 Apple GPU，仅 CPU 运行**；需要实用速度时请在 Mac 原生运行 Ollama，选主机模板。

<a id="network-policy-local_llm_only"></a>

### 网络政策（`local_llm_only`）

默认关闭，云端供应商仍可用。于「系统配置 → 仅限本地 LLM 网络」或 `SCOPEWEAVER_LOCAL_LLM_ONLY=true` 启用后：

- LLM HTTP（推论、模型清单、连接测试、预热、裁决）仅允许**环回地址及明确信任 CIDR**（`local_llm_allowed_cidrs`）。私有 IPv4、IPv6 ULA、LAN、Docker 不再自动信任；仅加入自己控制的推论地址，优先 IPv4 `/32` 或 IPv6 `/128`。
- 链路本地、组播、未指定地址即使在信任 CIDR 仍拒绝。连接开启时检查 DNS 解析，全部地址均须允许。拒绝所有重新导向与 LLM 配置代理。
- **封锁全部后端的智能体网页搜寻、搜寻连接探测、更新检查与下载。** 更新页区分刻意政策与网络故障。本地版本、更新进度、回滚仍可用；不以缓存远程发行信息绕过政策。
- 运行时变更无需重启，含既有会话。启用前已准入下载于下次受保护网络操作／本文读取停止并移除部分暂存；已送出流量无法收回。
- `SCOPEWEAVER_LOCAL_LLM_ALLOWED_CIDRS`（或保留的 `ARTEX_LOCAL_LLM_ALLOWED_CIDRS`）亦生效，不复制至保存清单，也无法在界面编辑。环境 `true` 为界面无法停用的政策下限。
- 目标流量、目标工具、全域代理仍可连网。这是应用政策，不是密码学或整台主机离线保证。

**既有严格 LAN／Docker 配置迁移：** 同机 `http://127.0.0.1:11434` 无需 CIDR；`http://192.168.1.20:11434` 需 `192.168.1.20/32`；IPv6 `fd00::20` 需 `fd00::20/128`。Docker 必须**从 ScopeWeaver 容器内**解析名称，因主机与容器可能看到不同地址：



```bash
# Existing Compose service; read its actual name resolution and network addresses.
docker compose exec scopeweaver getent ahosts host.docker.internal
docker compose exec scopeweaver getent ahosts ollama
docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}} {{.GlobalIPv6Address}}{{end}}' "$(docker compose ps -q ollama)"
```



若镜像缺 `getent`，以 `docker compose exec scopeweaver cat /etc/hosts` 检查既有主机网关，并以 `docker inspect` 检查 Ollama 容器地址。使用 ScopeWeaver 实际到达的推论 IP，勿照抄示例或允许整段 Docker 子网。加入「其他受信任 CIDR」或环境允许清单，如 `SCOPEWEAVER_LOCAL_LLM_ALLOWED_CIDRS=192.168.1.20/32`。`host.docker.internal` 与 `ollama` 名称均不自动授予信任。Docker 重建服务或地址变更后重查，在重试「加载模型」／「测试连接」前更新狭窄允许清单。

ScopeWeaver 看不到 Ollama 后续行为。**Ollama 可将请求转发至云端模型**；内附 Compose 设 `OLLAMA_NO_CLOUD=1`，但主机／远程 Ollama 须另行停用云端：`OLLAMA_NO_CLOUD=1` 或依 [FAQ](https://docs.ollama.com/faq) 在 `~/.ollama/server.json` 设 `disable_ollama_cloud`。

<a id="environment-and-configuration"></a>

### 环境与配置

`SCOPEWEAVER_LLM_*` 优先于 `ARTEX_LLM_*`；明确空白的现代变量亦覆写旧值。两者都覆写 `config.json` 的 `llm` 区块，文件路径来自 `SCOPEWEAVER_CONFIG`／`ARTEX_CONFIG`。

| 后缀 | 含义 |
| --- | --- |
| `PROVIDER` | `anthropic`、`openai`、`openai-responses`、`opencode-go`、`ollama` |
| `MODEL`, `BASE_URL`, `PROXY` | 模型、端点、选用 LLM 代理 |
| `API_KEY` | Anthropic／OpenAI 必填，除非基础明确本地；Ollama 为选用令牌 |
| `CONTEXT_WINDOW_K`, `MAX_TOKENS`, `KEEP_ALIVE`, `MAX_CONCURRENT` | 上述控制 |
| `THINKING_TYPE`, `REASONING_EFFORT`, `STREAM` | 既有推理与流式选项 |

政策变量：`SCOPEWEAVER_LOCAL_LLM_ONLY`／`ARTEX_LOCAL_LLM_ONLY`，以及逗号分隔的 `SCOPEWEAVER_LOCAL_LLM_ALLOWED_CIDRS`／`ARTEX_LOCAL_LLM_ALLOWED_CIDRS`。

启动时 Anthropic／OpenAI 在无通用密钥时使用 `ANTHROPIC_API_KEY`／`OPENAI_API_KEY`，含需验证本地网关。`opencode-go` 改用 `OPENCODE_API_KEY`，绝不使用 `OPENAI_API_KEY`；只有 `OPENCODE_API_KEY` 时会推断此供应商。若已设供应商密钥但刻意要免密钥本地配置，明确设 `SCOPEWEAVER_LLM_API_KEY=`（或 `ARTEX_LLM_API_KEY=`），覆写文件凭证并阻止供应商密钥回退。原生 Ollama 仅回退 `OLLAMA_API_KEY`，绝不采 Anthropic、OpenAI、OpenCode 密钥。

界面以空密钥保存新配置不会导入环境凭证。空密钥「测试连接」若有 profile ID，可重用该已存密钥。无 ID 时，本地 Anthropic／OpenAI 测试仅在格式、正规化基础 URL、代理全部相符于目前服务时，才重用启用密钥，含新草稿测试。不同本地端点、格式、代理不继承；原生 Ollama 绝不继承启用的云端密钥。非本地测试保留既有启用密钥回退。



```bash
# .env for ScopeWeaver in Docker with Ollama on the host
# Replace your-installed-tool-model with a tool-capable tag shown by Load models.
SCOPEWEAVER_LLM_PROVIDER=ollama
SCOPEWEAVER_LLM_BASE_URL=http://host.docker.internal:11434
SCOPEWEAVER_LLM_MODEL=your-installed-tool-model
SCOPEWEAVER_LLM_CONTEXT_WINDOW_K=32
SCOPEWEAVER_LLM_MAX_CONCURRENT=1
SCOPEWEAVER_LLM_KEEP_ALIVE=30m
SCOPEWEAVER_LLM_MAX_TOKENS=4096
SCOPEWEAVER_LOCAL_LLM_ONLY=false   # add the actual host /32 or /128 before enabling strict mode
# SCOPEWEAVER_LOCAL_LLM_ALLOWED_CIDRS=192.168.1.20/32  # replace with the resolved inference IP
```





```json
{
  "llm": { "provider": "ollama", "base_url": "http://127.0.0.1:11434", "model": "your-installed-tool-model", "context_window_k": 32, "llm_max_concurrent": 1, "keep_alive": "30m", "max_tokens": 4096 },
  "local_llm_only": false,
  "local_llm_allowed_cidrs": []
}
```



<a id="bounded-diagnostics-and-telemetry"></a>

### 有界诊断与遥测

「测试连接」刻意进行推论探测；选模板仅探索安装元数据。除工具／上下文、加载时间／速度外，原生诊断以所选 runner `/api/ps` 的 `size`、`size_vram` 显示 GPU 内存百分比。总大小缺漏／为零或 VRAM 缺漏表示**未知**，不是 0%；总量正数且 VRAM 明确为零才是 0%。此比率为内存估算，不是精确权重切分、GPU 容量测试或驻留保证。

百分比 ≤80% 时警告 CPU 工作可能拖慢生成。其他已载模型，以及同端点但不同模型／上下文的明确裁决配置，另有内存／重载警告。共用模型／上下文可能减少重载，但不保证驻留。诊断不下载模型、不保存／启用配置。

接近上下文容量警告是基于回报 token 数的服务器记录启发式，不是截断侦测器。完整原始 Ollama 时序字段仅于启用原始录制时保存；token／缓存计量独立。小模型仍需任务特定验证，下述可靠性限制仍适用。

<a id="hardware-tiers"></a>

### 硬体分级

以下依 GPU VRAM 或 Apple 统一内存提供工具能力模型的起点，**仅估算，非基准测试**。上下文为配置 **Context window (K)**，须同时符合模型上限与内存（权重 + 随上下文及 `OLLAMA_NUM_PARALLEL` 增长的 KV 缓存）。以「测试连接」确认所要求上下文是否获分配。

| 内存 | 模型示例 | 上下文视窗（K） | 角色 |
| --- | --- | --- | --- |
| 8–12 GB | `qwen2.5-coder:7b`, `qwen3:8b` | `16`–`32` | 裁决或辅助配置；主要代理能力较弱 |
| 16–24 GB | `qwen2.5-coder:14b`, `qwen3:14b` | `32` | 工作站主要代理 |
| 32–48 GB | `qwen2.5-coder:32b`, `qwen3:32b` | `32` | 主要代理；仅模型上限允许时用 `64`，这些模型列 32K／40K |
| 64–128 GB+ | `llama3.3:70b` | `64`（模型列 128K） | 大内存主机主要代理；64K 需要此范围较高端 |

模板从所选服务器已安装模型挑选，使用 32K。硬体示例不是自动模型选择。

<a id="model-candidates"></a>

### 模型候选

仅列 [Ollama 模型库](https://ollama.com/library) 页声明 `tools` 的模型（2026-10-15 检查），下载大小为默认量化。**内存指引为估算**：权重加随上下文／并行增长的 KV 缓存。ScopeWeaver 未对这些模型／硬体做基准，工具呼叫品质依模型而异。

| 模型 | 下载 | 说明 |
| --- | --- | --- |
| `qwen2.5-coder:7b` | 4.7 GB | 最小实用候选，亦可作裁决 |
| `qwen2.5-coder:14b` | 9.0 GB | 工作站候选 |
| `qwen2.5-coder:32b` | 20 GB | 较大主机候选 |
| `qwen3:8b`, `qwen3:14b`, `qwen3:32b` | 5.2 / 9.3 / 20 GB | 亦声明 `thinking` |
| `llama3.3:70b` | 43 GB | 大内存主机 |

Ollama 模型库没有 `qwen3:72b`、`llama3.2:8b`；`llama3.2` 提供 1B／3B。使用前在自己的服务器以「加载模型」确认 `tools`。

**已观察限制（2026-10-15）：** 声明 `tools` 不保证传输层工具呼叫。分支测试中，`qwen2.5-coder:7b` 将工具形状 JSON 当纯文字，因此无工具运行；`qwen3:4b` 通过工具诊断却无法完成长自主任务。小模型作主要代理仍未验证；见[验证](VERIFICATION.md#autonomous-task-reliability-is-not-validated)。

无论供应商为何，[README](../../README.zh-Hans.md) 使用限制仍适用：仅在本地隔离环境运行 ScopeWeaver。

[韩文指南](../ko/llm-providers.md)
