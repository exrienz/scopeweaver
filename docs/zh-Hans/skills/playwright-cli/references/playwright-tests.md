[English](../../../../../skills/playwright-cli/references/playwright-tests.md) · [한국어](../../../../ko/skills/playwright-cli/references/playwright-tests.md) · 简体中文 · [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/playwright-tests.md) · [日本語](../../../../ja/skills/playwright-cli/references/playwright-tests.md) · [Español](../../../../es/skills/playwright-cli/references/playwright-tests.md)

<a id="running-playwright-tests"></a>

# 运行 Playwright 测试

各语言版本请见本页上方导览。

使用 `npx playwright test` 或包管理器脚本运行 Playwright 测试。若不想开启互动 HTML 报告，请配置 `PLAYWRIGHT_HTML_OPEN=never`。



```bash
# Run all tests
PLAYWRIGHT_HTML_OPEN=never npx playwright test

# Run all tests through a custom npm script
PLAYWRIGHT_HTML_OPEN=never npm run special-test-command
```



<a id="debugging-playwright-tests"></a>

# 侦错 Playwright 测试

以 `--debug=cli` 运行失败的 Playwright 测试；命令会在开始时暂停并印出侦错说明。

**重要**：在背景运行命令，检查输出直到出现「Debugging Instructions」。完成后务必停止命令。

印出含会话名称的说明后，以 `playwright-cli` 附加该会话并探索页面。



```bash
# Run the test
PLAYWRIGHT_HTML_OPEN=never npx playwright test --debug=cli
# ...
# ... debugging instructions for "tw-abcdef" session ...
# ...

# Attach to the test
playwright-cli attach tw-abcdef
```



探索与寻找修正时，保持测试在背景运行。测试起始即暂停，请逐步运行或在最可能出问题的位置暂停。

每次 `playwright-cli` 操作都产生相应 Playwright TypeScript 代码，显示于输出，可直接复制至测试。通常需更新特定 locator 或 expectation，但也可能是应用程序错误，请依证据判断。

修正后停止背景测试，再重新运行确认通过。
