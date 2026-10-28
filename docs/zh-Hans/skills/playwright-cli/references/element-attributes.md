[English](../../../../../skills/playwright-cli/references/element-attributes.md) · [한국어](../../../../ko/skills/playwright-cli/references/element-attributes.md) · 简体中文 · [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/element-attributes.md) · [日本語](../../../../ja/skills/playwright-cli/references/element-attributes.md) · [Español](../../../../es/skills/playwright-cli/references/element-attributes.md)

<a id="inspecting-element-attributes"></a>

# 检查元素属性

各语言版本请见本页上方导览。

快照未显示元素的 `id`、`class`、`data-*` 属性或其他 DOM 属性时，请用 `eval` 检查。

<a id="examples"></a>

## 示例



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
