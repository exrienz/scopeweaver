# 요소 속성 조회

[English](../../../../../skills/playwright-cli/references/element-attributes.md) · 한국어 · [简体中文](../../../../zh-Hans/skills/playwright-cli/references/element-attributes.md) · [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/element-attributes.md) · [日本語](../../../../ja/skills/playwright-cli/references/element-attributes.md) · [Español](../../../../es/skills/playwright-cli/references/element-attributes.md)

스냅샷에 `id`, `class`, `data-*` 등 DOM 속성이 없으면 `eval`로 확인합니다.

## 예제

```bash
playwright-cli snapshot
# snapshot shows a button as e7 but doesn't reveal its id or data attributes

# get the element's id
playwright-cli eval "el => el.id" e7

# get all CSS classes
playwright-cli eval "el => el.className" e7

# get a specific attribute
playwright-cli eval "el => el.getAttribute('data-testid')" e7
playwright-cli eval "el => el.getAttribute('aria-label')" e7

# get a computed style property
playwright-cli eval "el => getComputedStyle(el).display" e7
```
