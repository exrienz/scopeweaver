[English](../traffic-evidence.md) · [한국어](../ko/traffic-evidence.md) · 简体中文 · [繁體中文](../zh-Hant/traffic-evidence.md) · [日本語](../ja/traffic-evidence.md) · [Español](../es/traffic-evidence.md)

<a id="multiple-traffic-records-as-finding-evidence"></a>

# 多笔流量记录作为发现证据

各语言版本请见本页上方导览。

发现详情支持跨页选取流量、配置角色与备注、排序及移除绑定。流量页亦可将多笔记录附加至既有发现。继承自其他任务的证据为只读，请到来源任务编辑。

「智能体自动绑定流量」默认关闭。通过 `/api/settings` 读取或修改 `agent_traffic_binding`。启用会增加请求／响应审查、工具呼叫与指示的 token 成本；下一个智能体回合采用新配置。关闭时隐藏自动绑定参数与补充绑定工具、不注入自动绑定指示，并拒绝既有运行中会话提出的新自动绑定。手动绑定、捕获、读取已保存证据与导出仍可使用。

启用时一般流程为：记录发现、自动启动报告智能体、验证并绑定流量，再依最新证据版本撰写报告。回报智能体在 `evidence` 保留验证命令、重要输出、既有真实流量 ID 与用途。报告智能体检查发现与运行记录，通过 `traffic_search`／`traffic_get` 验证流量、呼叫 `bind_finding_traffic`，再读取最新 `version` 后保存。停用时不会提供额外原始流量搜寻／读取工具或自动绑定，但仍可读取手动绑定快照并撰写报告。

为兼容性，呼叫端仍可通过 `report_finding.traffic_refs`／`evidence_hint_id` 明确立即绑定。绑定为选填：TCP 等非 HTTP 发现、未捕获流量或缺少精确记录时仍可回报。保留命令、记录及其他可验证证据，适当说明缺漏；不新增必填字段。提交的每个 ID 均须有效且本文完整。任何失败都会回滚整个绑定操作；回报时的明确绑定失败会回滚整次回报。再次附加既有快照不会重复绑定或覆写备注。

<a id="agent-identifiers-and-report-versions"></a>

## 智能体识别码与报告版本

`report_finding` 接受下列选填参数，阵列顺序决定初始证据顺序。



```json
{
  "traffic_refs": [
    {"traffic_id": "real-traffic-id", "role": "baseline", "note": "Normal account request"},
    {"traffic_id": "another-real-traffic-id", "role": "proof", "note": "Reproduction request"}
  ]
}
```



角色为 `baseline`（正常比较）、`proof`（发现证明）、`verification`（额外验证）与 `supporting`（默认佐证）。先用 `traffic_search`／`traffic_get` 验证真实记录。网域与时间仅缩小候选范围，不能证明任务归属。

指示参考 [CyberStrikeAI 发现回报指南](https://github.com/RuoJi6/CyberStrikeAI/blob/54d56774b8bd285817d16d48b70a4a5e6e0963f7/internal/app/vulnerability_tools.go)，并适配本专案选填绑定契约。不强制填写缺少流量的原因。绝不猜测 ID，亦不得只为取得捕获记录而重复探测。

响应第一行维持 `finding recorded: <exploration node ID>`。后续 JSON 提供独立发现记录 `finding_id`、探索节点 `finding_node_id` 与绑定摘要。

- `get_finding_traffic(finding_id)` 接受独立发现记录 ID，传回排序清单与 `version`。提供 `binding_id`、`side=request|response`、`offset`、`length`，可读取每段最多 8192 字节。
- `update_finding_report.finding_id` 仍接受探索节点 ID。`evidence_version` 应设为实际读取版本。生成期间证据若变更，旧版本写入将遭拒；须重新读取并生成。
- 未提供版本的旧报告呼叫不宣称涵盖既有流量证据。绑定、备注、角色或排序变更都会将既有报告标记为待更新。

启用自动绑定时，`add_hint`／`add_task_hint` 可在单一提示或批次 `hints` 阵列的各项保存 `traffic_refs`。规划器代表其他智能体回报时，可用 `evidence_hint_id` 从本任务提示选取参照，绝不可使用继承提示。系统不会从网域、时间或浏览历史推断绑定。提交失败不会创建部分发现，也不会提前启动报告生成。

使用 `bind_finding_traffic(finding_id, traffic_refs)` 为既有发现补上缺漏绑定，无须再次登记。`list_findings`、`list_task_findings`、`node_detail`、`get_task_node_detail` 同时提供 `finding_id` 与 `finding_node_id`；旧版 `id` 保留探索节点含义。

启动时仅为旧工具结构新增选填属性。原始默认流量工具绑定延伸至报告智能体，默认亦提供补充绑定工具。自定义绑定清单、提示词、描述与启用状态保持不变。最终工具组装完成后才附加指示：回报角色交接既有证据，报告智能体验证、绑定并撰写。无任务上下文的平台对话应将结构化提示交给任务智能体，勿直接回报。声明任务完成前先交接现有证据；缺少捕获不强迫等待。`report_finding` 失败不启动报告智能体。

<a id="api"></a>

## API

基础路径：`/api/exploration/findings/{finding_id}/traffic`，使用独立发现 ID。沿用既有验证；`context_task` 检查任务可见性与继承只读存取。

| 方法／相对路径 | 请求／响应 |
| --- | --- |
| `GET` | 排序摘要、证据版本、报告证据版本 |
| `POST` | 附加整批 `{"traffic_refs":[...]}` |
| `PATCH /{binding_id}` | `{"version":1,"role":"proof","note":"Description"}` |
| `DELETE /{binding_id}` | `{"version":1}` |
| `PUT /order` | `{"version":1,"binding_ids":["2","1"]}`；须包含完整清单 |
| `GET /{binding_id}` | 快照元数据与有界本文预览 |
| `GET /{binding_id}/body` | `side`、`offset`、`length`；`download=1` 下载全部原始字节 |

版本／排序集合冲突及封存期间写入传回 `409`；继承写入传回 `403`；绑定不存在或不属于该发现传回 `404`。流量／附件读取与验证失败皆明确报错。

<a id="storage-and-migration"></a>

## 保存与迁移

启动时幂等迁移 PostgreSQL：`traffic_evidence_snapshots`、`finding_traffic_bindings` 及 `findings.evidence_version`／`report_evidence_version`（默认 0）。不从历史叙述推断绑定。

快照保存原始流量 ID、捕获时间、URL、方法、状态、请求／响应头、本文长度与 SHA-256。本文位于 `<data>/evidence/blobs/<first-two-characters>/<hash>.bin`，独立于可抛弃的 `data/traffic`。多个发现可共用快照／本文。不提供更新快照内容 API；完整性不符会使读取与导出失败。

完整本文（含大型 blob 与旧版目录记录）在原流量写入锁下读取。先持久化并验证文件，再由单一 PostgreSQL 交易写入探索节点、意图关系、发现、快照与绑定。提交后才通知规划器。失败可能留下未参照文件，但不会留下部分业务记录。

PostgreSQL advisory lock `7337741004` 协调证据文件与 SQL 参照。封存排入队列后，任务列锁禁止修改证据。恢复从安装本文到提交元数据期间持有证据锁。删除发现会串连删除其绑定。

清理器每小时运行，仅回收至少 24 小时、无参照且不在进行中操作内的内容。一般流量清理不动证据目录。备份热数据时，请一并备份 PostgreSQL 与 `data/evidence`。

<a id="export-and-archives"></a>

## 导出与封存

Markdown 包含排序证据清单与版本；JSON 包含元数据；CSV 增加数量与绑定 ID。`md-zip` 保留发现 Markdown 并包含：



```text
evidence/<finding_id>/<binding_id>/
  manifest.json
  request.http
  response.http
  request.bin
  response.bin
```



Markdown 以相对路径链接消息。送出下载前，服务器复制附件、验证哈希、压缩并同步封存档，再读取每个 ZIP 项目检查 CRC。内容缺漏或损坏会使整次下载失败。完整附件保留原始二进位字节。

封存 v3 依发现绑定搜集快照与本文，不依赖原始流量或网域。封存验证完成后才清理热数据；共用证据保留。恢复先验证并安装本文，再以交易恢复元数据与绑定，支持失败重试。v1／v2 仍可恢复，缺少的新字段明确设为 0。

<a id="verification-and-boundaries"></a>

## 验证与边界

各包通过 `ARTEX_PG_DSN` 使用独立且全新创建的 PostgreSQL 测试数据库，避免残留任务／模型数据启动背景工作。运行完整相关包，确认未因缺少配置而略过：



```sh
# Set ARTEX_PG_DSN to the package's isolated database before each run.
# Explicit configuration failures must fail the test.
go test ./<package> -count=1
go test -race -p 1 ./evidence ./db ./agent ./server -run 'TestEvidence|TestFindingTraffic|TestFindingEvidence|TestReportFindingAtomicContract|TestTaskArchive'
```



前端检查包括 `npx tsc --noEmit`、受影响文件的 Biome、Webpack 构建与 `NEXT_EXPORT=1` 静态导出。使用独立缓存目录，避免覆写运行中的开发服务器。

本地端对端验收使用专用连接埠、受控 HTTP／以网域为基础的 HTTPS 目标及暂存数据目录。涵盖两个绑定入口、跨页选取、排序／备注、错误、继承只读、下载，以及删除原始流量、封存、回收热本文、恢复与哈希验证后的导出。

初版采全域证据协调锁。大型绑定／导出或缓慢附件下载可能使其他证据操作等待。不得捏造缺少的捕获或不完整本文。本功能不改变捕获配置，也不改变直接连至 IP 地址的 HTTPS 凭证处理。
