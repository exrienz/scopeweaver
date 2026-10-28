[English](../../../sidequestion/CONTEXT_BUDGET.md) · [한국어](../../ko/sidequestion/CONTEXT_BUDGET.md) · 简体中文 · [繁體中文](../../zh-Hant/sidequestion/CONTEXT_BUDGET.md) · [日本語](../../ja/sidequestion/CONTEXT_BUDGET.md) · [Español](../../es/sidequestion/CONTEXT_BUDGET.md)

<a id="long-side-conversations-and-context-budgets"></a>

# 长侧问对话与上下文预算

各语言版本请见本页上方导览。

上游历史修正，2026-09-11。原实作将请求 JSON 字符当作 token，并继承主任务 32K 输出预留，因此当 HTML、JavaScript 与工具结果占用大量上下文时，过早拒绝有效侧问。

<a id="open-source-implementations-reviewed"></a>

## 已检视的开源实作

- [Grok CLI 侧问上下文](https://github.com/superagent-ai/grok-cli/blob/fb97af83f06dca873281d60168430f06c8de6324/src/agent/agent.ts#L739)：捕获最近用户／助理文字，预算约 2000 字符、每则最多 400 字符。此路径不维护持续侧问历史。
- [Grok CLI 独立请求](https://github.com/superagent-ai/grok-cli/blob/fb97af83f06dca873281d60168430f06c8de6324/src/utils/side-question.ts)：独立取消讯号、支持时采 2048 token 输出上限、不含工具。
- [Grok CLI 主会话压缩](https://github.com/superagent-ai/grok-cli/blob/fb97af83f06dca873281d60168430f06c8de6324/src/agent/compaction.ts)：估算 token、保留近期内容、以新数据更新旧摘要，并处理跨回合截断。
- [OpenCode 会话压缩](https://github.com/anomalyco/opencode/blob/b3f1a96c6dd7adeb28b36dd11add1998fc84d67b/packages/core/src/session/compaction.ts)：估算完整请求、预留输出／缓冲空间、结合近期内容与滚动摘要，摘要请求不含工具。该实作默认近期上下文 8000 token、摘要输出 4096 token。
- [OpenCode 溢位回滚](https://github.com/anomalyco/opencode/blob/b3f1a96c6dd7adeb28b36dd11add1998fc84d67b/packages/core/src/session/runner/llm.ts)：仅在助理开始输出前尝试回滚，回滚呼叫不再次进入同一路径。

ARTEX 借用独立输出预算、近期内容加滚动摘要及有界回滚，保留 norma v0.3.6 结构化消息与工具配对，而非 Grok 文字摘录。不将 OpenCode 式主要会话压缩事件写入 ARTEX 主要对话记录。ScopeWeaver 保留此设计；引用版本描述历史实作。

<a id="request-budget-and-execution"></a>

## 请求预算与运行

- 消息使用 norma 内容区块 UTF-8 字节估算，加 4/3 裕度，再加系统指示、工具结构与消息封装开销。这是估算，不是精确模型 token 数。
- 侧问输出默认最多 8192 token，绝不超过主配置输出上限。`ARTEX_BTW_MAX_OUTPUT_TOKENS` 接受 256 至 32768 的服务器端上限，不变更默认模型或主任务参数。
- 输入预算为上下文视窗减输出上限及安全裕度。未知视窗采平台默认 200K。安全裕度为视窗 5%，限制在 128–8192 token。
- 成功交换依递增序位加载，每批最多 20 次。最多保留 20 次原始交换，且不超过输入预算四分之一或 16K token。
- 超额交换更新滚动摘要，保留历史来源与上下文时间。旧助理回答不是新的工具证据；冲突时以最新主要快照优先。
- 主要上下文仍过长时，仅摘要副本中的旧消息，保留最多 8K 近期 token。切分点保留工具呼叫／结果配对；过大群组整组纳入摘要。
- 摘要输入依剩余视窗空间在 UTF-8 安全边界切分，输出上限 2048 token。空白、截断、工具呼叫或超预算摘要不缓存。每个侧问在同一 120 秒逾时内最多 12 次摘要呼叫，达上限明确失败，不无限循环。
- 首次模型上下文溢位且尚未输出文字／工具呼叫时，允许一次更小请求重试。估算大小未减少时立即停止回滚。其他模型错误与部分流式不触发回滚。
- 所有返回用量（含摘要、失败尝试与取消）累计于同一侧问请求。供应商未返回用量时仅记 0；不得以估算冒充实测用量。

<a id="persistence-and-interface"></a>

## 持久化与界面

`side_question_sessions.memory` 保存较旧交换摘要、涵盖序位，以及依快照身分缓存的主要上下文摘要。`side_question_requests.context_info` 保存准备阶段、实际重放次数、摘要使用情况与预算估算。

只有原请求仍运行且清除版本一致时才保存摘要。清除亦移除缓存，延迟写入无法恢复已清除数据。新快照不重用旧快照摘要。封存 v3 保留摘要字段；恢复缺少字段的旧 v3 时填入空对象，v1／v2 保持兼容。

POST 先接受并返回请求，再于背景准备／压缩；期间不持有准入锁或数据库交易。SSE／历史显示准备、整理交换、压缩副本及回答阶段。压缩失败持久化为侧问终止失败。前端保留错误，并将失败问题恢复为草稿，不在输入栏上方显示 toast。历史轮询不再清除提交错误。

<a id="historical-verification-record"></a>

## 历史验证记录

以下为 2026-09-11 修正所记录的上游结果，并非新的 ScopeWeaver 结果。

- 19／20／21／50 次交换重放、保留超过 20 次交换前的结论、重启后摘要缓存重用：自动检查通过。
- 极长中文回答／代码上下文、分块预算、工具配对、快照不变及新快照缓存失效：自动检查通过。
- 摘要失败／取消／截断／超长／工具返回、清除竞态、呼叫上限、单次溢位回滚与部分流式后不重试：自动检查通过。
- 隔离 PostgreSQL 分页、重启、v1／v2／v3 封存、摘要／预算元数据恢复、旧 v3 缺字段及 20 个父资源共用四个并行槽：通过。
- 候选 Go 构建、前端 TypeScript、变更元件 Biome 与隔离目录的 Next.js 正式构建：通过。
- 应用程序内浏览器以 1280×720 及 390×844 隔离界面测试数据，验证准备、摘要范围提示、失败草稿恢复、无错误 toast、无水平溢出与控制台错误。暂存数据已移除。
- 只读重放既有本地 Worker 快照通过新预算检查。例如 Worker #3 的 293085 字符快照不再被本地字符计数拒绝。此检查未呼叫外部模型。
- 自动审批审查拒绝将私人 Worker 快照传给 Grok 进行真实对话；该测试未运行，也未算通过。
- 应用户要求，本地后端于 2026-09-11 00:37 使用原数据库、数据目录及登录配置重启。可执行文件与候选 SHA-256 一致；后端与前端代理的 `/api/health` 均回报健康。

验证命令（仅限隔离测试数据库）：



```sh
go test -race ./sidequestion ./db ./server -run 'TestSide|TestCheckpoint|TestSnapshot|TestBuildRequest|TestService|TestMainSide|TestTaskArchive' -count=1
go build ./cmd/artex
npx tsc --noEmit
npm run build -- --webpack
```



历史前端正式构建使用独立副本，避免覆写运行中预览的 `.next`。候选可执行文件为 `/private/tmp/artex-btw-budget-candidate`，复制至 `/private/tmp/artex-btw-preview/artex` 并启动；旧档在该处备份为 `artex.before-context-budget`。这些路径描述当时运行，并非随 ScopeWeaver 发行的文件。
