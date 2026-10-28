[English](../../../../../skills/playwright-cli/references/tracing.md) · [한국어](../../../../ko/skills/playwright-cli/references/tracing.md) · 简体中文 · [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/tracing.md) · [日本語](../../../../ja/skills/playwright-cli/references/tracing.md) · [Español](../../../../es/skills/playwright-cli/references/tracing.md)

<a id="tracing"></a>

# 追踪

各语言版本请见本页上方导览。

捕获详细运行追踪以供侦错与分析。追踪包含 DOM 快照、画面、网络活动与控制台记录。

<a id="basic-usage"></a>

## 基本用法



```bash
# Start trace recording
playwright-cli tracing-start

# Perform actions
playwright-cli open https://example.com
playwright-cli click e1
playwright-cli fill e2 "test"

# Stop trace recording
playwright-cli tracing-stop
```



<a id="trace-output-files"></a>

## 追踪输出文件

开始追踪时，Playwright 会创建含多个文件的 `traces/` 目录：

<a id="trace-timestamptrace"></a>

### `trace-{timestamp}.trace`

**操作记录**：主要追踪档包含：
- 每个操作（点选、填写、导览）
- 每次操作前后 DOM 快照
- 各步骤画面
- 时序信息
- 控制台消息
- 源代码位置

<a id="trace-timestampnetwork"></a>

### `trace-{timestamp}.network`

**网络记录**：完整网络活动：
- 所有 HTTP 请求与响应
- 请求头与本文
- 响应头与本文
- 时序（DNS、连接、TLS、TTFB、下载）
- 资源大小
- 失败请求与错误

<a id="resources"></a>

### `resources/`

**资源目录**：缓存资源：
- 图片、字型、样式表、脚本
- 重放用响应本文
- 重建页面状态所需素材

<a id="what-traces-capture"></a>

## 追踪捕获内容

| 类别 | 详情 |
|----------|---------|
| **操作** | 点选、填写、悬停、键盘输入、导览 |
| **DOM** | 每次操作前后完整 DOM 快照 |
| **画面** | 各步骤视觉状态 |
| **网络** | 全部请求、响应、头部、本文与时序 |
| **控制台** | 全部 console.log、warn、error 消息 |
| **时序** | 每项操作精确时间 |

<a id="use-cases"></a>

## 使用情境

<a id="debugging-failed-actions"></a>

### 侦错失败操作



```bash
playwright-cli tracing-start
playwright-cli open https://app.example.com

# This click fails - why?
playwright-cli click e5

playwright-cli tracing-stop
# Open trace to see DOM state when click was attempted
```



<a id="analyzing-performance"></a>

### 分析效能



```bash
playwright-cli tracing-start
playwright-cli open https://slow-site.com
playwright-cli tracing-stop

# View network waterfall to identify slow resources
```



<a id="capturing-evidence"></a>

### 捕获证据



```bash
# Record a complete user flow for documentation
playwright-cli tracing-start

playwright-cli open https://app.example.com/checkout
playwright-cli fill e1 "4111111111111111"
playwright-cli fill e2 "12/25"
playwright-cli fill e3 "123"
playwright-cli click e4

playwright-cli tracing-stop
# Trace shows exact sequence of events
```



<a id="trace-vs-video-vs-screenshot"></a>

## 追踪、影片与画面比较

| 功能 | 追踪 | 影片 | 画面 |
|---------|-------|-------|------------|
| **格式** | .trace 档 | .webm 影片 | .png/.jpeg 图片 |
| **DOM 检查** | 是 | 否 | 否 |
| **网络详情** | 是 | 否 | 否 |
| **逐步重放** | 是 | 连续 | 单张 |
| **文件大小** | 中 | 大 | 小 |
| **最适用途** | 侦错 | 示范 | 快速捕获 |

<a id="best-practices"></a>

## 最佳实务

<a id="1-start-tracing-before-the-problem"></a>

### 1. 在问题发生前开始追踪



```bash
# Trace the entire flow, not just the failing step
playwright-cli tracing-start
playwright-cli open https://example.com
# ... all steps leading to the issue ...
playwright-cli tracing-stop
```



<a id="2-clean-up-old-traces"></a>

### 2. 清理旧追踪

追踪可能占用大量磁碟空间：



```bash
# Remove traces older than 7 days
find .playwright-cli/traces -mtime +7 -delete
```



<a id="limitations"></a>

## 限制

- 追踪增加自动化开销。
- 大型追踪可能占用大量磁碟空间。
- 部分动态内容可能无法完整重放。
