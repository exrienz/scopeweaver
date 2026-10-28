[English](../../../sidequestion/VALIDATION.md) · [한국어](../../ko/sidequestion/VALIDATION.md) · 简体中文 · [繁體中文](../../zh-Hant/sidequestion/VALIDATION.md) · [日本語](../../ja/sidequestion/VALIDATION.md) · [Español](../../es/sidequestion/VALIDATION.md)

<a id="btw-verification-record"></a>

# `/btw` 验证记录

各语言版本请见本页上方导览。

上游历史记录。日期：2026-09-10。分支：`codex/btw-side-question`。基准：`8dae851b9b622f2ff2631f332fde9719d0b16fba`。这些并非新的 ScopeWeaver 结果。

使用隔离 PostgreSQL 测试数据库与数据目录。真实模型凭证仅注入该环境，未写入源代码或本记录。产品默认模型未变更。Go 1.26.3、norma v0.3.6、Next.js 16.2.9。

模型对话、返回对象、工程断言与 Qwen 审查存于 [validation-2026-09-10.json](../../../sidequestion/validation-2026-09-10.json)，不含 API 凭证。叙述文字已翻译，识别码与实测值保留。

<a id="engineering-checks"></a>

## 工程检查

| 范围 | 结果 | 证据 |
| --- | --- | --- |
| 结构化消息与工具参数深层复制 | 通过 | `TestCheckpointDeepCopyAndBoundaries` |
| 摘要／压缩请求不覆写；完整回复与终止状态发布；排除部分回复 | 通过 | `TestCheckpointDeepCopyAndBoundaries`, `TestSnapshotExcludesPartialStreamAndSelectsPoolMember` |
| 实际模型池成员身分 | 通过 | `TestSnapshotExcludesPartialStreamAndSelectsPoolMember` |
| 工具配对、20 次交换重放、预算裁减与溢位错误 | 通过 | `TestBuildRequestCompactionToolPairingAndBudget` |
| 主／侧并行与双向取消隔离 | 通过 | 阻塞供应商，`TestMainSideConcurrencyAndIndependentCancellation` |
| 无工具运行、流式／非流式、失败时可取得用量 | 通过 | `TestServiceNoToolsAndUsageOnFailure` |
| 真实 norma ChatAgent + 本地 Read 工具；主要对话记录／活动隔离 | 通过 | `TestSideActualChatCheckpointToolResultAndTranscriptIsolation`, 流式与非流式子测试 |
| 持久化、分页、幂等性、重启后部分回答 | 通过 | `TestSideHistoryIdempotencyPagingAndRecovery` |
| 清除／延迟写入竞态、父资源删除、版本比较 | 通过 | `TestSideClearLateWritersAndDeletedParent` |
| MainAgent／Worker 封存与恢复，v1／v2／v3 | 通过 | `TestSideTaskArchiveVersions` |
| 三种父 API、验证、拥有权、Worker 逻辑删除 | 通过 | `TestSideHTTPGlobalLimitTaskWorkerAndDeletion`, `TestSideCheckpointPersistsBeforeAdmissionAndRestart` |
| 忙碌主会话中的侧问；独立 SSE 重连／断线、取消与清除 | 通过 | `TestSideHTTPBusyIsolationClearAndReconnect` |
| 每个父资源一个并行请求／全域四个 | 通过 | 两个 `TestSideHTTP…` 测试 |
| 准入前保存快照、重启后追问、不捏造旧会话快照 | 通过 | `TestSideCheckpointPersistsBeforeAdmissionAndRestart` |
| 拒绝已删缓存配置或已变更模型 | 通过 | `TestSideRejectsDeletedOrChangedCachedProfile` |
| 封存前取消并等待最终回答／用量持久化 | 通过 | `TestSideTaskDrainPersistsBeforeArchive` |
| 流式消费者提早取消时仅记一次用量与侧问归属 | 通过 | `TestSideUsageRecordedOnceOnConsumerCancellation` |
| 重启恢复的 Worker／deadline 运行上下文发布新快照 | 通过 | `TestSideRestoredWorkerRuntimePublishesNewCheckpoint` |
| 相关包竞态检查 | 通过 | 下列命令 |
| TypeScript 与正式构建 | 通过 | `npx tsc --noEmit`, `npm run build` |
| 新增前端模组的 Biome | 通过 | `biome check`, 三个新模组 |

重现自动检查时，将 `ARTEX_PG_DSN` 设为独立可抛弃数据库，绝不用正式数据库：



```sh
go test -race ./agent ./db ./server ./sidequestion ./llmrec ./llmpool \
  -run 'Test(Side|Checkpoint|Snapshot|BuildRequest|Service|MainSide|CaptureRun|TaskArchive|CompleteForwards|StopIntent|CancelIntent)' -count=1
cd web
npx tsc --noEmit
npx biome check src/lib/side-questions.ts src/hooks/use-side-questions.ts src/components/side-question-workspace.tsx
npm run build
```



完整 Go 回归并未全数通过。两个既有 `server` 测试在暂存目录清理时因 `TempDir RemoveAll … directory not empty` 失败：

- `TestInheritedActivityDetailAndRelationDeletion`
- `TestTaskMetadataPatchReturnsRenameAndPin`

导出未修改基准，在同一隔离环境重跑 `server`，亦重现两项清理失败。基准另在 `TestCoreTaskLifecyclePG` 目标节点数断言失败；最终修改版服务器回归没有此失败。其他包、侧问案例与竞态检查通过。基准问题未算作验收通过，也未修改既有断言来隐藏问题。

Next.js 显示既有多 lockfile／工作区根目录推断警告；构建与全部页面生成仍完成。

<a id="browser-checks"></a>

## 浏览器检查

Codex 应用程序内浏览器连至隔离本地 Go 与 Next.js 开发服务器。桌面及 390 × 844 窄屏幕检查使用浏览器自动化与画面／记录审查：

- 一般对话中，`/btw` 在桌面侧栏同时显示主要与侧问内容。
- 追问正常；停止后保留部分侧问输出，主流程持续。
- 面板关闭时请求持续；重开恢复完整回答，刷新后空白 `/btw` 恢复历史。
- 窄屏幕 Drawer 输入、按钮、历史与关闭皆正常，无水平溢出。
- 清除需确认，移除历史但保留主要对话记录／快照。
- MainAgent 与两个 Worker 间切换问题时，标签与历史各自分离。
- 阻塞本地模型测试数据保持 Worker 运行。从主输入提交 `/btw` 并停止侧问后，Worker 仍显示即时运行与自己的暂停按钮，部分侧问回答亦持久化。
- 浏览器错误／警告记录为空。

受控测试数据独立于模型速度验证精确并行时序。最初两次 Worker 检查因任务／回答已结束而缺少有效并行视窗。修正测试数据后重复检查通过；前两次未算通过。

<a id="real-model-conversations"></a>

## 真实模型对话

优先模型 `grok-4.6` 在 OpenAI 兼容端点 `http://127.0.0.1:12580/tingly/openai` 的探测于 2.82 秒传回 HTTP 200、模型 `grok-4.6` 与 `READY`。既然可用，便未使用或验证 Tingly `glm` 或智谱 `glm-5.3` 回退。

| 情境 | 实际结果 |
| --- | --- |
| 主要运行期间询问资产、目标与标记 | 回答 `redhaze.top`、首页阅读／摘要目标、`BTW-REAL-0910`；16.97 秒完成 |
| 读取首页后询问工具证据 | 正确回答 WebFetch 200、curl 301 → 302 → 200 重新导向与标题；7.24 秒 |
| 要求侧问通过 Bash 创建文件 | 拒绝；未创建目标文件；7.74 秒 |
| 侧问完成后主要上下文不变 | 主要对话记录 SHA-256 与活动不变；侧问工具运行为零 |
| 实际停止／重启 Go 后追问 | 保留前三笔侧问记录；无须重跑主要智能体，即可从持久快照回答资产、标记与标题 |
| 新对话使用非流式 Grok | 正确资产与 `ATOMIC-0910`；返回／保存用量：input 11734、output 138、cache_read 11520 |

主要对话使用 WebFetch 与 Bash／curl 读取公开首页，最终到达 `https://id.redhaze.top/home`，标题为「RedHaze Technology RedHaze Group · Global integrated group portal」（译文）。Bash 保存本地暂存响应档，没有远程写入。此事已与侧问无工具运行分开验证。

主要对话记录校验值：`e7e61f135a4a120954b539f357e8c4205d7d5cd7460dcaf3dc0fd066463e1d00`。

用量限制：Tingly Grok 流式未返回用量。直接以 `stream_options.include_usage=true` 探测，传回 HTTP 200、12 个数据框及零个用量框。因此流式的零表示端点未提供用量，不表示免费。非流式用量及测试数据的失败／取消用量均正确持久化。

<a id="qwen-review"></a>

## Qwen 审查

审查模型 `qwen-flash` 在 OpenAI 兼容端点 `https://dashscope.aliyuncs.com/compatible-mode/v1` 传回 HTTP 200。它接收前三次真实侧问对话、主要工具证据与工程断言，返回 `verdict: accept`、`concerns: []`；判定回答与资产、标记、页面证据一致，拒绝工具亦符合限制。用量：prompt 6625、completion 312、total 6937。

审查未涵盖后续重启与非流式测试。Qwen 的「无写入」描述过于宽泛：如上所述，主要会话 curl 确实创建本地响应档。工程断言证明并行、零侧问工具运行与对话记录隔离；模型审查仅辅助评估回答品质。
