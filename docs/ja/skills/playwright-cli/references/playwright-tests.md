[English](../../../../../skills/playwright-cli/references/playwright-tests.md) | [한국어](../../../../ko/skills/playwright-cli/references/playwright-tests.md) | [简体中文](../../../../zh-Hans/skills/playwright-cli/references/playwright-tests.md) | [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/playwright-tests.md) | [日本語](playwright-tests.md) | [Español](../../../../es/skills/playwright-cli/references/playwright-tests.md)

<a id="running-playwright-tests"></a>
# Playwright のテストを実行する



Playwright のテストは `npx playwright test` またはパッケージマネージャーのスクリプトで実行します。対話型の HTML レポートを開かないようにするには、環境変数 `PLAYWRIGHT_HTML_OPEN=never` を使用してください。

```bash
# Run all tests
PLAYWRIGHT_HTML_OPEN=never npx playwright test

# Run all tests through a custom npm script
PLAYWRIGHT_HTML_OPEN=never npm run special-test-command
```

<a id="debugging-playwright-tests"></a>
# Playwright のテストをデバッグする

失敗した Playwright のテストをデバッグするには、`--debug=cli` オプションで実行します。開始時にテストを一時停止し、デバッグの指示を表示します。

**重要**: コマンドをバックグラウンドで実行し、「Debugging Instructions」が表示されるまで出力を確認してください。作業が終わったら、必ずコマンドを停止してください。

セッション名を含む指示が表示されたら、`playwright-cli` でセッションに接続してページを調べます。

```bash
# Run the test
PLAYWRIGHT_HTML_OPEN=never npx playwright test --debug=cli
# ...
# ... debugging instructions for "tw-abcdef" session ...
# ...

# Attach to the test
playwright-cli attach tw-abcdef
```

ページを調べて修正を探す間、テストはバックグラウンドで動かしたままにしてください。開始時に一時停止しているため、問題がありそうな箇所までステップ実行するか、その箇所で一時停止します。

`playwright-cli` の各操作は、対応する Playwright の TypeScript コードを生成します。出力からそのままテストにコピーできます。多くの場合は特定のロケーターや期待値の更新が必要ですが、アプリのバグである可能性もあります。状況に応じて判断してください。

テストを修正したら、バックグラウンドの実行を停止し、再実行して合格することを確認してください。
