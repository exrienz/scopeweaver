[English](../../DESIGN.md) · [한국어](../ko/DESIGN.md) · 简体中文 · [繁體中文](../zh-Hant/DESIGN.md) · [日本語](../ja/DESIGN.md) · [Español](../es/DESIGN.md)

<a id="scopeweaver-landing--designmd"></a>

# ScopeWeaver 介绍页 — DESIGN.md

各语言版本请见本页上方导览。

<!-- impeccable:design-schema 1 -->

ScopeWeaver 的静态 GitHub Pages 介绍页。默认英文（`landing/index.html`）；韩文、简体中文、繁体中文、日文与西班牙文为对等版本（`landing/{ko,zh-Hans,zh-Hant,ja,es}/index.html`）。纯 HTML／CSS／JS，无框架、CDN、追踪器或后端。在专案基础 `/scopeweaver/` 及其语言子目录提供服务，使用相对素材路径。

<a id="mode"></a>

## 模式

说服。此页为产品展示橱窗：开发者或安全研究者应能了解 ScopeWeaver、看到运行画面、理解任务流程与编程智能体的操作方式、信任限制说明，并找到发行版或配置命令。

<a id="visual-world"></a>

## 视觉风格

将应用程序控制台延伸为整页尺度。

- **底色** `#fafafa`、**表面** `#ffffff`、凹陷区 `#f4f4f3`。
- **墨色** `#171717`（亦用于标志方块与满版智能体区块），次要／第三层文字为 `#3f3f3f`／`#595959`。深色区块使用 `#fafafa` 文字与 `#adb0b8` 次要文字（带色中性色，在 `#171717` 上 ≥ 6:1）。
- **单一节制蓝色**，仅用于链接、重点与焦点：浅色背景 `#2563eb`，悬停 `#1b45c9`；深色区块 `#7cafff`／`#a8c8ff`。链接颜色使用 `--link`／`--link-hover`，深色区块重新指定，维持悬停可读性。
- **细线**：1px 的 `#e6e6e4`／`#d4d4d2` 分隔各区，不用粗重分隔线。
- **范围格线图样**：呼应标志 3×3 格子的淡双轴格线，置于首屏（径向遮罩）与智能体区块背景。这是已确认的品牌图样；侦测器默认标记装饰格线，但此已确认风格在此优先。
- **深度**：克制的阴影，具有真实偏移与柔和模糊；不使用零偏移彩色光晕。画面放在浅色应用程序边框内。

<a id="type"></a>

## 字型

- **Schibsted Grotesk**（自行托管可变 woff2、字重 400–900、OFL）为主要字型：标题字重 640–660，字距 −0.028em，平衡换行；展示字级上限约 3.7rem。
- **JetBrains Mono**（自行托管可变 woff2、OFL）用于识别码、命令、来源与标签，启用 `tnum`／`zero`。
- 仅自行托管拉丁字型；韩文版在拉丁展示字型之后加入系统韩文字型（`Apple SD Gothic Neo`、`Pretendard`、`Malgun Gothic`、`Noto Sans KR`），让两版 ASCII 维持同一风格。韩文使用 `word-break: keep-all` + `overflow-wrap: break-word`；正文中的行内 `<code>` 识别码使用 `overflow-wrap: anywhere`，避免长测试名称撑宽格线（堆叠格线使用 `minmax(0, 1fr)`）。
- 正文行宽约 33–68ch；功能文字 ≥ 11px。

<a id="structure"></a>

## 结构

以细线分隔的单栏区块，刻意变化密度：

1. **首屏**：5/7 分栏；左侧为产品文案与操作，右侧为大型真实仪表板画面边框。
2. **操作导览**：7/5 分栏。垂直 ARIA tablist 切换五个真实画面（仪表板、任务、发现、系统配置、LLM 模板），更新固定画面并展开各画面的单行说明。双图、规划器 → 单意图工作智能体 → 事实／发现、人工审查与复测皆以正文说明，不生成示意图。
3. **智能体桥接**：满版墨色区块。水平 ARIA tablist 切换 Claude Code／Codex／Pi 安装命令与复制按钮；八个适配器工具及「控制界面，不取代内部智能体」的如实说明，与有边框的移动版画面并列。
4. **安装**：一键脚本、Docker Compose、发行封存档三种具体方式，可复制命令、五个平台封存标签与 GHCR。
5. **需求**：四个细线格：PostgreSQL、另设模型供应商、Node.js 22+ 适配器、本地实验室目标。
6. **限制与致谢**：直接列出限制（仅本地隔离；历史范围概览不一致已于 `75a706d` 修正（目前状态另见[验证文件](VERIFICATION.md)，不宣称目前全部测试通过）；不宣称正式环境／成效；Docker／Windows 未验证；npm audit 问题；适配器测试用本地数据，编程智能体至供应商的真实运行未独立验证，链接 VERIFICATION），旁列去识别化的拥有者流程与 ARTEX／AGPL-3.0、字型／画面来源。
7. **页尾**：品牌、资源链接、授权及「介绍页，非即时服务」声明。

<a id="signature-interaction"></a>

## 代表互动

操作导览 tablist。选取画面时，以 opacity 交叉淡化有边框的画面，并通过 `grid-template-rows` 收合动画显示说明；只有这个精心设计的互动，不让每个区块都入场。使用移动 tabindex 及 Up／Down／Home／End；水平智能体 tablist 使用 Left／Right。

<a id="accessibility--states"></a>

## 无障碍与状态

- 跳转链接、地标区域、带标签导览与明显的主题焦点框（浅色底用蓝，深色底用浅蓝）。
- ARIA tabs 正确配置 `aria-selected`／`tabindex`／`hidden` 并可用键盘操作；复制按钮播报本地化状态（`Copy`／`Copied` · `복사`／`복사됨`），不支持剪贴簿时可手动选取。
- 行动导览使用 `aria-expanded` 展开控制。页首项目不缩小；≤420px 时缩减间距，支持至 320px。纯图示 GitHub 链接保留视觉隐藏的 GitHub 文字作为无障碍名称。
- 浏览器表面配合主题：选取、滚动条、焦点框、链接底线偏移及等宽表格数字。
- `prefers-reduced-motion` 停用转场与平滑滚动。
- 图片具有 width／height 与 `aspect-ratio` 避免版面位移；首屏画面设 `fetchpriority="high"`，其余在首屏下方时延迟加载。

<a id="honesty--provenance"></a>

## 诚实呈现与来源

每张画面均标示为虚构示范数据。拥有者工作流段落仅为去识别化的协作证据，不含目标、发现、消息、模型用量或凭证。宣称可追溯至 README、适配器 README、PROVENANCE、VERIFICATION 与 v0.1.0 变更记录。ScopeWeaver 衍生自 ARTEX，依 AGPL-3.0 授权；仅限本地隔离研究。

<a id="verification-performed-build-time-non-browser"></a>

## 已运行验证（构建时，非浏览器）

- `tools/validate-landing.mjs`（位于 `landing/` 外）：素材存在、相对基础路径安全、EN／KO id／tab／control 对等、语言互连、示范图说、未发布规划元数据、存在 `.nojekyll`，共 17/17。
- 对 JS 与验证脚本运行 `node --check`。
- Impeccable 侦测器：19 → 4 个残留（单一重点悬停颜色与深色区块静态配对；已确认满版区块由内层 wrap 内缩），两者实际渲染正确。格线图样提示属已确认品牌风格。
- 协调者在 Chrome 1440px 与 390px 检视英文及韩文，应用一批修正后，确认两语在 320px 也无水平溢出。画面分页、智能体分页、键盘切换、行动选单、语言链接与本地化复制回馈通过，未观察到浏览器控制台错误；导览时加载首屏下方的延迟画面。
- Pages 仅部署 `landing/`；上传前工作流可执行文件案完整性与 JavaScript 语法检查。开发文件与私人运行数据不包含在公开产物中。

<a id="implementation-attribution"></a>

## 实作归属

原请求为 Claude Code 使用 `claude-opus-5-5`、medium effort。主要实作会话在供应商保护中断后自动回退至 `claude-opus-4-8`；有界的浏览器 QA 修正阶段全程使用 `claude-opus-5-5`、medium effort。因此不得将整页完全归属 Opus 5.5。
