# Inspeccionar atributos de elementos

[English](../../../../../skills/playwright-cli/references/element-attributes.md) · [한국어](../../../../ko/skills/playwright-cli/references/element-attributes.md) · [简体中文](../../../../zh-Hans/skills/playwright-cli/references/element-attributes.md) · [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/element-attributes.md) · [日本語](../../../../ja/skills/playwright-cli/references/element-attributes.md) · Español

Si la instantánea no muestra `id`, `class`, atributos `data-*` u otras propiedades DOM, usa `eval` para inspeccionarlas.

## Ejemplos

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
