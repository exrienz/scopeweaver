# Playwright 테스트 실행

[English](../../../../../skills/playwright-cli/references/playwright-tests.md) · 한국어 · [简体中文](../../../../zh-Hans/skills/playwright-cli/references/playwright-tests.md) · [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/playwright-tests.md) · [日本語](../../../../ja/skills/playwright-cli/references/playwright-tests.md) · [Español](../../../../es/skills/playwright-cli/references/playwright-tests.md)

`npx playwright test` 또는 패키지 관리자 스크립트로 테스트합니다. 대화형 HTML 보고서가 열리지 않게 하려면 `PLAYWRIGHT_HTML_OPEN=never`를 설정합니다.

```bash
# Run all tests
PLAYWRIGHT_HTML_OPEN=never npx playwright test

# Run all tests through a custom npm script
PLAYWRIGHT_HTML_OPEN=never npm run special-test-command
```

# Playwright 테스트 디버깅

실패한 테스트를 `--debug=cli`로 실행하면 시작 지점에서 멈추고 디버깅 지침을 출력합니다.

명령을 백그라운드에서 실행하고 “Debugging Instructions”가 출력될 때까지 확인합니다. 작업 후 반드시 명령을 중지하세요.

세션 이름이 출력되면 `playwright-cli`로 연결해 페이지를 탐색합니다.

```bash
# Run the test
PLAYWRIGHT_HTML_OPEN=never npx playwright test --debug=cli
# ...
# ... debugging instructions for "tw-abcdef" session ...
# ...

# Attach to the test
playwright-cli attach tw-abcdef
```

탐색하고 수정 방법을 찾는 동안 테스트를 백그라운드에서 유지합니다.
시작 지점에서 멈춘 상태이므로 단계별로 실행하거나 문제가 있을 법한 위치에서 멈춥니다.


`playwright-cli`의 각 조작은 해당 Playwright TypeScript 코드를 생성합니다.
출력 코드를 테스트에 복사할 수 있습니다. 대개 locator나 기대값을 수정하지만 앱 오류일 수도 있으므로 증거를 보고 판단하세요.

수정 후 백그라운드 테스트를 중지하고 다시 실행해 통과를 확인합니다.
