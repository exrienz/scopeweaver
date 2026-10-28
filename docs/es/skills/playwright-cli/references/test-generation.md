# Generación de pruebas (planificar → generar → reparar)

[English](../../../../../skills/playwright-cli/references/test-generation.md) · [한국어](../../../../ko/skills/playwright-cli/references/test-generation.md) · [简体中文](../../../../zh-Hans/skills/playwright-cli/references/test-generation.md) · [繁體中文](../../../../zh-Hant/skills/playwright-cli/references/test-generation.md) · [日本語](../../../../ja/skills/playwright-cli/references/test-generation.md) · Español

Flujo completo para crear y mantener pruebas Playwright con `playwright-cli`. Cada acción de `playwright-cli` emite TypeScript equivalente; ese código es la materia prima de cada prueba. Las secciones se pueden usar independientemente:

- **Cómo funciona la generación**: mecanismo central: acciones a TypeScript y cómo añadir aserciones.
- **Planificar**: explorar y producir una especificación de qué probar.
- **Generar**: convertir especificación en pruebas. Actualizarla si es vaga u obsoleta.
- **Reparar**: diagnosticar fallos, corregir código y conciliar especificación con realidad.

Planificar/generar/reparar usan el mismo mecanismo: ejecutar `npx playwright test --debug=cli` en segundo plano y `playwright-cli attach tw-XXXX` para controlar la página pausada. Consulta [playwright-tests.md](playwright-tests.md).

---

## 0. Cómo funciona la generación

Cada acción con `playwright-cli` genera TypeScript Playwright correspondiente. Aparece en la salida y se puede copiar directamente a los archivos de prueba.

```bash
# Start a session
playwright-cli open https://example.com/login

# Take a snapshot to see elements
playwright-cli snapshot
# Output shows: e1 [textbox "Email"], e2 [textbox "Password"], e3 [button "Sign In"]

# Fill form fields - generates code automatically
playwright-cli fill e1 "user@example.com"
# Ran Playwright code:
# await page.getByRole('textbox', { name: 'Email' }).fill('user@example.com');

playwright-cli fill e2 "password123"
# Ran Playwright code:
# await page.getByRole('textbox', { name: 'Password' }).fill('password123');

playwright-cli click e3
# Ran Playwright code:
# await page.getByRole('button', { name: 'Sign In' }).click();
```

### Construir un archivo de prueba

Reúne el código generado en una prueba Playwright:

```typescript
import { test, expect } from '@playwright/test';

test('login flow', async ({ page }) => {
  // Generated code from playwright-cli session:
  await page.goto('https://example.com/login');
  await page.getByRole('textbox', { name: 'Email' }).fill('user@example.com');
  await page.getByRole('textbox', { name: 'Password' }).fill('password123');
  await page.getByRole('button', { name: 'Sign In' }).click();

  // Add assertions
  await expect(page).toHaveURL(/.*dashboard/);
});
```

### Usar localizadores semánticos

El código usa localizadores por rol cuando es posible, más resistentes:

```typescript
// Generated (good - semantic)
await page.getByRole('button', { name: 'Submit' }).click();

// Avoid (fragile - CSS selectors)
await page.locator('#submit-btn').click();
```

### Explorar antes de grabar

Captura instantáneas para entender la estructura antes de grabar acciones:

```bash
playwright-cli open https://example.com
playwright-cli snapshot
# Review the element structure
playwright-cli click e5
```

### Añadir aserciones manualmente

El código captura acciones, no aserciones. Añade expectativas con alguno de los comparadores recomendados:

- `toBeVisible()` — elemento renderizado y visible
- `toHaveText(text)` — texto del elemento coincidente
- `toHaveValue(value) / toBeEmpty()` — valor de entrada/selector coincidente
- `toBeChecked() / toBeUnchecked()` — estado de casilla coincidente
- `toMatchAriaSnapshot(snapshot)` — página o localizador coincide con instantánea parcial de accesibilidad

Usa `playwright-cli generate-locator <target>` para producir la expresión del localizador, y snapshot/eval para obtener el valor esperado.

Al comprobar texto, evita que el localizador contenga el texto del propio elemento. `getByTestId()` o `getByLabel()` suelen servir. Para localizadores textuales, prefiere `toBeVisible()`.

La instantánea esperada no necesita toda la información: captura solo lo necesario para la aserción. Puedes usar expresiones regulares para valores inestables.

```bash
# Get a stable locator for an element ref to use in the assertion
playwright-cli --raw generate-locator e5
# getByRole('button', { name: 'Submit' })

# Capture expected text content for toHaveText
playwright-cli --raw eval "el => el.textContent" e5

# Capture expected input value for toHaveValue/toBeEmpty
playwright-cli --raw eval "el => el.value" e5

# Capture expected aria snapshot for toMatchAriaSnapshot/toBeChecked
# (whole page, or use a ref to scope to a region)
playwright-cli --raw snapshot
playwright-cli --raw snapshot e5
```

```typescript
// Generated action
await page.getByRole('button', { name: 'Submit' }).click();

// Manual assertions using the outputs above:
await expect(page.getByRole('alert', { name: 'Success' })).toBeVisible();
await expect(page.getByTestId('main-header')).toHaveText('Welcome, user');
await expect(page.getByRole('textbox', { name: 'Email' })).toHaveValue('user@example.com');
await expect(page.getByRole('checkbox', { name: 'Enable notifications' })).toBeChecked();

// toMatchAriaSnapshot on the whole page, finds a matching region
await expect(page).toMatchAriaSnapshot(`
  - heading "Welcome, user"
  - link /\\d+ new messages?/
  - button "Sign out"
`);

// toMatchAriaSnapshot scoped to a region
await expect(page.getByRole('navigation')).toMatchAriaSnapshot(`
  - link "Home"
  - link /\\d+ new messages?/
  - link "Profile"
`);
```

---

## 1. Planificación

Objetivo: producir una especificación (p. ej., `specs/<feature>.plan.md`) que enumere escenarios. **Siempre** escríbela a un archivo.

### 1.1 Requisito previo: espacio de trabajo

Comprueba primero que Playwright esté instalado:

```bash
# Either of these confirms a workspace:
test -f playwright.config.ts || test -f playwright.config.js
npx --no-install playwright --version
```

Si no está, inicialízalo y deja elegir al usuario los valores predeterminados:

```bash
npm init playwright@latest
```

### 1.2 Requisito previo: prueba inicial

Una **prueba inicial** deja la página en el estado de inicio común: navegación, acceso necesario, opciones de funciones, etc. Los escenarios parten de un estado nuevo *después* de ella. `--debug=cli` pausa *dentro* de esta prueba; allí empiezan planificación y generación.

Prueba inicial mínima:

```ts
// tests/seed.spec.ts
import { test } from '@playwright/test';

test('seed', async ({ page }) => {
  await page.goto('https://example.com/');
});
```

Preferido: mueve navegación a una fixture para reutilizarla:

```ts
// tests/fixtures.ts
import { test as baseTest } from '@playwright/test';
export { expect } from '@playwright/test';

export const test = baseTest.extend({
  page: async ({ page }, use) => {
    await page.goto('https://example.com/');
    await use(page);
  },
});
```

```ts
// tests/seed.spec.ts
import { test } from './fixtures';

test('seed', async ({ page }) => {
  // Fixture already navigates. This empty body tells agents where to start.
});
```

Si no existe, crea una que al menos abra la aplicación.

### 1.3 Explorar la aplicación

Inicia la aplicación mediante la prueba inicial en segundo plano y conecta:

```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test tests/seed.spec.ts --debug=cli
# wait for "Debugging Instructions" and the session name tw-XXXX
playwright-cli attach tw-XXXX
```

Reanuda para ejecutarla y después explora:

```bash
playwright-cli resume                   # resume so that seed test runs fully
playwright-cli snapshot                 # inventory of interactive elements
playwright-cli click e5                 # follow a flow
playwright-cli eval "location.href"     # read URL / state
playwright-cli show --annotate          # ask the user to point at something
```

Identifica:

- Superficies interactivas (formularios, botones, listas, filtros, modales).
- Recorridos principales completos del usuario.
- Extremos: estados vacíos, validación, entradas largas y valores límite.
- Persistencia: recarga, almacenamiento local/de sesión, fragmentos URL.
- Navegación: controles que cambian URL y comportamiento atrás/adelante.

**Importante**: no abras solo la URL con playwright-cli; pasa siempre por la prueba para conservar su preparación personalizada.
**Importante**: detén la prueba de fondo al terminar de explorar.

### 1.4 Escribir especificación

Guarda en `specs/<feature>.plan.md`. Usa esta estructura:

```markdown
# <Feature> Test Plan

## Application Overview

<One paragraph describing what the feature does and why it matters.>

## Test Scenarios

### 1. <Group Name>

**Seed:** `tests/seed.spec.ts`

#### 1.1. <kebab-case-scenario-name>

**File:** `tests/<group>/<kebab-case-scenario-name>.spec.ts`

**Steps:**
  1. <Concrete user step>
    - expect: <observable outcome>
    - expect: <another observable outcome>
  2. <Next step>
    - expect: <outcome>

#### 1.2. <next-scenario>
...

### 2. <Next Group>

**Seed:** `tests/seed.spec.ts`
...
```

Directrices:

- Cada escenario es independiente y parte del estado nuevo inicial; nunca encadenes escenarios.
- Los nombres son kebab-case y coinciden con archivo (`should-add-single-todo` → `should-add-single-todo.spec.ts`).
- Cubre flujo normal, extremos, validación, negativos y persistencia.
- Describe pasos del usuario («Escribir 'Comprar leche'»), no API («llamar `fill`»).
- Pon resultados observables en viñetas `- expect:`; cada una se convierte en aserción.

---

## 2. Generar

Objetivo: tomar una especificación y producir archivos de pruebas. Actualízala si se ha desviado.

### 2.1 Entradas

- **Especificación**, p. ej., `specs/basic-operations.plan.md`.
- **Destino**: un escenario (p. ej., `1.2`), grupo completo (`1`) o todos.
- **Archivo inicial**, leído de `**Seed:**` del grupo.

### 2.2 Generar un escenario

Para cada escenario, secuencialmente (nunca en paralelo: comparten sesión inicial):

```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test <seed-file> --debug=cli   # background
playwright-cli attach tw-XXXX
# resume
```

**No** abras solo la URL con playwright-cli; pasa por la prueba para capturar su preparación.

Recorre `Steps:` uno por uno con `playwright-cli`, usando la especificación como plan y la aplicación activa como verdad. Si un paso es vago («pulsar botón»: ¿cuál?), referencia algo ausente o contradice el comportamiento real, usa tu criterio: actualiza la especificación a la realidad y continúa. Se espera editarla durante generación.

Cada acción imprime TypeScript equivalente (consulta [Cómo funciona](#0-cómo-funciona-la-generación)):

```bash
playwright-cli snapshot                         # find refs
playwright-cli fill e3 "John Doe"               # -> page.getByRole('textbox', {...}).fill(...)
playwright-cli press Enter
playwright-cli click e7
```

Para cada viñeta `- expect:`, añade una aserción explícita. Consulta [Cómo funciona](#0-cómo-funciona-la-generación).

Reúne el código y escribe la prueba en la ruta indicada:

```ts
// spec: specs/basic-operations.plan.md
// seed: tests/seed.spec.ts
import { test, expect } from './fixtures';   // or '@playwright/test' if no fixtures file

test.describe('Signing in and out', () => {
  test('should sign in', async ({ page }) => {
    // 1. Navigate to the application
    // (handled by the seed fixture)

    // 2. Type 'John Doe' into the username field
    await page.getByRole('textbox', { name: 'username' }).fill('John Doe');

    // 3. Type password
    await page.getByRole('textbox', { name: 'password' }).fill('TestPassword');

    // 4. Press Enter to submit
    await page.getByRole('textbox', { name: 'password' }).press('Enter');

    await expect(page.getByRole('heading')).toContainText('Welcome, John Doe!');
  });
});
```

Reglas:

- **Una prueba por archivo.** Ruta, nombre describe y nombre de prueba se copian literalmente de la especificación (sin ordinal).
- Antepon a las acciones de cada paso un comentario `// N. <step text>`.
- Usa literalmente el nombre de grupo describe (sin ordinal `1.`).
- Importa de `./fixtures` si existe; si no, `@playwright/test`.
- **Importante**: cierra la sesión CLI y detén la prueba de fondo antes del siguiente escenario.

### 2.3 Generar varios escenarios

Repite 2.2 uno a uno, reiniciando la prueba inicial entre escenarios para partir de página limpia. Se puede paralelizar por nombres de sesión únicos; asegúrate de detener cada ejecución.

### 2.4 Ejecutar pruebas generadas

Tras generar, ejecuta las pruebas nuevas una vez:

```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test tests/<group>/<scenario>.spec.ts
```

Cualquier fallo pasa a sección 3.

---

## 3. Reparar

Objetivo: corregir pruebas fallidas y actualizar especificación si cambió el comportamiento previsto.

### 3.1 Encontrar pruebas fallidas

```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test
```

Registra entradas `<file>:<line>` fallidas y procésalas una a una. No corrijas en paralelo: el estado compartido y la única sesión CLI lo hacen frágil.

### 3.2 Depurar un fallo

Ejecuta esa prueba en depuración y segundo plano; después conecta:

```bash
PLAYWRIGHT_HTML_OPEN=never npx playwright test tests/<group>/<scenario>.spec.ts:<line> --debug=cli
# wait for "Debugging Instructions" and the tw-XXXX session name
playwright-cli attach tw-XXXX
```

Se pausa al inicio. Avanza o ejecuta hasta justo antes de la acción/aserción fallida y diagnostica:

```bash
playwright-cli snapshot                # did the element change / move / rename?
playwright-cli console                 # app-side errors?
playwright-cli requests                # failed request? wrong payload?
playwright-cli show --annotate         # ask the user to point somewhere
```

Causas comunes: selector cambiado, envoltorio nuevo, etiqueta/ARIA renombrada, tiempos (transición/carga), texto de aserción actualizado o datos filtrados entre ejecuciones.

Ensaya la interacción corregida con `playwright-cli`; pega en la prueba el código generado.

### 3.3 Aplicar corrección

Edita localizador, aserción, orden o entradas para coincidir con el comportamiento correcto. Detén la depuración de fondo. Repite la prueba para confirmar que pasa.

Nunca omitas hooks ni añadas esperas fijas como solución. Nunca uses `networkidle`.

### 3.4 Conciliar con especificación

Abre la especificación de `// spec:` en la cabecera y localiza el escenario.

- **Corrección solo técnica** (localizador o aserción mejorados) y comportamiento de usuario aún correcto → no cambies especificación.
- **Cambian pasos visibles, entradas, orden o resultados** descritos → actualiza especificación a la realidad. Conserva ID y ruta; cambia solo pasos/expectativas.
- **No está claro si el cambio es intencionado** (especificación obsoleta) **o regresión** (prueba correcta, aplicación incorrecta) → **detente y pregunta al usuario**. Incluye:
  - ID de escenario (p. ej., `2.3`),
  - líneas que ya no coinciden,
  - comportamiento observado (cita una instantánea o resultado concreto).

Solo tras su respuesta, actualiza la especificación (cambio intencionado) o registra/señala la prueba como cobertura de un fallo (regresión).

### 3.5 Iteración y abandono

- Corrige fallos uno a uno; repite tras cada cambio.
- Si tras investigar a fondo confirmas que la prueba es correcta y la aplicación falla *y* el usuario confirma el fallo: marca `test.fixme(...)` con comentario a su decisión o incidencia. Nunca omitas silenciosamente.

---

## Referencias cruzadas

| Para... | Consulta |
|---|---|
| `--debug=cli` / conexión | [playwright-tests.md](playwright-tests.md) |
| Simular solicitudes durante exploración/generación | [request-mocking.md](request-mocking.md) |
| Gestionar sesión CLI de navegador | [session-management.md](session-management.md) |
