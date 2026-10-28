[English](../../PRODUCT.md) · [한국어](../ko/PRODUCT.md) · 简体中文 · [繁體中文](../zh-Hant/PRODUCT.md) · [日本語](../ja/PRODUCT.md) · [Español](../es/PRODUCT.md)

<a id="scopeweaver-product"></a>

# ScopeWeaver 产品

各语言版本请见本页上方导览。

<!-- impeccable:product-schema 1 -->

<a id="platform"></a>

## 平台
网页

<a id="stack"></a>

## 技术组合
应用程序使用 Go 后端、内嵌 Next.js 前端及 PostgreSQL。另行审批的 GitHub Pages 介绍页采纯 HTML、CSS 与 JavaScript，必须不依赖应用程序后端即可运行。

<a id="users"></a>

## 用户
评估 AI 辅助工作流、进行本地隔离研究与验证的开发者及安全研究者。此受众与六语应用程序／文件范围均已获拥有者审批。

<a id="product-purpose"></a>

## 产品目的
在同一自行托管的研究控制台中，协调目标规划、智能体、资产范围、覆盖率、发现记录与人工审查。介绍页协助访客理解产品、检视如实呈现的证据，并找到下载／配置文件。

<a id="positioning"></a>

## 定位
共用资产图与各任务探索图将目标链接至智能体方向、观察及证据。外部智能体控制适配器向 Claude Code、Codex、Pi 提供任务与发现工作流。

<a id="operating-context"></a>

## 操作情境
自行托管应用程序，需要 PostgreSQL 与另行配置的模型供应商才能运行智能体。GitHub Pages 仅托管公开静态介绍，不提供即时安全测试服务或后端。

<a id="capabilities-and-constraints"></a>

## 能力与限制
功能与配置依据 README.md、adapters/agent/README.md 及代码。v0.1.0 提供五个平台封存档与多架构 Docker 镜像。适配器为控制界面，不取代内部智能体，也不会将编程订阅转为 API 密钥。仓库使用限制允许本地隔离研究／验证，禁止线上目标测试。历史遗留的后端范围概览不一致问题已在提交 `75a706d` 中修复。验证证据与限制请参阅[VERIFICATION.md](VERIFICATION.md)；这不代表目前全部测试皆已通过，也不代表已可供正式环境使用。

<a id="brand-commitments"></a>

## 品牌承诺
保留 ScopeWeaver 名称与既有范围格线标志。应用程序画面创建精确、中性的控制台风格，仅使用节制的蓝色重点。默认英文，可选韩文、简体中文、繁体中文、日文与西班牙文。尊重 ARTEX 作者归属与 AGPL-3.0。

<a id="evidence-on-hand"></a>

## 现有证据
仓库的 screenshots/en 与 screenshots/ko 画面展示虚构示范数据，必须标示为示范画面。既有标志：web/public/scopeweaver.svg。README、来源、验证说明、适配器文件与发行产物为主要证据。已在登录的本地界面检视拥有者实际运行的对话，记录了创建任务、检查探索图与工作智能体追踪及韩文进度更新。公开页面只可使用此保护隐私的流程摘要，不得包含目标名称、原始消息、漏洞宣称、token 数或私人画面。这是协作流程证据，并非独立安全验证。未提供经验证客户推荐、采用统计、效能基准或外部审计安全宣称。

<a id="product-principles"></a>

## 产品原则
产品宣称必须可查证。区分公开介绍、示范数据与实际运行环境。入口附近说明配置与模型需求。保留范围／审批机制。绝不发布凭证、目标详情或私人对话。

<a id="accessibility--inclusion"></a>

## 无障碍与包容
支持桌面与行动网页的响应式设计、可用键盘操作的控制、可读对比、减少动态效果，以及完整英文、韩文、简体中文、繁体中文、日文与西班牙文文案。
