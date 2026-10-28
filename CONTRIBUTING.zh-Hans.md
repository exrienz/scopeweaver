[English](CONTRIBUTING.md) · [한국어](CONTRIBUTING.ko.md) · 简体中文 · [繁體中文](CONTRIBUTING.zh-Hant.md) · [日本語](CONTRIBUTING.ja.md) · [Español](CONTRIBUTING.es.md)

<a id="contributing-to-scopeweaver"></a>

# 贡献 ScopeWeaver

各语言版本请见本页上方导览。

欢迎提交错误修正、ARTEX 上游回移、依赖包升级、文件与测试。

<a id="before-you-start"></a>

## 开始之前

- 阅读[使用限制](README.zh-Hans.md#license-and-disclaimer)。贡献不得新增以真实线上系统为目标的功能，亦不得削弱这些限制。
- 大型变更请先开 issue，以便就方案达成共识。
- 每个 PR 聚焦单一主题。较小的 PR 更容易快速审查。

<a id="requirements-for-a-pull-request"></a>

## PR 要求

- **完整支持已发布语言及对应文件。** 用户可见文字与维护中的用户／贡献者指南，需提供英文、韩文、简体中文（`zh-Hans`）、繁体中文（`zh-Hant`）、日文（`ja`）与西班牙文（`es`）：
  - 服务器消息：保持 `locale.Register(key, en, ko)` 兼容，通过 `locale.RegisterAll` 与 `locale/` 中依语言索引的资源加入其他语言。
  - 网页界面：EN／KO 表格与 `messages/{zh-hans,zh-hant,ja,es}.ts` 必须涵盖全部目前英文键。
  - 文件：`README.md`、`CONTRIBUTING.md` 与维护中的指南须有 `ko`、`zh-Hans`、`zh-Hant`、`ja`、`es` 对应版本；保留可运行的英文技能入口。
- **测试变更行为。** 新增或更新在未应用变更时会失败的测试。不得为通过测试而停用、略过或放宽既有检查。
- **仅使用可抛弃数据库。** 会写入 PostgreSQL 的测试必须使用全新的 `scopeweaver_test_*` 数据库（智能体适配器 E2E 使用 `scopeweaver_adapter_*`）。
- **注明上游贡献。** ARTEX 回移须加入 `Upstream-commit: mhtsec/ARTEX@<sha>` 与原作者的 `Co-authored-by:` trailer，并更新 `docs/PROVENANCE.md` 与 `docs/ko/PROVENANCE.md`。
- **授权。** 贡献依仓库的 [AGPL-3.0](LICENSE) 授权接受。

<a id="adding-or-changing-a-locale"></a>

## 新增或修改语言区域

依照[语言区域指南](docs/zh-Hans/i18n.md)更新共用的 `locale/definitions.json`（标准 ID、原生名称、Intl 对应与别名）、比对测试向量、完整前后端目录、date-fns 对应、聚焦测试与浏览器基本测试。`zh`、`zh-CN`、`zh-SG` 对应 `zh-Hans`；`zh-TW`、`zh-HK`、`zh-MO` 对应 `zh-Hant`。明确文字系统优先于冲突地区。英文为键、含义、占位符与 rich tag 的来源。后端运行时英文／键回退是兼容性保护，不代表语言已可发布。不得翻译用户内容、草稿、原始证据、识别码或历史数据。请审查译文并记录疑虑；无证据时不得宣称经母语者审批。六种语言的维护中功能指南均须完整，包括英文／韩文配对文件。 亦须同步独立适配器的标准 ID 清单与根目录 shell／批次消息辅助。运行 `node e2e/script-locales.mjs`，并以原生 Windows 的 `node e2e/windows-launcher.mjs`／Windows CI 验证；封装适配器不得在运行时依赖仓库根目录 JSON。

<a id="checks-to-run"></a>

## 必须运行的检查

以下对应 [.github/workflows/ci.yml](.github/workflows/ci.yml) 的 CI 工作：



```bash
# patched Norma SDK (a separate Go module)
(cd third_party/norma && go test ./... -count=1)

# backend database-backed suites (db, server, agent): against a throwaway database
# that is created and dropped for you. They require ARTEX_TEST_DSN and skip (never
# fall back to the live DSN) without it — see docs/VERIFICATION.md.
make test-db

# backend: every Go package (set ARTEX_TEST_DSN to a disposable scopeweaver_test_* database)
ARTEX_TEST_DSN=postgres://sw_user_test@/scopeweaver_test_1?host=/tmp&sslmode=disable go test ./... -count=1

# frontend
cd web && npm ci && npm test && npm run build:static

# agent adapter (and its real-backend E2E test against a scopeweaver_adapter_* database)
cd adapters/agent && npm ci && npm test && npm run check
SCOPEWEAVER_ADAPTER_E2E=1 go test ./server -run '^TestAgentAdapterE2E$' -count=1
```



`e2e` 工作流亦会在每个 PR 与合并后，针对真实可执行文件运行必要浏览器基本测试。若要在本地针对运行中的服务器及全新的 `scopeweaver_test_*` 数据库运行，需要 `psql` 与同一签出版本构建的内嵌可执行文件：



```bash
cd e2e && npm ci && npx playwright install chromium
SCOPEWEAVER_URL=http://127.0.0.1:8787 ARTEX_PG_DSN=postgres://.../scopeweaver_test_e2e \
  SCOPEWEAVER_BINARY=/absolute/path/to/scopeweaver npm run smoke
```



既有基本检查初始化密码后，仪表板回归测试会针对同一测试数据库，在空闲环回连接埠启动该可执行文件的独立副本。它检查英文／韩文 Legacy／New token 堆叠与配置筛选，之后停止服务器并移除测试数据。仪表板证据与子服务器记录保存在 `e2e/e2e-results/dashboard-token-chart/`。

必要基本测试涵盖六种语言的偏好恢复、配置／登录／任务／配置／对话草稿、首次 API 语言、SSE 重连／筛选保留，以及实际呈现的数字格式。

请在 PR 说明运行了哪些检查，以及哪些无法运行。

<a id="review-and-merging"></a>

## 审查与合并

每个 `feat/*` 或 `fix/*` 分支皆由最新 `dev` 创建，PR 目标设为 `dev`。整合后，等待该确切 `dev` 提交上的所有检查及 E2E，然后从本仓库 `dev` 向 `main` 开升版 PR。升版需使用合并提交，让 `dev` 维持为 `main` 的祖先。升版后再次验证检查；后续工作仍从 `dev` 开始。`branch-policy` 会拒绝其他 PR 路由。

1. 目前候选版的全部必要检查必须通过：`branch-policy`、`frontend`、`agent-adapter`、`backend`、`windows`、`chat-browser`、`smoke`。`dev` 与 `main` 合并后都运行 CI 及 E2E。首次贡献者需由维护者审批 CI 运行。
2. 依差异检查正确性、安全性、已发布语言完整度，以及六语文件对等性。
3. **每个 PR 合并前，维护者均须进行最终人工检查。** 未经检查不得合并。

版本由维护者在合并前后完成完整端对端测试后发布。
