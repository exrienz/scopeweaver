# Ejecutar pruebas Playwright

[English](../../../../../skills/playwright-cli/references/playwright-tests.md) · [한국어](../../../../ko/skills/playwright-cli/references/playwright-tests.md) · [简体中文](../../../../zh-Hans/skills/playwright-cli/references/playwright-tests.md) · [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/playwright-tests.md) · [日本語](../../../../ja/skills/playwright-cli/references/playwright-tests.md) · Español

Ejecuta pruebas con `npx playwright test` o un script del gestor de paquetes. Para no abrir el informe HTML interactivo, usa la variable `PLAYWRIGHT_HTML_OPEN=never`.

```bash
# Run all tests
PLAYWRIGHT_HTML_OPEN=never npx playwright test

# Run all tests through a custom npm script
PLAYWRIGHT_HTML_OPEN=never npm run special-test-command
```

# Depurar pruebas Playwright

Para depurar una prueba fallida, ejecútala con `--debug=cli`. Se pausa al inicio e imprime instrucciones.

**IMPORTANTE**: ejecuta en segundo plano y revisa salida hasta que aparezca «Debugging Instructions». Detén el comando al terminar.

Cuando se impriman instrucciones con nombre de sesión, usa `playwright-cli` para conectarte y explorar.

```bash
# Run the test
PLAYWRIGHT_HTML_OPEN=never npx playwright test --debug=cli
# ...
# ... debugging instructions for "tw-abcdef" session ...
# ...

# Attach to the test
playwright-cli attach tw-abcdef
```

Mantén la prueba en segundo plano mientras exploras y buscas una solución.
La prueba se pausa al inicio; avanza o pausa en la ubicación
donde sea más probable el problema.

Cada acción con `playwright-cli` genera código TypeScript Playwright correspondiente.
Aparece en la salida y se puede copiar a la prueba. Normalmente hay que actualizar un localizador o expectativa, aunque también puede ser un fallo de la aplicación. Usa tu criterio.

Tras corregir, detén la ejecución de fondo. Repite para comprobar que pasa.
