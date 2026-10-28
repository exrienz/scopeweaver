[English](../../../../../skills/playwright-cli/references/playwright-tests.md) · [한국어](../../../../ko/skills/playwright-cli/references/playwright-tests.md) · [简体中文](../../../../zh-Hans/skills/playwright-cli/references/playwright-tests.md) · 繁體中文 · [日本語](../../../../ja/skills/playwright-cli/references/playwright-tests.md) · [Español](../../../../es/skills/playwright-cli/references/playwright-tests.md)

<a id="running-playwright-tests"></a>

# 執行 Playwright 測試

各語言版本請見本頁上方導覽。

使用 `npx playwright test` 或套件管理器指令碼執行 Playwright 測試。若不想開啟互動 HTML 報告，請設定 `PLAYWRIGHT_HTML_OPEN=never`。



```bash
# Run all tests
PLAYWRIGHT_HTML_OPEN=never npx playwright test

# Run all tests through a custom npm script
PLAYWRIGHT_HTML_OPEN=never npm run special-test-command
```



<a id="debugging-playwright-tests"></a>

# 偵錯 Playwright 測試

以 `--debug=cli` 執行失敗的 Playwright 測試；指令會在開始時暫停並印出偵錯說明。

**重要**：在背景執行指令，檢查輸出直到出現「Debugging Instructions」。完成後務必停止指令。

印出含工作階段名稱的說明後，以 `playwright-cli` 附加該工作階段並探索頁面。



```bash
# Run the test
PLAYWRIGHT_HTML_OPEN=never npx playwright test --debug=cli
# ...
# ... debugging instructions for "tw-abcdef" session ...
# ...

# Attach to the test
playwright-cli attach tw-abcdef
```



探索與尋找修正時，保持測試在背景執行。測試起始即暫停，請逐步執行或在最可能出問題的位置暫停。

每次 `playwright-cli` 操作都產生相應 Playwright TypeScript 程式碼，顯示於輸出，可直接複製至測試。通常需更新特定 locator 或 expectation，但也可能是應用程式錯誤，請依證據判斷。

修正後停止背景測試，再重新執行確認通過。
