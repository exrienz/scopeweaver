[English](../VERIFICATION.md) · [한국어](../ko/VERIFICATION.md) · 简体中文 · [繁體中文](../zh-Hant/VERIFICATION.md) · [日本語](../ja/VERIFICATION.md) · [Español](../es/VERIFICATION.md)

<a id="verification-and-known-limitations"></a>

# 验证与已知限制

各语言版本请见本页上方导览。

<a id="current-dev-pr-40-reconstruction--2026-10-25"></a>

## 基于当前 dev 重建 PR #40 — 2026-10-25

最初仅将 `aebea3e..9b04754` 中预期的四个提交重建到 `f354c920` 上；未重放分叉的分支历史。随后，候选版变基到 `7af06f1`，保留其六语言整合。目前的整合要求恰好 **1,987** 个正式键具有英文、韩文、简体中文、繁体中文、日文和西班牙文的原始译文。既有 OpenCode 环境键隔离、会话标头优先级及 User-Agent 默认值均予以保留。

更广的取消／竞态测试覆盖揭露了既有规划器结果未同步的问题：流式会话可能在暂停时返回，而真实的 `prove_goal` 处理程序仍在设置 `GoalMet` 及其原因。终结工具的写入方与规划器的结果快照现在共用一个私有互斥锁。这保留了导出字段、真实工具／终结回执，以及目前暂停与完成之间的优先次序；不增加供应商请求，也不改变正式数据库 schema。

终结证明关卡还要求**真实且成功的 SDK 工具结果回执**，且须与标记的工具使用 ID 及发起它的模型响应相符。私有捕获观察器在下一次压缩或推理边界之前运行；仅处理程序完成或 `PostToolUse` hook 本身不能满足资格。对抗回归使用真实的 `goal_met`、**30 ms** 运行预算与 **120 ms** 工具后 hook：旧关卡在收到中断回执后吞掉了结算，使**供应商调用为 1 次而非 2 次**，而原生原因仍为 `timeout`，错误仍为 nil。修正后的路径保留该中断回执，允许真实结算调用及其压缩，并拒绝延迟 hook 赋予资格。第二个同步回归将 hook 保持等待，直到捕获观察到中断，以证明真实处理程序在超时回执之前运行；正常成功证明、父上下文取消、反应式溢出与延迟来源防护仍在覆盖范围内。详见[终结证明回归](../../agent/terminal_proof_test.go)。

最初的重建仍有继承的关闭缺口：在未改动的 `f354c920` 上，`TestMainServeFailureRunsShutdown` 在 **10 次独立运行中失败 3 次**，包括数据库关闭后排程器仍写入。后续修复在启动前登记排程器归属。`Manager.Close` 在管理器锁之外取消并等待排程器结束，最多 **5 秒**，之后才停止日志写入器并关闭数据库。如果排程器 SQL 仍然阻塞，Close 返回 `context.DeadlineExceeded` 并**保持数据库开启**；工作结束后，可以重试关闭。CLI 原本就会报告关闭失败并以 1 退出。这不表示既有不带上下文的 SQL 调用变得可取消。已预先取消的排程器不再初始化水位或启动下一轮 tick。[阻塞 SQL 回归](../../server/scheduler_shutdown_test.go)检查并发关闭超时、工作仍活跃时数据库保持开启，以及解除阻塞后并发重试成功。首次并行全包运行还出现日志接收器测试的全局行数断言失败，当时其他包的子进程使用同一个数据库；跨包干扰仍属假设，因此最终验证遵循 CI，为每个包使用独立的可抛弃数据库。这些历史回执不代表最终确切提交的代管 CI、原生 Windows 或真实供应商验证。

整合候选版在网页端使用 Next.js **16.4.3** 与 Zod **4.6.7**，适配器使用 Zod **4.6.7**，并在锁定文件中以兼容方式修复 body-parser **2.4.0**、fast-uri **3.1.9**、ip-address **10.7.4** 和 qs **6.16.1**。适配器 SDK 仍固定为 **1.32.0**。六语言变基后，两个包的全新 `npm ci` 均通过；网页测试通过 **75/75**，冷启动静态构建导出 **31 页**，适配器测试通过 **18/18**，语法检查也通过。先前 72/7 测试回执仍作为历史证据保留。目前两项 `npm audit --audit-level=low` 检查均报告 **0 个漏洞**。这些结果取代了最初 Next.js 16.4.0 公告／构建失败的限制；原始回执仍为历史证据。浏览器原生贴上／撤销按键现在使用 `ControlOrMeta+V` 与 `ControlOrMeta+Z`，保留断言和超时。最终标准浏览器运行及确切提交的代管 CI 是独立关卡，本节不宣称已通过。

六语言整合后的兼容性检查必须覆盖每一种已发布来源语言及每一种请求输出语言的旧版内置规划器／工作器提示词与 `add_intent` schema。新增 zh-Hant／ja／es 测试数据来自不可变的 PR40 之前 dev `7af06f1`。精确识别必须保留已编辑提示词、自定义 schema 字段／默认值、已存行与原始历史。这扩展了下方原先的三语言验证。

<a id="patched-sdk-and-model-specific-context-windows"></a>

### 已修补 SDK 与各模型的上下文窗口

后端 CI 工作必须运行嵌套的 Norma 测试套件。请在根包之外单独运行 `(cd third_party/norma && go test ./... -count=1)`；详见[来源与修改声明](PROVENANCE.md#norma-execution-and-compaction-fixes--2026-10-25)。

回归集覆盖原始的 30 ms / 120 ms 工具中断竞态、小窗口节奏与长会话边界、有实质收益的摘要合并、真实溢出重试视图、单调学习上限，以及不得触发上下文恢复的输出错误。既有数值边界与原始历史保持不变。旧断言认为 40K 无法进入第二层，这在未改动的上游就已不成立；替代测试使用同一测试数据，要求真正执行合并、回收 token 并限制摘要大小。

`TestNoaRetainsConfiguredModelContextWindow` 实际运行 40K、128K、200K、250K、400K 与 1M 的 Noa 会话。主应用修正前，所有非 200K 情况都会变成 200K。`enableNoa` 现在先传入已解析配置中的正数窗口，再清除被替换的压缩器。其他测试分别覆盖窗口缺失时的回退、停用 Noa、损坏状态恢复，以及既有本地 Ollama 策略。`MaxTokens` 仍独立限制输出。200K 回退适用于未设置窗口的情况；不是通用模型上限。SDK 在 200K 及以上的默认值保持不变，而有实质收益的摘要按数量决定是否合并的条件则有意作了调整，该调整适用于所有上下文窗口，与窗口大小无关。

<a id="trigger-evidence-in-the-first-planner-request-2026-10-18"></a>

## 首次规划器请求中的触发证据（2026-10-18）

这是原始提交 `ffe260435fed6576394d613991aaf6e8010d3f48` 上的本地候选版，已回移共用 CLI 修正。通常，finding 或工作器完成事件会携带触发摘要及精简的 `graph_overview` 唤醒规划器；`prove_goal` 仍要求每个目标条件均有已记录证据。精简条目缺少完整的 finding／fact 字段，因此证明前读取 `node_detail` 是合理的。

[Planner.Plan](../../agent/planner.go) 的实际变更节录如下，省略了中间未变动的任务上下文设置：

```go
// Before
situational := renderTriggers(ts, triggers) + renderGraphOverview(tsx.graphOverviewData())
// After
overview := tsx.graphOverviewData()
situational := renderTriggers(ts, triggers) + renderGraphOverview(overview)
// After ensureRunDir and WithReviewContext:
situational += renderPlannerEvidence(ctx, overview, triggers, tools, ts.ID(), taskDir)
```

第一行新代码只捕获一次既有的有界概览；第二行呈现同一快照。最后一行在任务／审查上下文建立后，为具体 `finding`／`done` 触发 intent 所产生、属于目前 exploration 的 fact／finding 候选节点附上完整详情。它不遍历 `related_tasks`，也不扩展全局谱系。[附加实现](../../agent/planner_evidence.go)最多进行 **4 次读取尝试**，且仅当**整个附件不超过 32,768 个 UTF-8 字节**时才纳入完整节点，包含本地化标头、JSON 外层与 `omitted_count`；过大的节点会被省略，不会截断。重复候选会去重。

只有最终解析出的 `node_detail` 能力可以读取：其 schema 必须接受输入，`IsReadOnly` 必须成立，权限必须明确允许未改动的输入。包装器、默认值、计量、受限结果与错误均仍有效；不安全或不完整的结果会被省略，不会回退到原始辅助函数。标头在 EN／KO／zh-Hans 中均完整，并将记录内容标记为不可信观察。既有显式读取（包括关联的直接来源节点）保留原本的访问规则。

具体数据行来自 **CLI 所属的合成环回 R22 实验**，并非原始正式环境事件：intent `3` 完成一次自有 GET 并持久化 fact `4` 与 finding `5` 后，目标 `2` 仍未达成。目标要求 finding 名称精确为 `SYNTHETIC LAB ONLY public debug marker`、严重程度 `low`、标记与观察内容。在 **2026-10-17 22:42:31+09:00**，规划器为取得这些缺少的完整字段，合理地读取了 `node_detail` `5` 与 `4`。测试框架清理在证明前的 **22:42:54+09:00** 将其暂停；这不是人工暂停。R22 在未变动的 **420 秒任务预算**下，于**总计 470.50 秒**后失败。底层模型／供应商延迟的原因**未知**。修正后，同一快照候选中符合资格的完整详情会进入触发回合的首次请求；详情本身并不构成证明。

| 本地候选版检查 | 观察结果 |
| --- | --- |
| 原始完整 `agent` 包，无 overlay 的确切正式源代码 | 598 个测试／子测试通过，0 失败／略过；等效 CLI 为 607 通过，0 失败／略过 |
| 三个 `planner_evidence*test.go` 文件中的新增回归 | 31 个测试／子测试通过；实际 `Planner.Plan` 流式／非流式首次请求相等、九种权限情况、数量／字节边界、去重／隔离、缺少／已删／延迟／隐藏证据、显式恢复、不自动证明与多个目标 |
| 原始聚焦服务器证明／语言区域检查及完整 `locale` 包 | 分别通过 5 与 19 项，0 失败／略过；保留 **1,986 个键 × 3 种语言**的严格完整性，因新标头将预期数量由 1,985 更新 |
| 原始 `go build ./...`、`go vet ./...` | 两者原生退出码均为 0 |
| Linux `-race`，目前等效的 CLI 源代码 | 聚焦检查 49 项与完整 `agent` 包 607 项通过，0 失败／略过；两者实际 Go 退出码均为 0，无 `DATA RACE`。已验证全部 1,175 个快照源文件；这不是原始模块运行 |
| 真实封装 CLI 环回验收，全部五种情况 | core 19.14s、GLM LOW 82.63s、Flash LOW 114.84s、OAuth LOW 108.12s 与 Flash HIGH 317.83s 原生退出码均为 0，各自使用未变动的 420 秒任务预算。这些单次实验运行不是性能比较 |

未改动的原始基线在流式及非流式缺少上下文断言上失败（两个叶节点及其父项失败），并非设置失败。回归测试数据中的受控供应商用量为合成数据，不是付费 token 证据。仍须使用真实的 `prove_goal`、真实用量／回执与既有终结证明关卡；阶段、工作器、努力级别、截止时间与重试行为均不变。反例：停用、受限、缺少、过大、未触发或继承的节点不会自动获得完整详情。可用时，显式读取可以恢复被省略／新产生的证据；单凭读取数据不能完成目标，证明多个目标中的一个也不能完成任务。

以上为本地工作树结果，并非确切新提交的 CI。未改动的 Windows `TestChatBranchCopyRejectsDestinationSymlink` 权限失败也在 CLI 基线 `650dc82910e5e36668161a5276455506f64713fe` 重现；未放宽任何检查，也不宣称完整原生服务器成功。新候选版全部七项必要代管检查仍待完成。PR #40 仍未合并，最终人工审查待完成；本节不宣称已公开或发行。不引入公共 API、schema 或迁移变更。较早日期的章节保留其历史结果。

<a id="chat-context-readiness-2026-10-17"></a>

## 对话上下文就绪状态（2026-10-17）

这是基于 PR #40 提交 c9fb642 的本地界面候选版。检查器可以在首次 controls 请求完成之前打开。先前它会将尚未解析的会话视为草稿，再将首次解析出的会话当作切换而关闭面板。在[检查运行 39031048389](https://github.com/cskwork/scopeweaver/actions/runs/39031048389) 中，中文桌面任务案例因关闭按钮先不稳定、后脱离文档而失败。该代管提交仍是失败证据；本次新结果为本地结果。

重设逻辑节录（binding 是请求的 controls 键；session 是解析出的会话标识）：
~~~ts
// Before: every change from draft to resolved identity ran the reset.
epoch.current++;
setOpen(false);
// After: qualify the change before the existing reset.
const previous = hydrated.current;
hydrated.current = { binding, session };
if (previous.binding === binding && !previous.session) return;
epoch.current++;
setOpen(false);
~~~
第一行新增代码读取先前的 binding 与 session；第二行记住目前这组值。第三行仅在同一 binding 尚未解析出首个会话时保留已打开的面板。不同请求 binding 或后续已解析会话变化，仍会增加 epoch 并完整重设视图／表单，包括焦点与保留数量。hook 暴露既有请求键；没有服务器 API 或翻译标签变更。

具体受控数据行：任务 1、exploration 1、segment 0 通过真实任务 API 暂停。其首次 controls 响应被保持等待，直到 /context 打开。先前释放该真实响应会使触发按钮关闭，导致正常点击在 15 秒后超时（Go 退出码 1）；现在逐字节相同的浏览器判定检查会保留打开的触发按钮并正常关闭（退出码 0），供应商调用为零。任务暂停有别于对话关卡的 paused 字段。

| 冻结修正源代码上的本地检查 | 观察结果 |
| --- | --- |
| 既有前端测试 | 65 通过，0 失败／略过 |
| 全新静态前端构建 | TypeScript 与 31 页完成；已验证原生退出码 0 |
| 未改动的独立首次响应延迟判定检查 | 1 个 Go 测试通过，0 失败／略过；供应商调用为零 |
| 标准上下文浏览器测试数据 | 全部 12 个 EN/KO/zh-Hans × 1440/390 × 独立／任务案例，加上 hydration 与请求 binding 防护均通过（14 个浏览器步骤；1 个 Go 测试，0 失败／略过） |

新增防护先使用真实的首次后端响应，再使用合成路由状态来检查后续已解析会话变化，以及被保持等待的旧上下文响应。每个替代响应均保持等待，直到旧内容断言完成，避免新结果掩盖泄漏。非默认焦点与保留数量 7 会重设为空焦点与默认值 2。反例：真实请求 binding 或后续已解析会话变化仍须重设；延迟结果不得恢复旧上下文。第二次真实任务查询切换（完整运行中从任务 7 到 8）在新 controls 响应保持等待时观察到了新 URL，保留同一个仍连接的检查器触发按钮与文档，重设焦点／保留数量，并在替代数据可能掩盖问题前拒绝延迟旧上下文。这证明了已挂载状态下请求 binding 的变化，而非重新挂载。延迟错误／压缩成功分支仍仅经源代码审查，未单独运行。合成标识防护证明的是客户端处理，不是服务器 exploration 替换。其他上下文案例使用受控环回供应商；未调用任何付费／外部供应商。

最初的 shell 包装尝试因环境／捕获设置而停止；最终构建／测试退出码另行记录，保留警告，且未以源代码／断言变通来通过。先前规划器／关闭变更予以保留。配对文件记录这些本地结果；下一次已提交候选版仍须独立最终审查及全部七项代管检查。PR 就绪不等于维护者批准、合并或公开发行。

<a id="log-writer-shutdown-ownership-2026-10-17"></a>

## 日志写入器的关闭归属（2026-10-17）

服务器将日志保存在内存中，并通过异步数据库写入器持久化。先前取消服务器上下文时不会等待该写入器结束：使用后台上下文的插入可能在 `Manager.Close` 关闭连接池时仍然活跃。PR #40 提交 `14b5403` 的最近后端运行，在未改动的连接埠占用关闭测试中揭露了此问题。

写入器与管理器之间的流程节录：
```go
// Before: writer, then manager shutdown
pg.InsertLog(level, tag, text) // background context
return m.pg.Close()           // no writer join
// After: writer, then manager shutdown
pg.InsertLogContext(writeCtx, level, tag, text)
logSink.StopDB(m.pg)
return m.pg.Close()
```
第一行新代码使实际插入可取消；首次历史恢复也使用同一个上下文。第二行只解除这个确切连接池的绑定，取消未完成 SQL，并在接收器锁之外等待写入器结束。每次同一拥有者的停止操作都共用该完成信号；另一个管理器的连接池不能停止它。只有之后第三行才关闭连接池。既有不带上下文的 DB 方法委派给同一 SQL；环形缓冲区、stderr 与 SSE 传递仍可用，入队通道从不关闭。

一条观察数据：后端运行 [39028154916](https://github.com/cskwork/scopeweaver/actions/runs/39028154916)，job `117383132313`，在 **2026-10-17 11:18:45 UTC** 的 `TestMainServeFailureRunsShutdown` 失败。预期的连接埠占用绑定失败之后出现关闭及 `[logsink] db write: sql: database is closed`。另一个独立真实 PostgreSQL 表锁检查，在未改动的 `14b5403` 上重现了 `Manager.Close` 返回时其写入器仍阻塞（Go 退出码 1）。现在完全相同的测试数据会在锁仍持有、连接池尚未关闭时通过；没有过滤任何诊断。

| 原始本地 R19 检查 | 观察结果 |
| --- | --- |
| 未改动的独立阻塞插入回归 | 1 通过，0 失败／略过；保留基线行为 RED |
| 真实 SQL 拒绝防护 | 1 通过，0 失败／略过；可见 SQLSTATE `23514`，被拒行不存在，后续有效行已持久化 |
| 冻结的 `TestStopDB*` 测试数据 | 6 通过，0 失败／略过：无绑定、阻塞插入、其他连接池、已取消上下文、并发停止与 stderr 连续性 |
| 未改动的连接埠占用关闭测试 | 1 次运行加上 20 次重复运行通过；保留预期子进程失败，父 Go 退出码 0 |
| `go build ./...`、`go vet ./...` | 两者退出码均为 0 |

反例：真实的活跃上下文 SQL 拒绝仍必须报告；取消／等待结束并不会将它变成成功插入。这仍是尽力而为的日志记录，没有为队列中的行新增刷写保证。并发测试数据检查总体完成；每个等待方共用等待结束的行为也经源代码审查验证，不宣称为单独的运行时判定检查。全部数据库运行均使用可抛弃测试数据库；独立测试数据现在使用所提供可抛弃数据库中的自有 schema，不需要 CREATE DATABASE 权限。正式权限、HTTP／schema 合约与既有规划器测试不变。这些为本地结果，不是完整原生服务器或发行声明：继承的 Windows 符号链接权限限制仍存在，下一次已提交候选版必须通过全部七项代管检查。

<a id="terminal-goal-completion-2026-10-17"></a>

## 目标最终完成（2026-10-17）

以下为本地 R17 候选版结果；最终独立源代码审查与确切候选版的代管关卡另计。R17 共用修正让已完成的证明回合沿正常 SDK 路径走完，再于另一上游请求前在本地结束规划器。通常，`prove_goal` 将已确认的 fact／finding 关联到目标，并标记该目标已达成；只有全部目标均已证明才设置 `GoalMet`。先前 `Plan` 会等待 `captureRun` 完成后才将该标记返回引擎，因此会话等待另一次模型响应时，引擎无法完成。

控制流程节录（`...` 是未变动的活动回调）：
```go
// Before
_, _, err = captureRun(ctx, opts, input, ...)
return tsx.GoalMet, tsx.Reason, err
// After
installTerminalProofGateOn(&opts, tsx)
_, _, err = captureRun(ctx, opts, input, ...)
return tsx.GoalMet, tsx.Reason, err
```
第一行旧代码等待整个会话；第二行才对外提供目标结果。新增的一行在捕获开始前安装下一次请求的边界。它让目前响应、实际工具结果、延迟用量及对话记录完成，再提供不含内容、上游用量为零的本地 `end_turn`，并跳过证明后不必要的 `Compactor.Pre`。它保留所选压缩器、可选 `ContextView`、正常证明前／恢复行为，以及既有供应商／工具包装器。原始 `ToolUseID` 及成功／失败响应资格判定，防止延迟处理程序将旧失败证明归属于后续成功响应。最新证明来源通过 compare-and-swap 保留；较旧或未知来源不能抹掉较新的有效证明。失败记录在产出错误前执行，即使其消费者立即停止亦然。

具体实验数据行：在**独立 CLI R11 候选版**中（并非原始正式环境事件），finding 节点 `5` 于 **2026-10-17 18:36:24+09:00** 证明目标 `2`。未变动的 420 秒测试框架在 84 秒后的 **18:37:48+09:00** 暂停时，模型输出仍活跃。该次运行恰有一个 GET、一个 intent 与一个 finding。回执为 `scopeweaver-cli/dist/pi-glm-flash-fix-20261017/r11-flash-high-digest.txt`；它说明了完成等待，不能说明底层供应商延迟原因。反例：部分／被拒证明必须正常继续；失败证明响应保留其真实错误／恢复路径，不能赋予本地完成资格。

原始源代码证明使用受控供应商与可抛弃 PostgreSQL，不使用付费推理：

| 本地候选版检查 | 观察结果 |
| --- | --- |
| 完整 `agent` 包，R17 使用已修正的自有 fact 测试数据 | 567 个测试／子测试通过，0 失败、0 略过；包含流式／非流式、部分／被拒证明、父上下文取消、错误／恢复、压缩／视图及延迟来源案例 |
| `TestWireTools`、内置运行时 S7 与实际引擎 S18 | 28 通过，0 失败、0 略过；S18 在一次真实供应商调用后持久化 done，具有真实证明／终结回执，以及精确的输入／输出／缓存读取／缓存写入用量 **11/7/3/2**，原始对话记录亦包含这些数据 |
| S18 中真实暂停／供应商错误防护 | 如实的错误／原因回执保留已发出用量：用户暂停为 **11/0/3/0**，供应商失败为 **11/7/3/2**；观察证明后暂停与任务完成的状态优先次序，未新增策略断言 |
| 完整 `locale`、构建与静态分析 | 19 个测试通过，0 失败、0 略过；`go build ./...` 与 `go vet ./...` 退出码 0 |

已保存的原始 S18 基线失败，因为它尝试了第 2 次供应商调用，且没有终结结果／完成；其取消／供应商错误回执防护通过。独立不可变 R14 回归也曾失败：旧失败处理程序借用了后续响应的标识，导致 calls=2 而非 3；候选版保留了三次调用回归。审查者随后在原始 R16 候选版重现相反顺序：旧的失败 response-1 处理程序在有效 response-2 证明之后完成，抹掉了其来源，导致不必要的第三次请求。未改动的实际会话 overlay 在 R17 前以 calls=3 失败，现在以 calls=2/origin=2 及一份真实新证明结果通过。两种延迟处理程序顺序均通过[边界回归](../../agent/terminal_boundary_test.go)；[实际引擎回归](../../server/engine_terminal_proof_test.go)另行保留完成／回执／用量检查。首次整合 agent 运行失败，仅因新测试数据预期 `CreateExploration` 会创建一个来源 fact。R16 自行建立已确认 fact；这次测试数据修正没有改变任何断言或正式代码。较早失败回执仍予以保留。

S18 现在复用 S7 在所提供可抛弃数据库内的自有 schema 辅助函数，在删除自己的 schema 前关闭存储区；拥有数据库的 NOCREATEDB 测试角色就足够，正式权限保持不变。未改动的 `TestChatBranchCopyRejectsDestinationSymlink` 仍在本地 `server/chat_history_test.go:270` 失败，因为 Windows 拒绝创建符号链接。不宣称完整原生服务器成功或已通过确切候选版全部七项代管检查；CI／前端／浏览器交付仍待完成。根协调方负责另行进行竞态／真实运行验收；本节仅报告这些原始本地检查。

<a id="plannertool-contract-and-stored-stock-compatibility-2026-10-17"></a>

## 规划器／工具合约与已存内置内容兼容性（2026-10-17）

这是 `fix/glm-agent-contracts` 基于 `dev` 提交 `d7365ec` 的本地 Windows 结果。规划器的 `parent_ids` 指引现在与既有仅接受 fact／finding 的处理程序一致，批次 `add_intent` 元数据说明了字段与必要的 `summary`。规划器／工作器指引保留全新验证与重测，要求每个目标条件都有证据，且仅在成功派发或确认最终目标证明后结束回合。EN／KO／zh-Hans 资源均有配对。

完全相同的旧版内置规划器／工作器提示词与 `add_intent` schema 会解析为修正后的运行时默认值。用户编辑、绑定、启用标记、已存行与提示词历史均不变；没有正式数据库迁移，也没有处理程序语义变更。详见[合约回归](../../agent/planner_contract_test.go)、[内置内容兼容性回归](../../agent/legacy_stock_compat_test.go)及 [HTTP／运行时 PostgreSQL 证明](../../server/tools_stock_schema_test.go)。

测试先行的证明成功编译，并在基线上产生 **8 个顶层 agent 断言失败**与 **9 个 S7 内置 schema 子案例失败**。仅为编译测试而临时使用的未变动尾部提取与直通解析器，在应用修正前已移除。候选版结果：

| 本地检查 | 观察结果 |
| --- | --- |
| 完整 `agent` 包与可抛弃 PostgreSQL | 549 个测试／子测试通过，0 失败、0 略过 |
| 完整 `locale` 包 | 19 通过，0 失败、0 略过 |
| 聚焦服务器语言区域／工具／来源检查 | 11 通过，0 失败、0 略过；1,985 个正式键保留原始三语言完整性检查 |
| `TestExplorationDirectSourceReadView` | 使用可抛弃 PostgreSQL 通过 |
| `TestStockMetadataRuntimePostgres` 加 `TestWireTools`，`-count=2` | 48 个测试／子测试通过，0 失败、0 略过；实际环回 HTTP 处理程序、运行时解析及已存行／历史回读 |
| 已保存的公开工具情境 | S11：全部 10 个主智能体／规划器回调案例通过；S12：历史谱系、来源及无关／间接父节点拒绝通过；S3：无效 kind／空白 summary 拒绝、部分错误及省略／空值／null 来源关联通过 |
| `go build ./...`、`go vet ./...`、差异空白检查 | 通过 |

`TestStockMetadataRuntimePostgres` 与 `TestEngineTerminalProofPersistsAuthenticReceipt` 要求显式提供选择 `scopeweaver_test_*` 的可抛弃 `ARTEX_TEST_DSN`。拥有数据库的测试角色可以具有 **NOCREATEDB**，与 `make test-db` 一致。每份测试数据创建自己的 `stock_*` schema，并将所有池化连接的 `search_path` 仅设置为该 schema，保留所选数据库与其他连接选项。测试在使用 `CASCADE` 仅删除自己的 schema 前关闭存储区；清理错误会报告，但删除 schema 本身无法侦测空闲泄漏连接。`TestStockFixtureIsolationAndCleanup` 检查多个池化连接、父表不可见／保留及 schema 删除。可用 `go test ./server -run '^Test(WireTools|StockMetadataRuntimePostgres|StockFixtureIsolationAndCleanup|EngineTerminalProofPersistsAuthenticReceipt)$' -count=2` 重现隔离。正式数据库权限与 schema 不变。这项测试数据修复取代了先前对子数据库／CREATE DATABASE 的要求。

首次 S3 测试框架尝试在建立 hint 时使用无效状态 `open` 而失败；在通过的重跑前，测试数据修正为既有 `active` 状态，没有产品／断言变更。最终 S7 设置在其他测试填入父数据库后也通过了两次。这些检查确立合约与兼容行为，不能说明先前真实 Flash 重复／超时的原因。此分支完整 `go test ./...`、前端／浏览器检查及确切候选版全部七项必要 CI 工作仍待完成；不宣称已合并或发行。

<a id="shared-runtime-fixes-from-standalone-cli-verification-2026-10-17"></a>

## 独立 CLI 验证发现的共用运行时修正（2026-10-17）

候选分支 `fix/cli-discovered-runtime-bugs`，基于 `dev` 提交 `3f0ef25`。以下为本地结果；PR 确切提交的托管检查另计。此变更未将独立 CLI 或订阅供应商加入原始应用程序。

- HTTP 自定义工具现在使用 `Manager.ProxyAddr` 传回的完整记录代理 URL。先前 `"http://" + addr` 将测试的 `http://127.0.0.1:<port>` 变成 `http://http://127.0.0.1:<port>`，尚未抵达代理便失败；现在 `proxyStr = addr` 保留 URL。真实直接、明确配置与自动代理路由皆通过；自动路由修正前失败。
- 启动时先确认既有对话记录路径是目录再读取。Windows 一般文件回归修正前失败，现会拒绝路径并保留文件；不存在的目录仍有效。新诊断提供英文、韩文、简体中文。
- 关闭时先取消工作、排空 HTTP，再等待已准入对话、侧问与快照发布，最后关闭保存区。先前仅排空 HTTP，可能让背景结束处理在数据库关闭后写入。最终快照／上下文写入错误现会向上传递、保留待写项目，并使进程明确关闭失败；已删／冲突记录仍视为无害。此有界排空不宣称涵盖所有排程器或引擎生产者。
- 取消用量测试数据现创建既有选用用量表，符合正式初始化。修正前重现缺表失败；修正后全新数据库测试保留精确用量断言。

真实 PostgreSQL 关闭检查通过 11 个测试／子测试，含关闭重开持久化与注入最终写入失败。主进程通过 4 项，涵盖正常结束 0、重启 75、有界失败及真实连接埠占用服务失败的清理。聚焦代理／启动／语言区域回归通过。完整 Windows 后端为 1,975 通过、2 失败、15 略过；一项为未改变的符号链接权限限制，另一项揭露目录预期数，因新增两诊断由 1,980 改为 1,982，未移除完整性检查。修正目录另行验证；其余平台结果须由 Linux／托管检查创建。

前端 65 测试与静态导出通过，既有 POSIX npm 脚本使用 Git Bash。适配器 7 测试与语法检查通过。聚焦后端静态分析通过。未使用用户数据库、真实线上目标或通知；各写入测试均创建并删除独立 `scopeweaver_test_*`。不宣称合并或发行。

<a id="chat-controls-and-context-2026-10-16"></a>

## 对话控制与上下文（2026-10-16）

[PR #29](https://github.com/cskwork/scopeweaver/pull/29) 作者针对上游 `6366429` 回报以下检查。这是 PR #29／#30 合并审查前的贡献者历史 Linux 结果，不证明后续候选状态或 Windows 附件复制行为。

| 检查 | 结果 |
| --- | --- |
| 可抛弃 PostgreSQL 17 竞态包（`db`、`server`、`agent`） | 667 顶层测试通过，无竞态报告 |
| Go 包与静态分析 | `go test ./... -count=1`、`go vet ./...` 通过 |
| 前端测试、翻译、审计、静态导出 | 65 测试；3,263 EN／KO／zh-Hans 键对等；0 审计问题；TypeScript 与 31 页通过 |
| 后端翻译 | 1,978 正式键在三语通过完整性、对等与 printf |
| 对话浏览器关卡 | EN／KO／zh-Hans、1440／390 px、独立／任务对话共 60 情境通过，无未捕捉页面错误 |
| 内嵌应用基本测试与仪表板 | Smoke E2E 与 14 仪表板情境通过，无未捕捉浏览器错误 |
| 智能体适配器 | 7 单元、语法及 3 MCP／Pi 情境通过 |
| 安装与封装 | 8 安装数据、11 Compose 案例、每语 EN／KO 各 5 封存档、6 发行声明测试通过 |

对话浏览器案例涵盖队列编辑／排序、重启确认、重试／编辑／分支、上下文检查／压缩、工具引导与文件贴上。后端回归涵盖已接受语言持久化、预检错误、中断工具交换、放弃分支准备与文件读取归属。

可将模型目前上下文与计费对话用量分开检视。压缩保留您选择的近期完整交换；完整显示历史与之前尝试用量仍保留。

贡献者回报一次审查找出六个整合缺陷，之后修复生成输入准入与复测回滚、压缩后附件路径、旧版显示历史及引导投递控制。一项附件测试在子分支下次提示前删父分支，另一项检查 rebase 保留 manifest 字面文字。各修复前均回报失败回归，之后完整本地重跑。仪表板推理总数断言限缩至摘要列，因悬停提示有相同标签；重跑回报 14 仪表板情境通过。

回报推论测试使用环回模型与可抛弃数据库。Windows 运行、付费供应商、远程 CI 需另验证。声明测试使用 Python 3，符合 CI 解释器需求。

<a id="native-hardening-candidate--2026-10-15"></a>

## 原生强化候选版 — 2026-10-15

在基于 main `33b131e` 的 `feat/ollama-zero-egress-hardening` 进行本地与独立检查。这是候选结果，确切提交的远程 CI 与 PR 交付仍待完成。以下历史段落保留原始结果。

| 实际运行检查 | 结果 |
| --- | --- |
| 独立 Go 运行，20 包各用隔离可抛弃 PostgreSQL | 1,366 测试／子测试通过、0 失败；`enrich` 无测试；2 选择性略过（`TestAgentAdapterE2E`、`TestLiveContextReview`） |
| 另测智能体适配器 | 7 单元及 3 真实 MCP／Pi 情境通过；刻意不运行外部真实供应商审查 |
| 聚焦后端 `-race` | 传输、wire／用量、上下文、来源选择／失败、搜寻／更新、准入皆通过，无竞态 |
| `go vet ./...` | 通过 |
| 前端单元／本地化与静态导出 | 45 通过；3,168 EN／KO／zh-Hans 键保留对等、占位符、rich tag；TypeScript 与全部 31 页通过 |
| 后端目录 | 1,943 正式键通过原始三语完整性／对等与 printf |
| 安装、Compose、声明 | 8 安装、11 Compose 凭证、EN／KO 每语 5 封装测试封存的声明检查、Python 3.12 的 6 发行声明测试通过 |
| 静态构建与依赖检查 | macOS 内嵌与 Windows amd64 交叉构建通过；`go.mod`／`go.sum` 无漂移；未运行原生 Windows |
| 内嵌浏览器基本测试 | 最终 UI 重建后 60 通过（48 smoke + 12 仪表板）、0 失败。EN／KO／zh-Hans 稳定桌面／行动画面及水平边界通过：390px sheet [0,390]，1440px [864,1440]，全部 20 项关键内容可容纳。已重现并修正先前 420px 最小宽裁切 |

[原生 HTTP 优先级回归](../../server/ollama_priority_test.go) 将 worker 1 留在 `/api/chat`，排队 worker 2／3，再呼叫真实接线裁决，不由呼叫端标记紧急。`-race` 通过，HTTP 顺序 **worker 1 → judge → worker 2 → worker 3**，峰值并行 1，原生裁决 schema／verdict 有效，全部完成且无名额泄漏。既有队列测试涵盖 FIFO 角色、取消与降低上限。

[原生来源失败回归](../../server/provider_origin_failure_test.go) 修正前失败：原生工具待运行后流式出错，可能继承全域云端 `allow`。修正后 Stream／Complete 与终止路由错误保留具体原生配置、90 秒／`ask` 模型政策、300 秒／`deny` 人工审批到期，不呼叫审查者。实际云端容错仍清除原生偏好。回归与受影响竞态包通过。

已验证 API 对不转发请求的本地代理切换严格模式 **false → true → false**。四种搜寻后端 `ddgs`、`brave-free`、`tavily`、`deepseek` 仅在关闭时到达该接收端，严格模式各新增请求为零。严格更新检查／应用遭封锁，本地状态可用。[更新回归](../../server/update_policy_test.go) 亦通过缓存、重新导向、即时本文读取，以及已准入异步下载被后续切换阻挡后的清理／失败完成。这不代表整台主机出口控制。

主机 Ollama **0.32.15** 使用既有 `qwen3:4b`、**4,096 token 输出预算**，通过原生诊断、decision／comment schema、结构工具呼叫／结果重放、用量与要求 32K（`/api/ps` **32,768**）。先前 256／512 预算失败仍保留证据，未丢弃或算成功。加载／速度／卸载数值只是一次热运行快照，非基准或驻留保证。未下载模型或改主机配置，不宣称自主任务完成或安全评估品质。

浏览器涵盖三语诊断／警告、严格更新状态、裁决加载／失败保护及既有仪表板／用量。视觉 QA 重现固定 420px 最小宽编辑器在 390px 视窗左溢 30px。最小宽现符合视窗；重建应用通过全部 60 项，六张桌面／行动画面标签可读，稳定内容在视窗内。独立内嵌重建与浏览器可执行文件逐字节一致。确切提交远程 CI 与 PR 读回另为交付关卡。不宣称合并、发行、付费供应商呼叫、外部目标扫描或真实通知。

<a id="provider-authentication-follow-up--2026-10-15"></a>

## 供应商验证后续 — 2026-10-15

基于已合并 main `38fde23` 的 `fix/provider-env-auth`，本地回归重现并修正两个需验证的本地网关失败。旧版 Anthropic、OpenAI Chat Completions、OpenAI Responses 的启动供应商密钥回退与空密钥连接测试皆回 HTTP 401；相同数据在修正后通过。

| 检查 | 结果 |
| --- | --- |
| [`FromEnv → NewProvider` 本地 HTTP 验证](../../agent/provider_env_test.go) | 21 情境通过：18 一般格式涵盖供应商密钥、文件密钥、现代／旧版覆写、明确空白；3 原生涵盖云端密钥隔离、Ollama 令牌、明确空白 |
| [通过本地 HTTP 数据的连接测试 handler](../../server/llm_gateway_test.go) | 16 情境通过：15 一般格式涵盖相符正规化服务、不同端点／格式／代理、保留非本地回退；1 原生拒绝继承启用云端密钥 |
| 两聚焦组 `-race` | 通过，无竞态 |

上述 **37 个案例**为叶情境，不含父测试。各修正前重现验证失败，之后同测试通过。非本地 handler 案例中的公开文件地址经明确本地测试代理，不联络公开供应商。

无 profile ID 的空密钥连接测试，仅在一般格式、正规化基础 URL、代理全相符时重用目前本地服务密钥，含新草稿。不同本地服务与原生 Ollama 保持隔离。空密钥新配置不导入环境凭证。

这些检查创建已测协定与凭证边界，不代表付费帐号可用或全部历史功能。修正后完整包、浏览器、CI、已合并 main、发行产物结果另为关卡。未呼叫付费供应商。

<a id="installed-model-selection-follow-up--2026-10-15"></a>

## 已安装模型选择后续 — 2026-10-15

在 `feat/ollama-native-flexible-providers` 本地验证新配置 Ollama 默认选择，取代先前模板行为：选 Ollama 现运行只读模型探索。下方历史 37 检查浏览器快照早于此行为；「无隐含请求」为历史断言，不是目前契约。

| 检查 | 结果 |
| --- | --- |
| 前端单元与本地化（`npm --prefix web test`） | 45 通过、0 失败 |
| 正式静态导出（`npm --prefix web run build:static`） | TypeScript 与全部 31 页通过 |
| 浏览器与已验证后端真实主机模型探索 | 未存新配置、同机 Ollama、空代理／密钥通过 |
| 内嵌可执行文件与全新 `scopeweaver_test_default_final` 完整扩展浏览器 smoke | 43 通过、0 失败（31 smoke + 12 仪表板）；涵盖 EN／KO 三模板安装默认、直接格式选择、云端兼容、仅元数据选择、失败／空探索、手改／拒绝延迟响应、重开草稿／既有配置，以及既有配置、登录、本地化、SSE、仪表板回归 |

主机 `/api/tags` 传回 `qwen2.5-coder:7b`、`qwen3:4b`、`embeddinggemma:latest`。浏览器选 **`qwen2.5-coder:7b`**，第一个声明 `tools` 的已安装模型。元数据为 `completion`、`tools`、`insert`、7.6B、Q4_K_M；`qwen3:4b` 亦声明 `tools`，`embeddinggemma:latest` 仅 `embedding`。

浏览器恰好送一次 LLM 操作 `POST /api/llm/models`，主机端点、无密钥／代理。新草稿关闭，未保存、启用、测连接或推论。模型／机器配置不变。这仅证明安装模型选择与元数据探索，不代表工具运行或自主可靠性。

<a id="historical-native-ollama-branch--2026-10-15-branch-local-not-a-release"></a>

## 历史原生 Ollama 分支 — 2026-10-15（仅分支，非发行）

`feat/ollama-native-flexible-providers` 开 PR 前的本地检查。远程 CI 另于 GitHub Actions 追踪；本节不代表发行、合并或 CI 结果。

| 检查 | 结果 |
| --- | --- |
| 20 个 Go 包各用全新可抛弃 PostgreSQL 16 | 1,197 测试／子测试通过；2 选择性略过（`TestAgentAdapterE2E`、`TestLiveContextReview`） |
| 另测适配器 | 3 真实 MCP／Pi 情境通过 |
| `go vet ./...` | 通过 |
| 前端单元／本地化 | 43 通过；英／韩／简体中文各 3,159 唯一键 |
| 后端消息目录 | 三语各 1,938 唯一键；printf／对等通过 |
| TypeScript／正式静态导出 | 通过，最终 health UI 后重建 |
| `npm audit` | 0 漏洞 |
| 构建 | 最终 health UI 后重建 `CGO_ENABLED=0` 内嵌可执行文件；Windows amd64 `CGO_ENABLED=0` 交叉构建通过，未运行原生 Windows |
| 真实内嵌可执行文件与全新 `scopeweaver_test_ollama_final_health` 的 `npm run smoke` | 37 通过、0 失败（25 smoke + 12 仪表板）。含 EN／KO 三 Ollama 模板无隐含请求、免密钥保存／启用、本地免密钥兼容 health 与缺密钥公开云端对比、云端格式／可编辑 GLM、模型元数据／工具警告、缺模型精确 pull 命令、工具／上下文／加载／速度诊断、默认关闭本地政策，以及既有三语配置、登录、语言、草稿、SSE、审批用量回归 |
| Compose 凭证 | 真实 `network_mode: none` 容器经环境与 `FromEnv`，11 案例通过 |
| 安装与封装 | 8 安装数据共通过，含英／韩选择；英／韩各 5 封存档通过 |
| 发行声明（Python 3.12） | 6 测试通过 |
| 独立后端安全／正确性审查 | 通过，含 DeepSeek 本地政策、即时 CIDR、404 分类 |

<a id="real-ollama-run"></a>

### 真实 Ollama 运行

主机 Ollama 0.32.15、`qwen3:4b`，经原生流式 API：

- SDK 层级流式检查的合成工具数据中，模型呼叫 `record_fact`，ScopeWeaver 在本地运行并重放结果，回合完成；这不是平台数据库的 `record_fact`。
- 要求 32K，`/api/ps` 回报分配 32,768。
- 精确用量：流式回合输入 186／输出 750；完成回合输入 826／输出 193。
- 真实 API 免密钥保存／启用通过。诊断 `tool_call_verified`、`context_verified` 皆 true，上下文 32,768、加载 2.225375 ms、29.231 token/s、延迟 18,958 ms。**仅一次热运行，非冷启基准或硬体泛化。**
- 缺模型的精确韩文 pull 提示及严格政策切换封锁公开 IP 通过。
- **平台任务 Worker Bash 证据**：task 1 的 `qwen3:4b`（要求 32K、分配 32,768；并行 1；keep-alive 30m；思考关闭；上限 4,096）发出原生 Bash `call_219yv72x`。活动 9（call）、10（`tool_result`）同 ID、无错误，输出含 `LOCAL_TASK_OLLAMA_OK`。帐本精确记 6,704 输入／4,096 输出；下一原生呼叫重放相同 `tool_call_id`。意图通过既有 `db.AddIntent` 手动植入，非 LLM 创建。输出亦含测试数据的字面 `\n` 跳脱，不据此推断换行处理或模型错误。可执行文件来自最终 UI／DTO 前的未提交分支源代码。

<a id="autonomous-task-reliability-is-not-validated"></a>

### 自主任务可靠性未验证

工具诊断通过仅证明传输路径，不代表模型可完成任务。

- `qwen2.5-coder:7b` 声明 `tools`，却将工具形状 JSON 写入 `assistant.content`，不是传输 `tool_calls`，因此无 Bash／事实工具运行；ScopeWeaver 正确保留纯文字。
- `qwen3:4b` 作主要代理时，长提示下反复思考并以文字写出预期标记。作任务 Worker 确实运行上述 Bash，但 360 秒时任务暂停，未 `record_fact` 或自主完成；一次成功工具呼叫不代表自主完成。

**不对任何本地模型宣称端对端自主完成或安全评估品质，7B 级模型亦未验证可作完整智能体。** [硬体分级](llm-providers.md#hardware-tiers) 为估算，非基准测试。

<a id="not-exercised"></a>

### 未运行项目

无付费云端呼叫、外部目标扫描、真实通知、发布、部署、发行或合并。Anthropic、OpenAI Chat／Responses、自定义网关经传输契约测试、回归与可编辑浏览器流程验证，不使用真实供应商帐号。测试数据库皆可抛弃，主机 Ollama 配置未改。选用 `local_llm_only` 默认关闭；服务器仍需另设 `OLLAMA_NO_CLOUD`。

<a id="running-database-backed-tests"></a>

## 运行数据库测试

`db`、`server`、`agent` 包需要 PostgreSQL，连接字串**只**从 `ARTEX_TEST_DSN` 取得，绝不回退应用程序的 `ARTEX_PG_DSN`／`config.json`。此设计防止 `go test` 修改运行中应用的即时／正式数据库。

- **未设 `ARTEX_TEST_DSN`**：全部数据库测试略过，不接触数据库。
- **`ARTEX_TEST_DSN` 设为可抛弃数据库**：包对该库运行。
- **解析至与应用相同数据库**（同主机、连接埠、库名）：任何测试连接前即快速失败，错误 `refusing to run tests against the live database; point ARTEX_TEST_DSN at a throwaway DB`。

最简单的运行方式：



```bash
make test-db
```



`make test-db`（通过 `scripts/test-db.sh`）创建由 `sw_user_test` 角色拥有的可抛弃 `scopeweaver_test_<pid>`，将 `ARTEX_TEST_DSN` 指向该数据库，再由 `ARTEX_TEST_DSN` 指定测试连接，运行 `go test ./db/ ./server/ ./agent/ -count=1`，并在结束时删除数据库，即使测试失败亦同。可覆写 `TEST_DB_ROLE`、`TEST_DB_HOST`、`TEST_DB_NAME` 配合本地 PostgreSQL。库名必须以 `scopeweaver_test_` 开头，适配器 E2E 可用 `scopeweaver_adapter_`。

若要针对自行管理数据库运行部分测试，导出 `ARTEX_TEST_DSN` 并直接呼叫 `go test`：



```bash
export ARTEX_TEST_DSN='postgres://sw_user_test@/scopeweaver_test_1?host=/tmp&sslmode=disable'
go test ./server/ -run TestSomething -count=1
```



<a id="v021-candidate--2026-10-11"></a>

## v0.2.1 候选版 — 2026-10-11

Windows 候选版以真实静态前端构建，针对可抛弃 PostgreSQL 16.15 测试。结果取代下方历史依赖与 Windows 运行限制；旧记录保留供追溯。

| 检查 | 结果 |
| --- | --- |
| 前端依赖审计 | 全严重度 0 漏洞，含开发依赖 |
| 前端回归／本地化 | 26 通过、无略过 |
| 正式静态导出 | TypeScript 通过，生成 31 页 |
| 原生 Windows Go、18 包 | 1,016 测试／子测试通过、0 失败、13 既有平台／选择性略过 |
| 原生 Windows 完整 server | 311 测试／子测试通过、3 既有选择性／平台略过 |
| 适配器 | 7 单元／语法通过；真实已验证后端与新库的 3 MCP／Pi 情境通过 |
| 本地安装 | 4 隔离数据皆通过：新安装、旧档替换、依赖失败、前端构建失败 |
| 后端翻译目录 | 1,868 正式键通过英／韩完整性与 printf 参数 |
| 通知隔离 | 确定性旧 Worker 回归修正前失败、之后通过；管理／通知选择连续 5 次通过 |
| Windows 证据 | 完整 PostgreSQL 包通过，含并行写入、提交失败、长路径 |
| Windows Python／技能档 | 真实 Python 与无法使用启动器的回退通过；巢状 Unicode 路径通过清单／读取 handler |
| 内嵌 Windows 浏览器 E2E | 8 通过：两语配置服务中断保护、配置／登录、同源 SSE、用量 API 隔离、两语用量卡、无未捕捉错误 |

Go 数量含父测试与子测试。13 原生略过为十个 Unix 自更新数据、一个缺符号链接权限、选择性适配器整合及真实供应商。适配器另验证，Unix 数据由 Linux CI 涵盖。未为产生结果停用测试。

回归重现原 Windows 目录同步失败、本地化分隔符不符、不可用 Python 启动器、通知 Worker 泄漏与配置清理顺序。E2E SQL 辅助现将选项置连接 URL 前；直接中断／恢复检查确认 Windows psql 运行陈述式。

发行关卡另要求合并前后的 `frontend`、`windows`、`backend`、`agent-adapter`、浏览器 `smoke`。发布结果、平台封存与校验值见 [v0.2.1](https://github.com/cskwork/scopeweaver/releases/tag/v0.2.1)；此来源记录不宣称后续工作或下载资产已通过。

保留既有 Python 解释器路径。若旧安装存入不可用启动器别名，请在配置选可运行解释器。原生 Windows 测试不证明 Windows 自更新或 macOS 运行。未使用付费供应商、外部目标或真实通知收件人。

<a id="historical-verification--2026-10-03"></a>

## 历史验证 — 2026-10-03

**2026-10-03** 记录的英／韩 ScopeWeaver 验证，基于上游 ARTEX `160fe13`，使用可抛弃 PostgreSQL 与本地数据。此为源代码交付，当时未发布二进位或容器映像。

| 检查 | 结果 |
| --- | --- |
| 前端回归／本地化 | 21 通过，TypeScript 通过 |
| 正式静态导出／Go 内嵌可执行文件 | 构建成功 |
| Go 回归、18 包 | 986 测试／子测试通过，1 继承失败，1 外部供应商略过 |
| 完整 server | 285 测试／子测试通过，仅选择性真实供应商略过 |
| 后端翻译 | 1,865 正式键通过英／韩完整性与 printf |
| 自撰中文审计 | 无未翻译自撰来源／文件／Go 注解，保留刻意兼容／数据例外 |
| 桌面浏览器 | 每语 24 路由，无缺标签或页面水平溢出 |
| 行动浏览器 | 两语仪表板、任务、发现、配置，390px 无页面水平溢出 |
| 真实应用画面 | 两语登录／通知表单检查；配置验证／初始化通过 |
| 语言切换 | 导览／状态更新，保留选择与未送草稿 |
| HTTP 本地化 | Markdown／CSV 导出、协商、偏好保存、无效语言拒绝通过 |
| GLM-5.3 契约 | 本地 HTTP／SSE 21 聚焦测试／子测试通过，含两端点、推理、工具重放 |
| GLM 模板浏览器 | 两语两模板，保留名称／密钥／代理；无保存、启用或供应商请求 |
| 封装／启动脚本 | Shell 语法、双语封装数据、10 监督案例通过 |
| 来源／依赖 | 原授权、依赖版本、Go 模组不变；文件链接与差异检查通过 |

Go 数量含父测试与子测试。各包用独立数据库，初始化与一般启动相同的补充计量表。原始测试与检查保持启用。

<a id="inherited-issues-at-the-time"></a>

### 当时继承问题

`TestGraphOverviewExpandsAssociatedCompanyScope` 在 ScopeWeaver 与未改上游皆因 `task scope missing: <nil>` 失败。测试预期图概览含详细范围，但上游 `06a43f3` 刻意精简。`75a706d` 改为 `TestGraphOverviewKeepsAssociatedCompanyScopeCompact`，检查精简概览并直接验证保存的公司范围。此后完整后端 CI 已通过。

上游验证曾在 `TestTaskMetadataPatchReturnsRenameAndPin` 出现间歇暂存目录清理失败；最终 ScopeWeaver server 运行通过，未以无关生命周期变更隐藏。

上游 mock 预览无通知元数据 mock，因此通知页报错。**真实后端的通知页与新增通道表单在两语皆通过**，验证未传送通知。

v0.1.0 记录 **14 个 npm audit 问题：1 严重、9 高、4 中**。框架／工具更新及移除未用 shadcn CLI 依赖，已于上方 v0.2.1 候选解决。运行 `cd web && npm audit` 查看 npm 软件包仓库的最新审计报告；新公告可能改变结果。

<a id="historical-scope-and-limits"></a>

### 历史范围与限制

- 选择性 `TestLiveContextReview` 需私人外部供应商配置，未运行。无付费模型、外部目标扫描或真实通知。
- GLM 测试仅证明本地契约，不代表帐号存取／供应商可用。Coding Plan 仅支持官方工具，ScopeWeaver 未列入，见[供应商配置](llm-providers.md)。
- 未验证 Docker 映像／Windows 批次运行；封装数据不代表真实 Windows 测试。
- 配置进入已验证路由，但一次等待 window load 逾时；之后以真实 API 确认初始化与新密码。
- 用户文字、保存证据、编辑提示词、传输识别码、旧剖析标记、Unicode 数据与上游原画面均保留。英／韩应用不翻译任意用户数据或第三方工具输出。
- `sidequestion/` 历史验证描述上游工作，不是新的 ScopeWeaver 运行。

目前远程检查见 [GitHub Actions](https://github.com/cskwork/scopeweaver/actions/workflows/ci.yml)。

<a id="chat-document-paste-2026-10-16"></a>

## 对话文件贴上（2026-10-16）

PR #29 记录对可抛弃 PostgreSQL 17 与本地后端的以下贴上检查。这是贡献者于 2026-10-16 回报结果；目前整合证据另记：

- 前端 52 测试与正式静态导出通过。
- 剪贴簿浏览器 12 组通过：独立／任务对话、EN／KO／zh-Hans、1440／390 px。
- 涵盖单一草稿拥有者的快速文件贴上、伴随文字、原生 3,000 字符插入／撤销、Unicode 预览、草稿恢复、上传重试、导览、mention、IME 与两种送出偏好。贴上未提交任何消息。
- 贡献者停用 React paste handler，确认因缺第二附件标签而失败，再恢复重跑。
- 后端 smoke 通过剪贴簿与 Ollama 环回数据，无未捕捉页面错误。

文件案例通过 paste event 提供浏览器 File 对象；文字与撤销案例使用 Chromium 剪贴簿与键盘贴上。
